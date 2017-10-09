package rpc

import "code.aliyun.com/chain33/chain33/types"

//提供系统rpc接口
//sendTx
//status

type IClient interface {
	SendTx(tx []byte) (err error)
	Status() (status *types.ChainStatus, err error)
	GetBlock(start int, end int) (blocks []*types.Block, err error)
}

type channelClient struct {
}

type jsonClient struct {
}

type grpcClient struct {
}

func NewClient(name string) IClient {
	if name == "channel" {
		return &channelClient{}
	} else if name == "jsonrpc" {
		return &jsonClient{}
	} else if name == "grpc" {
		return &grpcClient{}
	}
	panic("client name not support")
}

//channel
func (client *channelClient) SendTx(tx []byte) (err error) {
	return nil
}

func (client *channelClient) Status() (status *types.ChainStatus, err error) {
	return nil, nil
}

func (client *channelClient) GetBlock(start int, end int) (blocks []*types.Block, err error) {
	return nil, nil
}

//grpc
func (client *grpcClient) SendTx(tx []byte) (err error) {
	return nil
}

func (client *grpcClient) Status() (status *types.ChainStatus, err error) {
	return nil, nil
}

func (client *grpcClient) GetBlock(start int, end int) (blocks []*types.Block, err error) {
	return nil, nil
}

//jsonrpc
func (client *jsonClient) SendTx(tx []byte) (err error) {
	return nil
}

func (client *jsonClient) Status() (status *types.ChainStatus, err error) {
	return nil, nil
}

func (client *jsonClient) GetBlock(start int, end int) (blocks []*types.Block, err error) {
	return nil, nil
}
