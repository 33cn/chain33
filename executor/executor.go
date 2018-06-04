package executor

//store package store the world - state data
import (
	"bytes"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/coins"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/hashlock"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/manage"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/none"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/privacy"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/retrieve"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/ticket"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/token"
	_ "gitlab.33.cn/chain33/chain33/executor/drivers/trade"
	privExec "gitlab.33.cn/chain33/chain33/executor/drivers/privacy"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var elog = log.New("module", "execs")
var coinsAccount = account.NewCoinsAccount()
var runningHeight int64 = 0

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	elog.SetHandler(log.DiscardHandler())
}

type Executor struct {
	client queue.Client
}

func New(cfg *types.Exec) *Executor {
	//设置区块链的MinFee，低于Mempool和Wallet设置的MinFee
	//在cfg.MinExecFee == 0 的情况下，必须 cfg.IsFree == true 才会起效果
	if cfg.MinExecFee == 0 && cfg.IsFree == true {
		elog.Warn("set executor to free fee")
		types.SetMinFee(0)
	}
	if cfg.MinExecFee > 0 {
		types.SetMinFee(cfg.MinExecFee)
	}
	exec := &Executor{}
	return exec
}

func (exec *Executor) SetQueueClient(client queue.Client) {
	exec.client = client
	exec.client.Sub("execs")

	//recv 消息的处理
	go func() {
		for msg := range client.Recv() {
			elog.Debug("exec recv", "msg", msg)
			if msg.Ty == types.EventExecTxList {
				exec.procExecTxList(msg)
			} else if msg.Ty == types.EventAddBlock {
				exec.procExecAddBlock(msg)
			} else if msg.Ty == types.EventDelBlock {
				exec.procExecDelBlock(msg)
			} else if msg.Ty == types.EventCheckTx {
				exec.procExecCheckTx(msg)
			}
		}
	}()
}

func (exec *Executor) procExecCheckTx(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec.client.Clone(), datas.Height, datas.BlockTime)
	//返回一个列表表示成功还是失败
	result := &types.ReceiptCheckTxList{}
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		err := execute.execCheckTx(tx, i)
		if err != nil {
			result.Errs = append(result.Errs, err.Error())
		} else {
			result.Errs = append(result.Errs, "")
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventReceiptCheckTx, result))
}

func (exec *Executor) procExecTxList(msg queue.Message) {
	datas := msg.GetData().(*types.ExecTxList)
	execute := newExecutor(datas.StateHash, exec.client.Clone(), datas.Height, datas.BlockTime)
	var receipts []*types.Receipt
	index := 0
	var privacyKV []*types.PrivacyKVToken
	for i := 0; i < len(datas.Txs); i++ {
		tx := datas.Txs[i]
		if execute.height == 0 { //genesis block 不检查手续费
			receipt, err := execute.Exec(tx, i)
			if err != nil {
				panic(err)
			}
			receipts = append(receipts, receipt)
			continue
		}
		//交易检查规则：
		//1. mempool 检查区块，尽量检查更多的错误
		//2. 打包的时候，尽量打包更多的交易，只要基本的签名，以及格式没有问题
		err := execute.checkTx(tx, index)
		if err != nil {
			receipt := types.NewErrReceipt(err)
			receipts = append(receipts, receipt)
			continue
		}
		//处理交易手续费(先把手续费收了)
		//如果收了手续费，表示receipt 至少是pack 级别
		//收不了手续费的交易才是 error 级别
		feelog := &types.Receipt{Ty: types.ExecPack}
		e, err := LoadDriver(string(tx.Execer))
		if err != nil {
			e, err = LoadDriver("none")
			if err != nil {
				panic(err)
			}
		}
		if !e.IsFree() && types.MinFee > 0 {
			feelog, err = execute.processFee(tx)
			if err != nil {
				receipt := types.NewErrReceipt(err)
				receipts = append(receipts, receipt)
				continue
			}
		}
		//只有到pack级别的，才会增加index
		receipt, err := execute.Exec(tx, index)
		index++
		if err != nil {
			elog.Error("exec tx error = ", "err", err, "tx", tx)
			//add error log
			errlog := &types.ReceiptLog{types.TyLogErr, []byte(err.Error())}
			feelog.Logs = append(feelog.Logs, errlog)
		} else {
			//合并两个receipt，如果执行不返回错误，那么就认为成功
			if receipt != nil {
				feelog.KV = append(feelog.KV, receipt.KV...)
				feelog.Logs = append(feelog.Logs, receipt.Logs...)
				feelog.Ty = receipt.Ty

				//如果是隐私交易,且目的方为隐私地址，则需要将其KV单独挑出来，以供blockchain使用
				if types.PrivacyX == string(tx.GetExecer()) {
					var action types.PrivacyAction

					if nil == types.Decode(tx.Payload, &action) {
						if action.Ty == types.ActionPublic2Privacy || action.Ty == types.ActionPrivacy2Privacy {
							privacyKVToken := &types.PrivacyKVToken{
								KV:      receipt.KV,
								TxIndex: int32(index),
								Txhash:  tx.Hash(),
							}
							if pub2priv := action.GetPublic2Privacy(); pub2priv != nil {
								privacyKVToken.Token = pub2priv.Tokenname
							} else if priv2priv := action.GetPrivacy2Privacy(); priv2priv != nil {
								privacyKVToken.Token = priv2priv.Tokenname
							}
							privacyKV = append(privacyKV, privacyKVToken)
						}
					}
				}
			}
		}
		receipts = append(receipts, feelog)
		elog.Debug("exec tx = ", "index", index, "execer", string(tx.Execer))
	}

	receiptsAndPrivacyKV := &types.ReceiptsAndPrivacyKV{
		Receipts:  &types.Receipts{receipts},
		PrivacyKV: &types.PrivacyKV{privacyKV},
	}

	msg.Reply(exec.client.NewMessage("", types.EventReceipts, receiptsAndPrivacyKV))
}

func (exec *Executor) procExecAddBlock(msg queue.Message) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, exec.client.Clone(), b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		kv, err := execute.execLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		if kv != nil && kv.KV != nil {
			err := exec.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}
	}
	msg.Reply(exec.client.NewMessage("", types.EventAddBlock, &kvset))
}

func (exec *Executor) procExecDelBlock(msg queue.Message) {
	datas := msg.GetData().(*types.BlockDetail)
	b := datas.Block
	execute := newExecutor(b.StateHash, exec.client.Clone(), b.Height, b.BlockTime)
	var kvset types.LocalDBSet
	privacyKV := &types.PrivacyKV{}
	i := len(b.Txs) - 1
	for ; i >= 0; i-- {
		tx := b.Txs[i]
		kv, err := execute.execDelLocal(tx, datas.Receipts[i], i)
		if err == types.ErrActionNotSupport {
			continue
		}
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
			return
		}

		if kv != nil && kv.KV != nil {
			err := exec.checkPrefix(tx.Execer, kv.KV)
			if err != nil {
				msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
				return
			}
			kvset.KV = append(kvset.KV, kv.KV...)
		}

		//如果是隐私交易,且目的方为隐私地址，则需要将其KV单独挑出来，以供blockchain使用
		if types.PrivacyX == string(tx.GetExecer()) {
			var action types.PrivacyAction
			if nil == types.Decode(tx.Payload, &action) {
				if action.Ty == types.ActionPublic2Privacy ||  action.Ty == types.ActionPrivacy2Privacy {
					privacyKVToken := &types.PrivacyKVToken{
						TxIndex:int32(i),
						Txhash:tx.Hash(),
					}
					for _, ele_kv := range kv.KV {
						if bytes.Contains(ele_kv.Key, []byte(privExec.PrivacyUTXOKEYPrefix)){
							privacyKVToken.KV = append(privacyKVToken.KV, ele_kv)
						}
					}

					if pub2priv := action.GetPublic2Privacy(); pub2priv != nil {
						privacyKVToken.Token = pub2priv.Tokenname
					} else if priv2priv := action.GetPrivacy2Privacy(); priv2priv != nil{
						privacyKVToken.Token = priv2priv.Tokenname
					}
					privacyKV.PrivacyKVToken = append(privacyKV.PrivacyKVToken, privacyKVToken)
				}
			}
		}
	}

	localDBSetWithPrivacy := &types.LocalDBSetWithPrivacy {
		LocalDBSet:&kvset,
		PrivacyLocalDBSet:privacyKV,
	}
	msg.Reply(exec.client.NewMessage("", types.EventDelBlock, localDBSetWithPrivacy))
}

func (exec *Executor) checkPrefix(execer []byte, kvs []*types.KeyValue) error {
	if kvs == nil {
		return nil
	}
	if bytes.HasPrefix(execer, []byte("user.")) {
		for j := 0; j < len(kvs); j++ {
			if !bytes.HasPrefix(kvs[j].Key, execer) {
				return types.ErrLocalDBPerfix
			}
		}
	}
	return nil
}

func (exec *Executor) Close() {
	elog.Info("exec module closed")
}

//执行器 -> db 环境
type executor struct {
	stateDB      dbm.KVDB
	localDB      dbm.KVDB
	coinsAccount *account.AccountDB
	execDriver   *drivers.ExecDrivers
	height       int64
	blocktime    int64
}

func newExecutor(stateHash []byte, client queue.Client, height, blocktime int64) *executor {
	e := &executor{
		stateDB:      NewStateDB(client.Clone(), stateHash),
		localDB:      NewLocalDB(client.Clone()),
		coinsAccount: account.NewCoinsAccount(),
		execDriver:   drivers.CreateDrivers4CurrentHeight(height),
		height:       height,
		blocktime:    blocktime,
	}
	e.coinsAccount.SetDB(e.stateDB)
	runningHeight = height
	return e
}

//隐私交易费扣除规则：
//1.公对私交易：直接从coin合约中扣除
//2.私对私交易或者私对公交易：交易费的扣除从隐私合约账户在coin合约中的账户中扣除
func (e *executor) processFee(tx *types.Transaction) (*types.Receipt, error) {
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if types.PrivacyX != string(tx.Execer) {
		return e.cutFeeFromAccount(from, tx.Fee)
	} else {
		var action types.PrivacyAction
		err := types.Decode(tx.Payload, &action)
		if err != nil {
			return nil, err
		}
		if action.Ty == types.ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
			return e.cutFeeFromAccount(from, tx.Fee)
		} else { //如果是私到私 或者私到公，交易费扣除则需要utxo实现,交易费并不生成真正的UTXO,也是即时燃烧掉而已
			if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
				totalInput := int64(0)
				keys := make([][]byte, len(action.GetPrivacy2Privacy().Input.Keyinput))
				for i, input := range action.GetPrivacy2Privacy().Input.Keyinput {
					totalInput += input.Amount
					keys[i] = input.KeyImage
				}
				if !e.checkUTXOValid(keys) {
					elog.Error("exec UTXO double spend check failed")
					return nil, types.ErrDoubeSpendOccur
				}
				totalOutput := int64(0)
				for _, output := range action.GetPrivacy2Privacy().Output.Keyoutput {
					totalOutput += output.Amount
				}
				feeAmount := totalInput - totalOutput
				if feeAmount >= tx.Fee {
					//从隐私合约在coin的账户中扣除，同时也保证了相应的utxo差额被燃烧
					execaddr := drivers.ExecAddress(types.PrivacyX)
					return e.cutFeeFromAccount(execaddr, feeAmount)
				}

			} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
				totalInput := int64(0)
				keys := make([][]byte, len(action.GetPrivacy2Public().Input.Keyinput))
				for i, input := range action.GetPrivacy2Public().Input.Keyinput {
					totalInput += input.Amount
					keys[i] = input.KeyImage
				}
				if !e.checkUTXOValid(keys) {
					elog.Error("exec UTXO double spend check failed")
					return nil, types.ErrDoubeSpendOccur
				}

				totalOutput := int64(0)
				for _, output := range action.GetPrivacy2Privacy().Output.Keyoutput {
					totalOutput += output.Amount
				}

				feeAmount := totalInput-action.GetPrivacy2Public().Amount-totalOutput
				if feeAmount >= tx.Fee {
					//从隐私合约在coin的账户中扣除，同时也保证了相应的utxo差额被燃烧
					execaddr := drivers.ExecAddress(types.PrivacyX)
					return e.cutFeeFromAccount(execaddr, feeAmount)
				}
			}
		}
	}
	return nil, types.ErrNoBalance
}

//通过keyImage确认是否存在双花，有效即不存在双花，返回true，反之则返回false
func (e *executor) checkUTXOValid(keyImages [][]byte) bool {
	if values, err := e.stateDB.BatchGet(keyImages); err == nil {
		if len(values) != len(keyImages) {
			elog.Error("exec module", "checkUTXOValid return different count value with keys")
			return false
		}
		for _, value := range values {
			if value != nil {
				return false
			}
		}
		return true
	}
	return false
}

func (e *executor) cutFeeFromAccount(AccAddr string, feeAmount int64)(*types.Receipt, error) {
	accFrom := e.coinsAccount.LoadAccount(AccAddr)
	if accFrom.GetBalance()- feeAmount >= 0 {
		copyfrom := *accFrom
		accFrom.Balance = accFrom.GetBalance() - feeAmount
		receiptBalance := &types.ReceiptAccountTransfer{&copyfrom, accFrom}
		e.coinsAccount.SaveAccount(accFrom)
		return e.cutFeeReceipt(accFrom, receiptBalance), nil
	}

	return nil, types.ErrNoBalance
}

func (e *executor) cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptAccountTransfer) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, e.coinsAccount.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func (e *executor) checkTx(tx *types.Transaction, index int) error {
	if e.height > 0 && e.blocktime > 0 && tx.IsExpire(e.height, e.blocktime) {
		//如果已经过期
		return types.ErrTxExpire
	}
	if err := tx.Check(types.MinFee); err != nil {
		return err
	}
	return nil
}

func (e *executor) checkPrivacyTxFee(tx *types.Transaction) error {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return err
	}
	if action.Ty == types.ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		// 公开对隐私,只需要直接计算隐私合约中的余额即可
		from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
		execaddr := drivers.ExecAddress(types.PrivacyX)
		accFrom := e.coinsAccount.LoadExecAccount(from, execaddr)
		if accFrom.GetBalance()-tx.Fee < 0 {
			return types.ErrBalanceLessThanTenTimesFee
		}

	} else if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		if tx.Fee < types.PrivacyUTXOFee {
			return types.ErrPrivacyTxFeeNotEnough
		}
		// 隐私对隐私,需要检查输入和输出值,条件 输入-输入>=手续费
		var totalInput, totalOutput int64
		for _, value := range action.GetPrivacy2Privacy().GetInput().GetKeyinput() {
			if value.GetAmount()<=0 {
				return types.ErrAmount
			}
			totalInput += value.GetAmount()
		}
		for _, value := range action.GetPrivacy2Privacy().GetOutput().GetKeyoutput() {
			if value.GetAmount() <= 0 {
				return types.ErrAmount
			}
			totalOutput += value.GetAmount()
		}
		if totalInput-totalOutput < tx.Fee {
			return types.ErrPrivacyTxFeeNotEnough
		}

	} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		if tx.Fee < types.PrivacyUTXOFee {
			return types.ErrPrivacyTxFeeNotEnough
		}

		var totalInput, totalOutput int64
		for _, value := range action.GetPrivacy2Public().GetInput().GetKeyinput() {
			if value.GetAmount()<=0 {
				return types.ErrAmount
			}
			totalInput += value.GetAmount()
		}
		for _, value := range action.GetPrivacy2Public().GetOutput().GetKeyoutput() {
			if value.GetAmount() <= 0 {
				return types.ErrAmount
			}
			totalOutput += value.GetAmount()
		}
		if totalInput-totalOutput-action.GetPrivacy2Public().GetAmount() < tx.Fee {
			return types.ErrPrivacyTxFeeNotEnough
		}
	}

	return nil
}

func (e *executor) execCheckTx(tx *types.Transaction, index int) error {
	//基本检查
	err := e.checkTx(tx, index)
	if err != nil {
		return err
	}
	//checkInExec
	exec := e.loadDriverForExec(string(tx.Execer))
	//手续费检查
	if !exec.IsFree() && types.MinFee > 0 {
		if types.PrivacyX == string(tx.Execer) {
			err = e.checkPrivacyTxFee(tx)
			if err != nil {
				return err
			}
		} else {
			from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
			accFrom := e.coinsAccount.LoadAccount(from)
			if accFrom.GetBalance() < types.MinBalanceTransfer {
				return types.ErrBalanceLessThanTenTimesFee
			}
		}
	}

	exec.SetDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.CheckTx(tx, index)
}

func (e *executor) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	exec := e.loadDriverForExec(string(tx.Execer))
	exec.SetDB(e.stateDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.Exec(tx, index)
}

func (e *executor) execLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriverForExec(string(tx.Execer))
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecLocal(tx, r, index)
}

func (e *executor) execDelLocal(tx *types.Transaction, r *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	exec := e.loadDriverForExec(string(tx.Execer))
	exec.SetLocalDB(e.localDB)
	exec.SetEnv(e.height, e.blocktime)
	return exec.ExecDelLocal(tx, r, index)
}

func (e *executor) loadDriverForExec(exector string) (c drivers.Driver) {
	exec, err := e.execDriver.LoadDriver(exector)
	if err != nil {
		exec, err = e.execDriver.LoadDriver("none")
		if err != nil {
			panic(err)
		}
	}
	exec.SetExecDriver(e.execDriver)

	return exec
}

func LoadDriver(name string) (c drivers.Driver, err error) {
	execDrivers := drivers.CreateDrivers4CurrentHeight(runningHeight)
	return execDrivers.LoadDriver(name)
}
