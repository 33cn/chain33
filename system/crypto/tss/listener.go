package tss

import (
	"context"
	"fmt"
	"time"

	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/getamis/alice/types"
)

// Listener state listen
type Listener interface {
	types.StateChangedListener

	Wait() error
}

// NewListener  new listener
func NewListener(protocol string, timeout time.Duration) Listener {
	ctx := cryptocli.GetCryptoContext().Ctx
	if ctx == nil {
		ctx = context.TODO()
	}
	if timeout > 0 {
		ctx, _ = context.WithTimeout(ctx, timeout)
	}
	return &listener{
		protocol: protocol,
		errCh:    make(chan error, 1),
		ctx:      ctx,
	}
}

type listener struct {
	protocol string
	errCh    chan error
	ctx      context.Context
}

func (l *listener) OnStateChanged(oldState types.MainState, newState types.MainState) {
	switch newState {
	case types.StateFailed:
		l.errCh <- fmt.Errorf("state %s -> %s", oldState.String(), newState.String())
	case types.StateDone:
		l.errCh <- nil
	}
}

func (l *listener) Wait() error {

	select {
	case err := <-l.errCh:
		return err
	case <-l.ctx.Done():
		return fmt.Errorf("tss listener context done: %s", l.ctx.Err())
	}
}
