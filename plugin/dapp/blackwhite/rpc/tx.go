package rpc

type BlackwhiteCreateTx struct {
	PlayAmount  int64  `json:"amount"`
	PlayerCount int32  `json:"playerCount"`
	Timeout     int64  `json:"timeout"`
	GameName    string `json:"gameName"`
	Fee         int64  `json:"fee"`
}

type BlackwhitePlayTx struct {
	GameID     string   `json:"gameID"`
	Amount     int64    `json:"amount"`
	HashValues [][]byte `json:"hashValues"`
	Fee        int64    `json:"fee"`
}

type BlackwhiteShowTx struct {
	GameID string `json:"gameID"`
	Secret string `json:"secret"`
	Fee    int64  `json:"fee"`
}

type BlackwhiteTimeoutDoneTx struct {
	GameID string `json:"GameID"`
	Fee    int64  `json:"fee"`
}
