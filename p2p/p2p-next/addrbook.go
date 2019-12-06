package p2p_next

import (
	"encoding/hex"
	"sync"

	"github.com/33cn/chain33/types"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
)

const (
	addrkeyTag = "addrs"
	privKeyTag = "privkey"
)

// AddrBook peer address manager
type AddrBook struct {
	mtx     sync.Mutex
	cfg     *types.P2P
	privkey string
	pubkey  string
	bookDb  db.DB
	Quit    chan struct{}
}

func NewAddrBook(cfg *types.P2P) *AddrBook {
	a := &AddrBook{
		cfg:  cfg,
		Quit: make(chan struct{}, 1),
	}

	if !a.loadDb() {
		a.initKey()
	}
	return a

}

func (a *AddrBook) loadDb() bool {
	a.bookDb = db.NewDB("addrbook", a.cfg.Driver, a.cfg.DbPath, a.cfg.DbCache)
	privkey, err := a.bookDb.Get([]byte(privKeyTag))
	if len(privkey) != 0 && err == nil {
		pubkey, err := genPubkey(string(privkey))
		if err != nil {
			logger.Error("genPubkey", "err", err.Error())
			panic(err)
		}
		a.setKey(string(privkey), pubkey)
		return true
	}

	return false

}

func (a *AddrBook) initKey() {

	priv, pub, err := GenPrivPubkey()
	if err != nil {
		var maxRetry = 10
		for i := 0; i < maxRetry; i++ {
			priv, pub, err = GenPrivPubkey()
			if err == nil {
				break
			}
			if i == maxRetry-1 && err != nil {
				panic(err.Error())
			}
		}

	}

	a.setKey(hex.EncodeToString(priv), hex.EncodeToString(pub))
}

func (a *AddrBook) SaveKey(priv, pub string) {
	a.setKey(priv, pub)
	err := a.bookDb.Set([]byte(privKeyTag), []byte(priv))
	if err != nil {
		panic(err)
	}
}

// GetPrivPubKey return privkey and pubkey
func (a *AddrBook) GetPrivPubKey() (string, string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return a.privkey, a.pubkey
}

func (a *AddrBook) setKey(privkey, pubkey string) {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	a.privkey = privkey
	a.pubkey = pubkey

}

// GenPrivPubkey return key and pubkey in bytes
func GenPrivPubkey() ([]byte, []byte, error) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		logger.Error("CryPto Error", "Error", err.Error())
		return nil, nil, err
	}

	key, err := cr.GenKey()
	if err != nil {
		logger.Error("GenKey", "Error", err)
		return nil, nil, err
	}
	return key.Bytes(), key.PubKey().Bytes(), nil
}

func genPubkey(privkey string) (string, error) {

	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		logger.Error("CryPto Error", "Error", err.Error())
		return "", err
	}

	pribyts, err := hex.DecodeString(privkey)
	if err != nil {
		logger.Error("DecodeString Error", "Error", err.Error())
		return "", err
	}
	priv, err := cr.PrivKeyFromBytes(pribyts)
	if err != nil {
		logger.Error("Load PrivKey", "Error", err.Error())
		return "", err
	}

	return hex.EncodeToString(priv.PubKey().Bytes()), nil

}
