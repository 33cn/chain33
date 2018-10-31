package executor

import (
	"fmt"
)

var (
	tokenCreated          = "mavl-token-"
	tokenPreCreatedOT     = "mavl-create-token-ot-"
	tokenPreCreatedSTO    = "mavl-create-token-sto-"
	tokenPreCreatedOTNew  = "mavl-token-create-ot-"
	tokenPreCreatedSTONew = "mavl-token-create-sto-"

	tokenPreCreatedSTONewLocal = "LODB-token-create-sto-"
)

func calcTokenKey(token string) (key []byte) {
	return []byte(fmt.Sprintf(tokenCreated+"%s", token))
}

func calcTokenAddrKeyS(token string, owner string) (key []byte) {
	return []byte(fmt.Sprintf(tokenPreCreatedOT+"%s-%s", owner, token))
}

func calcTokenStatusKey(token string, owner string, status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTO+"%d-%s-%s", status, token, owner))
}

func calcTokenAddrNewKeyS(token string, owner string) (key []byte) {
	return []byte(fmt.Sprintf(tokenPreCreatedOTNew+"%s-%s", owner, token))
}

func calcTokenStatusNewKeyS(token string, owner string, status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONew+"%d-%s-%s", status, token, owner))
}

func calcTokenStatusKeyLocal(token string, owner string, status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONewLocal+"%d-%s-%s", status, token, owner))
}

func calcTokenStatusKeyPrefixLocal(status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONewLocal+"%d", status))
}

func calcTokenStatusTokenKeyPrefixLocal(status int32, token string) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONewLocal+"%d-%s-", status, token))
}

//存储地址上收币的信息
func calcAddrKey(token string, addr string) []byte {
	return []byte(fmt.Sprintf("LODB-token-%s-Addr:%s", token, addr))
}
