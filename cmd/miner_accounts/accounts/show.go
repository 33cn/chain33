package accounts

import (

	"time"

	l "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/types"

	//"encoding/json"
	//"io/ioutil"
	"strconv"
	"fmt"
	"gitlab.33.cn/chain33/chain33/rpc"
)

var log = l.New("module", "accounts")

type ShowMinerAccount struct {
	DataDir string
	Addrs []string
}

func (*ShowMinerAccount) Echo(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	*out = *in
	return nil
}

type TimeAt struct {
	// YYYY-mm-dd-HH
	TimeAt string `json:"timeAt"`
}

func (show *ShowMinerAccount) Get(in *TimeAt, out *interface{}) error {
	if in == nil {
		log.Error("show", "in", "nil")
		return types.ErrInputPara
	}
	seconds := time.Now().Unix()
	if len(in.TimeAt) != 0 {
		tm, err := time.Parse("2006-01-02-15", in.TimeAt)
		if err != nil {
			log.Error("show", "in.TimeAt Parse", err)
			return types.ErrInputPara
		}
		seconds = tm.Unix()
	}
	log.Info("show", "utc", seconds)

	addrs := show.Addrs
	log.Info("show", "miners", addrs)
	realTs, curAcc, err := cache.getBalance(addrs, "ticket", seconds)
	if err != nil {
		return nil
	}
	lastReadTs, lastAcc, err := cache.getBalance(addrs, "ticket", realTs-3600)
	if err != nil {
		return nil
	}
	fmt.Print(curAcc, lastAcc)
	miner := calcResult(curAcc, lastAcc)
	miner.Seconds = realTs - lastReadTs
	*out = &miner

	return nil
}

func calcResult(acc1, acc2 []*rpc.Account) *MinerAccounts {
	type minerAt struct {
		addr string
		curAcc *rpc.Account
		lastAcc *rpc.Account
	}
	miners := map[string]*minerAt{}
	for _, a := range acc1 {
		miners[a.Addr] = &minerAt{a.Addr, a, nil}
	}
	for _, a := range acc2 {
		if _, ok := miners[a.Addr]; ok {
			miners[a.Addr].lastAcc = a
		}
	}
	miner := MinerAccounts{}
	totalIncrease := float64(0)
	for k, v := range miners {
		if v.lastAcc != nil && v.curAcc != nil {
			total := v.curAcc.Balance + v.curAcc.Frozen
			increase := total - v.lastAcc.Balance - v.lastAcc.Frozen
			m :=  &MinerAccount{
				Addr: k,
				Total: strconv.FormatFloat(float64(total)/float64(types.Coin), 'f', 4, 64),
				Increase: strconv.FormatFloat(float64(increase)/float64(types.Coin), 'f', 4, 64),
			}
			miner.MinerAccounts = append(miner.MinerAccounts, m)
			totalIncrease += float64(increase)/float64(types.Coin)
		}
	}
	miner.TotalIncrease = strconv.FormatFloat(totalIncrease, 'f', 4, 64)

	return &miner

}

/*
func readJson(jsonFile string) (*Accounts, error) {
	d1, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Error("show", "read json", jsonFile, "err", err)
		return nil, err
	}
	var acc Accounts
	err = json.Unmarshal([]byte(d1), &acc)
	if err != nil {
		log.Error("show", "read json", jsonFile, "err", err)
		return nil, err
	}
	return &acc, nil
}

func parseAccounts(acc *Accounts) (*map[string]float64, error) {
	result := map[string]float64{}
	for _, a := range acc.Accounts {
		f1, e1 := strconv.ParseFloat(a.Balance, 64)
		f2, e2 := strconv.ParseFloat(a.Frozen, 64)
		if e1 != nil || e2 != nil {
			log.Error("show", "account2  len", e1, "account  len", e2)
			return nil, types.ErrNotFound
		}
		result[a.Addr] = f1 + f2
	}
	return &result, nil
}

func getAccountDetail(jsonFile string) (*map[string]float64, error) {
	acc, err := readJson(jsonFile)
	if err != nil {
		return nil, err
	}
	return parseAccounts(acc)
}

*/

