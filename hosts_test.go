package gotdx

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
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
		conn, err := ln.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	client := New(
		WithTCPAddress("127.0.0.1:1"),
		WithTCPAddressPool(ln.Addr().String()),
		WithTimeoutSec(1),
	)
	defer client.Disconnect()

	if err := client.connect(); err != nil {
		t.Fatalf("connect failed: %v", err)
	}
	if got := client.CurrentAddress(); got != ln.Addr().String() {
		t.Fatalf("unexpected current address: %q", got)
	}
	<-done
}

type stubConn struct{}

func (stubConn) Read(b []byte) (int, error)       { return 0, io.EOF }
func (stubConn) Write(b []byte) (int, error)      { return len(b), nil }
func (stubConn) Close() error                     { return nil }
func (stubConn) LocalAddr() net.Addr              { return stubAddr("local") }
func (stubConn) RemoteAddr() net.Addr             { return stubAddr("remote") }
func (stubConn) SetDeadline(time.Time) error      { return nil }
func (stubConn) SetReadDeadline(time.Time) error  { return nil }
func (stubConn) SetWriteDeadline(time.Time) error { return nil }

type stubAddr string

func (a stubAddr) Network() string { return "tcp" }
func (a stubAddr) String() string  { return string(a) }
