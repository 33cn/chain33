package types

import (
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/inconshreveable/log15"
	"reflect"
)

//game action ty
const (
	PBGameActionStart = iota + 1
	PBGameActionContinue
	PBGameActionQuit
	PBGameActionQuery
)

const (
	//log for PBgame
	TyLogPBGameStart    = 721
	TyLogPBGameContinue = 722
	TyLogPBGameQuit     = 723
	TyLogPBGameQuery    = 724
)

//包的名字可以通过配置文件来配置
//建议用github的组织名称，或者用户名字开头, 再加上自己的插件的名字
//如果发生重名，可以通过配置文件修改这些名字
var (
	JRPCName        = "PokerBull"
	PokerBullX      = "pokerbull"
	ExecerPokerBull = []byte(PokerBullX)
    logger = log15.New("module", "execs.pokerbull")

	actionName       = map[string]int32{
		"Start":      PBGameActionStart,
		"Continue":   PBGameActionContinue,
		"Quit":       PBGameActionQuit,
		"Query":      PBGameActionQuery,
	}
	logInfo = map[int64]*types.LogInfo{
		TyLogPBGameStart:    {reflect.TypeOf(ReceiptPBGame{}), "LogPokerBullStart"},
		TyLogPBGameContinue: {reflect.TypeOf(ReceiptPBGame{}), "LogPokerBullContinue"},
		TyLogPBGameQuit:     {reflect.TypeOf(ReceiptPBGame{}), "LogPokerBullQuit"},
		TyLogPBGameQuery:    {reflect.TypeOf(ReceiptPBGame{}), "LogPokerBullQuery"},
	}
)

const (
	//查询方法名
	FuncName_QueryGameListByIds = "QueryGameListByIds"
	FuncName_QueryGameById      = "QueryGameById"
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerPokerBull)
}
