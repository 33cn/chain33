package tss

import (
	"testing"

	"github.com/getamis/alice/types"
	"github.com/stretchr/testify/require"
)

func TestNewListener(t *testing.T) {
	l := NewListener("/gg18/dkg")
	require.NotNil(t, l)
	require.NotNil(t, l.Done())
}

func TestListener_OnStateChanged_StateDone(t *testing.T) {
	l := NewListener("/gg18/dkg")
	l.(*listener).OnStateChanged(types.StateInit, types.StateDone)
	err := <-l.Done()
	require.NoError(t, err)
}

func TestListener_OnStateChanged_StateFailed(t *testing.T) {
	l := NewListener("/gg18/dkg")
	l.OnStateChanged(types.StateInit, types.StateFailed)
	err := <-l.Done()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Init")
	require.Contains(t, err.Error(), "Failed")
	l.OnStateChanged(types.StateInit, types.StateDone)
	err = <-l.Done()
	require.NoError(t, err)
}

func TestListener_OnStateChanged_OtherState(t *testing.T) {
	l := NewListener("/gg18/dkg")
	// Other state does not send to errCh; Done() would block. So we just ensure no panic.
	l.(*listener).OnStateChanged(types.StateInit, types.StateInit)
	// Channel should be empty; select would block. Use a short timeout or leave as is.
	doneCh := l.Done()
	select {
	case <-doneCh:
		t.Fatal("unexpected value on Done() for non-terminal state")
	default:
		// expected: no send yet
	}
}
