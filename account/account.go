package account

//package for account manger
//1. load from db
//2. save to db
//3. KVSet
//4. Transfer
//5. Add
//6. Sub
//7. Account balance query
//8. gen a private key -> private key to address (bitcoin likes)

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"

	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

var genesisKey = []byte("mavl-acc-genesis")
var addrSeed = []byte("address seed bytes for public key")

func LoadAccount(db dbm.KVDB, addr string) *types.Account {
	value, err := db.Get(AccountKey(addr))
	if err != nil {
		return &types.Account{Addr: addr}
	}
	var acc types.Account
	err = types.Decode(value, &acc)
	if err != nil {
		panic(err) //数据库已经损坏
	}
	return &acc
}

func GetGenesis(db dbm.KVDB) *types.Genesis {
	value, err := db.Get(genesisKey)
	if err != nil {
		return &types.Genesis{}
	}
	var g types.Genesis
	err = types.Decode(value, &g)
	if err != nil {
		panic(err) //数据库已经损坏
	}
	return &g
}

func SaveGenesis(db dbm.KVDB, g *types.Genesis) {
	set := GetGenesisKVSet(g)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func SaveAccount(db dbm.KVDB, acc *types.Account) {
	set := GetKVSet(acc)
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func GetKVSet(acc *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc)
	kvset = append(kvset, &types.KeyValue{AccountKey(acc.Addr), value})
	return kvset
}

func GetGenesisKVSet(g *types.Genesis) (kvset []*types.KeyValue) {
	value := types.Encode(g)
	kvset = append(kvset, &types.KeyValue{genesisKey, value})
	return kvset
}

func PubKeyToAddress(in []byte) *Address {
	a := new(Address)
	a.Pubkey = make([]byte, len(in))
	copy(a.Pubkey[:], in[:])
	a.Version = 0
	a.Hash160 = common.Rimp160AfterSha256(in)
	return a
}

var bname [200]byte

func ExecAddress(name string) *Address {
	if len(name) > 100 {
		panic("name too long")
	}
	buf := append(bname[:0], addrSeed...)
	buf = append(buf, []byte(name)...)
	hash := common.Sha2Sum(buf)
	return PubKeyToAddress(hash[:])
}

func CheckAddress(addr string) error {
	_, err := NewAddrFromString(addr)
	return err
}

func NewAddrFromString(hs string) (a *Address, e error) {
	dec := Decodeb58(hs)
	if dec == nil {
		e = errors.New("Cannot decode b58 string '" + hs + "'")
		return
	}
	if len(dec) < 25 {
		e = errors.New("Address too short " + hex.EncodeToString(dec))
		return
	}
	if len(dec) == 25 {
		sh := common.Sha2Sum(dec[0:21])
		if !bytes.Equal(sh[:4], dec[21:25]) {
			e = errors.New("Address Checksum error")
		} else {
			a = new(Address)
			a.Version = dec[0]
			copy(a.Hash160[:], dec[1:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, dec[21:25])
			a.Enc58str = hs
		}
	}
	return
}

func LoadAccounts(q *queue.Queue, addrs []string) (accs []*types.Account, err error) {
	client := q.GetClient()
	//get current head ->
	msg := client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	get := types.StoreGet{}
	get.StateHash = msg.GetData().(*types.Header).GetStateHash()
	for i := 0; i < len(addrs); i++ {
		get.Keys = append(get.Keys, AccountKey(addrs[i]))
	}
	msg = client.NewMessage("store", types.EventStoreGet, &get)
	client.Send(msg, true)
	msg, err = client.Wait(msg)
	if err != nil {
		return nil, err
	}
	values := msg.GetData().(*types.StoreReplyValue)
	for i := 0; i < len(values.Values); i++ {
		value := values.Values[i]
		if value == nil {
			accs = append(accs, &types.Account{})
		} else {
			var acc types.Account
			err := types.Decode(value, &acc)
			if err != nil {
				return nil, err
			}
			accs = append(accs, &acc)
		}
	}
	return accs, nil
}

type CacheDB struct {
	header *types.Header
	q      queue.IClient
	cache  map[string][]byte
}

func NewCacheDB(q *queue.Queue, header *types.Header) *CacheDB {
	return &CacheDB{header, q.GetClient(), make(map[string][]byte)}
}

func (db *CacheDB) Get(key []byte) (value []byte, err error) {
	if value, ok := db.cache[string(key)]; ok {
		return value, nil
	}
	value, err = db.get(key)
	if err != nil {
		return nil, err
	}
	db.cache[string(key)] = value
	return value, err
}

func (db *CacheDB) Set(key []byte, value []byte) error {
	db.cache[string(key)] = value
	return nil
}

func (db *CacheDB) get(key []byte) (value []byte, err error) {
	query := &types.StoreGet{db.header.StateHash, [][]byte{key}}
	msg := db.q.NewMessage("store", types.EventStoreGet, query)
	db.q.Send(msg, true)
	resp, err := db.q.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		return nil, types.ErrNotFound
	}
	return value, nil
}

func LoadAccountsDB(db dbm.KVDB, addrs []string) (accs []*types.Account, err error) {
	for i := 0; i < len(addrs); i++ {
		acc := LoadAccount(db, addrs[i])
		accs = append(accs, acc)
	}
	return accs, nil
}

//address to save key
func AccountKey(address string) (key []byte) {
	key = append(key, []byte("mavl-acc-")...)
	key = append(key, []byte(address)...)
	return key
}

type Address struct {
	Version  byte
	Hash160  [20]byte // For a stealth address: it's HASH160
	Checksum []byte   // Unused for a stealth address
	Pubkey   []byte   // Unused for a stealth address
	Enc58str string
}

func (a *Address) String() string {
	if a.Enc58str == "" {
		var ad [25]byte
		ad[0] = a.Version
		copy(ad[1:21], a.Hash160[:])
		if a.Checksum == nil {
			sh := common.Sha2Sum(ad[0:21])
			a.Checksum = make([]byte, 4)
			copy(a.Checksum, sh[:4])
		}
		copy(ad[21:25], a.Checksum[:])
		a.Enc58str = Encodeb58(ad[:])
	}
	return a.Enc58str
}

var b58set []byte = []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

func b58chr2int(chr byte) int {
	for i := range b58set {
		if b58set[i] == chr {
			return i
		}
	}
	return -1
}

var bn0 *big.Int = big.NewInt(0)
var bn58 *big.Int = big.NewInt(58)

func Encodeb58(a []byte) (s string) {
	idx := len(a)*138/100 + 1
	buf := make([]byte, idx)
	bn := new(big.Int).SetBytes(a)
	var mo *big.Int
	for bn.Cmp(bn0) != 0 {
		bn, mo = bn.DivMod(bn, bn58, new(big.Int))
		idx--
		buf[idx] = b58set[mo.Int64()]
	}
	for i := range a {
		if a[i] != 0 {
			break
		}
		idx--
		buf[idx] = b58set[0]
	}

	s = string(buf[idx:])

	return
}

func Decodeb58(s string) (res []byte) {
	bn := big.NewInt(0)
	for i := range s {
		v := b58chr2int(byte(s[i]))
		if v < 0 {
			return nil
		}
		bn = bn.Mul(bn, bn58)
		bn = bn.Add(bn, big.NewInt(int64(v)))
	}

	// We want to "restore leading zeros" as satoshi's implementation does:
	var i int
	for i < len(s) && s[i] == b58set[0] {
		i++
	}
	if i > 0 {
		res = make([]byte, i)
	}
	res = append(res, bn.Bytes()...)
	return
}
