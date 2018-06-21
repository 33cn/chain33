package drivers

//store package store the world - state data
import (
	"strings"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/types"
)

var elog = log.New("module", "execs")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type DriverCreate func() Driver

type driverWithHeight struct {
	create DriverCreate
	height int64
}

var (
	execDrivers        = make(map[string]*driverWithHeight)
	execAddressNameMap = make(map[string]string)
	registedExecDriver = make(map[string]*driverWithHeight)
)

func Register(name string, create DriverCreate, height int64) {
	if create == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := registedExecDriver[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	driverWithHeight := &driverWithHeight{
		create: create,
		height: height,
	}
	registedExecDriver[name] = driverWithHeight
	registerAddress(name)
	execDrivers[ExecAddress(name)] = driverWithHeight
}

func LoadDriver(name string, height int64) (driver Driver, err error) {
	//user.evm 的交易，使用evm执行器
	if strings.HasPrefix(name, "user.evm.") {
		name = "evm"
	}
	c, ok := registedExecDriver[name]
	if !ok {
		return nil, types.ErrUnknowDriver
	}
	if height >= c.height || height == -1 {
		return c.create(), nil
	}
	return nil, types.ErrUnknowDriver
}

func IsDriverAddress(addr string, height int64) bool {
	c, ok := execDrivers[addr]
	if !ok {
		return false
	}
	if height >= c.height || height == -1 {
		return true
	}
	return false
}

func registerAddress(name string) {
	if len(name) == 0 {
		panic("empty name string")
	}
	addr := ExecAddress(name)
	execAddressNameMap[name] = addr
}

func ExecAddress(name string) string {
	if addr, ok := execAddressNameMap[name]; ok {
		return addr
	}
	return address.ExecAddress(name)
}
