package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

func main() {
	level := pebble.LevelOptions{
		FilterPolicy: bloom.FilterPolicy(10),
	}
	db, err := pebble.Open("/home/yann/test/pebble", &pebble.Options{
		BytesPerSync: 4 << 20,
		Levels:       []pebble.LevelOptions{level},
	})
	if err != nil {
		log.Fatal(err)
	}

	testValue := []byte("giu398egfisugdiwegfaiulfffffffffffglgieruiugfiuadgugfuihfd3ugfogeiof;hdfheodgfeugfdj;gf;3orfigfdagfuiehrfuiehiafbdjafhariugfhuaiegfriuhfeagauiffaeferhi;hgoaigf")
	start := time.Now()
	for i := 0; i < 1e8; i++ {
		fmt.Println(i)
		sum := sha256.Sum256(IntToBytes(i))
		key := sum[:]
		if err := db.Set(key, testValue, pebble.NoSync); err != nil {
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
