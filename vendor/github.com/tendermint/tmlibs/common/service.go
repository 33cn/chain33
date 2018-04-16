package common

import (
	"sync/atomic"

	"github.com/tendermint/tmlibs/log"
)

type Service interface {
	Start() (bool, error)
	OnStart() error

	Stop() bool
	OnStop()

	Reset() (bool, error)
	OnReset() error

	IsRunning() bool

	String() string

	SetLogger(log.Logger)
}

/*
Classical-inheritance-style service declarations. Services can be started, then
stopped, then optionally restarted.

Users can override the OnStart/OnStop methods. In the absence of errors, these
methods are guaranteed to be called at most once. If OnStart returns an error,
service won't be marked as started, so the user can call Start again.

A call to Reset will panic, unless OnReset is overwritten, allowing
OnStart/OnStop to be called again.

The caller must ensure that Start and Stop are not called concurrently.

It is ok to call Stop without calling Start first.

Typical usage:

	type FooService struct {
		BaseService
		// private fields
	}

	func NewFooService() *FooService {
		fs := &FooService{
			// init
		}
		fs.BaseService = *NewBaseService(log, "FooService", fs)
		return fs
	}

	func (fs *FooService) OnStart() error {
		fs.BaseService.OnStart() // Always call the overridden method.
		// initialize private fields
		// start subroutines, etc.
	}

	func (fs *FooService) OnStop() error {
		fs.BaseService.OnStop() // Always call the overridden method.
		// close/destroy private fields
		// stop subroutines, etc.
	}
*/
type BaseService struct {
	Logger  log.Logger
	name    string
	started uint32 // atomic
	stopped uint32 // atomic
	Quit    chan struct{}

	// The "subclass" of BaseService
	impl Service
}

func NewBaseService(logger log.Logger, name string, impl Service) *BaseService {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	return &BaseService{
		Logger: logger,
		name:   name,
		Quit:   make(chan struct{}),
		impl:   impl,
	}
}

func (bs *BaseService) SetLogger(l log.Logger) {
	bs.Logger = l
}

// Implements Servce
func (bs *BaseService) Start() (bool, error) {
	if atomic.CompareAndSwapUint32(&bs.started, 0, 1) {
		if atomic.LoadUint32(&bs.stopped) == 1 {
			bs.Logger.Error(Fmt("Not starting %v -- already stopped", bs.name), "impl", bs.impl)
			return false, nil
		} else {
			bs.Logger.Info(Fmt("Starting %v", bs.name), "impl", bs.impl)
		}
		err := bs.impl.OnStart()
		if err != nil {
			// revert flag
			atomic.StoreUint32(&bs.started, 0)
			return false, err
		}
		return true, err
	} else {
		bs.Logger.Debug(Fmt("Not starting %v -- already started", bs.name), "impl", bs.impl)
		return false, nil
	}
}

// Implements Service
// NOTE: Do not put anything in here,
// that way users don't need to call BaseService.OnStart()
func (bs *BaseService) OnStart() error { return nil }

// Implements Service
func (bs *BaseService) Stop() bool {
	if atomic.CompareAndSwapUint32(&bs.stopped, 0, 1) {
		bs.Logger.Info(Fmt("Stopping %v", bs.name), "impl", bs.impl)
		bs.impl.OnStop()
		close(bs.Quit)
		return true
	} else {
		bs.Logger.Debug(Fmt("Stopping %v (ignoring: already stopped)", bs.name), "impl", bs.impl)
		return false
	}
}

// Implements Service
// NOTE: Do not put anything in here,
// that way users don't need to call BaseService.OnStop()
func (bs *BaseService) OnStop() {}

// Implements Service
func (bs *BaseService) Reset() (bool, error) {
	if !atomic.CompareAndSwapUint32(&bs.stopped, 1, 0) {
		bs.Logger.Debug(Fmt("Can't reset %v. Not stopped", bs.name), "impl", bs.impl)
		return false, nil
	}

	// whether or not we've started, we can reset
	atomic.CompareAndSwapUint32(&bs.started, 1, 0)

	bs.Quit = make(chan struct{})
	return true, bs.impl.OnReset()
}

// Implements Service
func (bs *BaseService) OnReset() error {
	PanicSanity("The service cannot be reset")
	return nil
}

// Implements Service
func (bs *BaseService) IsRunning() bool {
	return atomic.LoadUint32(&bs.started) == 1 && atomic.LoadUint32(&bs.stopped) == 0
}

func (bs *BaseService) Wait() {
	<-bs.Quit
}

// Implements Servce
func (bs *BaseService) String() string {
	return bs.name
}

//----------------------------------------

type QuitService struct {
	BaseService
}

func NewQuitService(logger log.Logger, name string, impl Service) *QuitService {
	if logger != nil {
		logger.Info("QuitService is deprecated, use BaseService instead")
	}
	return &QuitService{
		BaseService: *NewBaseService(logger, name, impl),
	}
}
