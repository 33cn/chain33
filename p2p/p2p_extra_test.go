package p2p

import (
	"testing"
)

func TestManagerWait(t *testing.T) {
	mgr := &Manager{}
	mgr.Wait() // no-op, just verify no panic
}
