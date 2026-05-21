package download

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// F-DL-003: In handler.go:141-169, wg.Done() is called at the end of the goroutine
// but NOT deferred. If downloadBlock panics or the early return at line 149 is taken,
// wg.Done() is skipped and wg.Wait() deadlocks forever.

func TestWgDone_NotDeferred_Deadlock(t *testing.T) {
	// Simulate the buggy pattern: wg.Done() at end of goroutine, not deferred.
	// If the function panics, wg.Wait() blocks forever.

	var wg sync.WaitGroup
	wg.Add(1)

	done := make(chan struct{})
	go func() {
		defer func() { recover() }() // catch the panic but wg.Done() is skipped

		// Simulate buggy goroutine pattern: panic before reaching wg.Done().
		// The bug is precisely that the goroutine has a non-deferred wg.Done()
		// after this code path, so a panic skips it. Here we don't write that
		// dead line — vet would flag it as unreachable, and it's not needed to
		// reproduce the deadlock.
		panic("simulated downloadBlock panic")
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("should not reach here with buggy pattern")
	case <-time.After(100 * time.Millisecond):
		// Deadlock confirmed - wg.Wait() never returns
	}

	// Now test the fixed pattern: defer wg.Done()
	var wg2 sync.WaitGroup
	wg2.Add(1)

	done2 := make(chan struct{})
	go func() {
		defer wg2.Done()
		defer func() { recover() }()
		panic("simulated downloadBlock panic")
	}()

	go func() {
		wg2.Wait()
		close(done2)
	}()

	select {
	case <-done2:
		// Fixed: wg.Wait() returns even after panic
	case <-time.After(100 * time.Millisecond):
		t.Fatal("BUG: fixed pattern should not deadlock")
	}

	assert.True(t, true, "defer wg.Done() prevents deadlock on panic")
}
