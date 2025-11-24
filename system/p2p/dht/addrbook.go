package dht

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/33cn/chain33/common/db"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet/bipwallet"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	addrkeyTag              = "mutiaddrs"
	privKeyTag              = "privkey"
	privKeyCompressBytesLen = 32
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
	a.bookDb = db.NewDB("addrBook", a.cfg.Driver, dbPath, a.cfg.DbCache)
	a.loadDb()
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

// Randkey Rand keypair
func (a *AddrBook) Randkey() crypto.PrivKey {
	a.initKey()
	return a.GetPrivkey()
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
func (a *AddrBook) GetPrivkey() crypto.PrivKey {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	if a.privkey == "" {
		return nil
	}
	keybytes, err := hex.DecodeString(a.privkey)
	if err != nil {
		log.Error("GetPrivkey", "DecodeString Error", err.Error())
		return nil
	}
	log.Debug("GetPrivkey", "privsize", len(keybytes))

	privkey, err := crypto.UnmarshalSecp256k1PrivateKey(keybytes)
	if err != nil {
		privkey, err = crypto.UnmarshalPrivateKey(keybytes)
		if err != nil {
			log.Error("GetPrivkey", "UnmarshalPrivateKey", err.Error())
			return nil
		}

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
		return err
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
	var peerInfo = make(map[string]interface{})
	var info = make(map[string]interface{})
	pubkey, err := PeerIDToPubkey(id.Pretty())
	if err == nil {
		info["pubkey"] = pubkey
		pbytes, _ := hex.DecodeString(pubkey)
		addr, _ := bipwallet.PubToAddress(pbytes)
		info["address"] = addr
	}

	peerInfo[id.Pretty()] = info
	bs, err := json.MarshalIndent(peerInfo, "", "\t")
	if err != nil {
		log.Error("StoreHostId", "MarshalIndent", err.Error())
		return
	}
	_, err = pf.Write(bs)
	if err != nil {
		return
	}

}

// GenPrivPubkey return key and pubkey in bytes
func GenPrivPubkey() ([]byte, []byte, error) {
	priv, pub, err := crypto.GenerateSecp256k1Key(rand.Reader)
	//priv, pub, err := crypto.GenerateKeyPairWithReader(crypto.Secp256k1, 2048, rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	privkey, err := priv.Raw()
	if err != nil {
		return nil, nil, err

	}
	pubkey, err := pub.Raw()
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

	log.Info("GenPubkey", "key size", len(keybytes))
	privkey, err := crypto.UnmarshalSecp256k1PrivateKey(keybytes)
	if err != nil {

		privkey, err = crypto.UnmarshalPrivateKey(keybytes)
		if err != nil {
			//切换
			log.Error("genPubkey", "UnmarshalPrivateKey", err.Error())
			return "", err
		}
	}

	pubkey, err := privkey.GetPublic().Raw()
	if err != nil {
		log.Error("genPubkey", "GetPubkey", err.Error())
		return "", err
	}
	return hex.EncodeToString(pubkey), nil

}

// PeerIDToPubkey 提供节点ID转换为pubkey,进而通过pubkey创建chain33 地址的功能
func PeerIDToPubkey(id string) (string, error) {
	//encodeIdStr := "16Uiu2HAm7vDB7XDuEv8XNPcoPqumVngsjWoogGXENNDXVYMiCJHM"
	//hexpubStr:="02b99bc73bfb522110634d5644d476b21b3171eefab517da0646ef2aba39dbf4a0"
	//chain33 address:13ohj5JH6NE15ENfuQRneqGdg29nT27K3k
	pID, err := peer.Decode(id)
	if err != nil {
		return "", err
	}

	pub, err := pID.ExtractPublicKey()
	if err != nil {
		return "", err
	}

	pbs, err := pub.Raw()
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(pbs), nil

}
