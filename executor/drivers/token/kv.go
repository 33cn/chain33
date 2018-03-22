package token

import "fmt"

const (
	tokenCreated       = "mavl-token-"
	tokenPreCreatedOT  = "mavl-create-token-ot-"
	tokenPreCreatedSTO = "mavl-create-token-sto-"
)

func calcTokenKey(token string) (key []byte) {
	return []byte(fmt.Sprintf(tokenCreated+"%s", token))
}

func calcTokenAddrKey(token string, owner string) (key []byte) {
	return []byte(fmt.Sprintf(tokenPreCreatedOT+"%s-%s", owner, token))
}

func calcTokenStatusKey(token string, owner string, status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTO+"%d-%s-%s", status, token, owner))
}

func calcTokenStatusKeyPrefix(status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTO+"%d", status))
}

func calcTokenStatusSymbolPrefix(status int32, token string) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTO+"%d-%s-", status, token))
}
