package tss

import (
	"fmt"

	"github.com/getamis/alice/types"
)

// Listener state listen
type Listener interface {
	types.StateChangedListener

	Done() <-chan error
}

// NewListener  new listener
func NewListener(protocol string) Listener {
	return &listener{
		protocol: protocol,
		errCh:    make(chan error, 1),
	}
}

type listener struct {
	protocol string
	errCh    chan error
}

func (l *listener) OnStateChanged(oldState types.MainState, newState types.MainState) {
	if newState == types.StateFailed {
		l.errCh <- fmt.Errorf("state %s -> %s", oldState.String(), newState.String())
		return
	} else if newState == types.StateDone {
		l.errCh <- nil
		return
	}

	log.Debug("State changed", "protocol", l.protocol,
		"old", oldState.String(), "new", newState.String())
}

func (l *listener) Done() <-chan error {
	return l.errCh
}
