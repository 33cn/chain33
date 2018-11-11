package address

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/33cn/chain33/common/crypto"

	_ "github.com/33cn/chain33/system/crypto/init"
)

func TestAddress(t *testing.T) {
	c, err := crypto.New("secp256k1")
	if err != nil {
		t.Error(err)
		return
	}
	key, err := c.GenKey()
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%X", key.Bytes())
	addr := PubKeyToAddress(key.PubKey().Bytes())
	t.Log(addr)
}

func TestPubkeyToAddress(t *testing.T) {
	pubkey := "024a17b0c6eb3143839482faa7e917c9b90a8cfe5008dff748789b8cea1a3d08d5"
	b, err := hex.DecodeString(pubkey)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%X", b)
	addr := PubKeyToAddress(b)
	t.Log(addr)
}

func TestCheckAddress(t *testing.T) {
	c, err := crypto.New("secp256k1")
	if err != nil {
		t.Error(err)
		return
	}
	key, err := c.GenKey()
	if err != nil {
		t.Error(err)
		return
	}
	addr := PubKeyToAddress(key.PubKey().Bytes())
	err = CheckAddress(addr.String())
	require.NoError(t, err)
}

func BenchmarkExecAddress(b *testing.B) {
	start := time.Now().UnixNano() / 1000000
	fmt.Println(start)
	for i := 0; i < b.N; i++ {
		ExecAddress("ticket")
	}
	end := time.Now().UnixNano() / 1000000
	fmt.Println(end)
	duration := end - start
	fmt.Println("duration with cache:", strconv.FormatInt(duration, 10))

	start = time.Now().UnixNano() / 1000000
	fmt.Println(start)
	for i := 0; i < b.N; i++ {
		GetExecAddress("ticket")
	}
	end = time.Now().UnixNano() / 1000000
	fmt.Println(end)
	duration = end - start
	fmt.Println("duration without cache:", strconv.FormatInt(duration, 10))
}
