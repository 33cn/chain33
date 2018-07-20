package paracross

type ParacrossCommitTx struct {
	Fee       int64  `json:"fee"`
	Title     string `json:"title"`
	Height    int64  `json:"height"`
	StateHash string `json:"stateHash"`
}
