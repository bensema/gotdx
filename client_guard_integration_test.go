// 文件 client_guard_integration_test.go 用本地 mock 行情服务器覆盖 `execute`
// 受保护路径的端到端行为：并发同参请求经防击穿合并只发一次网络请求、熔断打开
// 快速失败、源恢复后自动回归。MAC 模式无握手协议，mock 服务器只需应答业务请求。

package gotdx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensema/gotdx/proto"
)

// mockKLineServer 为本地 mock 行情服务器：读请求帧（12 字节请求头 + 负载）后回
// 空 K 线应答（`Count=0`），并统计收到的 K 线请求数。MAC 模式无握手，无需应答
// 建连握手帧。
type mockKLineServer struct {
	ln         net.Listener
	addr       string
	mu         sync.Mutex
	conns      map[net.Conn]struct{}
	klineCount atomic.Int32
}

// newMockKLineServer 在 127.0.0.1 随机端口启动 mock 服务器。
func newMockKLineServer(t *testing.T) *mockKLineServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("mock listen: %v", err)
	}
	s := &mockKLineServer{ln: ln, addr: ln.Addr().String(), conns: make(map[net.Conn]struct{})}
	go s.serve()
	return s
}

// serve 接受连接并为每条连接开独立读写 goroutine。
func (s *mockKLineServer) serve() {
	for {
		c, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.conns[c] = struct{}{}
		s.mu.Unlock()
		go s.handle(c)
	}
}

// handle 循环读请求帧并回非空 K 线应答，连接关闭或读写失败即退出。请求帧为
// 10 字节请求头（Zip/SeqID/PacketType/PkgLen1/PkgLen2）+ `PkgLen1` 字节负载
// （负载以 2 字节命令码开头，后随业务参数）。应答含 2 根日 K 线：解析器丢弃
// 首根昨收基准条后净返回 1 根，`Count=1` 判为非空、不走空数据换节点。
//
// 注意：不能回 `Count=0` 的空应答——`GetSecurityBars.ParseResponse` 末尾按
// "多要一条昨收基准"执行 `Count-1`，空应答会下溢成 65535（既有缺陷，另报）。
func (s *mockKLineServer) handle(c net.Conn) {
	defer c.Close()
	head := make([]byte, 10)
	for {
		if _, err := io.ReadFull(c, head); err != nil {
			return
		}
		pkgLen := binary.LittleEndian.Uint16(head[6:8])
		body := make([]byte, pkgLen)
		if _, err := io.ReadFull(c, body); err != nil {
			return
		}
		method := binary.LittleEndian.Uint16(body[0:2])
		if method == proto.KMSG_SECURITYBARS {
			s.klineCount.Add(1)
		}
		// 非空应答：16 字节响应头（回显序号与命令码）+ K 线负载。
		var hdr proto.RespHeader
		hdr.SeqID = binary.LittleEndian.Uint32(head[1:5])
		hdr.Method = method
		payload := mockKLinePayload()
		hdr.ZipSize, hdr.UnZipSize = uint16(len(payload)), uint16(len(payload))
		var buf bytes.Buffer
		if err := binary.Write(&buf, binary.LittleEndian, &hdr); err != nil {
			return
		}
		if _, err := buf.Write(payload); err != nil {
			return
		}
		if _, err := c.Write(buf.Bytes()); err != nil {
			return
		}
	}
}

// encodeMockPrice 按 `getprice` 变长整数编码写入价格（实际值 ×1000）。
func encodeMockPrice(v int) []byte {
	first := v & 0x3F
	v >>= 6
	if v > 0 {
		first |= 0x80
	}
	out := []byte{byte(first)}
	for v > 0 {
		b := v & 0x7F
		v >>= 7
		if v > 0 {
			b |= 0x80
		}
		out = append(out, byte(b))
	}
	return out
}

// mockKLinePayload 构造 2 根日 K 线负载：首根为昨收基准（解析时丢弃），
// 第二根为保留的数据条。日线日期按年月日各两位打包，价格按 ×1000 编码。
func mockKLinePayload() []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2)) // Count
	// 条 1（丢弃）：昨日 2026-05-05，价格取 1 字节小值。
	_ = binary.Write(&buf, binary.LittleEndian, uint32(20260505))
	buf.Write(encodeMockPrice(10))
	buf.Write(encodeMockPrice(10))
	buf.Write(encodeMockPrice(10))
	buf.Write(encodeMockPrice(10))
	_ = binary.Write(&buf, binary.LittleEndian, float32(100))
	_ = binary.Write(&buf, binary.LittleEndian, float32(1000))
	// 条 2（保留）：今日 2026-05-06，开 12.000 收 12.050 高 12.100 低 11.990。
	_ = binary.Write(&buf, binary.LittleEndian, uint32(20260506))
	buf.Write(encodeMockPrice(12000))
	buf.Write(encodeMockPrice(12050))
	buf.Write(encodeMockPrice(12100))
	buf.Write(encodeMockPrice(11990))
	_ = binary.Write(&buf, binary.LittleEndian, float32(1000))
	_ = binary.Write(&buf, binary.LittleEndian, float32(5000000))
	return buf.Bytes()
}

// stop 关闭监听并断开全部连接。
func (s *mockKLineServer) stop() {
	_ = s.ln.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.conns {
		_ = c.Close()
	}
}

// restart 在原地址重启监听（用于熔断恢复验证）。
func (s *mockKLineServer) restart(t *testing.T) *mockKLineServer {
	t.Helper()
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		t.Fatalf("mock re-listen: %v", err)
	}
	ns := &mockKLineServer{ln: ln, addr: s.addr, conns: make(map[net.Conn]struct{})}
	go ns.serve()
	return ns
}

// TestGuardMergeRealPath 覆盖防击穿合并的端到端路径：并发同参请求只发一次网络
// 请求；不同参数互不合并。
func TestGuardMergeRealPath(t *testing.T) {
	srv := newMockKLineServer(t)
	defer srv.stop()

	client := NewMAC(
		WithMacTCPAddress(srv.addr),
		WithMacTCPAddressPool(srv.addr),
		WithPoolSize(3),
		WithTimeoutSec(2),
	)
	defer client.Disconnect()
	if err := client.ConnectMAC(); err != nil {
		t.Fatalf("connect: %v", err)
	}

	// 并发 20 个同参请求：防击穿合并为一次网络请求，全部成功且结果一致。
	// 用启动栅栏让各请求同时进入 `execute`，最大化在途重叠；`singleflight`
	// 只合并在途请求，本地 mock 毫秒级响应下断言合并后网络请求数显著少于
	// 并发数（≤3），而非精确 1。
	const n = 20
	var wg sync.WaitGroup
	okCount := atomic.Int32{}
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			rep, err := client.GetKLine(4, 1, "600519", 0, 5, 1, 0)
			if err != nil {
				t.Errorf("same-param request err: %v", err)
				return
			}
			if rep.Count != 1 || len(rep.List) != 1 {
				t.Errorf("expected 1 kept kline, got count %d list %d", rep.Count, len(rep.List))
			}
			okCount.Add(1)
		}()
	}
	close(start)
	wg.Wait()
	if got := srv.klineCount.Load(); got > 3 {
		t.Fatalf("merged requests sent %d network calls, want <= 3", got)
	}
	if okCount.Load() != n {
		t.Fatalf("successful replies %d, want %d", okCount.Load(), n)
	}

	// 并发 20 个不同 start 参数：互不合并，各发一次。
	srv.klineCount.Store(0)
	var wg2 sync.WaitGroup
	for i := 0; i < n; i++ {
		wg2.Add(1)
		go func(i int) {
			defer wg2.Done()
			if _, err := client.GetKLine(4, 1, "600519", uint16(i), 5, 1, 0); err != nil {
				t.Errorf("distinct-param request err: %v", err)
			}
		}(i)
	}
	wg2.Wait()
	if got := srv.klineCount.Load(); got != n {
		t.Fatalf("distinct params sent %d network calls, want %d", got, n)
	}
}

// TestGuardCircuitOpensAndRecovers 覆盖熔断打开与恢复的端到端路径：服务器故障
// 连续失败打开熔断后请求快速失败返回 `ErrCircuitOpen`；服务器恢复并冷却期满后
// 试探请求成功、熔断关闭回归正常。
func TestGuardCircuitOpensAndRecovers(t *testing.T) {
	srv := newMockKLineServer(t)

	client := NewMAC(
		WithMacTCPAddress(srv.addr),
		WithMacTCPAddressPool(srv.addr),
		WithPoolSize(3),
		WithTimeoutSec(1),
		WithGuardConfig(3, 150*time.Millisecond),
	)
	defer client.Disconnect()
	if err := client.ConnectMAC(); err != nil {
		t.Fatalf("connect: %v", err)
	}
	// 服务器正常时一次请求成功。
	if _, err := client.GetKLine(4, 1, "600519", 0, 5, 1, 0); err != nil {
		t.Fatalf("healthy request err: %v", err)
	}

	// 服务器故障：连续失败达到阈值打开熔断，随后请求快速失败。
	srv.stop()
	var gotOpen bool
	for i := 0; i < 10; i++ {
		_, err := client.GetKLine(4, 1, "600519", 0, 5, 1, 0)
		if errors.Is(err, ErrCircuitOpen) {
			gotOpen = true
			break
		}
	}
	if !gotOpen {
		t.Fatal("expected ErrCircuitOpen after repeated failures")
	}

	// 服务器恢复 + 冷却期满：重连后试探请求成功，熔断关闭回归正常。
	srv2 := srv.restart(t)
	defer srv2.stop()
	if err := client.ConnectMAC(); err != nil { // 幂等补建池内连接
		t.Fatalf("reconnect: %v", err)
	}
	time.Sleep(200 * time.Millisecond) // 越过冷却期，进入半开放行试探
	if _, err := client.GetKLine(4, 1, "600519", 0, 5, 1, 0); err != nil {
		t.Fatalf("probe after recovery err: %v", err)
	}
	// 试探成功已关闭熔断：后续请求正常。
	if _, err := client.GetKLine(4, 1, "600519", 0, 5, 1, 0); err != nil {
		t.Fatalf("post-recovery request err: %v", err)
	}
}
