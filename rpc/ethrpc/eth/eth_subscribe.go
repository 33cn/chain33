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
	//HeadEvent 获取block header 事件
	HeadEvent = 1
	//EvmEvent 获取evm 事件
	EvmEvent = 4
)

// NewHeads ...
// eth_subscribe
// params:["newHeads"]
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
	in.Type = HeadEvent
	stream, err := e.grpcCli.SubEvent(context.Background(), &in)
	if err != nil {
		return nil, err
	}
	go func() {

		for {
			select {
			case <-subscription.Err():
				//取消订阅
				e.grpcCli.UnSubEvent(context.Background(), &ctypes.ReqString{Data: string(subscription.ID)})
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

// Logs ...
// eth_subscribe
// params:["logs",{"address":"","topics":[""]}]
// address：要监听日志的源地址或地址数组，可选
// topics：要监听日志的主题匹配条件，可选
func (e *ethHandler) Logs(ctx context.Context, options *types.FilterQuery) (*rpc.Subscription, error) {
	log.Info("eth_subscribe", "Logs,fromBlock:", options.FromBlock, "address:", options.Address)
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return nil, rpc.ErrNotificationsUnsupported
	}
	subscription := notifier.CreateSubscription()
	logs := make(chan []*types.EvmLog)
	err := e.subscribeLogs(options, logs, subscription.ID)
	if err != nil {
		log.Error("eth_subscribe Logs", "err:", err)
		return nil, err
	}

	go func() {
		for {
			select {
			case err = <-subscription.Err():
				//取消订阅
				log.Info("eth_unsubscribe", "subid:", subscription.ID)
				e.grpcCli.UnSubEvent(context.Background(), &ctypes.ReqString{Data: string(subscription.ID)})
				return
			case evmlogs := <-logs:
				//推送到订阅者
				for _, evmlog := range evmlogs {
					if err := notifier.Notify(subscription.ID, evmlog); err != nil {
						log.Error("Logs Notify", "err", err)
						return

					}
				}

			}
		}

	}()
	return subscription, nil
}

func (e *ethHandler) subscribeLogs(options *types.FilterQuery, logs chan []*types.EvmLog, subID rpc.ID) error {

	//通过Grpc 客户端
	var in ctypes.ReqSubscribe
	in.Name = string(subID)
	in.Contract = make(map[string]bool)
	switch options.Address.(type) {
	case []interface{}:
		for _, addr := range options.Address.([]interface{}) {
			in.Contract[strings.ToLower(addr.(string))] = true
		}
	case string:
		in.Contract[strings.ToLower(options.Address.(string))] = true
	default:
		return errors.New("no support address types")
	}

	// 4 EvmEvent,2 Tx Receipt
	in.Type = EvmEvent
	fromblock, _ := hexutil.DecodeUint64(options.FromBlock)
	in.FromBlock = int64(fromblock)

	stream, err := e.grpcCli.SubEvent(context.Background(), &in)
	if err != nil {
		return err
	}
	filter, err := types.NewFilter(e.grpcCli, e.cfg, options)
	if err != nil {
		return err
	}
	go func() {

		for {

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
			logs <- evmlogs

		}
	}()

	return nil
}
