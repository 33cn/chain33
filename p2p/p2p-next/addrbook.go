package p2pnext

import (
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
		a.setKey(string(privkey), a.genPubkey(string(privkey)))
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

func (a *AddrBook) SaveKey(key, tag string) {
	a.setKey(privkey, pubkey)
	err := a.bookDb.Set([]byte(privKeyTag), []byte(privkey))
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
		log.Error("CryPto Error", "Error", err.Error())
		return nil, nil, err
	}

	key, err := cr.GenKey()
	if err != nil {
		log.Error("GenKey", "Error", err)
		return nil, nil, err
	}
	return key.Bytes(), key.PubKey().Bytes(), nil
}
