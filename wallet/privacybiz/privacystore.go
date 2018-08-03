package privacybiz

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"

	"github.com/golang/protobuf/proto"
)

const (
	Privacy4Addr     = "Privacy4Addr"
	AvailUTXOs       = "UTXO"
	FrozenUTXOs      = "FTXOs4Tx"
	PrivacySTXO      = "STXO-"
	PrivacyTokenMap  = "PrivacyTokenMap"
	STXOs4Tx         = "STXOs4Tx"
	RevertSendtx     = "RevertSendtx"
	RecvPrivacyTx    = "RecvPrivacyTx"
	SendPrivacyTx    = "SendPrivacyTx"
	ScanPrivacyInput = "ScanPrivacyInput-"
	ReScanUtxosFlag  = "ReScanUtxosFlag-"
)

// calcPrivacyAddrKey 获取隐私账户私钥对保存在钱包中的索引串
func calcPrivacyAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s", Privacy4Addr, addr))
}

//calcAddrKey 通过addr地址查询Account账户信息
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("Addr:%s", addr))
}

// calcPrivacyUTXOPrefix4Addr 获取指定地址下可用UTXO信息索引的KEY值前缀
func calcPrivacyUTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-", AvailUTXOs, token, addr))
}

// calcFTXOsKeyPrefix 获取指定地址下由于交易未被确认而让交易使用到的UTXO处于冻结状态信息的KEY值前缀
func calcFTXOsKeyPrefix(token, addr string) []byte {
	var prefix string
	if len(token) > 0 && len(addr) > 0 {
		prefix = fmt.Sprintf("%s:%s-%s-", FrozenUTXOs, token, addr)
	} else if len(token) > 0 {
		prefix = fmt.Sprintf("%s:%s-", FrozenUTXOs, token)
	} else {
		prefix = fmt.Sprintf("%s:", FrozenUTXOs)
	}
	return []byte(prefix)
}

// privacyStore 隐私交易数据库存储操作类
type privacyStore struct {
	db db.DB
}

func (store *privacyStore) getAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {
	if len(addr) == 0 {
		bizlog.Error("getAccountByPrefix addr is nil")
		return nil, types.ErrInputPara
	}
	list := db.NewListHelper(store.db)
	accbytes := list.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		bizlog.Error("getAccountByPrefix addr not exist")
		return nil, types.ErrAccountNotExist
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := proto.Unmarshal(accbyte, &walletaccount)
		if err != nil {
			bizlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

func (store *privacyStore) getWalletAccountPrivacy(addr string) (*types.WalletAccountPrivacy, error) {
	if len(addr) == 0 {
		bizlog.Error("GetWalletAccountPrivacy addr is nil")
		return nil, types.ErrInputPara
	}

	privacyByte, err := store.db.Get(calcPrivacyAddrKey(addr))
	if err != nil {
		bizlog.Error("GetWalletAccountPrivacy", "db Get error ", err)
		return nil, err
	}
	if nil == privacyByte {
		return nil, types.ErrPrivacyNotEnabled
	}
	var accPrivacy types.WalletAccountPrivacy
	err = proto.Unmarshal(privacyByte, &accPrivacy)
	if err != nil {
		bizlog.Error("GetWalletAccountPrivacy", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &accPrivacy, nil
}

func (store *privacyStore) getAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		bizlog.Error("GetAccountByAddr addr is nil")
		return nil, types.ErrInputPara
	}
	data, err := store.db.Get(calcAddrKey(addr))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			bizlog.Debug("GetAccountByAddr addr", "err", err)
		}
		return nil, types.ErrAddrNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		bizlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

func (store *privacyStore) setWalletAccountPrivacy(addr string, privacy *types.WalletAccountPrivacy) error {
	if len(addr) == 0 {
		bizlog.Error("SetWalletAccountPrivacy addr is nil")
		return types.ErrInputPara
	}
	if privacy == nil {
		bizlog.Error("SetWalletAccountPrivacy privacy is nil")
		return types.ErrInputPara
	}

	privacybyte, err := proto.Marshal(privacy)
	if err != nil {
		bizlog.Error("SetWalletAccountPrivacy proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	newbatch := store.db.NewBatch(true)
	store.db.Set(calcPrivacyAddrKey(addr), privacybyte)
	newbatch.Write()

	return nil
}
func (store *privacyStore) listAvailableUTXOs(token, addr string) ([]*types.PrivacyDBStore, error) {
	if 0 == len(addr) {
		bizlog.Error("listWalletPrivacyAccount addr is nil")
		return nil, types.ErrInputPara
	}

	list := db.NewListHelper(store.db)
	onetimeAccbytes := list.PrefixScan(calcPrivacyUTXOPrefix4Addr(token, addr))
	if len(onetimeAccbytes) == 0 {
		bizlog.Error("listWalletPrivacyAccount ", "addr not exist", addr)
		return nil, nil
	}

	privacyDBStoreSlice := make([]*types.PrivacyDBStore, len(onetimeAccbytes))
	for index, acckeyByte := range onetimeAccbytes {
		var accPrivacy types.PrivacyDBStore
		accByte, err := store.db.Get(acckeyByte)
		if err != nil {
			bizlog.Error("listWalletPrivacyAccount", "db Get err:", err)
			return nil, err
		}
		err = proto.Unmarshal(accByte, &accPrivacy)
		if err != nil {
			bizlog.Error("listWalletPrivacyAccount", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		privacyDBStoreSlice[index] = &accPrivacy
	}
	return privacyDBStoreSlice, nil
}

func (store *privacyStore) listFrozenUTXOs(token, addr string) ([]*types.FTXOsSTXOsInOneTx, error) {
	if 0 == len(addr) {
		bizlog.Error("listFrozenUTXOs addr is nil")
		return nil, types.ErrInputPara
	}
	list := db.NewListHelper(store.db)
	values := list.List(calcFTXOsKeyPrefix(token, addr), nil, 0, 0)
	if len(values) == 0 {
		bizlog.Error("listFrozenUTXOs ", "addr not exist", addr)
		return nil, nil
	}

	ftxoslice := make([]*types.FTXOsSTXOsInOneTx, 0)
	for _, acckeyByte := range values {
		var ftxotx types.FTXOsSTXOsInOneTx
		accByte, err := store.db.Get(acckeyByte)
		if err != nil {
			bizlog.Error("listFrozenUTXOs", "db Get err:", err)
			return nil, err
		}

		err = proto.Unmarshal(accByte, &ftxotx)
		if err != nil {
			bizlog.Error("listFrozenUTXOs", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		ftxoslice = append(ftxoslice, &ftxotx)
	}
	return ftxoslice, nil

}
