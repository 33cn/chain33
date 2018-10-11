package types

type TokenPreCreateTx struct {
	Price        int64  `json:"price"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Introduction string `json:"introduction"`
	OwnerAddr    string `json:"ownerAddr"`
	Total        int64  `json:"total"`
	Fee          int64  `json:"fee"`
	ParaName     string `json:"paraName"`
}

type TokenFinishTx struct {
	OwnerAddr string `json:"ownerAddr"`
	Symbol    string `json:"symbol"`
	Fee       int64  `json:"fee"`
	ParaName  string `json:"paraName"`
}

type TokenRevokeTx struct {
	OwnerAddr string `json:"ownerAddr"`
	Symbol    string `json:"symbol"`
	Fee       int64  `json:"fee"`
	ParaName  string `json:"paraName"`
}

type TokenAccountResult struct {
	Token    string `json:"Token,omitempty"`
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}