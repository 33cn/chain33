package p2pnext

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"sync"

	//"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
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
	a.bookDb = db.NewDB("addrbook", a.cfg.Driver, a.cfg.DbPath, a.cfg.DbCache)

	if !a.loadDb() {
		a.initKey()
	}
	return a

}

func (a *AddrBook) AddrsInfo() []peer.AddrInfo {
	var addrsInfo []peer.AddrInfo
	if a.bookDb == nil {
		return addrsInfo
	}

	addrsInfoBs, err := a.bookDb.Get([]byte(addrkeyTag))
	if err != nil {
		logger.Error("LoadAddrsInfo", " Get addrkeyTag", err.Error())
		return addrsInfo
	}

	err = json.Unmarshal(addrsInfoBs, &addrsInfo)
	if err != nil {
		logger.Error("LoadAddrsInfo", "Unmarshal err", err.Error())
	}

	return addrsInfo
}

func (a *AddrBook) loadDb() bool {

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

	//loading addrinfos

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

	a.SaveKey(hex.EncodeToString(priv), hex.EncodeToString(pub))
}

func (a *AddrBook) SaveKey(priv, pub string) {
	a.setKey(priv, pub)
	err := a.bookDb.Set([]byte(privKeyTag), []byte(priv))
	if err != nil {
		panic(err)
	}
}

func (a *AddrBook) GetPrivkey() p2pcrypto.PrivKey {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	keybytes, err := hex.DecodeString(a.privkey)
	if err != nil {
		logger.Error("GetPrivkey", "DecodeString Error", err.Error())
		return nil
	}
	privkey, err := p2pcrypto.UnmarshalPrivateKey(keybytes)
	if err != nil {
		logger.Error("GetPrivkey", "PrivKeyUnmarshaller", err.Error())
		return nil
	}
	return privkey
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

func (a *AddrBook) SaveAddr(addrinfos []peer.AddrInfo) {

	jsonBytes, err := json.Marshal(addrinfos)
	if err != nil {
		logger.Error("Failed to save AddrBook to file", "err", err)
		return
	}
	logger.Debug("saveToDb", "addrs", string(jsonBytes))
	err = a.bookDb.Set([]byte(addrkeyTag), jsonBytes)

}

// GenPrivPubkey return key and pubkey in bytes
func GenPrivPubkey() ([]byte, []byte, error) {
	priv, pub, err := p2pcrypto.GenerateKeyPairWithReader(p2pcrypto.Secp256k1, 2048, rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	privkey, err := priv.Bytes()
	if err != nil {
		return nil, nil, err

	}
	pubkey, err := pub.Bytes()
	if err != nil {
		return nil, nil, err

	}
	return privkey, pubkey, nil

}

func genPubkey(key string) (string, error) {
	keybytes, err := hex.DecodeString(key)
	if err != nil {
		logger.Error("DecodeString Error", "Error", err.Error())
		return "", err
	}
	privkey, err := p2pcrypto.UnmarshalPrivateKey(keybytes)
	if err != nil {
		logger.Error("genPubkey", "PrivKeyUnmarshaller", err.Error())
		return "", err
	}
	pubkey, err := privkey.GetPublic().Bytes()
	if err != nil {
		logger.Error("genPubkey", "GetPubkey", err.Error())
		return "", err
	}
	return hex.EncodeToString(pubkey), nil

}
