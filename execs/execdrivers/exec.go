package execdrivers

//store package store the world - state data
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")
var zeroHash [32]byte

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Executer interface {
	SetDB(dbm.KVDB)
	SetEnv(height, blocltime int64)
	Exec(tx *types.Transaction) (*types.Receipt, error)
}

var (
	drivers = make(map[string]Executer)
)

func Register(name string, driver Executer) {
	if driver == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

func LoadExecute(name string) (c Executer, err error) {
	c, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}
	return c, nil
}
