package executor

/*
privacy执行器支持隐私交易的执行，

主要提供操作有以下几种：
1）公开地址转账到一次性地址中，即：public address -> one-time addrss
2）隐私转账，隐私余额会被继续转账到一次性地址中 one-time address -> one-time address；
3）隐私余额转账到公开地址中， 即：one-time address -> public address

操作流程：
1）如果需要进行coin或token的隐私转账，则需要首先将其balance转账至privacy合约账户中；
2）此时用户可以发起隐私交易，在交易信息中指定接收的公钥对(A,B),执行成功之后，balance会被存入到一次性地址中；
3）发起交易，

*/

import (
	"bytes"
	"math/rand"
	"sort"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var privacylog = log.New("module", "execs.privacy")

var driverName = "privacy"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&privacy{}))
}

func Init(name string, sub []byte) {
	drivers.Register(GetName(), newPrivacy, types.GetDappFork(driverName, "Enable"))
	// 如果需要在开发环境下使用隐私交易，则需要使用下面这行代码，否则用上面的代码
	//drivers.Register(newPrivacy().GetName(), newPrivacy, 0)
}

func GetName() string {
	return newPrivacy().GetName()
}

type privacy struct {
	drivers.DriverBase
}

func newPrivacy() drivers.Driver {
	t := &privacy{}
	t.SetChild(t)
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

func (p *privacy) GetDriverName() string {
	return driverName
}

func (p *privacy) getUtxosByTokenAndAmount(tokenName string, amount int64, count int32) ([]*pty.LocalUTXOItem, error) {
	localDB := p.GetLocalDB()
	var utxos []*pty.LocalUTXOItem
	prefix := CalcPrivacyUTXOkeyHeightPrefix(tokenName, amount)
	values, err := localDB.List(prefix, nil, count, 0)
	if err != nil {
		return utxos, err
	}

	for _, value := range values {
		var utxo pty.LocalUTXOItem
		err := types.Decode(value, &utxo)
		if err != nil {
			privacylog.Info("getUtxosByTokenAndAmount", "decode to LocalUTXOItem failed because of", err)
			return utxos, err
		}
		utxos = append(utxos, &utxo)
	}

	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Height <= utxos[j].Height
	})
	return utxos, nil
}

func (p *privacy) getGlobalUtxoIndex(getUtxoIndexReq *pty.ReqUTXOGlobalIndex) (types.Message, error) {
	debugBeginTime := time.Now()
	utxoGlobalIndexResp := &pty.ResUTXOGlobalIndex{}
	tokenName := getUtxoIndexReq.Tokenname
	currentHeight := p.GetHeight()
	for _, amount := range getUtxoIndexReq.Amount {
		utxos, err := p.getUtxosByTokenAndAmount(tokenName, amount, types.UTXOCacheCount)
		if err != nil {
			return utxoGlobalIndexResp, err
		}

		index := len(utxos) - 1
		for ; index >= 0; index-- {
			if utxos[index].GetHeight()+types.ConfirmedHeight <= currentHeight {
				break
			}
		}

		mixCount := getUtxoIndexReq.MixCount
		totalCnt := int32(index + 1)
		if mixCount > totalCnt {
			mixCount = totalCnt
		}

		utxoIndex4Amount := &pty.UTXOIndex4Amount{
			Amount: amount,
		}

		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		positions := random.Perm(int(totalCnt))
		for i := int(mixCount - 1); i >= 0; i-- {
			position := positions[i]
			item := utxos[position]
			utxoGlobalIndex := &pty.UTXOGlobalIndex{
				Outindex: item.GetOutindex(),
				Txhash:   item.GetTxhash(),
			}
			utxo := &pty.UTXOBasic{
				UtxoGlobalIndex: utxoGlobalIndex,
				OnetimePubkey:   item.GetOnetimepubkey(),
			}
			utxoIndex4Amount.Utxos = append(utxoIndex4Amount.Utxos, utxo)
		}
		utxoGlobalIndexResp.UtxoIndex4Amount = append(utxoGlobalIndexResp.UtxoIndex4Amount, utxoIndex4Amount)
	}

	duration := time.Since(debugBeginTime)
	privacylog.Debug("getGlobalUtxoIndex cost", duration)
	return utxoGlobalIndexResp, nil
}

//获取指定amount下的所有utxo，这样就可以查询当前系统不同amout下存在的UTXO,可以帮助查询用于混淆用的资源
//也可以确认币种的碎片化问题
//显示存在的各种不同的额度的UTXO,如1,3,5,10,20,30,100...
func (p *privacy) ShowAmountsOfUTXO(reqtoken *pty.ReqPrivacyToken) (types.Message, error) {
	querydb := p.GetLocalDB()

	key := CalcprivacyKeyTokenAmountType(reqtoken.Token)
	replyAmounts := &pty.ReplyPrivacyAmounts{}
	value, err := querydb.Get(key)
	if err != nil {
		return replyAmounts, err
	}
	if value != nil {
		var amountTypes pty.AmountsOfUTXO
		err := types.Decode(value, &amountTypes)
		if err == nil {
			for amount, count := range amountTypes.AmountMap {
				amountDetail := &pty.AmountDetail{
					Amount: amount,
					Count:  count,
				}
				replyAmounts.AmountDetail = append(replyAmounts.AmountDetail, amountDetail)
			}
		}

	}
	return replyAmounts, nil
}

//显示在指定额度下的UTXO的具体信息，如区块高度，交易hash，输出索引等具体信息
func (p *privacy) ShowUTXOs4SpecifiedAmount(reqtoken *pty.ReqPrivacyToken) (types.Message, error) {
	querydb := p.GetLocalDB()

	var replyUTXOsOfAmount pty.ReplyUTXOsOfAmount
	values, err := querydb.List(CalcPrivacyUTXOkeyHeightPrefix(reqtoken.Token, reqtoken.Amount), nil, 0, 0)
	if err != nil {
		return &replyUTXOsOfAmount, err
	}
	if len(values) != 0 {
		for _, value := range values {
			var localUTXOItem pty.LocalUTXOItem
			err := types.Decode(value, &localUTXOItem)
			if err == nil {
				replyUTXOsOfAmount.LocalUTXOItems = append(replyUTXOsOfAmount.LocalUTXOItems, &localUTXOItem)
			}
		}
	}

	return &replyUTXOsOfAmount, nil
}

func (p *privacy) CheckTx(tx *types.Transaction, index int) error {
	txhashstr := common.Bytes2Hex(tx.Hash())
	var action pty.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "Decode tx.Payload error", err)
		return err
	}
	privacylog.Debug("PrivacyTrading CheckTx", "txhash", txhashstr, "action type ", action.Ty)
	if pty.ActionPublic2Privacy == action.Ty {
		return nil
	}
	input := action.GetInput()
	output := action.GetOutput()
	if input == nil || output == nil {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "input", input, "output", output)
		return nil
	}
	//如果是私到私 或者私到公，交易费扣除则需要utxo实现,交易费并不生成真正的UTXO,也是即时燃烧掉而已
	var amount int64
	keyinput := input.Keyinput
	keyOutput := output.Keyoutput
	token := action.GetTokenName()
	if action.Ty == pty.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		amount = action.GetPrivacy2Public().Amount
	}

	if tx.Fee < pty.PrivacyTxFee {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "fee set:", tx.Fee, "required:", pty.PrivacyTxFee, " error ErrPrivacyTxFeeNotEnough")
		return pty.ErrPrivacyTxFeeNotEnough
	}

	var ringSignature types.RingSignature
	if err := types.Decode(tx.Signature.Signature, &ringSignature); err != nil {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "Decode tx.Signature.Signature error ", err)
		return err
	}

	totalInput := int64(0)
	totalOutput := int64(0)
	inputCnt := len(keyinput)
	keyImages := make([][]byte, inputCnt)
	keys := make([][]byte, 0)
	pubkeys := make([][]byte, 0)
	for i, input := range keyinput {
		totalInput += input.Amount
		keyImages[i] = calcPrivacyKeyImageKey(token, input.KeyImage)
		for j, globalIndex := range input.UtxoGlobalIndex {
			keys = append(keys, CalcPrivacyOutputKey(token, input.Amount, common.ToHex(globalIndex.Txhash), int(globalIndex.Outindex)))
			pubkeys = append(pubkeys, ringSignature.Items[i].Pubkey[j])
		}
	}
	res, errIndex := p.checkUTXOValid(keyImages)
	if !res {
		if errIndex >= 0 && errIndex < int32(len(keyinput)) {
			input := keyinput[errIndex]
			privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "UTXO spent already errindex", errIndex, "utxo amout", input.Amount/types.Coin, "utxo keyimage", common.ToHex(input.KeyImage))
		}
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "checkUTXOValid failed ")
		return pty.ErrDoubleSpendOccur
	}

	res, errIndex = p.checkPubKeyValid(keys, pubkeys)
	if !res {
		if errIndex >= 0 && errIndex < int32(len(pubkeys)) {
			privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "Wrong pubkey errIndex ", errIndex)
		}
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "checkPubKeyValid ", false)
		return pty.ErrPubkeysOfUTXO
	}

	for _, output := range keyOutput {
		totalOutput += output.Amount
	}

	feeAmount := int64(0)
	if action.Ty == pty.ActionPrivacy2Privacy {
		feeAmount = totalInput - totalOutput
	} else {
		feeAmount = totalInput - totalOutput - amount
	}

	if feeAmount < pty.PrivacyTxFee {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "fee available:", feeAmount, "required:", pty.PrivacyTxFee)
		return pty.ErrPrivacyTxFeeNotEnough
	}
	return nil
}

//通过keyImage确认是否存在双花，有效即不存在双花，返回true，反之则返回false
func (p *privacy) checkUTXOValid(keyImages [][]byte) (bool, int32) {
	stateDB := p.GetStateDB()
	values, err := stateDB.BatchGet(keyImages)
	if err != nil {
		privacylog.Error("exec module", "checkUTXOValid failed to get value from statDB")
		return false, Invalid_index
	}
	if len(values) != len(keyImages) {
		privacylog.Error("exec module", "checkUTXOValid return different count value with keys")
		return false, Invalid_index
	}
	for i, value := range values {
		if value != nil {
			privacylog.Error("exec module", "checkUTXOValid i=", i, " value=", value)
			return false, int32(i)
		}
	}

	return true, Invalid_index
}

func (p *privacy) checkPubKeyValid(keys [][]byte, pubkeys [][]byte) (bool, int32) {
	values, err := p.GetStateDB().BatchGet(keys)
	if err != nil {
		privacylog.Error("exec module", "checkPubKeyValid failed to get value from statDB with err", err)
		return false, Invalid_index
	}

	if len(values) != len(pubkeys) {
		privacylog.Error("exec module", "checkPubKeyValid return different count value with keys")
		return false, Invalid_index
	}

	for i, value := range values {
		var keyoutput pty.KeyOutput
		types.Decode(value, &keyoutput)
		if !bytes.Equal(keyoutput.Onetimepubkey, pubkeys[i]) {
			privacylog.Error("exec module", "Invalid pubkey for tx with hash", string(keys[i]))
			return false, int32(i)
		}
	}

	return true, Invalid_index
}
