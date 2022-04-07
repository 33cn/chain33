package ethrpc

import (
	"errors"
	clientMocks "github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/dapp/coins/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"math/big"
	"testing"
)

var(

	//mockEthObj *mocks.Eth
	//tojb *Tobj
	ethOjb *EthApi
	qapi *clientMocks.QueueProtocolAPI
	q = queue.New("test")
)


func init(){
	//mockEthObj=&mocks.Eth{}
	qapi =&clientMocks.QueueProtocolAPI{}
	cfg:=ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	q.SetConfig(cfg)
	ethOjb=&EthApi{}
	ethOjb.cfg=cfg
	ethOjb.cli.Init(q.Client(),qapi)

}



func TestEthApi_Accounts(t *testing.T) {
	var account =&ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xa42431Da868c58877a627CC71Dc95F01bf40c196"},Label: "test1"}
	var account2 =&ctypes.WalletAccount{Acc: &ctypes.Account{Addr: "0xBe807Dddb074639cD9fA61b47676c064fc50D62C"},Label: "test2"}
	var resp =&ctypes.WalletAccounts{
		Wallets: []*ctypes.WalletAccount{account,account2},
	}
	req := &ctypes.ReqAccountList{WithoutBalance: true}
	qapi.On("ExecWalletFunc","wallet","WalletGetAccountList",req).Return(nil,errors.New("wrong param"))
	accs,err:= ethOjb.Accounts()
	assert.NotNil(t, err)

	qapi =&clientMocks.QueueProtocolAPI{}
	qapi.On("ExecWalletFunc","wallet","WalletGetAccountList",req).Return(resp,nil)
	var addrs =[]string{"0xa42431Da868c58877a627CC71Dc95F01bf40c196","0xBe807Dddb074639cD9fA61b47676c064fc50D62C"}
	ethOjb.cli.Init(q.Client(),qapi)
	accs,err = ethOjb.Accounts()
	assert.Nil(t, err)
	assert.Equal(t, accs,addrs)
}

func TestEthApi_BlockNumber(t *testing.T) {
	var header ctypes.Header
	header.Height=102400
	qapi.On("GetLastHeader",mock.Anything).Return(&header,nil)
	bnum,err:= ethOjb.BlockNumber()
	assert.Nil(t, err)
	t.Log("blocknum:",bnum)
	bn:=big.NewInt(header.Height)
	assert.Equal(t, hexutil.EncodeBig(bn),bnum)
}

func TestEthApi_Mining(t *testing.T) {
	var status =new(ctypes.WalletStatus)
	status.IsAutoMining=true
	qapi.On("ExecWalletFunc","wallet","GetWalletStatus",&ctypes.ReqNil{}).Return(status,nil)
	mining,err:= ethOjb.Mining()
	assert.Nil(t, err)
	t.Log("mining:",mining)
	assert.Equal(t, true,mining)
}

func TestEthApi_GetBalance(t *testing.T) {
	var req ctypes.ReqBalance
	req.Execer="coins"
	req.AssetSymbol="bty"
	addr:="0x1E79307966B830bCdfEAB1Ed09871d248c2fE171"
	req.Addresses=[]string{addr}

	var header ctypes.Header
	header.Height=102400
	head := &ctypes.Header{StateHash: []byte("sdfadasds")}
	qapi.On("GetLastHeader").Return(head, nil)
	qapi.On("GetConfig").Return(ethOjb.cfg, nil)
	var acc = &ctypes.Account{Addr: "0x1E79307966B830bCdfEAB1Ed09871d248c2fE171", Balance: 5e8}
	accv := ctypes.Encode(acc)
	storevalue := &ctypes.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	qapi.On("StoreGet", mock.Anything).Return(storevalue, nil)

	var addrs = make([]string, 1)
	addrs = append(addrs, "0x1E79307966B830bCdfEAB1Ed09871d248c2fE171")

	var tag ="latest"
	balanceHexStr,err:= ethOjb.GetBalance(addr,&tag)
	assert.Nil(t, err)
	t.Log("balance",balanceHexStr)
	assert.Equal(t, balanceHexStr,"0x1dcd6500")
}

func TestEthApi_GetBlock(t*testing.T){
	var resp ctypes.BlockDetails
	var item ctypes.BlockDetail

	resp.Items=append(resp.GetItems(),&item)

	transfer:= &ctypes.AssetsTransfer{
		Cointoken: "",
		Amount: 102454299999,
		To: "",
		Note:nil,
	}

	v := &types.CoinsAction_Transfer{Transfer:transfer}
	action := &types.CoinsAction{Value: v, Ty:types.CoinsActionTransfer}
	item.Block=&ctypes.Block{
		Height: 70,
		BlockTime: 1648792721,
		Version: 0,
		ParentHash: common.FromHex("25bfcebbeedd16039bb22f0e15702f9e7a0f6d8a3ae05e2e7e6fd7ce66babb36"),
		TxHash: common.FromHex("0xfe4a45a5b946e7de1d6c0bcd3f47d0d4b68bb252009bd032f9102854346c3217"),
		Txs: []*ctypes.Transaction{{
			Execer: []byte("coins"),
			Payload:
				ctypes.Encode(action),
			Expire: 0,
			Nonce: 2871810575193867206,
			ChainID: 88,
			To: "0xde79a84dd3a16bb91044167075de17a1ca4b1d6b",
			Signature: &ctypes.Signature{
				Ty: 1,
				Pubkey:common.FromHex("0x02f5263862dae4e8516e08e5551f353be05f479c4d36ea754a7b2b359f81fbebb1"),
				Signature: common.FromHex("0x30450221008259d987850c34036a33ede7db25ea72dbe92edd3e9bfe6cbcabd0c43c427a9902206f410d921d6b0d09ce9a8cdf0db9f8eec64677c56404fcd2ccf0e2c5ef4dd06a"),
			},
		}},
		Difficulty: 520159231,
		MainHash: common.FromHex("0x660f78e492bf2630ecd4d8fdf09ec64f0e141bdfeb7636ed4992b31dd81338bd"),
		MainHeight: 70,
	}

	testEthApi_GetBlockByHash(t,&resp)
	testEthApi_GetBlockByNumber(t,&resp)

}
func testEthApi_GetBlockByHash(t *testing.T,resp *ctypes.BlockDetails) {
	var hash="0x660f78e492bf2630ecd4d8fdf09ec64f0e141bdfeb7636ed4992b31dd81338bd"
	var req ctypes.ReqHashes
	req.Hashes=append(req.Hashes,common.FromHex(hash))
	qapi.On("GetBlockByHashes",&req).Return(resp,nil)
	block,err:= ethOjb.GetBlockByHash(hash,true)
	assert.Nil(t, err)
	assert.Equal(t, block.Header.Number,"0x46")

}

func testEthApi_GetBlockByNumber(t*testing.T,resp *ctypes.BlockDetails){
	var req ctypes.ReqBlocks
	req.Start= 70
	req.End= 70
	req.IsDetail=true
	qapi.On("GetBlocks",&req).Return(resp,nil)

	block,err:= ethOjb.GetBlockByNumber("0x46",true)
	assert.Nil(t, err)
	//assert.Equal(t, block.Header.Number,"0x46")
	t.Log("block",block.Header)
}


func TestEthApi_ChainId(t *testing.T) {
	id,err:= ethOjb.ChainId()
	assert.Nil(t, err)
	assert.Equal(t, id,"0x21")
}




func TestWeb3_Sha3(t *testing.T) {
	w:=&Web3{}
	hash,err:= w.Sha3("0x68656c6c6f20776f726c64")
	if err!=nil{
		t.Log("err:",err)
		return
	}

	t.Log("hash:",hash)
	assert.Equal(t, hash,"0x47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad")
}


