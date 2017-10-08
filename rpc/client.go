package rpc

import "code.aliyun.com/chain33/chain33/types"

//提供系统rpc接口
//sendTx
//status

type IClient struct {
	SendTx(tx []byte) (err error)
	Status() (status *types.ChainStatus, err error)
	GetBlock(start int, end int) ([]*types.Block, err error)
}
