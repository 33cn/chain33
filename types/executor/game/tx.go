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
