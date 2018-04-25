package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"

	l "github.com/inconshreveable/log15"
	"github.com/rs/cors"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"

	"math/rand"
	"time"

	"encoding/hex"
)

var (
	log      = l.New("module", "token-approver")
	minFee   = types.MinFee
	signType = "secp256k1"
)

// 独立的服务， 提供两个功能
//   1. 帮忙做审核token的交易签名
//       1. a帐号有审核的权限， 客服核完需要用他的私钥对审核交易进行签名
//       1. 输入是生成好的， tokenfinishtx 类型的交易 （也可以选择输入 owner, symbol， 费用固定在server端 ）
//       1. 输出是签过名的交易
//   1. 给指定帐号打手续费    1bty
//       1. 给指定帐号打手续费
//       1. 给审核帐号打手续费
//  实现基于http 的 json rpc
//    app-proto
//      |
//      V
//     rpc      json format req/resp
//      |
//      V
//     http     serve listener --> conn, conn.io -> request, response
//      |
//      V
//     tcp ...  构建 listener （socket 支持io）
type HTTPConn struct {
	in  io.Reader
	out io.Writer
}

func (c *HTTPConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HTTPConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }
func (c *HTTPConn) Close() error                      { return nil }

type Signatory struct {
	privkey string
}

func (*Signatory) Close() {

}

func (*Signatory) Listen() {

}

func (*Signatory) Echo(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	*out = *in
	return nil
}

type TokenFinish struct {
	OwnerAddr string `json:"owner_addr"`
	Symbol    string `json:"symbol"`
	//	Fee       int64  `json:"fee"`
}

func (signatory *Signatory) SignApprove(in *TokenFinish, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	if checkAddr(in.OwnerAddr) != true || len(in.Symbol) == 0 {
		return types.ErrInputPara
	}
	v := &types.TokenFinishCreate{Symbol: in.Symbol, Owner: in.OwnerAddr}
	finish := &types.TokenAction{
		Ty:    types.TokenActionFinishCreate,
		Value: &types.TokenAction_Tokenfinishcreate{v},
	}

	tx := &types.Transaction{
		Execer:  []byte("token"),
		Payload: types.Encode(finish),
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      account.ExecAddress("token").String(),
	}

	var err error
	tx.Fee, err = tx.GetRealFee(minFee)
	if err != nil {
		log.Error("SignApprove", "calc fee failed", err)
		return err
	}
	err = signTx(tx, signatory.privkey)
	if err != nil {
		return err
	}
	txHex := types.Encode(tx)
	*out = hex.EncodeToString(txHex)
	return nil
}

func (signatory *Signatory) SignTranfer(in *string, out *interface{}) error {
	if in == nil {
		return types.ErrInputPara
	}
	if checkAddr(*in) != true{
		return types.ErrInputPara
	}

	amount := 1 * types.Coin
	v := &types.CoinsTransfer{
		Amount: amount,
		Note: "transfer 1 bty by signatory-server",
	}
	transfer := &types.CoinsAction{
		Ty: types.CoinsActionTransfer,
		Value: &types.CoinsAction_Transfer{v},
	}


	tx := &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(transfer),
		To:      *in,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
	}

	var err error
	tx.Fee, err = tx.GetRealFee(minFee)
	if err != nil {
		log.Error("SignTranfer", "calc fee failed", err)
		return err
	}
	err = signTx(tx, signatory.privkey)
	if err != nil {
		log.Error("SignTranfer", "signTx failed", err)
		return err
	}
	txHex := types.Encode(tx)
	*out = hex.EncodeToString(txHex)
	return nil

}


func main() {
	tcpAddr := "localhost:8888"
	listen, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}

	approver := Signatory{""} // 这样骨架就有了， 需要在这里加功能
	server := rpc.NewServer()
	server.Register(&approver)
	// type HandlerFunc func(ResponseWriter, *Request)
	var handler http.Handler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL, r.Header, r.Body)
			if r.URL.Path == "/" {
				serverCodec := jsonrpc.NewServerCodec(&HTTPConn{in: r.Body, out: w}) // wrap http to rpc
				w.Header().Set("Content-type", "application/json")
				w.WriteHeader(200)

				err := server.ServeRequest(serverCodec) // 处理io
				if err != nil {
					log.Debug("Error while serving JSON request: %v", err)
					return
				}
			}
		})

	co := cors.New(cors.Options{}) // TOOD 看看这个库是做什么的
	handler = co.Handler(handler)

	http.Serve(listen, handler)

	fmt.Println(handler)

}

func signTx(tx *types.Transaction, hexPrivKey string) error {
	signType := types.SECP256K1
	c, err := crypto.New(types.GetSignatureTypeName(signType))

	bytes, err := common.FromHex(hexPrivKey)
	if err != nil {
		log.Error("signTx", "err", err)
		return err
	}

	privKey, err := c.PrivKeyFromBytes(bytes)
	if err != nil {
		log.Error("signTx", "err", err)
		return err
	}

	tx.Sign(int32(signType), privKey)
	return nil
}

func checkAddr(addr string) bool {
	return true // TODO
}
