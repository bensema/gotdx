package gotdx

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/bensema/gotdx/proto"
)

func TestDefaultOptionsUseBuiltInHostPools(t *testing.T) {
	opt := defaultOptions()

	mainHosts := MainHostAddresses()
	exHosts := ExHostAddresses()
	macHosts := MACHostAddresses()
	macExHosts := MACExHostAddresses()

	if opt.TCPAddress != mainHosts[0] || len(opt.TCPAddressPool) != len(mainHosts)-1 {
		t.Fatalf("unexpected main defaults: primary=%q pool=%d", opt.TCPAddress, len(opt.TCPAddressPool))
	}
	if opt.ExTCPAddress != exHosts[0] || len(opt.ExTCPAddressPool) != len(exHosts)-1 {
		t.Fatalf("unexpected ex defaults: primary=%q pool=%d", opt.ExTCPAddress, len(opt.ExTCPAddressPool))
	}
	if opt.MacTCPAddress != macHosts[0] || len(opt.MacTCPAddressPool) != len(macHosts)-1 {
		t.Fatalf("unexpected mac defaults: primary=%q pool=%d", opt.MacTCPAddress, len(opt.MacTCPAddressPool))
	}
	if opt.MacExTCPAddress != macExHosts[0] || len(opt.MacExTCPAddressPool) != len(macExHosts)-1 {
		t.Fatalf("unexpected mac ex defaults: primary=%q pool=%d", opt.MacExTCPAddress, len(opt.MacExTCPAddressPool))
	}
}

func TestProbeAddressesWithDialSortsByLatency(t *testing.T) {
	addresses := []string{
		"127.0.0.1:7001",
		"127.0.0.1:7002",
		"127.0.0.1:7003",
	}

	// 拨号闭包模拟三个节点：前两个可达且耗时不同，第三个拨号失败。
	results := probeAddressesWithDial(addresses, 50*time.Millisecond, func(network, address string, timeout time.Duration) (net.Conn, error) {
		switch address {
		case addresses[0]:
			time.Sleep(20 * time.Millisecond)
			return stubConn{}, nil
		case addresses[1]:
			time.Sleep(5 * time.Millisecond)
			return stubConn{}, nil
		default:
			return nil, errors.New("dial failed")
		}
	})

	if len(results) != len(addresses) {
		t.Fatalf("unexpected result len: %d", len(results))
	}
	if results[0].Address != addresses[1] || !results[0].Reachable {
		t.Fatalf("expected fastest reachable first, got %+v", results[0])
	}
	if results[1].Address != addresses[0] || !results[1].Reachable {
		t.Fatalf("expected slower reachable second, got %+v", results[1])
	}
	if results[2].Address != addresses[2] || results[2].Reachable {
		t.Fatalf("expected failed address last, got %+v", results[2])
	}
}

func TestConnectUsesAddressPoolFallback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			// 每条连接读取请求头后回主站握手应答，保证测速探测与 `Connect` 建池握手都能成功。
			go func(conn net.Conn) {
				defer conn.Close()
				header := make([]byte, 12)
				if _, err := io.ReadFull(conn, header); err != nil {
					return
				}
				body := fakeHello1Body()
				_, _ = conn.Write(append(fakeRespHeader(0, proto.KMSG_CMD1, uint16(len(body))), body...))
			}(conn)
		}
	}()

	// 首选地址不可达，客户端应回退到地址池中的本地监听地址。
	client := New(
		WithTCPAddress("127.0.0.1:1"),
		WithTCPAddressPool(ln.Addr().String()),
		WithTimeoutSec(1),
		WithPoolSize(1),
	)
	defer client.Disconnect()

	if _, err := client.Connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if got := client.CurrentAddress(); got != ln.Addr().String() {
		t.Fatalf("unexpected current address: %q", got)
	}
}

// stubConn 是不进行真实网络读写的 `net.Conn` 桩，供拨号探测测试替换真实连接。
// `Read` 恒返回 `io.EOF`，`Write` 原样返回写入长度，关闭与三类超时设置均为空操作；
// 本地与远端地址分别返回固定的 `stubAddr` 桩。
type stubConn struct{}

func (stubConn) Read(b []byte) (int, error)       { return 0, io.EOF }
func (stubConn) Write(b []byte) (int, error)      { return len(b), nil }
func (stubConn) Close() error                     { return nil }
func (stubConn) LocalAddr() net.Addr              { return stubAddr("local") }
func (stubConn) RemoteAddr() net.Addr             { return stubAddr("remote") }
func (stubConn) SetDeadline(time.Time) error      { return nil }
func (stubConn) SetReadDeadline(time.Time) error  { return nil }
func (stubConn) SetWriteDeadline(time.Time) error { return nil }

// stubAddr 是 `net.Addr` 的最小桩实现；`Network` 恒返回 `tcp`，`String` 返回底层文本。
type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }
