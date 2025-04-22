package types

import (
	"errors"
	"strings"

	"github.com/33cn/chain33/common/address"

	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
)

// CommonAction ...
type CommonAction struct {
	Note   []byte //存放metamask 签名后的rawtx hexdata
	To     string //to地址（目的地址或者合约地址）
	Amount uint64 //解析组装后的chain33 tx 的amount
	Code   []byte //evm 数据
	Nonce  int64
}

// DecodeTxAction decode chain33Tx ethTx
func DecodeTxAction(msg []byte) (*CommonAction, error) {
	var tx TransactionChain33
	err := proto.Unmarshal(msg, &tx)
	if err != nil {
		return nil, err
	}
	if strings.Contains(string(tx.Execer), "evm") {
		//evm 合约操作
		var evmaction EVMAction4Chain33
		err = proto.Unmarshal(tx.Payload, &evmaction)
		if err == nil {
			var code []byte
			var to string
			if len(evmaction.GetCode()) != 0 {
				code = evmaction.GetCode() //部署合约

			} else {
				if evmaction.GetContractAddr() != address.ExecAddress(string(tx.GetExecer())) {
					code = evmaction.GetPara()
					to = evmaction.ContractAddr
				} else { //coins 转账

					to = common.Bytes2Hex(evmaction.GetPara())
				}

			}

			return &CommonAction{
				Note:   common.FromHex(evmaction.Note),
				To:     to,
				Amount: evmaction.GetAmount(),
				Code:   code,
				Nonce:  tx.Nonce,
			}, nil
		}
	}

	//coins 转账
	var coinsAction CoinsActionChain33
	err = proto.Unmarshal(tx.Payload, &coinsAction)
	if err == nil {
		transfer, ok := coinsAction.GetValue().(*CoinsActionChain33_Transfer)
		if ok {
			return &CommonAction{
				Note:   transfer.Transfer.Note,
				To:     transfer.Transfer.To,
				Amount: uint64(transfer.Transfer.GetAmount()),
				Nonce:  tx.Nonce,
			}, nil
		}
		err = errors.New("action no support")

	}

	return nil, err

}
