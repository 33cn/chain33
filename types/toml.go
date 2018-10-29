package types

type ConfigSubModule struct {
	Store     map[string][]byte
	Exec      map[string][]byte
	Consensus map[string][]byte
	Wallet    map[string][]byte
}
