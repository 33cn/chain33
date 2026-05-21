package queue

import (
	"sync"
	"testing"

	"github.com/33cn/chain33/types"
)

// F-QUE-001: queue.Close() concurrent panic.
// isClosed() check is outside the lock. Two goroutines can both pass the check,
// and the second one will close(q.done) on an already-closed channel → panic.

func TestQueueClose_ConcurrentPanic(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q := New("test-concurrent-close")
	q.SetConfig(cfg)
	go q.Start()

	var wg sync.WaitGroup
	panicked := make(chan struct{}, 10)

	// Launch multiple goroutines calling Close() concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicked <- struct{}{}
				}
			}()
			q.Close()
		}()
	}

	wg.Wait()
	close(panicked)

	count := 0
	for range panicked {
		count++
	}
	if count > 0 {
		t.Fatalf("BUG: queue.Close() panicked %d times under concurrent calls (send/close on closed channel)", count)
	}
}
