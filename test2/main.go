package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func main() {

	db, err := leveldb.OpenFile("/home/yann/test/leveldb", &opt.Options{
		OpenFilesCacheCapacity: 16,
		BlockCacheCapacity:     2 * opt.MiB,
		WriteBuffer:            4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if err != nil {
		log.Fatal(err)
	}
	tx := &types.Transaction{
		Execer:  []byte("dajfilwejwpohgg3"),
		Payload: []byte("fhieuwkytoehgoegowhgo324goi3oi34rehgothei4goh34itvndeoigl3hpgh p43thtoi3hto38t3g4oigeoitheirbfjksdhflj4lkgg43ig ehgoihewioghow;eg ihgewgo h eigerh go;"),
		To:      "hfi23uytowhfskhfuiegfskfhskjfgwu",
		Header:  []byte("gf2i3ur23u9urhoihfishfoehwf  fsoi hs go3 3 wo fyosfh owi fhs hifos hosf  "),
	}
	_ = tx

	start := time.Now()
	for i := 0; i < 1e8; i++ {
		fmt.Println(i)
		sum := sha256.Sum256(IntToBytes(i))
		key := sum[:]
		tx.Fee = rand.Int63()
		tx.Expire = rand.Int63()
		tx.Nonce = rand.Int63()
		tx.GroupCount = rand.Int31()
		if err := db.Put(key, types.Encode(tx), &opt.WriteOptions{}); err != nil {
			log.Fatal(err)
		}
	}
	fmt.Println("time cost:", time.Since(start))

	if err := db.Close(); err != nil {
		log.Fatal(err)
	}
}

func IntToBytes(n int) []byte {
	data := int64(n)
	bytebuf := bytes.NewBuffer([]byte{})
	binary.Write(bytebuf, binary.BigEndian, data)
	return bytebuf.Bytes()
}
