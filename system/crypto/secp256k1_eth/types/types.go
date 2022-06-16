package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/golang/protobuf/proto"
	"strings"
)

//import "github.com/33cn/chain33/types"

type CommonAction struct {
	Note   []byte
	To     string
	Amount uint64
	Code   []byte
}

func DecodeTxAction(msg []byte) *CommonAction {
	var tx TransactionChain33
	err := proto.Unmarshal(msg, &tx)
	if err != nil {
		return nil
	}
	if strings.Contains(string(tx.Execer), "evm") {
		//evm 合约操作
		var evmaction EVMAction4Chain33
		err = proto.Unmarshal(tx.Payload, &evmaction)
		if err == nil {
			var code []byte
			if len(evmaction.GetCode()) != 0 {
				code = evmaction.GetCode()
			} else {
				code = evmaction.GetPara()
			}
			return &CommonAction{
				Note:   common.FromHex(evmaction.Note),
				To:     evmaction.ContractAddr,
				Amount: evmaction.GetAmount(),
				Code:   code,
			}
		}
	} else {
		//coins 转账
		var coinAction CoinsActionChain33
		err = proto.Unmarshal(tx.Payload, &coinAction)
		if err == nil {
			transfer := coinAction.GetValue().(*CoinsActionChain33_Transfer)
			return &CommonAction{
				Note:   transfer.Transfer.Note,
				To:     transfer.Transfer.To,
				Amount: uint64(transfer.Transfer.GetAmount()),
			}
		}
	}
	return nil

}
