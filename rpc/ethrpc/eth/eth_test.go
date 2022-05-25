package eth

import (
	"encoding/json"
	"errors"
	"fmt"
	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	etypes "github.com/33cn/chain33/rpc/ethrpc/types"
	"github.com/33cn/chain33/system/dapp/coins/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"strings"
	"testing"
	"time"
)

var (
	ethCli *ethHandler
	qapi   *clientMocks.QueueProtocolAPI

	q = queue.New("test")
)

func init() {
	qapi = &clientMocks.QueueProtocolAPI{}
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	q.SetConfig(cfg)
	ethCli = &ethHandler{}
	ethCli.cfg = cfg
	ethCli.cli.Init(q.Client(), qapi)

}

func TestEthHandler_Accounts(t *testing.T) {
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

func TestEthHandler_BlockNumber(t *testing.T) {
	var header ctypes.Header
	header.Height = 102400
	qapi.On("GetLastHeader", mock.Anything).Return(&header, nil)
	bnum, err := ethCli.BlockNumber()
	assert.Nil(t, err)
	t.Log("blocknum:", bnum)
	bn := big.NewInt(header.Height)
	assert.Equal(t, hexutil.EncodeBig(bn), bnum.String())
}

func TestEthHandler_Mining(t *testing.T) {
	var status = new(ctypes.WalletStatus)
	status.IsAutoMining = true
	qapi.On("ExecWalletFunc", "wallet", "GetWalletStatus", &ctypes.ReqNil{}).Return(status, nil)
	mining, err := ethCli.Mining()
	assert.Nil(t, err)
	t.Log("mining:", mining)
	assert.Equal(t, true, mining)
}

func TestEthHandler_GetBalance(t *testing.T) {
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

func TestEthHandler_GetBlock(t *testing.T) {
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

	testEthAPIGetBlockByHash(t, &resp)
	testEthAPIGetBlockByNumber(t, &resp)

}
func testEthAPIGetBlockByHash(t *testing.T, resp *ctypes.BlockDetails) {
	var hash = "0x660f78e492bf2630ecd4d8fdf09ec64f0e141bdfeb7636ed4992b31dd81338bd"
	var req ctypes.ReqHashes
	req.Hashes = append(req.Hashes, common.FromHex(hash))
	qapi.On("GetBlockByHashes", &req).Return(resp, nil)
	block, err := ethCli.GetBlockByHash(common.HexToHash(hash), true)
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Number, "0x46")

}

func testEthAPIGetBlockByNumber(t *testing.T, resp *ctypes.BlockDetails) {
	var req ctypes.ReqBlocks
	req.Start = 70
	req.End = 70
	req.IsDetail = true
	qapi.On("GetBlocks", &req).Return(resp, nil)
	num := (*hexutil.Big)(big.NewInt(req.Start))
	block, err := ethCli.GetBlockByNumber(num.String(), true)
	assert.Nil(t, err)
	assert.Equal(t, num.String(), block.Header.Number)
}

func TestEthHandler_ChainId(t *testing.T) {
	id, err := ethCli.ChainId()
	assert.Nil(t, err)
	assert.Equal(t, id.String(), "0x21")
}

func TestEthHandler_EstimateGas(t *testing.T) {

	qapi.On("Query", mock.Anything).Return("0x12345", nil)
	var msg etypes.CallMsg
	msg.From = "1N2aNfWXqGe9kWcL8u9TYpj5RzVQbjwKAP"
	msg.To = "1QHKXSGAgmCyZXhmkjPR4eRy7vHeezeiD6"
	msg.Value = (*hexutil.Big)(big.NewInt(1))
	_, err := ethCli.EstimateGas(&msg)
	assert.NotNil(t, err)
}

func TestEthHandler_GetTransactionCount(t *testing.T) {
	qapi.On("Query", mock.Anything).Return("0x12", nil)
	_, err := ethCli.GetTransactionCount("1N2aNfWXqGe9kWcL8u9TYpj5RzVQbjwKAP", "latest")
	assert.NotNil(t, err)

}

func TestEthHandler_GetTxCount(t *testing.T) {
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
	count, err := ethCli.GetBlockTransactionCountByHash(common.HexToHash(hash))
	assert.Nil(t, err)
	assert.Equal(t, 1, int(count))

	qapi.On("GetBlocks", mock.Anything).Return(blockdetails, nil)
	count, err = ethCli.GetBlockTransactionCountByNumber((*hexutil.Big)(big.NewInt(70)))
	assert.Nil(t, err)
	assert.Equal(t, 1, int(count))
}

func TestEthHandler_GetTransaction(t *testing.T) {
	hexTxs := "0x0a8d110ad4030a0365766d12e8012a647750c9f0000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c4512000000000000000000000000096cf8ba17671e7aa4f76fb81813a6d713ba866b00000000000000000000000000000000000000000000000000005b21a46300803a5c6c6f636b28314e32614e665758714765396b57634c3875395459706a35527a5651626a774b41502c20317271644375547970557979645747674e7a6676314376794b474d364d456b78472c203130303230303035303030303030302942223151484b58534741676d43795a58686d6b6a50523465527937764865657a656944361a6e0801122103c404be076a4ffd87ce494d54029e840810a9e06e6dbfb72b8cb2d0906d1b02851a473045022100c2a822a60bfb0212197c86442a14a44619cd0c1447051708df70a56c81cadfd802200f7a3b71c7f0f817e57c8b1bc747f5d394535557e259be9698e517f9e883099230d6838ba9b18c9bee343a223139746a5335316b6a7772436f535153313355336f7765376759424c6653666f466d40054a201ff66560942c87e94112b99970915354b09f80e813d2c10ad41b47829862080c5220c5bba5174b2788b4241700fa9f50d4fe2b936fc7bcffac21afefeb9aa6e6517d12f10c08021abf0108dc0412b9010a734c4f44422d65766d2d73746174653a317271644375547970557979645747674e7a6676314376794b474d364d456b78473a307833313539636239376466353336306134643363386335386535613836613465323735383432366235623930313432396632633232613138353536363564343638122000000000000000000000000000000000000000000000000000005b21a46300801a2000000000000000000000000000000000000000000000000000000000000000001abf0108dc0412b9010a734c4f44422d65766d2d73746174653a317271644375547970557979645747674e7a6676314376794b474d364d456b78473a3078653330346636323139636439383166653532356234646536653462646165373165373838396262316363343831336162373334393635393536363630666437321220000000000000000000000000000000000000000000000000000ac23e4baaeb401a20000000000000000000000000000000000000000000000000000b1d5ff00debc01a8e0108dd041288010a20ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef0a20000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c45120a20000000000000000000000000ff610318e64e7f41517a689700bd9823a02736aa122000000000000000000000000000000000000000000000000000005b21a46300801abf0108dc0412b9010a734c4f44422d65766d2d73746174653a317271644375547970557979645747674e7a6676314376794b474d364d456b78473a307830373937623864333334386535323930646230643937303235653761303366656237633533303332616134343232633734613732663164346331633561326232122000000000000000000000000000000000000000000000000000005b21a46300801a2000000000000000000000000000000000000000000000000000000000000000001a8e0108dd041288010a208c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b9250a20000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c45120a20000000000000000000000000ff610318e64e7f41517a689700bd9823a02736aa122000000000000000000000000000000000000000000000000000000000000000001ac00108dc0412ba010a744c4f44422d65766d2d73746174653a3151484b58534741676d43795a58686d6b6a50523465527937764865657a656944363a307830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303038122000000000000000000000000000000000000000000000000000000000000003311a2000000000000000000000000000000000000000000000000000000000000003321ac00108dc0412ba010a744c4f44422d65766d2d73746174653a3151484b58534741676d43795a58686d6b6a50523465527937764865657a656944363a3078343030653030613561643837326264656533653536633231656561313064333564613366303665393730326333663463613366366534626564333332663165321220000000000000000000000000000000000000000000000000000ac23e4baaeb401a20000000000000000000000000000000000000000000000000000b1d5ff00debc01aab0208dd0412a5020a204150c77f33761fbde38a274b3cbecce67bc2060ac09687dd6eab0cffe90dfefe128002000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c4512000000000000000000000000e6a6ea8e06bad72af51c04e1ee930b40cc4c4512000000000000000000000000096cf8ba17671e7aa4f76fb81813a6d713ba866b00000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000005b21a46300800000000000000000000000000000000000000000000000000000000000000332000000000000000000000000000000000000000000000000000000000000000359434300000000000000000000000000000000000000000000000000000000001a5108db04124c0a22314e32614e665758714765396b57634c3875395459706a35527a5651626a774b41501a223151484b58534741676d43795a58686d6b6a50523465527937764865657a6569443620c0bf0220e592fa0128153089a1b992064222314e32614e665758714765396b57634c3875395459706a35527a5651626a774b41504a0f63616c6c45766d436f6e7472616374"
	var details ctypes.TransactionDetails
	err := ctypes.Decode(common.FromHex(hexTxs), &details)
	if err != nil {
		t.Log(err)
		return
	}
	qapi.On("GetTransactionByHash", mock.Anything).Return(&details, nil)

	hexhash := "0x8519b0db1e568724fd8f2d8bbce55b3ced2eaf9984f3ea08d1d8aefa328de513"

	txs, err := ethCli.GetTransactionByHash(common.HexToHash(hexhash))
	assert.Nil(t, err)
	assert.Equal(t, txs.Hash, hexhash)

	receipt, err := ethCli.GetTransactionReceipt(common.HexToHash(hexhash))
	assert.Nil(t, err)
	assert.Equal(t, "1N2aNfWXqGe9kWcL8u9TYpj5RzVQbjwKAP", receipt.From)
}

func Test_Eip712SignTypeData(t *testing.T) {
	deadline := big.NewInt(time.Now().Unix() + 3600)
	v := big.NewInt(1e8)
	n := big.NewInt(0)
	id := big.NewInt(22)
	t.Log("deadline:", deadline.Int64())
	var testTypedData = &apitypes.TypedData{
		Domain: apitypes.TypedDataDomain{
			Name:              "WLT",
			Version:           "1",
			ChainId:           (*math.HexOrDecimal256)(id),
			VerifyingContract: "0xde17a85756815bf5755173603f2b07e69455f654",
		},
		Message: apitypes.TypedDataMessage{
			"sender":    "0xd741c9f9e0A1F5bb1ed898115A683253F14c1F8b",
			"recipient": "0xDe79A84DD3A16BB91044167075dE17a1CA4b1d6b",
			"value":     (*math.HexOrDecimal256)(v),
			"nonce":     (*math.HexOrDecimal256)(n),
			"deadline":  (*math.HexOrDecimal256)(big.NewInt(1653046056)),
		},
		PrimaryType: "TransferFrom",
		Types: apitypes.Types{
			"EIP712Domain": {
				{
					Name: "name",
					Type: "string",
				},
				{
					Name: "version",
					Type: "string",
				},
				{
					Name: "chainId",
					Type: "uint256",
				},
				{
					Name: "verifyingContract",
					Type: "address",
				},
			},
			"TransferFrom": {
				{
					Name: "sender",
					Type: "address",
				},
				{
					Name: "recipient",
					Type: "address",
				},
				{
					Name: "value",
					Type: "uint256",
				},
				{
					Name: "nonce",
					Type: "uint256",
				},
				{
					Name: "deadline",
					Type: "uint256",
				},
			},
		},
	}

	jbs, _ := json.MarshalIndent(testTypedData, "", "\t")
	t.Log("jsonstr:", string(jbs))
	var privKey = "427da8655959736f02d0e4e557a6c343e5ccc20e8516c3980bf948b430d511fb"
	pk, err := crypto.ToECDSA(common.FromHex(privKey))
	if err != nil {
		t.Log(err)
		return
	}
	encodedData, err := testTypedData.EncodeData("EIP712Domain", testTypedData.Domain.Map(), 1)
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("encodeData:", common.Bytes2Hex(encodedData))
	domainSeparator, err := testTypedData.HashStruct("EIP712Domain", testTypedData.Domain.Map())
	if err != nil {
		t.Log(err)
		return
	}

	t.Log("domain hash", domainSeparator.String())
	typedDataHash, err := testTypedData.HashStruct(testTypedData.PrimaryType, testTypedData.Message)
	if err != nil {
		t.Log(err)
		return
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hash := crypto.Keccak256(rawData)
	sig, err := crypto.Sign(hash, pk)
	if err != nil {
		t.Log(err)
		return
	}
	//sig[64] += 27
	t.Log("sigdata:", common.Bytes2Hex(sig))
	//sig[64] -= 27
	pub, err := crypto.Ecrecover(hash, sig)
	if err != nil {
		t.Log("err", err)

		return
	}
	t.Log("pub", common.Bytes2Hex(pub))
	epub, _ := crypto.UnmarshalPubkey(pub)
	t.Log("addr:", crypto.PubkeyToAddress(*epub).String())
}

func TestSendSignTypeData(t *testing.T) {
	eabi, _ := abi.JSON(strings.NewReader(abidata))
	//parameter := fmt.Sprintf("transferFromWithSignature(%s, %s,%d,%d,%s)")
	/*
		transferFromWithSignature(
		        address sender,
		        address recipient,
		        uint256 amount,
		        uint256 deadline,
		        bytes memory signature
		    )

	*/

	var from = "0xd741c9f9e0A1F5bb1ed898115A683253F14c1F8b"
	var to = "0xDe79A84DD3A16BB91044167075dE17a1CA4b1d6b"
	var value = 1e8
	var sig = "d6266993b09c1ae68df69e417b431466778e595467c1dd40ae31bab6ea974f0e3485679b965d3f411077e443ba1b39fd098a5a8d4507aaa91af767106164a4b501"
	t.Log(len(common.FromHex(sig)))
	var sigdata [65]byte
	copy(sigdata[:], common.Hex2Bytes(sig)[:])
	//param:= fmt.Sprintf("transferFromWithSignature(%s, %d ,%d, %d ,%s)",from,to,value,time.Now().Unix()+3600,sig)
	var iargs []interface{}
	iargs = append(iargs, common.HexToAddress(from), common.HexToAddress(to), big.NewInt(int64(value)), big.NewInt(1653046056), sigdata[:])
	packdata, err := eabi.Pack("transferFromWithSignature", iargs...)
	if err != nil {
		t.Log("err", err)

		//return
	}
	t.Log("packdata", common.Bytes2Hex(packdata))
	return
	sigb := common.Hex2Bytes(sig)
	sigb[64] -= 27
	pub, err := crypto.Ecrecover(crypto.Keccak256(packdata), sigb)
	if err != nil {
		t.Log("err", err)

		return
	}
	t.Log("pub", common.Bytes2Hex(pub))
	epub, _ := crypto.UnmarshalPubkey(pub)
	t.Log("addr:", crypto.PubkeyToAddress(*epub).String())
	return
	packdata, err = eabi.Pack("nonces", common.HexToAddress(from))
	if err != nil {
		t.Log("err", err)

		//return
	}
	t.Log("packdata", common.Bytes2Hex(packdata))
	//hexpack := "c9bf726b000000000000000000000000d741c9f9e0a1f5bb1ed898115a683253f14c1f8b000000000000000000000000d741c9f9e0a1f5bb1ed898115a683253f14c1f8b0000000000000000000000000000000000000000000000000000000005f5e100000000000000000000000000000000000000000000000000000000006287161b00000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000416fc58c5db9182eeaae423a3a8138ae530595293211ced84468baf11fa0bb3f143a66cae70fcbaffe09a87ced31e8147a983b06d1475007cd79bd73e6ae5a76810100000000000000000000000000000000000000000000000000000000000000"

}

var abidata = "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"supply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"decimals_\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"chainid_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"hashStruct\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"Log\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PERMIT_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TRANSFERFROM_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"transferFromWithSignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
