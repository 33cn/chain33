package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	//	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	//	"io"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
	"github.com/piotrnar/gocoin/lib/btc"
)

var SeedLong int = 16
var WalletSeed = []byte("walletseed")
var seedlog = log.New("module", "wallet")

const BACKUPKEYINDEX = "backupkeyindex"

//通过指定语言类型生成seed种子，传入原始字典的所在的路径
//lang = 0 通过英语单词生成种子 ，英文字典文件english.txt
//lang = 1 通过中文生成种子 ，中文字典文件chinese_simplified.txt

func CreateSeed(folderpath string, lang int32) (string, error) {
	seedlog.Info("CreateSeed", "folderpath", folderpath, "lang", lang)
	var filepath string
	if lang == 0 {
		filepath = fmt.Sprintf("%venglish.txt", folderpath)
	} else if lang == 1 {
		filepath = fmt.Sprintf("%vchinese_simplified.txt", folderpath)
	} else {
		return "", errors.New("CreateSeed lang err!")
	}

	//打开原始种子字典文件
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fb, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	st := strings.TrimSpace(string(fb))
	strs := strings.Split(st, "\n")
	strnum := len(strs)
	var seed []string
	for i := 0; i < SeedLong; i++ {

		bigi, err := rand.Int(rand.Reader, big.NewInt(int64(strnum-1)))
		if err != nil {
			fmt.Println(err.Error())
			return "", err
		}
		index := bigi.Int64()
		word := strings.TrimSpace(strs[int(index)])
		seed = append(seed, word)
	}
	var seedS string
	seedsize := len(seed)
	for k, v := range seed {
		if k != (seedsize - 1) {
			seedS += v + " "
		} else {
			seedS += v
		}
	}
	return seedS, nil
}

//使用password加密seed存储到db中
func SaveSeed(db dbm.DB, seed string, password string) (bool, error) {
	if len(seed) == 0 || len(password) == 0 {
		return false, ErrInputPara
	}

	Encrypted, err := AesgcmEncrypter([]byte(password), []byte(seed))
	if err != nil {
		seedlog.Error("SaveSeed", "AesgcmEncrypter err", err)
		return false, err
	}
	db.SetSync(WalletSeed, Encrypted)
	//seedlog.Info("SaveSeed ok", "Encryptedseed", Encryptedseed)
	return true, nil
}

//使用password解密seed上报给上层
func GetSeed(db dbm.DB, password string) (string, error) {
	if len(password) == 0 {
		return "", ErrInputPara
	}
	Encryptedseed := db.Get(WalletSeed)
	if len(Encryptedseed) == 0 {
		return "", errors.New("seed is not exit!")
	}
	seed, err := AesgcmDecrypter([]byte(password), Encryptedseed)
	if err != nil {
		return "", err
	}
	return string(seed), nil
}

//判断钱包是否已经保存seed
func HasSeed(db dbm.DB) (bool, error) {
	Encryptedseed := db.Get(WalletSeed)
	if len(Encryptedseed) == 0 {
		return false, errors.New("seed is not exit!")
	}
	return true, errors.New("seed is exit!")
}

//通过seed生成子私钥十六进制字符串
func GetPrivkeyBySeed(db dbm.DB, seed string) (string, error) {
	var backupindex uint32

	pkx := btc.MasterKey([]byte(seed), false)
	masterprivkey := pkx.String() //主私钥字符串
	xpubkey := pkx.Pub().String() //主公钥字符串
	//seedlog.Info("GetPrivkeyBySeed", "seed", seed, "masterprivkey", masterprivkey, "xpubkey", xpubkey)

	//hexmpub := hex.EncodeToString(pkx.Pub().Key)
	//hexmpriv := hex.EncodeToString(pkx.Key)
	//seedlog.Info("GetPrivkeyBySeed", "seed", seed, "hexmpriv", hexmpriv, "hexmpub", hexmpub)

	//通过主私钥随机生成child私钥十六进制字符串
	backuppubkeyindex := db.Get([]byte(BACKUPKEYINDEX))
	if backuppubkeyindex == nil {
		backupindex = 0
	} else {
		if err := json.Unmarshal([]byte(backuppubkeyindex), &backupindex); err != nil {
			return "", err
		}
	}
	index := backupindex + 1
	//生成子私钥和子公钥字符串，并校验是否相同
	subprivkey := btc.StringChild(masterprivkey, index)
	subpubkey := btc.StringChild(xpubkey, index)
	//seedlog.Info("GetPrivkeyBySeed", "subprivkey", subprivkey, "subpubkey", subpubkey)

	//通过子私钥字符串生成对应的hex字符串
	wallet, _ := btc.StringWallet(subprivkey)
	rec := btc.NewPrivateAddr(wallet.Key, 0x80, true)
	Hexsubprivkey := common.ToHex(rec.Key[1:])

	//对生成的子公钥做交易
	creatpubkey := wallet.Pub().String()
	if subpubkey != creatpubkey {
		seedlog.Error("GetPrivkeyBySeed subpubkey != creatpubkeybypriv")
		return "", errors.New("subpubkey verify fail!")
	}

	// back up index in db
	var pubkeyindex []byte
	pubkeyindex, err := json.Marshal(index)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed", "Marshal err ", err)
		return "", err
	}

	db.SetSync([]byte(BACKUPKEYINDEX), pubkeyindex)
	//seedlog.Info("GetPrivkeyBySeed", "Hexsubprivkey", Hexsubprivkey, "index", index)
	return Hexsubprivkey, nil
}

//通过私钥生成对应的公钥地址，传入的私钥是十六进制字符串，输出addr
func GetAddrByPrivkey(HexPrivkey string) (string, error) {
	if len(HexPrivkey) == 0 {
		return "", ErrInputPara
	}
	//解码hex格式的私钥
	privkeybyte, err := common.FromHex(HexPrivkey)
	if err != nil {
		return "", err
	}
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		seedlog.Error("GetAddrByPrivkey", "err", err)
		return "", err
	}

	priv, err := cr.PrivKeyFromBytes(privkeybyte)
	if err != nil {
		seedlog.Error("GetAddrByPrivkey", "PrivKeyFromBytes err", err)
		return "", err
	}
	addr := account.PubKeyToAddress(priv.PubKey().Bytes())
	return addr.String(), nil
}

//使用钱包的password对seed进行aesgcm加密,返回加密后的seed
func AesgcmEncrypter(password []byte, seed []byte) ([]byte, error) {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		seedlog.Error("AesgcmEncrypter NewCipher err", "err", err)
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		seedlog.Error("AesgcmEncrypter NewGCM err", "err", err)
		return nil, err
	}

	Encrypted := aesgcm.Seal(nil, key[:12], []byte(seed), nil)
	//seedlog.Info("AesgcmEncrypter Seal", "seed", seed, "key", key, "Encrypted", Encrypted)
	return Encrypted, nil
}

//使用钱包的password对seed进行aesgcm解密,返回解密后的seed
func AesgcmDecrypter(password []byte, seed []byte) ([]byte, error) {
	key := make([]byte, 32)
	if len(password) > 32 {
		key = password[0:32]
	} else {
		copy(key, password)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		seedlog.Error("AesgcmDecrypter", "NewCipher err", err)
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		seedlog.Error("AesgcmDecrypter", "NewGCM err", err)
		return nil, err
	}
	decryptered, err := aesgcm.Open(nil, key[:12], seed, nil)
	if err != nil {
		seedlog.Error("AesgcmDecrypter", "aesgcm Open err", err)
		return nil, err
	}
	//seedlog.Info("AesgcmDecrypter", "password", string(password), "seed", seed, "decryptered", string(decryptered))
	return decryptered, nil
}
