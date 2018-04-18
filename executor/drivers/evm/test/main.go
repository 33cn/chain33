package main

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/account"
)

func getAddr() *account.Address{
	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		return nil
	}
	key, err := c.GenKey()
	if err != nil {
		return nil
	}
	return account.PubKeyToAddress(key.PubKey().Bytes())
}
func main() {
	vm := evm.NewFakeEVM()

	tx := types.Transaction{[]byte("evm"), []byte("code"), nil, int64(1000),int64(0),int64(0),"to"}


	vm.Exec(&tx,0)
}

