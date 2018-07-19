package paracross

import "gitlab.33.cn/chain33/chain33/account"
import dbm "gitlab.33.cn/chain33/chain33/common/db"
import "gitlab.33.cn/chain33/chain33/types"
import (
	pt "gitlab.33.cn/chain33/chain33/types/executor/paracross"
	"golang.org/x/tools/go/gcimporter15/testdata"
)


type action struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func newAction(t *Paracross, tx *types.Transaction) *action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &action{t.GetCoinsAccount(), t.GetStateDB(), hash, fromaddr,
		t.GetBlockTime(), t.GetHeight(), t.GetAddr()}
}


func getNodes(title string) ([]string, error) {
	return nil, nil
}

func validTitle(title string) bool {
	return true
}

func validNode(addr string, nodes []string) bool {
	return true
}

func checkCommitInfo(commit *types.ParacrossCommitAction) error {
	return nil
}

func isCommitDone(f interface{}, nodes []string) bool {
	return true
}

func makeCommitReceipt(addr string, commit *types.ParacrossCommitAction, prev, current *types.ParacrossHeightStatus) *types.Receipt {
	key := calcTitleHeightKey(commit.Status.Title, commit.Status.Height)
	log := &types.ReceiptParacrossCommit{
		Addr: addr,
		Status: commit.Status,
		Prev: prev,
		Current: current,
	}
	return &types.Receipt{
		Ty: types.ExecOk,
		KV: []*types.KeyValue{
			&types.KeyValue{key, types.Encode(current)},
		},
		Logs: []*types.ReceiptLog {
			&types.ReceiptLog{
				Ty: types.TyLogParacrossCommit,
				Log: types.Encode(log),
			},
		},
	}
}

func makeRecordReceipt(addr string, commit *types.ParacrossCommitAction) *types.Receipt {
	log := &types.ReceiptParacrossRecord{
		Addr: addr,
		Status: commit.Status,
	}
	return &types.Receipt{
		Ty: types.ExecOk,
		KV: nil,
		Logs: []*types.ReceiptLog {
			&types.ReceiptLog{
				Ty: types.TyLogParacrossRecord,
				Log: types.Encode(log),
			},
		},
	}
}

func makeDoneReceipt(addr string, commit *types.ParacrossCommitAction, current *types.ParacrossHeightStatus) *types.Receipt {

	log := &types.ReceiptParacrossDone {
		Title: commit.Status.Title,
		Height: commit.Status.Height,
		StateHash: commit.Status.StateHash,
	}
	return &types.Receipt{
		Ty: types.ExecOk,
		KV: nil,
		Logs: []*types.ReceiptLog {
			&types.ReceiptLog{
				Ty: types.TyLogParacrossDode,
				Log: types.Encode(log),
			},
		},
	}
}

func getMostCommit(stat *types.ParacrossHeightStatus) (int, string) {
	stats := make(map[string]int, 0)
	n := len(stat.Details.Addrs)
	for i := 0; i < n; i++ {
		if _, ok := stats[stat.Details.StateHash[i]]; ok {
			stats[stat.Details.StateHash[i]]++
		} else {
			stats[stat.Details.StateHash[i]] = 1
		}
	}
	most := -1
	var statHash string
	for k, v := range stats {
		if v > most {
			most = v
			statHash = k
		}
	}
	return  most, statHash
}

func (a *action) Commit(commit *types.ParacrossCommitAction) (*types.Receipt, error) {
	err := checkCommitInfo(commit)
	if err != nil {
		return nil, err
	}
	if !validTitle(commit.Status.Title) {
		return nil, types.ErrInputPara // TODO
	}

	nodes, err := getNodes(commit.Status.Title)
	if err != nil {
		return nil, types.ErrInputPara // TODO
	}

	if !validNode(a.fromaddr, nodes) {
		return nil, types.ErrInputPara // TODO
	}

	titleStatus, err := getTitle(a.db, calcTitleKey(commit.Status.Title))
	if err != nil {
		return nil, err
	}
	if commit.Status.Height > titleStatus.Height + 1 {
		// 也许是在分叉上挖矿， 也许是非正常的交易
		return nil, types.ErrInputPara // future height
	} else if commit.Status.Height <= titleStatus.Height {
		// 在完成共识之后来的， 增加 record log， 只记录不修改已经达成的共识
		return makeRecordReceipt(a.fromaddr, commit), nil
	}

	// 处理当前高度的共识
	stat, err := getTitleHeight(a.db, calcTitleHeightKey(commit.Status.Title, commit.Status.Height))
	// 三种情况， 第一个， 非第一个， 触发共识
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	if err == types.ErrNotFound {
		stat = &types.ParacrossHeightStatus{
			Status: pt.ParacrossStatusCommiting,
			Title: commit.Status.Title,
			Height: commit.Status.Height,
			Details: &types.ParacrossStatusDetails{
				Addrs:[]string{a.fromaddr},
				StateHash:[]string{commit.Status.StateHash},
			},
		}
		receipt = makeCommitReceipt(a.fromaddr, commit, nil, stat)
	} else {
		copyStat := *stat
		stat.Details.Addrs = append(stat.Details.Addrs, a.fromaddr)
		stat.Details.StateHash = append(stat.Details.StateHash, commit.Status.StateHash)
		receipt = makeCommitReceipt(a.fromaddr, commit, &copyStat, stat)
	}

	commitCount := len(stat.Details.Addrs)
	most, statHash := getMostCommit(stat)
	if isCommitDone(stat, nodes) {

	}



	return &types.Receipt{types.ExecOk, kv, logs}, nil

/*
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	order := newOrder(put, a.fromaddr)

	key := calcOrderKey(string(a.txhash))
	value := types.Encode(&order.g)
	order.save(a.db, key, value)

	kv = append(kv, &types.KeyValue{key, value})
	log := &types.ReceiptGuessing{Order:&order.g}
	logs = append(logs, &types.ReceiptLog{types.TyLogGuessingPut, types.Encode(log)})
*/
}

