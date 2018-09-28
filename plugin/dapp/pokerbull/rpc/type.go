package rpc

import (
	"encoding/hex"
	"encoding/json"

	log "github.com/inconshreveable/log15"
	pb "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

var pblog = log.New("module", pb.PokerBullX)
var name string
var jrpc = &Jrpc{}
var grpc = &Grpc{}

func InitRPC(s pluginmgr.RPCServer) {
	cli := channelClient{}
	cli.Init(s.GetQueueClient())
	jrpc.cli = cli
	grpc.channelClient = cli
	s.JRPC().RegisterName(pb.JRPCName, jrpc)
	pb.RegisterPokerbullServer(s.GRPC(), grpc)
}

func Init(s pluginmgr.RPCServer) {
	name = pb.PokerBullX
	// init executor type
	types.RegistorExecutor(name, &PokerBullType{})

	// init log
	types.RegistorLog(pb.TyLogPBGameStart, &PokerBullStartLog{})
	types.RegistorLog(pb.TyLogPBGameContinue, &PokerBullContinueLog{})
	types.RegistorLog(pb.TyLogPBGameQuit, &PokerBullQuitLog{})
	types.RegistorLog(pb.TyLogPBGameQuery, &PokerBullQueryLog{})

	// init query rpc
	//types.RegisterRPCQueryHandle(pb.FuncName_QueryGameListByIds, &GameGetList{})
	//types.RegisterRPCQueryHandle(pb.FuncName_QueryGameById, &GameGetInfo{})

	InitRPC(s)
}

type PokerBullType struct {
	types.ExecTypeBase
}

func (m PokerBullType) ActionName(tx *types.Transaction) string {
	var g pb.PBGameAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-Pokerbull-action-err"
	}
	if g.Ty == pb.PBGameActionStart && g.GetStart() != nil {
		return "PokerbullStart"
	} else if g.Ty == pb.PBGameActionContinue && g.GetContinue() != nil {
		return "PokerbullContinue"
	} else if g.Ty == pb.PBGameActionQuit && g.GetQuit() != nil {
		return "PokerbullQuit"
	} else if g.Ty == pb.PBGameActionQuery && g.GetQuery() != nil {
		return "PokerbullQuery"
	}
	return "unkown"
}

func (m PokerBullType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action pb.PBGameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (m PokerBullType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (m PokerBullType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	pblog.Debug("Pokerbull.CreateTx", "action", action)
	var tx *types.Transaction
	return tx, nil
}

type PokerBullStartLog struct {
}

func (l PokerBullStartLog) Name() string {
	return "LogPokerBullStart"
}

func (l PokerBullStartLog) Decode(msg []byte) (interface{}, error) {
	var logTmp pb.ReceiptPBGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type PokerBullContinueLog struct {
}

func (l PokerBullContinueLog) Name() string {
	return "LogPokerBullContinue"
}

func (l PokerBullContinueLog) Decode(msg []byte) (interface{}, error) {
	var logTmp pb.ReceiptPBGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type PokerBullQuitLog struct {
}

func (l PokerBullQuitLog) Name() string {
	return "LogPokerBullQuit"
}

func (l PokerBullQuitLog) Decode(msg []byte) (interface{}, error) {
	var logTmp pb.ReceiptPBGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type PokerBullQueryLog struct {
}

func (l PokerBullQueryLog) Name() string {
	return "LogPokerBullQuery"
}

func (l PokerBullQueryLog) Decode(msg []byte) (interface{}, error) {
	var logTmp pb.ReceiptPBGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type PokerBullStartTxRPC struct{}

func (t *PokerBullStartTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBStartTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullStartTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type PokerBullContinueTxRPC struct {
}

func (t *PokerBullContinueTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBContinueTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullContinueTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type PokerBullQuitTxRPC struct {
}

func (t *PokerBullQuitTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBQuitTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullQuitTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type PokerBullQueryTxRPC struct {
}

func (t *PokerBullQueryTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBQuitTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullQueryTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}
