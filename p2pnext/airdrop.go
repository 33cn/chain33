package p2pnext

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/33cn/chain33/types"
)

// RandStr return a rand string
func RandStr(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r := rand.New(rand.NewSource(types.Now().Unix()))
	b := make([]rune, n)
	for i := range b {

		b[i] = letters[r.Intn(len(letters))]
	}

	return string(b)
}

func (p *P2P) genAirDropKeyFromWallet() error {
	log.Info("genAirDropKeyFromWallet", "getpub", "before")

	_, savePub := p.addrbook.GetPrivPubKey()
	log.Info("genAirDropKeyFromWallet", "getpub", savePub)

	for {
		resp, err := p.api.ExecWalletFunc("wallet", "GetWalletStatus", &types.ReqNil{})
		if err != nil {
			log.Error("genAirDropKeyFromWallet", "GetWalletStatus", err.Error())
			time.Sleep(time.Second)
			continue
		}
		log.Info("genAirDropKeyFromWallet", "resp", resp)
		if resp.(*types.WalletStatus).GetIsWalletLock() { //上锁
			if savePub == "" {
				log.Warn("P2P Stuck ! Wallet must be unlock and save with mnemonics")

			}
			time.Sleep(time.Second)
			continue
		}

		if !resp.(*types.WalletStatus).GetIsHasSeed() { //无种子
			if savePub == "" {
				log.Warn("P2P Stuck ! Wallet must be imported with mnemonics")

			}
			time.Sleep(time.Second * 5)
			continue
		}

		break
	}

	r := rand.New(rand.NewSource(types.Now().Unix()))
	var minIndex int32 = 100000000
	randIndex := minIndex + r.Int31n(1000000)
	reqIndex := &types.Int32{Data: randIndex}
	msg, err := p.api.ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)
	if err != nil {
		log.Error("genAirDropKeyFromWallet", "err", err)
		return err
	}
	var hexPrivkey string
	if reply, ok := msg.(*types.ReplyString); !ok {
		log.Error("genAirDropKeyFromWallet", "wrong format data", "")
		panic(err)

	} else {
		hexPrivkey = reply.GetData()
	}
	if hexPrivkey[:2] == "0x" {
		hexPrivkey = hexPrivkey[2:]
	}

	hexPubkey, err := GenPubkey(hexPrivkey)
	if err != nil {
		log.Error("genAirDropKeyFromWallet", "gen pub error", err)
		panic(err)
	}

	log.Info("genAirDropKeyFromWallet", "pubkey", hexPubkey)

	if savePub == hexPubkey {
		return nil
	}

	if savePub != "" {
		//priv,pub是随机公私钥对，兼容老版本，先对其进行导入钱包处理
		err = p.loadP2PPrivKeyToWallet()
		if err != nil {
			log.Error("genAirDropKeyFromWallet", "loadP2PPrivKeyToWallet error", err)
			panic(err)
		}
		p.addrbook.SaveKey(hexPrivkey, hexPubkey)
		//重启p2p模块
		log.Info("genAirDropKeyFromWallet", "p2p will Restart....")
		p.ReStart()
		return nil
	}
	//覆盖addrbook 中的公私钥对
	p.addrbook.SaveKey(hexPrivkey, hexPubkey)

	return nil
}
func (p *P2P) loadP2PPrivKeyToWallet() error {
	var parm types.ReqWalletImportPrivkey
	parm.Privkey, _ = p.addrbook.GetPrivPubKey()
	parm.Label = "node award"

ReTry:
	resp, err := p.api.ExecWalletFunc("wallet", "WalletImportPrivkey", &parm)
	if err != nil {
		if err == types.ErrPrivkeyExist {
			return nil
		}
		if err == types.ErrLabelHasUsed {
			//切换随机lable
			parm.Label = fmt.Sprintf("node award-%v", RandStr(3))
			time.Sleep(time.Second)
			goto ReTry
		}
		log.Error("loadP2PPrivKeyToWallet", "err", err.Error())
		return err
	}

	log.Debug("loadP2PPrivKeyToWallet", "resp", resp.(*types.WalletAccount))
	return nil
}
