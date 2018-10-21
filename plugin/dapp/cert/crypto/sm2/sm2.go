package sm2

import (
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	"gitlab.33.cn/chain33/chain33/system/crypto/sm2"
)

type sm2Driver struct {
	sm2.Driver
}

func init() {
	crypto.Register(ty.SignNameAuthSM2, &sm2Driver{})
}
