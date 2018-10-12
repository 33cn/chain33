package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	"reflect"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
)

var (
	minPeriod int64 = 60
	rlog            = log.New("module", "execs.retrieve")
)

var (
	zeroDelay       int64
	zeroPrepareTime int64
	zeroRemainTime  int64
)

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = rt.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&Retrieve{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

//const maxTimeWeight = 2
func Init(name string) {
	drivers.Register(GetName(), newRetrieve, 0)
}

func GetName() string {
	return newRetrieve().GetName()
}

type Retrieve struct {
	drivers.DriverBase
}

func newRetrieve() drivers.Driver {
	r := &Retrieve{}
	r.SetChild(r)
	r.SetExecutorType(executorType)
	return r
}

func (r *Retrieve) GetDriverName() string {
	return "retrieve"
}

func (g *Retrieve) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (r *Retrieve) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func calcRetrieveKey(backupAddr string, defaultAddr string) []byte {
	key := fmt.Sprintf("Retrieve-backup:%s:%s", backupAddr, defaultAddr)
	return []byte(key)
}

func getRetrieveInfo(db dbm.KVDB, backupAddr string, defaultAddr string) (*rt.RetrieveQuery, error) {
	info := rt.RetrieveQuery{}
	retInfo, err := db.Get(calcRetrieveKey(backupAddr, defaultAddr))
	if err != nil {
		return nil, err
	}

	err = types.Decode(retInfo, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
