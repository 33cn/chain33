package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/33cn/chain33/common/address"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet"
	"github.com/33cn/chain33/wallet/bipwallet"
)

var seed = flag.String("seed", "", "source seed")
var targetaddr = flag.String("addr", "", "address of target")

func main() {
	flag.Parse()
	wallet.InitSeedLibrary()
	log.Println("seed", *seed)
	log.Println("target", *targetaddr)
	seedlist := strings.Split(*seed, " ")
	//第一种情况，用户写错一个字
	n := 0
	for k := range wallet.ChineseSeedCache {
		log.Println("change ", k, int(100*(float64(n)/float64(2048))))
		n++
		var seeds []string
		for i := 0; i < len(seedlist); i++ {
			item := seedlist[i]
			seedlist[i] = k
			newseed := strings.Join(seedlist, " ")
			seeds = append(seeds, newseed)
			seedlist[i] = item
		}
		checkmultithread(seeds)
	}
	log.Println("not found")
}

func checkmultithread(seeds []string) {
	done := make(chan struct{}, len(seeds))
	for i := 0; i < len(seeds); i++ {
		go func(seed string) {
			checkseed(seed)
			done <- struct{}{}
		}(seeds[i])
	}
	for i := 0; i < len(seeds); i++ {
		<-done
	}
}

func checkseed(newseed string) {
	addrlist, err := genaddrlist(newseed)
	if err != nil {
		return
	}
	if _, ok := addrlist[*targetaddr]; ok {
		log.Println("find new seed", newseed)
		os.Exit(0)
	}
}
func genaddrlist(seed string) (map[string]bool, error) {
	wallet, err := bipwallet.NewWalletFromMnemonic(bipwallet.TypeBty, seed)
	if err != nil {
		//log.Println("GetPrivkeyBySeed NewWalletFromMnemonic", "err", err)
		wallet, err = bipwallet.NewWalletFromSeed(bipwallet.TypeBty, []byte(seed))
		if err != nil {
			log.Println("GetPrivkeyBySeed NewWalletFromSeed", "err", err)
			return nil, types.ErrNewWalletFromSeed
		}
	}
	addrlist := make(map[string]bool)
	for index := 0; index <= 5; index++ {
		//通过索引生成Key pair
		_, pub, err := wallet.NewKeyPair(uint32(index))
		if err != nil {
			log.Println("GetPrivkeyBySeed NewKeyPair", "err", err)
			return nil, types.ErrNewKeyPair
		}
		addr := address.PubKeyToAddress(pub)
		addrlist[addr.String()] = true
	}
	return addrlist, err
}
