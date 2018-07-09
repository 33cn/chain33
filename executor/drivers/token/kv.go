package token

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/types"
)

var (
	tokenCreated          string
	tokenPreCreatedOT     string
	tokenPreCreatedSTO    string
	tokenPreCreatedOTNew  string
	tokenPreCreatedSTONew string
)

//const (
//	tokenCreated          = "mavl-token-"
//	tokenPreCreatedOT     = "mavl-create-token-ot-"
//	tokenPreCreatedSTO    = "mavl-create-token-sto-"
//	tokenPreCreatedOTNew  = "mavl-token-create-ot-"
//	tokenPreCreatedSTONew = "mavl-token-create-sto-"
//)

func setReciptPrefix() {
	tokenCreated = "mavl-" + types.ExecNamePrefix + "token-"
	tokenPreCreatedOT = "mavl-create-" + types.ExecNamePrefix + "token-ot-"
	tokenPreCreatedSTO = "mavl-create-" + types.ExecNamePrefix + "token-sto-"
	tokenPreCreatedOTNew = "mavl-" + types.ExecNamePrefix + "token-create-ot-"
	tokenPreCreatedSTONew = "mavl-" + types.ExecNamePrefix + "token-create-sto-"
}

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

func calcTokenAddrNewKey(token string, owner string) (key []byte) {
	return []byte(fmt.Sprintf(tokenPreCreatedOTNew+"%s-%s", owner, token))
}

func calcTokenStatusNewKey(token string, owner string, status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONew+"%d-%s-%s", status, token, owner))
}

func calcTokenStatusKeyNewPrefix(status int32) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONew+"%d", status))
}

func calcTokenStatusSymbolNewPrefix(status int32, token string) []byte {
	return []byte(fmt.Sprintf(tokenPreCreatedSTONew+"%d-%s-", status, token))
}
