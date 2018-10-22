package types

//game action ty
const (
	GameActionCreate = iota + 1
	GameActionMatch
	GameActionCancel
	GameActionClose

	//log for game
	TyLogCreateGame = 711
	TyLogMatchGame  = 712
	TyLogCancleGame = 713
	TyLogCloseGame  = 714
)

//包的名字可以通过配置文件来配置
//建议用github的组织名称，或者用户名字开头, 再加上自己的插件的名字
//如果发生重名，可以通过配置文件修改这些名字
var (
	GameX      = "game"
	ExecerGame = []byte(GameX)
)

const (
	Action_CreateGame = "createGame"
	Action_MatchGame  = "matchGame"
	Action_CancelGame = "cancelGame"
	Action_CloseGame  = "closeGame"
)

const (
	FuncName_QueryGameListByIds           = "QueryGameListByIds"
	FuncName_QueryGameListCount           = "QueryGameListCount"
	FuncName_QueryGameListByStatusAndAddr = "QueryGameListByStatusAndAddr"
	FuncName_QueryGameById                = "QueryGameById"
)
