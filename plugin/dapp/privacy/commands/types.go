package commands

type PrivacyAccountResult struct {
	Token         string `json:"Token,omitempty"`
	Txhash        string `json:"Txhash,omitempty"`
	OutIndex      int32  `json:"OutIndex,omitempty"`
	Amount        string `json:"Amount,omitempty"`
	OnetimePubKey string `json:"OnetimePubKey,omitempty"`
}

type PrivacyAccountInfoResult struct {
	AvailableDetail []*PrivacyAccountResult `json:"AvailableDetail,omitempty"`
	FrozenDetail    []*PrivacyAccountResult `json:"FrozenDetail,omitempty"`
	AvailableAmount string                  `json:"AvailableAmount,omitempty"`
	FrozenAmount    string                  `json:"FrozenAmount,omitempty"`
	TotalAmount     string                  `json:"TotalAmount,omitempty"`
}

type PrivacyAccountSpendResult struct {
	Txhash string                  `json:"Txhash,omitempty"`
	Res    []*PrivacyAccountResult `json:"Spend,omitempty"`
}

type ShowRescanResult struct {
	Addr       string `json:"addr"`
	FlagString string `json:"FlagString"`
}

type showRescanResults struct {
	RescanResults []*ShowRescanResult `json:"ShowRescanResults,omitempty"`
}

type ShowEnablePrivacy struct {
	Results []*ShowPriAddrResult `json:"results"`
}

type ShowPriAddrResult struct {
	Addr string `json:"addr"`
	IsOK bool   `json:"IsOK"`
	Msg  string `json:"msg"`
}
