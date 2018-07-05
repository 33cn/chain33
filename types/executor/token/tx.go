package token

type TokenPreCreateTx struct {
	Price        int64  `json:"price"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Introduction string `json:"introduction"`
	OwnerAddr    string `json:"ownerAddr"`
	Total        int64  `json:"total"`
	Fee          int64  `json:"fee"`
}

type TokenFinishTx struct {
	OwnerAddr string `json:"ownerAddr"`
	Symbol    string `json:"symbol"`
	Fee       int64  `json:"fee"`
}

type TokenRevokeTx struct {
	OwnerAddr string `json:"ownerAddr"`
	Symbol    string `json:"symbol"`
	Fee       int64  `json:"fee"`
}
