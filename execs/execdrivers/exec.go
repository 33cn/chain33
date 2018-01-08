package execdrivers

//store package store the world - state data
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var elog = log.New("module", "execs")
var zeroHash [32]byte

var bname [200]byte
var addrSeed = []byte("address seed bytes for public key")

func ExecAddress(name string) *account.Address {
	if len(name) > 100 {
		panic("name too long")
	}
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	return account.PubKeyToAddress(hash[:])
}

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Executer interface {
	SetDB(dbm.KVDB)
	GetName() string
	SetEnv(height, blocltime int64)
	Exec(tx *types.Transaction) (*types.Receipt, error)
	Query(tx *types.Transaction) (types.Message, error)
}

var (
	drivers     = make(map[string]Executer)
	execaddress = make(map[string]string)
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

func RegisterAddress(name string) {
	if len(name) == 0 {
		panic("empty name string")
	}
	if _, dup := execaddress[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	execaddress[ExecAddress(name).String()] = name
}

func IsExecAddress(addr string) bool {
	_, ok := execaddress[addr]
	return ok
}
