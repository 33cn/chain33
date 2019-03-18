package address

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	//"strconv"
	"strings"
	"testing"
	//"time"

	//"github.com/33cn/chain33/common/crypto"
	//"github.com/stretchr/testify/assert"
	//"github.com/stretchr/testify/require"

	_ "github.com/33cn/chain33/system/crypto/init"
)

func TestPubkeyToAddress(t *testing.T) {
	//ReadTaceAddr

	af, err := os.OpenFile("addrs.txt.txt", os.O_RDONLY, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	afb, err := ioutil.ReadAll(af)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	traceAddrs := strings.Split(string(afb), "\n")
	fmt.Println("追踪到的地址数量:", len(traceAddrs))
	var TraceMap = make(map[string]string)
	for _, addr := range traceAddrs {
		TraceMap[addr] = addr
	}

	//pubkey := "03034b0a44c31f1a062f64632b75b82d50dbeae5a76ccc4f49dbe5da1ced8fb1b3"
	f, err := os.OpenFile("onlinepids.txt", os.O_RDONLY, 0666)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fb, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	pubkeys := strings.Split(string(fb), "\n")
	fmt.Println("空投地址数量:", len(pubkeys))
	var RealAddr = make(map[string]string)
	for _, pub := range pubkeys {
		keyIp := strings.Split(pub, "@")
		key := keyIp[0]
		Ip := key[1]
		b, err := hex.DecodeString(key)
		if err != nil {
			t.Error(err)
			return
		}
		t.Logf("%X", b)
		addr := PubKeyToAddress(b)
		if _, ok := TraceMap[addr.String()]; ok {
			RealAddr[addr.String()] = addr.String()
		}
	}

	fmt.Println("匹配地址数量:", len(RealAddr))
	return

	//	t.Log(addr)
}
