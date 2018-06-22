package wallet

import (
	"sync/atomic"

	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

func (wallet *Wallet) setAutoMining(flag int32) {
	atomic.StoreInt32(&wallet.autoMinerFlag, flag)
}

func (wallet *Wallet) isAutoMining() bool {
	return atomic.LoadInt32(&wallet.autoMinerFlag) == 1
}

//检查周期 --> 10分
//开启挖矿：
//1. 自动把成熟的ticket关闭
//2. 查找超过1万余额的账户，自动购买ticket
//3. 查找mineraddress 和他对应的 账户的余额（不在1中），余额超过1万的自动购买ticket 挖矿
//
//停止挖矿：
//1. 自动把成熟的ticket关闭
//2. 查找ticket 可取的余额
//3. 取出ticket 里面的钱
func (wallet *Wallet) autoMining() {
	defer wallet.wg.Done()
	for {
		select {
		case <-wallet.miningTicket.C:
			if wallet.cfg.GetMinerdisable() {
				break
			}
			if !(wallet.IsCaughtUp() || wallet.cfg.GetForceMining()) {
				walletlog.Error("wallet IsCaughtUp false")
				break
			}
			//判断高度是否增长
			height := wallet.GetHeight()
			if height <= wallet.lastHeight {
				walletlog.Error("wallet Height not inc")
				break
			}
			wallet.lastHeight = height
			walletlog.Info("BEG miningTicket")
			if wallet.isAutoMining() {
				n1, err := wallet.closeTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("closeTicket", "err", err)
				}
				err = wallet.processFees()
				if err != nil {
					walletlog.Error("processFees", "err", err)
				}
				hashes1, n2, err := wallet.buyTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("buyTicket", "err", err)
				}
				hashes2, n3, err := wallet.buyMinerAddrTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("buyMinerAddrTicket", "err", err)
				}
				hashes := append(hashes1, hashes2...)
				if len(hashes) > 0 {
					wallet.waitTxs(hashes)
				}
				if n1+n2+n3 > 0 {
					wallet.flushTicket()
				}
			} else {
				n1, err := wallet.closeTicket(wallet.lastHeight + 1)
				if err != nil {
					walletlog.Error("closeTicket", "err", err)
				}
				err = wallet.processFees()
				if err != nil {
					walletlog.Error("processFees", "err", err)
				}
				hashes, err := wallet.withdrawFromTicket()
				if err != nil {
					walletlog.Error("withdrawFromTicket", "err", err)
				}
				if len(hashes) > 0 {
					wallet.waitTxs(hashes)
				}
				if n1 > 0 {
					wallet.flushTicket()
				}
			}
			walletlog.Info("END miningTicket")
		case <-wallet.done:
			return
		}
	}
}

func (wallet *Wallet) buyTicket(height int64) ([][]byte, int, error) {
	privs, err := wallet.getAllPrivKeys()
	if err != nil {
		walletlog.Error("buyTicket.getAllPrivKeys", "err", err)
		return nil, 0, err
	}
	count := 0
	var hashes [][]byte
	for _, priv := range privs {
		hash, n, err := wallet.buyTicketOne(height, priv)
		if err != nil {
			walletlog.Error("buyTicketOne", "err", err)
			continue
		}
		count += n
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes, count, nil
}

func (wallet *Wallet) buyMinerAddrTicket(height int64) ([][]byte, int, error) {
	privs, err := wallet.getAllPrivKeys()
	if err != nil {
		walletlog.Error("buyMinerAddrTicket.getAllPrivKeys", "err", err)
		return nil, 0, err
	}
	count := 0
	var hashes [][]byte
	for _, priv := range privs {
		hashlist, n, err := wallet.buyMinerAddrTicketOne(height, priv)
		if err != nil {
			if err != types.ErrNotFound {
				walletlog.Error("buyMinerAddrTicketOne", "err", err)
			}
			continue
		}
		count += n
		if hashlist != nil {
			hashes = append(hashes, hashlist...)
		}
	}
	return hashes, count, nil
}

func (wallet *Wallet) withdrawFromTicket() (hashes [][]byte, err error) {
	privs, err := wallet.getAllPrivKeys()
	if err != nil {
		walletlog.Error("withdrawFromTicket.getAllPrivKeys", "err", err)
		return nil, err
	}
	for _, priv := range privs {
		hash, err := wallet.withdrawFromTicketOne(priv)
		if err != nil {
			walletlog.Error("withdrawFromTicketOne", "err", err)
			continue
		}
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes, nil
}

func (wallet *Wallet) closeTicket(height int64) (int, error) {
	return wallet.closeAllTickets(height)
}

func (wallet *Wallet) forceCloseTicket(height int64) (*types.ReplyHashes, error) {
	return wallet.forceCloseAllTicket(height)
}

func (wallet *Wallet) flushTicket() {
	walletlog.Info("wallet FLUSH TICKET")
	hashList := wallet.client.NewMessage("consensus", types.EventFlushTicket, nil)
	wallet.client.Send(hashList, false)
}

func (wallet *Wallet) needFlushTicket(tx *types.Transaction, receipt *types.ReceiptData) bool {
	if receipt.Ty != types.ExecOk || string(tx.Execer) != "ticket" {
		return false
	}
	pubkey := tx.Signature.GetPubkey()
	addr := address.PubKeyToAddress(pubkey)
	return wallet.AddrInWallet(addr.String())
}
