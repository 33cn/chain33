package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
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

var driverName = "retrieve"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Retrieve{}))
}

//const maxTimeWeight = 2
func Init(name string, sub []byte) {
	drivers.Register(GetName(), newRetrieve, types.GetDappFork(driverName, "Enable"))
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
	r.SetExecutorType(types.LoadExecutorType(driverName))
	return r
}

func (r *Retrieve) GetDriverName() string {
	return driverName
}

func (r *Retrieve) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func calcRetrieveKey(backupAddr string, defaultAddr string) []byte {
	key := fmt.Sprintf("LODB-retrieve-backup:%s:%s", backupAddr, defaultAddr)
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
