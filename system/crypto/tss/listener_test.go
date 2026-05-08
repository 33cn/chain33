package tss

import (
	"testing"
	"time"

	"github.com/getamis/alice/types"
	"github.com/stretchr/testify/require"
)

func TestNewListener(t *testing.T) {
	l := NewListener("/gg18/dkg", time.Second)
	require.NotNil(t, l)
}

func TestListener_Wait_StateDone(t *testing.T) {
	l := NewListener("/gg18/dkg", time.Second)
	l.(*listener).OnStateChanged(types.StateInit, types.StateDone)
	require.NoError(t, l.Wait())
}

func TestListener_Wait_StateFailed(t *testing.T) {
	l := NewListener("/gg18/dkg", time.Second)
	l.OnStateChanged(types.StateInit, types.StateFailed)
	err := l.Wait()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Init")
	require.Contains(t, err.Error(), "Failed")
}

func TestListener_Wait_Timeout(t *testing.T) {
	l := NewListener("/gg18/dkg", 10*time.Millisecond)
	l.(*listener).OnStateChanged(types.StateInit, types.StateInit)
	err := l.Wait()
	require.Error(t, err)
	require.Contains(t, err.Error(), "context deadline exceeded")
}
