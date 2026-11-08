package gotdx

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensema/gotdx/proto"
)

// TestMACHeartbeatAcquireRace exercises heartbeatPool concurrently with business
// acquisitions on a 20-connection fake MAC pool. It is a regression test for the
// heartbeat/acquire lock race: heartbeat must remove a connection from the pool
// before checking it, and must never leave inUse corrupted.
func TestMACHeartbeatAcquireRace(t *testing.T) {
	heartbeat := make([]byte, 10)
	info := make([]byte, 87)
	addr, stop := fakeMACHeartbeatServer(t, heartbeat, info)
	defer stop()

	client := NewMAC(
		WithMacTCPAddress(addr),
		WithPoolSize(20),
		WithTimeoutSec(2),
	)
	if err := client.ConnectMAC(); err != nil {
		t.Fatalf("ConnectMAC: %v", err)
	}
	defer client.Disconnect()

	deadline := time.Now().Add(3 * time.Second)
	for {
		client.mu.Lock()
		filled := len(client.pool)
		client.mu.Unlock()
		if filled >= 20 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("pool did not fill, got %d", filled)
		}
		time.Sleep(10 * time.Millisecond)
	}

	var stopWork atomic.Bool
	var wg sync.WaitGroup
	var errs atomic.Int64
	var firstErr atomic.Value

	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stopWork.Load() {
			client.heartbeatPool()
			time.Sleep(time.Millisecond)
		}
	}()

	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; !stopWork.Load(); i++ {
				code := fmt.Sprintf("6%05d", (worker*1000+i)%100000)
				if _, err := client.MACQuotes(uint8(1), code); err != nil {
					errs.Add(1)
					firstErr.CompareAndSwap(nil, err.Error())
				}
			}
		}(worker)
	}

	time.Sleep(2 * time.Second)
	stopWork.Store(true)
	wg.Wait()

	client.mu.Lock()
	inUse := client.inUse
	poolLen := len(client.pool)
	client.mu.Unlock()
	if errs.Load() != 0 {
		first, _ := firstErr.Load().(string)
		t.Fatalf("business requests failed during heartbeat race: %d (first=%s)", errs.Load(), first)
	}
	if inUse != 0 {
		t.Fatalf("inUse corrupted after heartbeat race: %d", inUse)
	}
	if poolLen != 20 {
		t.Fatalf("pool size changed after heartbeat race: got %d, want 20", poolLen)
	}
}

// ensure proto package remains referenced if this test evolves.
var _ = proto.KMSG_MACQUOTES
