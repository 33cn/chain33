package types

type HashlockLockTx struct {
	Secret     string `json:"secret"`
	Amount     int64  `json:"amount"`
	Time       int64  `json:"time"`
	ToAddr     string `json:"toAddr"`
	ReturnAddr string `json:"returnAddr"`
	Fee        int64  `json:"fee"`
}

type HashlockUnlockTx struct {
	Secret string `json:"secret"`
	Fee    int64  `json:"fee"`
}

type HashlockSendTx struct {
	Secret string `json:"secret"`
	Fee    int64  `json:"fee"`
}
