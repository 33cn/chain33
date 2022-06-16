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

	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
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
	t.Log("balance", balanceHexStr.String())
	assert.Equal(t, balanceHexStr.String(), "0x4563918244f40000")
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
	assert.Equal(t, block.Header.Number.String(), "0x46")

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
	assert.Equal(t, num.String(), block.Header.Number.String())
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
	qapi.On("GetBlockHash", mock.Anything).Return(&ctypes.ReplyHash{Hash: common.FromHex("0x2685bc8e4cfcee3b307d58c58ce752b3f4a9d76dfda84f7a940040e3cc907a29")}, nil)
	hexhash := "0x8519b0db1e568724fd8f2d8bbce55b3ced2eaf9984f3ea08d1d8aefa328de513"

	txs, err := ethCli.GetTransactionByHash(common.HexToHash(hexhash))
	assert.Nil(t, err)
	assert.Equal(t, txs.Hash.String(), hexhash)
	assert.Equal(t, txs.BlockHash.String(), "0x2685bc8e4cfcee3b307d58c58ce752b3f4a9d76dfda84f7a940040e3cc907a29")
	_, err = ethCli.GetTransactionReceipt(common.HexToHash(hexhash))
	assert.Nil(t, err)

}

func TestEthHandler_GetCode(t *testing.T) {
	var ret struct {
		Creator  string         ` json:"creator,omitempty"`
		Name     string         ` json:"name,omitempty"`
		Alias    string         ` json:"alias,omitempty"`
		Addr     string         ` json:"addr,omitempty"`
		Code     *hexutil.Bytes ` json:"code,omitempty"`
		CodeHash []byte         ` json:"codeHash,omitempty"`
		// 绑定ABI数据 ForkEVMABI
		Abi string `json:"abi,omitempty"`
	}
	code := common.FromHex("0x6080604052600436106100555760003560e01c8063481c6a751461005a5780635d495aea146100855780635ec01e4d1461009c578063e97dcb62146100c7578063efa1c482146100d1578063f71d96cb146100fc575b600080fd5b34801561006657600080fd5b5061006f610139565b60405161007c91906107d0565b60405180910390f35b34801561009157600080fd5b5061009a61015d565b005b3480156100a857600080fd5b506100b161033b565b6040516100be919061084d565b60405180910390f35b6100cf610371565b005b3480156100dd57600080fd5b506100e661041f565b6040516100f391906107eb565b60405180910390f35b34801561010857600080fd5b50610123600480360381019061011e91906105a8565b6104ad565b60405161013091906107d0565b60405180910390f35b60008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b60008054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146101eb576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101e29061082d565b60405180910390fd5b60006001805490506101fb61033b565b610205919061096a565b905060018181548110610241577f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b9060005260206000200160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166108fc479081150290604051600060405180830381858888f193505050501580156102b1573d6000803e3d6000fd5b50600067ffffffffffffffff8111156102f3577f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b6040519080825280602002602001820160405280156103215781602001602082028036833780820191505090505b50600190805190602001906103379291906104ec565b5050565b60004442600160405160200161035393929190610797565b6040516020818303038152906040528051906020012060001c905090565b662386f26fc1000034116103ba576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103b19061080d565b60405180910390fd5b6001339080600181540180825580915050600190039060005260206000200160009091909190916101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b606060018054806020026020016040519081016040528092919081815260200182805480156104a357602002820191906000526020600020905b8160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019060010190808311610459575b5050505050905090565b600181815481106104bd57600080fd5b906000526020600020016000915054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b828054828255906000526020600020908101928215610565579160200282015b828111156105645782518260006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055509160200191906001019061050c565b5b5090506105729190610576565b5090565b5b8082111561058f576000816000905550600101610577565b5090565b6000813590506105a2816109ea565b92915050565b6000602082840312156105ba57600080fd5b60006105c884828501610593565b91505092915050565b60006105dd8383610601565b60208301905092915050565b60006105f5838361061f565b60208301905092915050565b61060a8161090a565b82525050565b6106198161090a565b82525050565b6106288161090a565b82525050565b60006106398261088d565b61064381856108bd565b935061064e83610868565b8060005b8381101561067f57815161066688826105d1565b9750610671836108a3565b925050600181019050610652565b5085935050505092915050565b600061069782610898565b6106a181856108ce565b93506106ac83610878565b8060005b838110156106e4576106c1826109ca565b6106cb88826105e9565b97506106d6836108b0565b9250506001810190506106b0565b5085935050505092915050565b60006106fe601a836108d9565b91507f746865206d696e20616d6f756e7420697320302e3031206574680000000000006000830152602082019050919050565b600061073e601b836108d9565b91507f796f7520617265206e6f7420616c6c6f77656420746f207069636b00000000006000830152602082019050919050565b61077a8161093c565b82525050565b61079161078c8261093c565b610960565b82525050565b60006107a38286610780565b6020820191506107b38285610780565b6020820191506107c3828461068c565b9150819050949350505050565b60006020820190506107e56000830184610610565b92915050565b60006020820190508181036000830152610805818461062e565b905092915050565b60006020820190508181036000830152610826816106f1565b9050919050565b6000602082019050818103600083015261084681610731565b9050919050565b60006020820190506108626000830184610771565b92915050565b6000819050602082019050919050565b60008190508160005260206000209050919050565b600081519050919050565b600081549050919050565b6000602082019050919050565b6000600182019050919050565b600082825260208201905092915050565b600081905092915050565b600082825260208201905092915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006109158261091c565b9050919050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b6000610959610954836109dd565b6108ea565b9050919050565b6000819050919050565b60006109758261093c565b91506109808361093c565b9250826109905761098f61099b565b5b828206905092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601260045260246000fd5b60006109d68254610946565b9050919050565b60008160001c9050919050565b6109f38161093c565b81146109fe57600080fd5b5056fea26469706673582212200803e202ec4ea4add02f57a64f3b9931a5363412f1190e8221074375d2ad81d764736f6c63430008000033")
	ret.Code = (*hexutil.Bytes)(&code)
	qapi.On("Query", mock.Anything).Return(&ret, nil)
	caddr := "0xd01c479dee5e61c52ded7422a634220ba91e2447"
	addr := common.HexToAddress(caddr)
	getCode, err := ethCli.GetCode(&addr, "")
	assert.Nil(t, err)
	t.Log("retCode", getCode)
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

	t.Log("sigdata:", common.Bytes2Hex(sig))
	//通过哈希和签名数据恢复出公钥
	pub, err := crypto.Ecrecover(hash, sig)
	if err != nil {
		t.Log("err", err)
		return
	}
	t.Log("pub", common.Bytes2Hex(pub))
	assert.Equal(t, "046d21eac1b7a11fdeb7e0b759e0f479240a8a3630d191808c072792611408c4580e5f57b96d9f073eb00f9355c301f31bf274705da1a0f969434106ed26cd7570", common.Bytes2Hex(pub))
	epub, _ := crypto.UnmarshalPubkey(pub)
	addr := crypto.PubkeyToAddress(*epub).String()
	t.Log("addr:", addr)
	assert.Equal(t, "0xd741c9f9e0A1F5bb1ed898115A683253F14c1F8b", addr)
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
	iargs = append(iargs, common.HexToAddress(from), common.HexToAddress(to), big.NewInt(int64(value)), big.NewInt(1654849059), sigdata[:])
	packdata, err := eabi.Pack("transferFromWithSignature", iargs...)
	assert.Nil(t, err)
	t.Log("packdata", common.Bytes2Hex(packdata))

}

var abidata = "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"supply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"uint8\",\"name\":\"decimals_\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"chainid_\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"hashStruct\",\"type\":\"bytes32\"},{\"indexed\":true,\"internalType\":\"bytes32\",\"name\":\"hash\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"Log\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DOMAIN_SEPARATOR\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"PERMIT_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"TRANSFERFROM_TYPEHASH\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"nonces\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"permit\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"deadline\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"}],\"name\":\"transferFromWithSignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
