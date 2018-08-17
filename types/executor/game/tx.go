package game

type GamePreCreateTx struct {
	//Secret     string `json:"secret"`
	//下注必须时偶数，不能时级数
	Amount int64 `json:"amount"`
	//暂时只支持sha256加密
	HashType  string `json:"hashType"`
	HashValue []byte `json:"hashValue,omitempty"`
	Fee       int64  `json:"fee"`
}

type GamePreMatchTx struct {
	GameId string `json:"gameId"`
	Guess  int32  `json:"guess"`
	Fee    int64  `json:"fee"`
}

type GamePreCancelTx struct {
	GameId string `json:"gameId"`
	Fee    int64  `json:"fee"`
}

type GamePreCloseTx struct {
	GameId string `json:"gameId"`
	Secret string `json:"secret"`
	Result int32  `json:"result"`
	Fee    int64  `json:"fee"`
}

type Game struct {
	// 默认是由创建这局游戏的txHash作为gameId
	GameId string `json:"gameId"`
	// create 1 -> Match 2 -> Cancel 3 -> Close 4
	Status int32 `json:"status"`
	// 创建时间
	CreateTime int64 `json:"createTime"`
	// 匹配时间(何时参与对赌）
	MatchTime int64 `json:"matchTime"`
	// 状态close的时间（包括cancel）
	Closetime int64 `json:"closetime"`
	// 赌注
	Value int64 `json:"value"`
	// 发起者账号地址
	CreateAddress string `json:"createAddress"`
	// 对赌者账号地址
	MatchAddress string `json:"matchAddress"`
	// hash 类型，预留字段
	HashType string `json:"hashType"`
	// 庄家创建游戏时，庄家自己出拳结果加密后的hash值
	HashValue []byte `json:"hashValue"`
	// 用来公布庄家出拳结果的私钥
	Secret string `json:"secret"`
	// 0 平局，1 庄家获胜，2 matcher获胜，3 庄家开奖超时，matcher获胜，并获得本局所有赌资
	Result int32 `json:"result"`
	// matcher 出拳结果
	Guess int32 `json:"guess"`
}

type ReplyGameList struct {
	Games []*Game `json:"games"`
}

type ReplyGame struct {
	Game *Game `json:"game"`
}
