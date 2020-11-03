package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

func main() {
	level := &pebble.LevelOptions{}
	level.EnsureDefaults()
	level.FilterPolicy = bloom.FilterPolicy(10)

	db, err := pebble.Open("demo", &pebble.Options{
		BytesPerSync: 4 << 20,
		Levels:       []pebble.LevelOptions{{FilterPolicy: bloom.FilterPolicy(10)}},
	})
	if err != nil {
		log.Fatal(err)
	}
	tx := &types.Transaction{
		Execer:    []byte("dajfilwejwpohgg3"),
		Payload:   []byte("fhieuwkytoehgoegowhgo324goi3oi34rehgothei4goh34itvndeoigl3hpgh p43thtoi3hto38t3g4oigeoitheirbfjksdhflj4lkgg43ig ehgoihewioghow;eg ihgewgo h eigerh go;"),
		To:        "hfi23uytowhfskhfuiegfskfhskjfgwu",
		Header:    []byte("gf2i3ur23u9urhoihfishfoehwf  fsoi hs go3 3 wo fyosfh owi fhs hifos hosf  "),
		HashCache: []byte("hfui23yfweoihfifgiogfiwegowlf "),
	}
	_ = tx

	_ = math.MaxInt64
	start := time.Now()
	for i := 0; i < 1e6; i++ {
		fmt.Println(i)
		sum := sha256.Sum256(IntToBytes(i + 1e10))
		key := sum[:]
		//tx.Fee = rand.Int63()
		//tx.Expire = rand.Int63()
		//tx.Nonce = rand.Int63()
		//tx.GroupCount = rand.Int31()
		//if err := db.Set(key, types.Encode(tx), pebble.NoSync); err != nil {
		//	log.Fatal(err)
		//}
		_, closer, err := db.Get(key)
		if err == pebble.ErrNotFound {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		_ = closer.Close()
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
