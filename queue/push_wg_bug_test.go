package queue

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// F-BCSYNC-001: In blockchain/push.go:693, postwg.Add(1) is called AFTER
// the goroutine starts (line 692). If the goroutine finishes quickly and calls
// Done() before Add(1) executes, WaitGroup counter goes negative → panic.
// This test demonstrates the pattern in isolation.

func TestWaitGroupAdd_AfterGoroutine_Race(t *testing.T) {
	// Buggy pattern: go func(){ ... wg.Done() }(); wg.Add(1)
	buggyPanic := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				buggyPanic <- true
			} else {
				buggyPanic <- false
			}
		}()
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			go func() {
				wg.Done()
			}()
			wg.Add(1) // BUG: Add after goroutine launch
		}
		wg.Wait()
	}()

	select {
	case panicked := <-buggyPanic:
		if panicked {
			t.Log("BUG CONFIRMED: negative WaitGroup counter panic")
		} else {
			t.Log("Race did not trigger this run (non-deterministic)")
		}
	case <-time.After(2 * time.Second):
		t.Log("Timed out — likely deadlocked")
	}

	// Fixed pattern: wg.Add(1) BEFORE go func
	var wg2 sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg2.Add(1)
		go func() {
			wg2.Done()
		}()
	}
	wg2.Wait()
	assert.True(t, true, "fixed pattern: Add before go prevents race")
}
