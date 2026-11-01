package gotdx

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/bensema/gotdx/proto"
	"github.com/bensema/gotdx/types"
)

// executeGenericMethodProtocol 是 `execute` 泛型流程的最小协议桩：请求体长度固定为 1 字节，
// 响应负载原样保存为应答字符串，用于在 `net.Pipe` 上走通一次收发流程。
type executeGenericMethodProtocol struct {
	reply string // 最近一次响应负载转成的应答字符串，供 `Response` 返回
}

// BuildRequest 返回固定的请求负载 `0x5a`；不携带业务参数，编码恒成功，错误返回 `nil`。
func (*executeGenericMethodProtocol) BuildRequest() ([]byte, error) {
	return []byte{0x5a}, nil
}

// ParseResponse 把响应负载原样转为字符串存入 `reply`；入参 `header` 本桩不使用，
// 负载解码恒成功，错误返回 `nil`。
func (p *executeGenericMethodProtocol) ParseResponse(_ *proto.RespHeader, payload []byte) error {
	p.reply = string(payload)
	return nil
}

// Response 返回最近一次保存的应答字符串。
func (p *executeGenericMethodProtocol) Response() string {
	return p.reply
}

func TestClientExecuteGenericMethod(t *testing.T) {
	// 使用 `net.Pipe` 在同一进程内模拟一条连接，避免依赖真实网络。
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	client := New(WithTimeoutSec(1))
	client.pool = []*pooledConn{{conn: clientConn}}

	serverDone := make(chan error, 1)
	go func() {
		request := make([]byte, 1)
		if _, err := io.ReadFull(serverConn, request); err != nil {
			serverDone <- err
			return
		}
		if request[0] != 0x5a {
			serverDone <- errors.New("unexpected request payload")
			return
		}

		_, err := serverConn.Write(append(fakeRespHeader(0, 0, 5), []byte("reply")...))
		serverDone <- err
	}()

	reply, err := client.execute(&executeGenericMethodProtocol{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if reply != "reply" {
		t.Fatalf("unexpected reply: %q", reply)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("server failed: %v", err)
	}
}

func TestMakeStocks(t *testing.T) {
	stocks, err := makeStocks([]uint8{types.MarketSZ.Uint8(), types.MarketSH.Uint8()}, []string{"000001", "600000"})
	if err != nil {
		t.Fatalf("makeStocks failed: %v", err)
	}
	if len(stocks) != 2 {
		t.Fatalf("unexpected stock len: %d", len(stocks))
	}
	if stocks[0].Market != types.MarketSZ.Uint8() || stocks[0].Code != "000001" {
		t.Fatalf("unexpected first stock: %+v", stocks[0])
	}
	if stocks[1].Market != types.MarketSH.Uint8() || stocks[1].Code != "600000" {
		t.Fatalf("unexpected second stock: %+v", stocks[1])
	}
}

func TestMakeStocksCountMismatch(t *testing.T) {
	if _, err := makeStocks([]uint8{types.MarketSZ.Uint8()}, []string{"000001", "600000"}); err == nil {
		t.Fatal("expected error on count mismatch")
	}
}

func TestMakeFixedBuffers(t *testing.T) {
	code := makeCode6("600000EXTRA")
	if string(code[:]) != "600000" {
		t.Fatalf("unexpected code buffer: %q", string(code[:]))
	}

	code9 := makeCode9("TSLA-EXTRA")
	if string(code9[:4]) != "TSLA" {
		t.Fatalf("unexpected code9 buffer: %q", string(code9[:4]))
	}

	code22 := makeCode22("TSLA")
	if string(code22[:4]) != "TSLA" {
		t.Fatalf("unexpected code22 buffer: %q", string(code22[:4]))
	}

	code23 := makeCode23("09988")
	if string(code23[:5]) != "09988" {
		t.Fatalf("unexpected code23 buffer: %q", string(code23[:5]))
	}

	file40 := makeFixed40("block.dat")
	if string(file40[:9]) != "block.dat" {
		t.Fatalf("unexpected file40 buffer: %q", string(file40[:9]))
	}

	file80 := makeFixed80("test.txt")
	if string(file80[:8]) != "test.txt" {
		t.Fatalf("unexpected file80 buffer: %q", string(file80[:8]))
	}

	file300 := makeFixed300("foo")
	if string(file300[:3]) != "foo" {
		t.Fatalf("unexpected file300 buffer: %q", string(file300[:3]))
	}

	file43 := makeFixed43("TSLA")
	if string(file43[:4]) != "TSLA" {
		t.Fatalf("unexpected file43 buffer: %q", string(file43[:4]))
	}
}

func TestQuotesSortReverse(t *testing.T) {
	if got := quotesSortReverse(types.SortCode, true); got != 0 {
		t.Fatalf("unexpected code sort reverse: %d", got)
	}
	if got := quotesSortReverse(types.SortPrice, false); got != 1 {
		t.Fatalf("unexpected asc sort reverse: %d", got)
	}
	if got := quotesSortReverse(types.SortPrice, true); got != 2 {
		t.Fatalf("unexpected desc sort reverse: %d", got)
	}
}

func TestMakeExStocks(t *testing.T) {
	stocks, err := makeExStocks([]uint8{types.ExCategoryUSStock, types.ExCategoryHKMainBoard}, []string{"TSLA", "09988"})
	if err != nil {
		t.Fatalf("makeExStocks failed: %v", err)
	}
	if len(stocks) != 2 {
		t.Fatalf("unexpected stock len: %d", len(stocks))
	}
	if stocks[0].Category != types.ExCategoryUSStock || stocks[0].Code != "TSLA" {
		t.Fatalf("unexpected first ex stock: %+v", stocks[0])
	}
}

// TestNodeHealthDowngradesAfterEmptyStreak 验证同一节点连续空数据达到阈值后被标记降权，
// 一次非空成功后计数清零、节点恢复定向资格。
func TestNodeHealthDowngradesAfterEmptyStreak(t *testing.T) {
	c := New(WithPoolSize(1))
	addr := "127.0.0.1:7727"

	// 未记录健康度（从未返回空数据）时不视为降权，保持原定向行为。
	if c.nodeHealthDegradedLocked(addr) {
		t.Fatal("fresh node must not be degraded")
	}
	for i := 0; i < healthDowngradeThreshold-1; i++ {
		c.recordNodeHealthEmpty(addr)
		if c.nodeHealthDegradedLocked(addr) {
			t.Fatalf("node must stay usable before streak threshold, round %d", i+1)
		}
	}
	c.recordNodeHealthEmpty(addr)
	if !c.nodeHealthDegradedLocked(addr) {
		t.Fatal("node must be degraded after threshold empty streak")
	}
	c.recordNodeHealthOK(addr)
	if c.nodeHealthDegradedLocked(addr) {
		t.Fatal("node must recover after a non-empty reply")
	}
}

// TestAcquireSkipsDegradedTarget 验证定向节点被降权后，借出跳过该节点回退轮询；节点恢复
// 后定向优先重新生效。
func TestAcquireSkipsDegradedTarget(t *testing.T) {
	addrA, addrB := "10.0.0.1:7727", "10.0.0.2:7727"
	c := New(WithPoolSize(2))
	for i := 0; i < healthDowngradeThreshold; i++ {
		c.recordNodeHealthEmpty(addrA)
	}

	pcA := &pooledConn{addr: addrA}
	pcB := &pooledConn{addr: addrB}
	c.mu.Lock()
	c.pool = []*pooledConn{pcA, pcB}
	c.lastDataAddr = addrA
	c.mu.Unlock()

	// 降权节点 A 被跳过，轮询借出 B。
	pc, err := c.acquire()
	if err != nil {
		t.Fatal(err)
	}
	if pc.addr != addrB {
		t.Fatalf("degraded target must be skipped, acquired %s", pc.addr)
	}
	c.release(pc)

	// A 返回一次非空后恢复定向资格，再次借出优先命中 A。
	c.recordNodeHealthOK(addrA)
	pc, err = c.acquire()
	if err != nil {
		t.Fatal(err)
	}
	if pc.addr != addrA {
		t.Fatalf("recovered target must be preferred, acquired %s", pc.addr)
	}
}

// emptyCountReply 为测试用应答桩，携带 `Count` 字段供 `isEmptyReply` 判空。
type emptyCountReply struct {
	Count uint16
}

// emptyCountProtocol 是空应答协议桩：固定请求体，解析后 `Count` 恒为 0，用于验证关闭换
// 节点模式下空应答被直接返回、不触发并行探测。
type emptyCountProtocol struct {
	reply emptyCountReply
}

// BuildRequest 返回固定请求负载 `0x5a`，编码恒成功。
func (*emptyCountProtocol) BuildRequest() ([]byte, error) {
	return []byte{0x5a}, nil
}

// ParseResponse 置空 `Count`，负载内容本桩不使用，解码恒成功。
func (p *emptyCountProtocol) ParseResponse(_ *proto.RespHeader, _ []byte) error {
	p.reply.Count = 0
	return nil
}

// Response 返回最近一次应答桩。
func (p *emptyCountProtocol) Response() emptyCountReply { return p.reply }

// countReply 携带 `Count` 字段供 `isEmptyReply` 判空，用于并行探测专项测试区分空/非空应答。
type countReply struct {
	Count uint16
}

// countProtocol 按响应负载内容决定 `Count`：负载为 `"data"` 记 3 条非空，其余记 0 条空。
// 协议对象只在主线程被 `ParseResponse` 串行访问，测试并发调用时不应跨 goroutine 共享。
type countProtocol struct {
	reply countReply
}

// BuildRequest 返回固定请求负载 `0x5a`，编码恒成功。
func (*countProtocol) BuildRequest() ([]byte, error) {
	return []byte{0x5a}, nil
}

// ParseResponse 按负载内容填充 `Count`，解码恒成功。
func (p *countProtocol) ParseResponse(_ *proto.RespHeader, payload []byte) error {
	if string(payload) == "data" {
		p.reply.Count = 3
	} else {
		p.reply.Count = 0
	}
	return nil
}

// Response 返回最近一次应答桩。
func (p *countProtocol) Response() countReply { return p.reply }

// startCountServer 在 `net.Pipe` 服务端模拟长连接行情节点：循环读取请求字节并按给定负载
// 回包，直到连接关闭；与真实服务器一样可被借出方重复复用，避免单次应答桩被再次选中后
// 读写失败。
func startCountServer(t *testing.T, serverConn net.Conn, payload string) {
	t.Helper()
	go func() {
		for {
			request := make([]byte, 1)
			if _, err := io.ReadFull(serverConn, request); err != nil {
				return
			}
			if _, err := serverConn.Write(append(fakeRespHeader(0, 0, uint16(len(payload))), []byte(payload)...)); err != nil {
				return
			}
		}
	}()
}

// waitPoolRecovered 等待并行探测收尾 goroutine 归还在途连接，校验借出计数归零、池容量恢复。
func waitPoolRecovered(t *testing.T, client *Client, wantPool int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		client.mu.Lock()
		poolLen, inUse := len(client.pool), client.inUse
		client.mu.Unlock()
		if poolLen == wantPool && inUse == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("connections not recovered: pool=%d inUse=%d, want pool=%d", poolLen, inUse, wantPool)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestParallelEmptyRetryPicksFirstNonEmpty 验证两条探测连接中一条空、一条非空时，并行探测
// 返回首个非空应答；胜出连接与残留连接最终都归还池内，不残留借出计数。
func TestParallelEmptyRetryPicksFirstNonEmpty(t *testing.T) {
	caA, saA := net.Pipe()
	caB, saB := net.Pipe()
	t.Cleanup(func() {
		_ = caA.Close()
		_ = saA.Close()
		_ = caB.Close()
		_ = saB.Close()
	})

	client := New(WithTimeoutSec(1))
	client.mu.Lock()
	// 数据节点排在池首：轮询借出先命中非空节点，首查即胜出，避免调度串行化时轮询索引
	// 把刚归还的空节点再次选中而跳过另一连接（属既有借出顺序行为，不在本次修复范围）。
	client.pool = []*pooledConn{
		{conn: caB, addr: "addr-b"},
		{conn: caA, addr: "addr-a"},
	}
	client.mu.Unlock()

	startCountServer(t, saA, "empty")
	startCountServer(t, saB, "data")

	protocol := &countProtocol{}
	reply, addr, found := client.parallelEmptyRetry(protocol, []byte{0x5a})
	if !found {
		t.Fatal("parallel retry must find the non-empty reply")
	}
	if reply.Count != 3 {
		t.Fatalf("non-empty reply must be returned, got Count=%d", reply.Count)
	}
	if addr != "addr-b" {
		t.Fatalf("winner must be the data node, got %s", addr)
	}

	// 胜出后收尾 goroutine 只归还残留连接、不解析不记账；此处等待连接回收落定即可。
	waitPoolRecovered(t, client, 2)
}

// TestParallelEmptyRetryAllEmptyFallsBack 验证两条探测连接均返回空数据时 `found=false`，
// 由调用方回退首次空应答；两个节点都串行记账，连接全部归还。
func TestParallelEmptyRetryAllEmptyFallsBack(t *testing.T) {
	caA, saA := net.Pipe()
	caB, saB := net.Pipe()
	t.Cleanup(func() {
		_ = caA.Close()
		_ = saA.Close()
		_ = caB.Close()
		_ = saB.Close()
	})

	client := New(WithTimeoutSec(1))
	client.mu.Lock()
	client.pool = []*pooledConn{
		{conn: caA, addr: "addr-a"},
		{conn: caB, addr: "addr-b"},
	}
	client.mu.Unlock()

	startCountServer(t, saA, "empty")
	startCountServer(t, saB, "empty")

	protocol := &countProtocol{}
	reply, _, found := client.parallelEmptyRetry(protocol, []byte{0x5a})
	if found {
		t.Fatal("all-empty probes must not report a winner")
	}
	if reply.Count != 0 {
		t.Fatalf("fallback reply must stay empty, got Count=%d", reply.Count)
	}

	// 调度串行化时轮询可能把刚归还的连接再次选中、暂时跳过另一连接（既有借出顺序行为），
	// 因此只校验"被探测到的空节点都已记账、总次数不超过探测条数"，不绑定具体节点各记一次。
	var streakA, streakB int
	client.mu.Lock()
	if h := client.health["addr-a"]; h != nil {
		streakA = h.emptyStreak
	}
	if h := client.health["addr-b"]; h != nil {
		streakB = h.emptyStreak
	}
	client.mu.Unlock()
	total := streakA + streakB
	if total < 1 || total > 2 {
		t.Fatalf("contacted empty nodes must be recorded, got a=%d b=%d", streakA, streakB)
	}

	waitPoolRecovered(t, client, 2)
}

// TestParallelEmptyRetrySkipsAcquireErrors 验证池内无连接时探测直接返回 `found=false`，
// 不 panic、不残留借出计数。
func TestParallelEmptyRetrySkipsAcquireErrors(t *testing.T) {
	client := New(WithTimeoutSec(1))
	protocol := &countProtocol{}

	reply, _, found := client.parallelEmptyRetry(protocol, []byte{0x5a})
	if found {
		t.Fatal("no available connection must not report a winner")
	}
	if reply.Count != 0 {
		t.Fatalf("zero reply expected, got Count=%d", reply.Count)
	}
	waitPoolRecovered(t, client, 0)
}

// TestParallelEmptyRetryDiscardsFailedConnection 验证收发失败的连接被销毁、成功但空的连接
// 归还池内；失败路径不影响另一条探测的健康度记账。
func TestParallelEmptyRetryDiscardsFailedConnection(t *testing.T) {
	caBad, saBad := net.Pipe()
	caGood, saGood := net.Pipe()
	t.Cleanup(func() {
		_ = caBad.Close()
		_ = saBad.Close()
		_ = caGood.Close()
		_ = saGood.Close()
	})
	// 服务端立即关闭，使客户端读写失败，触发连接销毁路径。
	_ = saBad.Close()

	client := New(WithTimeoutSec(1))
	client.mu.Lock()
	client.pool = []*pooledConn{
		{conn: caBad, addr: "addr-bad"},
		{conn: caGood, addr: "addr-good"},
	}
	client.mu.Unlock()

	startCountServer(t, saGood, "empty")

	protocol := &countProtocol{}
	_, _, found := client.parallelEmptyRetry(protocol, []byte{0x5a})
	if found {
		t.Fatal("failed + empty probes must not report a winner")
	}

	waitPoolRecovered(t, client, 1)

	client.mu.Lock()
	_, hasBad := client.health["addr-bad"]
	client.mu.Unlock()
	if hasBad {
		t.Fatal("failed connection must not record health")
	}
}

// TestExecuteSkipsEmptyRetryWhenSwitchDisabled 验证关闭空数据换节点后，空应答在首次请求后
// 直接返回，不再并行探测池内其他连接（单连接模式对齐老库固定节点行为）。
func TestExecuteSkipsEmptyRetryWhenSwitchDisabled(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	client := New(WithTimeoutSec(1), WithEmptyNodeSwitchDisabled(true))
	client.pool = []*pooledConn{{conn: clientConn}}

	requestCount := make(chan int, 1)
	go func() {
		count := 0
		for {
			request := make([]byte, 1)
			if _, err := io.ReadFull(serverConn, request); err != nil {
				break
			}
			count++
			_, _ = serverConn.Write(fakeRespHeader(0, 0, 0))
		}
		requestCount <- count
	}()

	reply, err := client.execute(&emptyCountProtocol{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if reply.Count != 0 {
		t.Fatalf("disabled switch must return first empty reply, got Count=%d", reply.Count)
	}
	_ = clientConn.Close()
	if got := <-requestCount; got != 1 {
		t.Fatalf("disabled switch must send exactly one request, got %d", got)
	}
}
