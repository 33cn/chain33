package dht

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	addrkeyTag = "mutiaddrs"
	privKeyTag = "privkey"
)

// AddrBook peer address manager
type AddrBook struct {
	mtx     sync.Mutex
	cfg     *types.P2P
	privkey string
	pubkey  string
	bookDb  db.DB
}

// NewAddrBook new addr book
func NewAddrBook(cfg *types.P2P) *AddrBook {
	a := &AddrBook{
		cfg: cfg,
	}
	dbPath := cfg.DbPath + "/" + p2pty.DHTTypeName
	a.bookDb = db.NewDB("addrbook", a.cfg.Driver, dbPath, a.cfg.DbCache)

	if !a.loadDb() {
		a.initKey()
	}
	return a

}

// AddrsInfo get addr infos
func (a *AddrBook) AddrsInfo() []peer.AddrInfo {
	var addrsInfo []peer.AddrInfo
	if a.bookDb == nil {
		return addrsInfo
	}

	addrsInfoBs, err := a.bookDb.Get([]byte(addrkeyTag))
	if err != nil {
		log.Error("LoadAddrsInfo", " Get addrkeyTag", err.Error())
		return addrsInfo
	}

	err = json.Unmarshal(addrsInfoBs, &addrsInfo)
	if err != nil {
		log.Error("LoadAddrsInfo", "Unmarshal err", err.Error())
		return nil
	}

	return addrsInfo
}

func (a *AddrBook) loadDb() bool {

	privkey, err := a.bookDb.Get([]byte(privKeyTag))
	if len(privkey) != 0 && err == nil {
		pubkey, err := GenPubkey(string(privkey))
		if err != nil {
			log.Error("genPubkey", "err", err.Error())
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

	a.saveKey(hex.EncodeToString(priv), hex.EncodeToString(pub))
}

func (a *AddrBook) saveKey(priv, pub string) {
	a.setKey(priv, pub)
	err := a.bookDb.Set([]byte(privKeyTag), []byte(priv))
	if err != nil {
		panic(err)
	}
}

// GetPrivkey get private key
func (a *AddrBook) GetPrivkey() p2pcrypto.PrivKey {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	keybytes, err := hex.DecodeString(a.privkey)
	if err != nil {
		log.Error("GetPrivkey", "DecodeString Error", err.Error())
		return nil
	}
	privkey, err := p2pcrypto.UnmarshalPrivateKey(keybytes)
	if err != nil {
		log.Error("GetPrivkey", "PrivKeyUnmarshaller", err.Error())
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

// SaveAddr save addr
func (a *AddrBook) SaveAddr(addrinfos []peer.AddrInfo) error {

	jsonBytes, err := json.Marshal(addrinfos)
	if err != nil {
		log.Error("Failed to save AddrBook to file", "err", err)
		return nil
	}
	log.Debug("saveToDb", "addrs", string(jsonBytes))
	return a.bookDb.Set([]byte(addrkeyTag), jsonBytes)

}

// StoreHostID store host id into file
func (a *AddrBook) StoreHostID(id peer.ID, path string) {
	dbPath := path + "/" + p2pty.DHTTypeName
	if os.MkdirAll(dbPath, 0755) != nil {
		return
	}

	pf, err := os.Create(fmt.Sprintf("%v/%v", dbPath, id.Pretty()))
	if err != nil {
		log.Error("StoreHostId", "Create file", err.Error())
		return
	}

	defer pf.Close()
	pf.WriteString(id.Pretty())

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

// GenPubkey generate public key
func GenPubkey(key string) (string, error) {
	keybytes, err := hex.DecodeString(key)
	if err != nil {
		log.Error("DecodeString Error", "Error", err.Error())
		return "", err
	}
	privkey, err := p2pcrypto.UnmarshalPrivateKey(keybytes)
	if err != nil {
		log.Error("genPubkey", "PrivKeyUnmarshaller", err.Error())
		return "", err
	}
	pubkey, err := privkey.GetPublic().Bytes()
	if err != nil {
		log.Error("genPubkey", "GetPubkey", err.Error())
		return "", err
	}
	return hex.EncodeToString(pubkey), nil

}
