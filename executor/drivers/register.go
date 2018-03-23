package drivers

//store package store the world - state data
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

var (
	execDrivers        = make(map[string]Driver)
	execAddress        = make(map[string]string)
	execAddressNameMap = make(map[string]string)
)

func Register(name string, driver Driver) {
	if driver == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := execDrivers[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	execDrivers[name] = driver
	registerAddress(name)
}

func LoadDriver(name string) (c Driver, err error) {
	c, ok := execDrivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}
	return c, nil
}

func registerAddress(name string) {
	if len(name) == 0 {
		panic("empty name string")
	}
	addr := ExecAddress(name)
	if _, dup := execAddress[addr]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	execAddress[addr] = name
	execAddressNameMap[name] = addr
}

func IsDriverAddress(addr string) bool {
	_, ok := execAddress[addr]
	return ok
}

func ExecAddress(name string) string {
	if addr, ok := execAddressNameMap[name]; ok {
		return addr
	}
	return account.ExecAddress(name).String()
}

func GetExecNameByAddr(addr string) string {
	if name, ok := execAddress[addr]; ok {
		return name
	} else {
		return ""
	}
}
