package ticket

const (
	keyWalletAutoMiner = "WalletAutoMiner"
)

func CalcWalletAutoMiner() []byte {
	return []byte(keyWalletAutoMiner)
}
