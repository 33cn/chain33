package test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"gitlab.33.cn/wallet/bipwallet"
)

func TestBipwallet(t *testing.T) {
	/*目前暂时支持这些币种创建地址

	    TypeEther:        "ETH",
		TypeEtherClassic: "ETC",
		TypeBitcoin:      "BTC",
		TypeLitecoin:     "LTC",
		TypeZayedcoin:    "ZEC",
		TypeBty:          "BTY",
		TypeYcc:          "YCC",
	*/
	//选择币种，填入种子创建wallet对象
	wallet, err := bipwallet.NewWalletFromMnemonic(bipwallet.TypeEther,
		"wish address cram damp very indicate regret sound figure scheme review scout")
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}
	var index uint32 = 0
	//通过索引生成Key pair
	priv, pub, err := wallet.NewKeyPair(index)
	fmt.Println("privkey:", hex.EncodeToString(priv))
	fmt.Println("pubkey:", hex.EncodeToString(pub))
	//通过索引生成对应的地址
	address, err := wallet.NewAddress(index)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("address:", address)

}
