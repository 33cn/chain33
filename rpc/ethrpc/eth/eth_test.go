package eth

import (
	"errors"
	"math/big"
	"testing"

	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	etypes "github.com/33cn/chain33/rpc/ethrpc/types"
	"github.com/33cn/chain33/system/dapp/coins/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	ethCli *ethHandler
	qapi   *clientMocks.QueueProtocolAPI
	q      = queue.New("test")
)

func init() {
	qapi = &clientMocks.QueueProtocolAPI{}
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	q.SetConfig(cfg)
	ethCli = &ethHandler{}
	ethCli.cfg = cfg
	ethCli.cli.Init(q.Client(), qapi)

}

func TestEthApi_Accounts(t *testing.T) {
	var account = &ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xa42431Da868c58877a627CC71Dc95F01bf40c196"}, Label: "test1"}
	var account2 = &ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xBe807Dddb074639cD9fA61b47676c064fc50D62C"}, Label: "test2"}
	var resp = &ctypes.WalletAccounts{
		Wallets: []*ctypes.WalletAccount{account, account2},
	}
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	qapi.On("ExecWalletFunc", "wallet", "WalletGetAccountList", req).Return(nil, errors.New("wrong param"))
	accs, err := ethCli.Accounts()
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(accs))
	qapi = &clientMocks.QueueProtocolAPI{}
	qapi.On("ExecWalletFunc", "wallet", "WalletGetAccountList", req).Return(resp, nil)
	var addrs = []string{"0xa42431Da868c58877a627CC71Dc95F01bf40c196", "0xBe807Dddb074639cD9fA61b47676c064fc50D62C"}
	ethCli.cli.Init(q.Client(), qapi)
	accs, err = ethCli.Accounts()
	assert.Nil(t, err)
	assert.Equal(t, accs, addrs)
}

func TestEthApi_BlockNumber(t *testing.T) {
	var header ctypes.Header
	header.Height = 102400
	qapi.On("GetLastHeader", mock.Anything).Return(&header, nil)
	bnum, err := ethCli.BlockNumber()
	assert.Nil(t, err)
	t.Log("blocknum:", bnum)
	bn := big.NewInt(header.Height)
	assert.Equal(t, hexutil.EncodeBig(bn), bnum.String())
}

func TestEthApi_Mining(t *testing.T) {
	var status = new(ctypes.WalletStatus)
	status.IsAutoMining = true
	qapi.On("ExecWalletFunc", "wallet", "GetWalletStatus", &ctypes.ReqNil{}).Return(status, nil)
	mining, err := ethCli.Mining()
	assert.Nil(t, err)
	t.Log("mining:", mining)
	assert.Equal(t, true, mining)
}

func TestEthApi_GetBalance(t *testing.T) {
	var req ctypes.ReqBalance
	req.Execer = "coins"
	req.AssetSymbol = "bty"
	addr := "0x1E79307966B830bCdfEAB1Ed09871d248c2fE171"
	req.Addresses = []string{addr}

	var header ctypes.Header
	header.Height = 102400
	head := &ctypes.Header{StateHash: []byte("sdfadasds")}
	qapi.On("GetLastHeader").Return(head, nil)
	qapi.On("GetConfig").Return(ethCli.cfg, nil)
	var acc = &ctypes.Account{Addr: "0x1E79307966B830bCdfEAB1Ed09871d248c2fE171", Balance: 5e8}
	accv := ctypes.Encode(acc)
	storevalue := &ctypes.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	qapi.On("StoreGet", mock.Anything).Return(storevalue, nil)
	var tag = "latest"
	balanceHexStr, err := ethCli.GetBalance(addr, &tag)
	assert.Nil(t, err)
	t.Log("balance", balanceHexStr)
	assert.Equal(t, balanceHexStr.String(), "0x1dcd6500")
}

func TestEthApi_GetBlock(t *testing.T) {
	var resp ctypes.BlockDetails
	var item ctypes.BlockDetail

	resp.Items = append(resp.GetItems(), &item)

	transfer := &ctypes.AssetsTransfer{
		Cointoken: "",
		Amount:    102454299999,
		To:        "",
		Note:      nil,
	}

	v := &types.CoinsAction_Transfer{Transfer: transfer}
	action := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	item.Block = &ctypes.Block{
		Height:     70,
		BlockTime:  1648792721,
		Version:    0,
		ParentHash: common.FromHex("25bfcebbeedd16039bb22f0e15702f9e7a0f6d8a3ae05e2e7e6fd7ce66babb36"),
		TxHash:     common.FromHex("0xfe4a45a5b946e7de1d6c0bcd3f47d0d4b68bb252009bd032f9102854346c3217"),
		Txs: []*ctypes.Transaction{{
			Execer:  []byte("coins"),
			Payload: ctypes.Encode(action),
			Expire:  0,
			Nonce:   2871810575193867206,
			ChainID: 88,
			To:      "0xde79a84dd3a16bb91044167075de17a1ca4b1d6b",
			Signature: &ctypes.Signature{
				Ty:        1,
				Pubkey:    common.FromHex("0x02f5263862dae4e8516e08e5551f353be05f479c4d36ea754a7b2b359f81fbebb1"),
				Signature: common.FromHex("0x30450221008259d987850c34036a33ede7db25ea72dbe92edd3e9bfe6cbcabd0c43c427a9902206f410d921d6b0d09ce9a8cdf0db9f8eec64677c56404fcd2ccf0e2c5ef4dd06a"),
			},
		}},
		Difficulty: 520159231,
		MainHash:   common.FromHex("0x660f78e492bf2630ecd4d8fdf09ec64f0e141bdfeb7636ed4992b31dd81338bd"),
		MainHeight: 70,
	}

	testEthApiGetBlockByHash(t, &resp)
	testEthApiGetBlockByNumber(t, &resp)

}
func testEthApiGetBlockByHash(t *testing.T, resp *ctypes.BlockDetails) {
	var hash = "0x660f78e492bf2630ecd4d8fdf09ec64f0e141bdfeb7636ed4992b31dd81338bd"
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, common.FromHex(hash))
	qapi.On("GetBlockByHashes", &req).Return(resp, nil)
	block, err := ethCli.GetBlockByHash(hash, true)
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Number, "0x46")

}

func testEthApiGetBlockByNumber(t *testing.T, resp *ctypes.BlockDetails) {
	var req ctypes.ReqBlocks
	req.Start = 70
	req.End = 70
	req.IsDetail = true
	qapi.On("GetBlocks", &req).Return(resp, nil)
	num := (*hexutil.Big)(big.NewInt(req.Start))
	block, err := ethCli.GetBlockByNumber(num, true)
	assert.Nil(t, err)
	assert.Equal(t, num.String(), block.Header.Number)
}

func TestEthApi_ChainId(t *testing.T) {
	id, err := ethCli.ChainId()
	assert.Nil(t, err)
	assert.Equal(t, id.String(), "0x21")
}

func TestEthApi_EstimateGas(t *testing.T) {

	qapi.On("Query", mock.Anything).Return("0x12345", nil)
	var msg etypes.CallMsg
	msg.From = "1N2aNfWXqGe9kWcL8u9TYpj5RzVQbjwKAP"
	msg.To = "1QHKXSGAgmCyZXhmkjPR4eRy7vHeezeiD6"
	msg.Value = (*hexutil.Big)(big.NewInt(1))
	_, err := ethCli.EstimateGas(&msg)
	assert.NotNil(t, err)
}

func TestEthApi_GetTxCount(t *testing.T) {
	blockdetails := &ctypes.BlockDetails{Items: []*ctypes.BlockDetail{
		{
			Block: &ctypes.Block{
				Txs: []*ctypes.Transaction{
					{},
				},
			},
		},
	}}

	qapi.On("GetBlockByHashes", mock.Anything).Return(blockdetails, nil)
	var hash = "0x660f78e492bf2630ecd4d8fdf09ec64f0e141bdfeb7636ed4992b31dd81338bd"
	count, err := ethCli.GetBlockTransactionCountByHash(hash)
	assert.Nil(t, err)
	assert.Equal(t, 1, int(count))

	qapi.On("GetBlocks", mock.Anything).Return(blockdetails, nil)
	count, err = ethCli.GetBlockTransactionCountByNumber((*hexutil.Big)(big.NewInt(70)))
	assert.Nil(t, err)
	assert.Equal(t, 1, int(count))
}

func TestEthApi_GetTransaction(t *testing.T) {
	hexTxs := "0x0a8d110ad4030a0365766d12e8012a647750c9f0000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c4512000000000000000000000000096cf8ba17671e7aa4f76fb81813a6d713ba866b00000000000000000000000000000000000000000000000000005b21a46300803a5c6c6f636b28314e32614e665758714765396b57634c3875395459706a35527a5651626a774b41502c20317271644375547970557979645747674e7a6676314376794b474d364d456b78472c203130303230303035303030303030302942223151484b58534741676d43795a58686d6b6a50523465527937764865657a656944361a6e0801122103c404be076a4ffd87ce494d54029e840810a9e06e6dbfb72b8cb2d0906d1b02851a473045022100c2a822a60bfb0212197c86442a14a44619cd0c1447051708df70a56c81cadfd802200f7a3b71c7f0f817e57c8b1bc747f5d394535557e259be9698e517f9e883099230d6838ba9b18c9bee343a223139746a5335316b6a7772436f535153313355336f7765376759424c6653666f466d40054a201ff66560942c87e94112b99970915354b09f80e813d2c10ad41b47829862080c5220c5bba5174b2788b4241700fa9f50d4fe2b936fc7bcffac21afefeb9aa6e6517d12f10c08021abf0108dc0412b9010a734c4f44422d65766d2d73746174653a317271644375547970557979645747674e7a6676314376794b474d364d456b78473a307833313539636239376466353336306134643363386335386535613836613465323735383432366235623930313432396632633232613138353536363564343638122000000000000000000000000000000000000000000000000000005b21a46300801a2000000000000000000000000000000000000000000000000000000000000000001abf0108dc0412b9010a734c4f44422d65766d2d73746174653a317271644375547970557979645747674e7a6676314376794b474d364d456b78473a3078653330346636323139636439383166653532356234646536653462646165373165373838396262316363343831336162373334393635393536363630666437321220000000000000000000000000000000000000000000000000000ac23e4baaeb401a20000000000000000000000000000000000000000000000000000b1d5ff00debc01a8e0108dd041288010a20ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef0a20000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c45120a20000000000000000000000000ff610318e64e7f41517a689700bd9823a02736aa122000000000000000000000000000000000000000000000000000005b21a46300801abf0108dc0412b9010a734c4f44422d65766d2d73746174653a317271644375547970557979645747674e7a6676314376794b474d364d456b78473a307830373937623864333334386535323930646230643937303235653761303366656237633533303332616134343232633734613732663164346331633561326232122000000000000000000000000000000000000000000000000000005b21a46300801a2000000000000000000000000000000000000000000000000000000000000000001a8e0108dd041288010a208c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9250a20000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c45120a20000000000000000000000000ff610318e64e7f41517a689700bd9823a02736aa122000000000000000000000000000000000000000000000000000000000000000001ac00108dc0412ba010a744c4f44422d65766d2d73746174653a3151484b58534741676d43795a58686d6b6a50523465527937764865657a656944363a307830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303038122000000000000000000000000000000000000000000000000000000000000003311a2000000000000000000000000000000000000000000000000000000000000003321ac00108dc0412ba010a744c4f44422d65766d2d73746174653a3151484b58534741676d43795a58686d6b6a50523465527937764865657a656944363a3078343030653030613561643837326264656533653536633231656561313064333564613366303665393730326333663463613366366534626564333332663165321220000000000000000000000000000000000000000000000000000ac23e4baaeb401a20000000000000000000000000000000000000000000000000000b1d5ff00debc01aab0208dd0412a5020a204150c77f33761fbde38a274b3cbecce67bc2060ac09687dd6eab0cffe90dfefe128002000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c4512000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c4512000000000000000000000000096cf8ba17671e7aa4f76fb81813a6d713ba866b00000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000005b21a46300800000000000000000000000000000000000000000000000000000000000000332000000000000000000000000000000000000000000000000000000000000000359434300000000000000000000000000000000000000000000000000000000001a5108db04124c0a22314e32614e665758714765396b57634c3875395459706a35527a5651626a774b41501a223151484b58534741676d43795a58686d6b6a50523465527937764865657a6569443620c0bf0220e592fa0128153089a1b992064222314e32614e665758714765396b57634c3875395459706a35527a5651626a774b41504a0f63616c6c45766d436f6e7472616374"
	var details ctypes.TransactionDetails
	err := ctypes.Decode(common.FromHex(hexTxs), &details)
	if err != nil {
		t.Log(err)
		return
	}
	qapi.On("GetTransactionByHash", mock.Anything).Return(&details, nil)

	hexhash := "0x8519b0db1e568724fd8f2d8bbce55b3ced2eaf9984f3ea08d1d8aefa328de513"
	txs, err := ethCli.GetTransactionByHash(hexhash)
	assert.Nil(t, err)
	assert.Equal(t, txs.Hash, hexhash)

	receipt, err := ethCli.GetTransactionReceipt(hexhash)
	assert.Nil(t, err)
	assert.Equal(t, "1N2aNfWXqGe9kWcL8u9TYpj5RzVQbjwKAP", receipt.From)
}
