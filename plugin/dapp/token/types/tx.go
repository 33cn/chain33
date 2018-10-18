package types

type TokenAccountResult struct {
	Token    string `json:"Token,omitempty"`
	Currency int32  `json:"currency,omitempty"`
	Balance  string `json:"balance,omitempty"`
	Frozen   string `json:"frozen,omitempty"`
	Addr     string `json:"addr,omitempty"`
}
