package common

import (
	"sync"
	"time"
)

// Used by RepeatTimer the first time,
// and every time it's Reset() after Stop().
type TickerMaker func(dur time.Duration) Ticker

// Ticker is a basic ticker interface.
type Ticker interface {

	// Never changes, never closes.
	Chan() <-chan time.Time

	// Stopping a stopped Ticker will panic.
	Stop()
}

//----------------------------------------
// defaultTickerMaker

func defaultTickerMaker(dur time.Duration) Ticker {
	ticker := time.NewTicker(dur)
	return (*defaultTicker)(ticker)
}

type defaultTicker time.Ticker

// Implements Ticker
func (t *defaultTicker) Chan() <-chan time.Time {
	return t.C
}

// Implements Ticker
func (t *defaultTicker) Stop() {
	((*time.Ticker)(t)).Stop()
}

//---------------------------------------------------------------------

/*
	RepeatTimer repeatedly sends a struct{}{} to `.Chan()` after each `dur`
	period. (It's good for keeping connections alive.)
	A RepeatTimer must be stopped, or it will keep a goroutine alive.
*/
type RepeatTimer struct {
	name string
	ch   chan time.Time
	tm   TickerMaker

	mtx    sync.Mutex
	dur    time.Duration
	ticker Ticker
	quit   chan struct{}
}

// NewRepeatTimer returns a RepeatTimer with a defaultTicker.
func NewRepeatTimer(name string, dur time.Duration) *RepeatTimer {
	return NewRepeatTimerWithTickerMaker(name, dur, defaultTickerMaker)
}

// NewRepeatTimerWithTicker returns a RepeatTimer with the given ticker
// maker.
func NewRepeatTimerWithTickerMaker(name string, dur time.Duration, tm TickerMaker) *RepeatTimer {
	var t = &RepeatTimer{
		name:   name,
		ch:     make(chan time.Time),
		tm:     tm,
		dur:    dur,
		ticker: nil,
		quit:   nil,
	}
	t.reset()
	return t
}

func (t *RepeatTimer) fireRoutine(ch <-chan time.Time, quit <-chan struct{}) {
	for {
		select {
		case t_ := <-ch:
			t.ch <- t_
		case <-quit: // NOTE: `t.quit` races.
			return
		}
	}
}

func (t *RepeatTimer) Chan() <-chan time.Time {
	return t.ch
}

func (t *RepeatTimer) Stop() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	t.stop()
}

// Wait the duration again before firing.
func (t *RepeatTimer) Reset() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	t.reset()
}

//----------------------------------------
// Misc.

// CONTRACT: (non-constructor) caller should hold t.mtx.
func (t *RepeatTimer) reset() {
	if t.ticker != nil {
		t.stop()
	}
	t.ticker = t.tm(t.dur)
	t.quit = make(chan struct{})
	go t.fireRoutine(t.ticker.Chan(), t.quit)
}

// CONTRACT: caller should hold t.mtx.
func (t *RepeatTimer) stop() {
	if t.ticker == nil {
		/*
			Similar to the case of closing channels twice:
			https://groups.google.com/forum/#!topic/golang-nuts/rhxMiNmRAPk
			Stopping a RepeatTimer twice implies that you do
			not know whether you are done or not.
			If you're calling stop on a stopped RepeatTimer,
			you probably have race conditions.
		*/
		panic("Tried to stop a stopped RepeatTimer")
	}
	t.ticker.Stop()
	t.ticker = nil
	/*
		XXX
		From https://golang.org/pkg/time/#Ticker:
		"Stop the ticker to release associated resources"
		"After Stop, no more ticks will be sent"
		So we shouldn't have to do the below.

		select {
		case <-t.ch:
			// read off channel if there's anything there
		default:
		}
	*/
	close(t.quit)
}
