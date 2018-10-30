package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
)

const fee = 1e6
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var r *rand.Rand
var TxHeightOffset int64 = 0

func main() {
	if len(os.Args) == 1 || os.Args[1] == "-h" {
		LoadHelp()
		return
	}
	fmt.Println("jrpc url:", os.Args[2]+":8801")
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	argsWithoutProg := os.Args[1:]
	switch argsWithoutProg[0] {
	case "-h": //使用帮助
		LoadHelp()
	case "perf":
		if len(argsWithoutProg) != 6 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Perf(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3], argsWithoutProg[4], argsWithoutProg[5])
	case "put":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Put(argsWithoutProg[1], argsWithoutProg[2], "")
	case "get":
		if len(argsWithoutProg) != 3 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		Get(argsWithoutProg[1], argsWithoutProg[2])
	case "valnode":
		if len(argsWithoutProg) != 4 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		ValNode(argsWithoutProg[1], argsWithoutProg[2], argsWithoutProg[3])
	}
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println("perf [ip, size, num, interval, duration] {offset}            : 写数据性能测试")
	fmt.Println("put  [ip, size]                                              : 写数据")
	fmt.Println("get  [ip, hash]                                              : 读数据")
	fmt.Println("valnode [ip, pubkey, power]                                  : 增加/删除/修改tendermint节点")
}

func Perf(ip, size, num, interval, duration string) {
	var numThread int
	numInt, err := strconv.Atoi(num)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	intervalInt, err := strconv.Atoi(interval)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	durInt, err := strconv.Atoi(duration)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	if numInt < 10 {
		numThread = 1
	} else if numInt > 100 {
		numThread = 10
	} else {
		numThread = numInt / 10
	}
	maxTxPerAcc := 50
	ch := make(chan struct{}, numThread)
	for i := 0; i < numThread; i++ {
		go func() {
			txCount := 0
			_, priv := genaddress()
			for sec := 0; durInt == 0 || sec < durInt; {
				setTxHeight(ip)
				for txs := 0; txs < numInt/numThread; txs++ {
					if txCount >= maxTxPerAcc {
						_, priv = genaddress()
						txCount = 0
					}
					Put(ip, size, common.ToHex(priv.Bytes()))
					txCount++
				}
				time.Sleep(time.Second * time.Duration(intervalInt))
				sec += intervalInt
			}
			ch <- struct{}{}
		}()
	}
	for j := 0; j < numThread; j++ {
		<-ch
	}
}

func Put(ip string, size string, privkey string) {
	sizeInt, err := strconv.Atoi(size)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	url := "http://" + ip + ":8801"
	if privkey == "" {
		_, priv := genaddress()
		privkey = common.ToHex(priv.Bytes())
	}
	payload := RandStringBytes(sizeInt)
	//fmt.Println("payload:", common.ToHex([]byte(payload)))

	tx := &types.Transaction{Execer: []byte("user.write"), Payload: []byte(payload), Fee: 1e6}
	tx.To = address.ExecAddress("user.write")
	tx.Expire = TxHeightOffset + types.TxHeightFlag
	tx.Sign(types.SECP256K1, getprivkey(privkey))
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.SendTransaction","params":[{"data":"%v"}]}`,
		common.ToHex(types.Encode(tx)))

	resp, err := http.Post(url, "application/json", bytes.NewBufferString(poststr))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("returned JSON: %s\n", string(b))
}

func Get(ip string, hash string) {
	url := "http://" + ip + ":8801"
	fmt.Println("transaction hash:", hash)

	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.QueryTransaction","params":[{"hash":"%s"}]}`, hash)
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(poststr))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("returned JSON: %s\n", string(b))
}

func setTxHeight(ip string) {
	url := "http://" + ip + ":8801"
	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.GetLastHeader","params":[]}`)
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(poststr))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Printf("returned JSON: %s\n", string(b))
	msg := &RespMsg{}
	err = json.Unmarshal(b, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	TxHeightOffset = msg.Result.Height
	fmt.Println("TxHeightOffset:", TxHeightOffset)
}

type RespMsg struct {
	Id     int64           `json:"id"`
	Result rpctypes.Header `json:"result"`
	Err    string          `json:"error"`
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	fmt.Println("addr:", addrto.String())
	return addrto.String(), privto
}

func RandStringBytes(n int) string {
	b := make([]byte, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func ValNode(ip, pubkey, power string) {
	url := "http://" + ip + ":8801"

	fmt.Println(pubkey, ":", power)
	pubkeybyte, err := hex.DecodeString(pubkey)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	powerInt, err := strconv.Atoi(power)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	_, priv := genaddress()
	privkey := common.ToHex(priv.Bytes())
	nput := &ty.ValNodeAction_Node{Node: &ty.ValNode{PubKey: pubkeybyte, Power: int64(powerInt)}}
	action := &ty.ValNodeAction{Value: nput, Ty: ty.ValNodeActionUpdate}
	tx := &types.Transaction{Execer: []byte("valnode"), Payload: types.Encode(action), Fee: fee}
	tx.To = address.ExecAddress("valnode")
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(privkey))

	poststr := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"Chain33.SendTransaction","params":[{"data":"%v"}]}`,
		common.ToHex(types.Encode(tx)))

	resp, err := http.Post(url, "application/json", bytes.NewBufferString(poststr))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("returned JSON: %s\n", string(b))
}
