package drivers

//store package store the world - state data
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/account"
	clog "code.aliyun.com/chain33/chain33/common/log"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type ExecDrivers struct {
	ExecName2Driver map[string]Driver //from name to driver
	ExecAddr2Name   map[string]string //from address to name
}

type driverWithHeight struct {
	driver Driver
	height int64
}

var (
	//execDrivers        = make(map[string]Driver)
	//execAddress        = make(map[string]string)
	execAddressNameMap = make(map[string]string)
	registedExecDriver = make(map[string]*driverWithHeight)
)

func Register(name string, driver Driver, height int64) {
	if driver == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := registedExecDriver[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	driverWithHeight := &driverWithHeight{
		driver,
		height,
	}
	registedExecDriver[name] = driverWithHeight
	registerAddress(name)
}

func CreateDrivers4CurrentHeight(height int64) *ExecDrivers {
	var drivers ExecDrivers
	drivers.ExecName2Driver = make(map[string]Driver)
	drivers.ExecAddr2Name = make(map[string]string)
	if len(registedExecDriver) > 0 {
		for name, driverInfo := range registedExecDriver {
			//height -> -1 only for test
			if height >= driverInfo.height || height < 0 {
				drivers.ExecName2Driver[name] = driverInfo.driver
				drivers.ExecAddr2Name[ExecAddress(name)] = name
			}
		}
	}
	return &drivers
}
func (execDrivers *ExecDrivers) LoadDriver(name string) (c Driver, err error) {
	c, ok := execDrivers.ExecName2Driver[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}
	return c, nil
}

func (execDrivers *ExecDrivers) IsDriverAddress(addr string) bool {
	_, ok := execDrivers.ExecAddr2Name[addr]
	return ok
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
	return account.ExecAddress(name).String()
}
