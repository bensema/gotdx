package gotdx

import (
	"encoding/binary"
	"io"
	"net"
	"testing"

	"github.com/bensema/gotdx/proto"
)

// fakeMACHeartbeatServer 模拟 MAC 服务端：按请求 method 返回可控心跳响应。
// heartbeatBody 用于 KMSG_HEARTBEAT，infoBody 用于 KMSG_MACSERVERINFO；
// 传 nil 时返回空负载，用于模拟不支持对应协议。
func fakeMACHeartbeatServer(t *testing.T, heartbeatBody, infoBody []byte) (addr string, stop func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				for {
					header := make([]byte, 12)
					if _, err := io.ReadFull(conn, header); err != nil {
						return
					}
					seq := binary.LittleEndian.Uint32(header[1:5])
					pkgLen := binary.LittleEndian.Uint16(header[8:10])
					method := binary.LittleEndian.Uint16(header[10:12])
					payloadLen := 0
					if pkgLen > 2 {
						payloadLen = int(pkgLen - 2)
					}
					if _, err := io.ReadFull(conn, make([]byte, payloadLen)); err != nil {
						return
					}
					body := []byte(nil)
					switch method {
					case proto.KMSG_HEARTBEAT:
						body = heartbeatBody
					case proto.KMSG_MACSERVERINFO:
						body = infoBody
					case proto.KMSG_MACQUOTES:
						// 35 字节头部 + 1 个分时点 + 120 字节 summary。
						body = make([]byte, 173)
						binary.LittleEndian.PutUint32(body[24:28], 20261108)
						binary.LittleEndian.PutUint32(body[29:33], 0x41200000) // 10.0f
						body[33] = 1                                           // 1 个分时点
					}
					if _, err := conn.Write(append(fakeRespHeader(seq, method, uint16(len(body))), body...)); err != nil {
						return
					}
				}
			}(conn)
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}

func newMACHeartbeatTestClient(t *testing.T, addr string) *Client {
	t.Helper()
	client := NewMAC(
		WithMacTCPAddress(addr),
		WithPoolSize(1),
		WithTimeoutSec(2),
	)
	if err := client.ConnectMAC(); err != nil {
		t.Fatalf("ConnectMAC: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect() })
	return client
}

func TestMACHeartbeatKeepsConnection(t *testing.T) {
	heartbeat := make([]byte, 10)
	binary.LittleEndian.PutUint32(heartbeat[6:10], 20261108)
	info := make([]byte, 87)
	addr, stop := fakeMACHeartbeatServer(t, heartbeat, info)
	defer stop()

	client := newMACHeartbeatTestClient(t, addr)
	client.heartbeatPool()

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pool) != 1 {
		t.Fatalf("healthy MAC connection should stay in pool, got %d", len(client.pool))
	}
	if client.macHeartbeatDisabled {
		t.Fatal("heartbeat should not be disabled when KMSG_HEARTBEAT succeeds")
	}
}

func TestMACHeartbeatFallsBackToServerInfo(t *testing.T) {
	// HeartBeat 返回非法短负载，MACServerInfo 返回合法负载：
	// 连接应继续保留，且 MAC 心跳不应被禁用。
	info := make([]byte, 87)
	addr, stop := fakeMACHeartbeatServer(t, []byte{1}, info)
	defer stop()

	client := newMACHeartbeatTestClient(t, addr)
	client.heartbeatPool()

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pool) != 1 {
		t.Fatalf("MAC connection should survive MACServerInfo fallback, got %d", len(client.pool))
	}
	if client.macHeartbeatDisabled {
		t.Fatal("heartbeat should remain enabled after MACServerInfo fallback")
	}
}

func TestMACHeartbeatFallsBackToTCPKeepalive(t *testing.T) {
	// 两种协议都返回非法负载：应禁用 MAC 应用层心跳，但保留连接，交给 TCP keepalive。
	addr, stop := fakeMACHeartbeatServer(t, []byte{1}, []byte{1})
	defer stop()

	client := newMACHeartbeatTestClient(t, addr)
	client.heartbeatPool()

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pool) != 1 {
		t.Fatalf("connection should stay in pool after heartbeat fallback, got %d", len(client.pool))
	}
	if !client.macHeartbeatDisabled {
		t.Fatal("MAC application heartbeat should be disabled after both protocols fail")
	}
}

func TestMACHeartbeatDiscardsClosedConnection(t *testing.T) {
	heartbeat := make([]byte, 10)
	binary.LittleEndian.PutUint32(heartbeat[6:10], 20261108)
	addr, stop := fakeMACHeartbeatServer(t, heartbeat, make([]byte, 87))
	defer stop()

	client := newMACHeartbeatTestClient(t, addr)
	client.mu.Lock()
	pc := client.pool[0]
	client.mu.Unlock()
	_ = pc.conn.Close()

	client.heartbeatPool()

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.pool) != 0 {
		t.Fatalf("closed MAC connection should be discarded, got pool=%d", len(client.pool))
	}
}
