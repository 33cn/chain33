package wallet

const (
	keyWalletPassKey = "WalletPassKey"
)

func CalcWalletPassKey() []byte {
	return []byte(keyWalletPassKey)
}
