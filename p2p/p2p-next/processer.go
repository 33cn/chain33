package p2p_next

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/queue"
	net "github.com/libp2p/go-libp2p-net"
)

type Driver interface {
	New(node *Node, cli queue.Client, done chan struct{}) Driver
	OnResp(s net.Stream)
	OnReq(s net.Stream)
	DoProcess(msg *queue.Message)
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Driver)
)

//对不同币种的Sender进行注册
func Register(name string, driver Driver) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("transformer: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("transformer: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

//提供名称返回相应的Sender对象
func NewDriver(name string) (t Driver, err error) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	t, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown Coins %q", name)
		return
	}

	return t, nil
}
