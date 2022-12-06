package eth

import (
	"context"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/33cn/chain33/rpc/ethrpc/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	headEvent = 1
	evmEvent  = 4
)

//NewHeads ...
//eth_subscribe
//params:["newHeads"]
func (e *ethHandler) NewHeads(ctx context.Context) (*rpc.Subscription, error) {
	log.Info("eth_subscribe", "NewHeads ", "")
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()
	//通过Grpc 客户端
	var in ctypes.ReqSubscribe
	in.Name = string(subscription.ID)
	in.Type = headEvent
	stream, err := e.grpcCli.SubEvent(context.Background(), &in)
	if err != nil {
		return nil, err
	}
	go func() {

		for {
			select {
			case <-subscription.Err():
				//取消订阅
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					log.Error("NewHeads read", "err", err)
					return
				}
				eheader, _ := types.BlockHeaderToEthHeader(msg.GetHeaderSeqs().GetSeqs()[0].GetHeader())
				if err := notifier.Notify(subscription.ID, eheader); err != nil {
					log.Error("notify", "err", err)
					return

				}
			}

		}
	}()

	return subscription, nil
}

//Logs ...
//eth_subscribe
//params:["logs",{"address":"","topics":[""]}]
//address：要监听日志的源地址或地址数组，可选
//topics：要监听日志的主题匹配条件，可选
func (e *ethHandler) Logs(ctx context.Context, options *types.FilterQuery) (*rpc.Subscription, error) {
	log.Info("eth_subscribe", "Logs,fromBlock:", options.FromBlock, "address:", options.Address)
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()
	//通过Grpc 客户端
	var in ctypes.ReqSubscribe
	in.Name = string(subscription.ID)
	in.Contract = make(map[string]bool)
	switch options.Address.(type) {
	case []interface{}:
		for _, addr := range options.Address.([]interface{}) {
			in.Contract[strings.ToLower(addr.(string))] = true
		}
	case string:
		in.Contract[strings.ToLower(options.Address.(string))] = true
	default:
		return nil, errors.New("no support address types")
	}

	// 4 EvmEvent,2 Tx Receipt
	in.Type = evmEvent
	fromblock, _ := hexutil.DecodeUint64(options.FromBlock)
	in.FromBlock = int64(fromblock)

	stream, err := e.grpcCli.SubEvent(context.Background(), &in)
	if err != nil {
		return nil, err
	}
	filter, err := types.NewFilter(e.grpcCli, e.cfg, options)
	if err != nil {
		return nil, err
	}
	go func() {

		for {
			select {
			case <-subscription.Err():
				//取消订阅
				return
			default:
				msg, err := stream.Recv()
				if err != nil {
					log.Error("Logs read", "err", err)
					return
				}
				var evmlogs []*types.EvmLog
				for _, item := range msg.GetEvmLogs().GetLogs4EVMPerBlk() {
					logs := filter.FilterEvmTxLogs(item)
					evmlogs = append(evmlogs, logs...)
				}
				//推送到订阅者
				for _, lg := range evmlogs {
					if err := notifier.Notify(subscription.ID, lg); err != nil {
						log.Error("Logs Notify", "err", err)
						return

					}
				}

			}

		}
	}()

	return subscription, nil
}
