package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	sccrypto "github.com/NebulousLabs/Sia/crypto"
	"github.com/NebulousLabs/Sia/modules"
	log "github.com/inconshreveable/log15"
	"github.com/piotrnar/gocoin/lib/btc"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	SeedLong     = 15
	SaveSeedLong = 12

	WalletSeed = []byte("walletseed")
	seedlog    = log.New("module", "wallet")

	ChineseSeedCache = make(map[string]string)
	EnglishSeedCache = make(map[string]string)
)

const BACKUPKEYINDEX = "backupkeyindex"

//通过指定语言类型生成seed种子，传入原始字典的所在的路径
//lang = 0 通过英语单词生成种子 ，英文字典文件english.txt
//lang = 1 通过中文生成种子 ，中文字典文件chinese_simplified.txt

func CreateSeed(folderpath string, lang int32) (string, error) {
	var strs []string
	if lang == 0 {
		strs = strings.Split(englishText, " ")
	} else if lang == 1 {
		strs = strings.Split(chineseText, " ")
	} else {
		return "", types.ErrSeedlang
	}

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

//初始化seed标准库的单词到map中，方便seed单词的校验
func InitSeedLibrary() {
	//首先将标准seed库转换成字符串数组
	englieshstrs := strings.Split(englishText, " ")
	chinesestrs := strings.Split(chineseText, " ")

	//中引文标准seed库保存到map中
	for _, wordstr := range chinesestrs {
		ChineseSeedCache[wordstr] = wordstr
	}

	for _, wordstr := range englieshstrs {
		EnglishSeedCache[wordstr] = wordstr
	}
}

//校验输入的seed字符串数组是否在标准库中，
//首先从第一个单词开始在英文和中文库中查找来区分seed是英文还是中文，
//都查找不到就直接返回校验错误,以及出错的单词或者汉字
func VerifySeed(seedarry []string) (bool, string) {
	//首先通过seed的第一个单词来确定seed是英文的还是中文组合
	//第一个单词是英文并存在英文标准库中
	if EnglishSeedCache[seedarry[0]] == seedarry[0] {
		for _, seedstr := range seedarry {
			if EnglishSeedCache[seedstr] != seedstr {
				return false, seedstr
			}
		}
		return true, ""
	} else { //第一个单词不存在英文标准中，需要尝试看是否存在中文库中
		for _, seedstr := range seedarry {
			if ChineseSeedCache[seedstr] != seedstr {
				return false, seedstr
			}
		}
		return true, ""
	}
}

//使用password加密seed存储到db中
func SaveSeed(db dbm.DB, seed string, password string) (bool, error) {
	if len(seed) == 0 || len(password) == 0 {
		return false, types.ErrInputPara
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
		return "", types.ErrInputPara
	}
	Encryptedseed, err := db.Get(WalletSeed)
	if len(Encryptedseed) == 0 || err != nil {
		return "", types.ErrSeedNotExist
	}
	seed, err := AesgcmDecrypter([]byte(password), Encryptedseed)
	if err != nil {
		return "", err
	}
	return string(seed), nil
}

//判断钱包是否已经保存seed
func HasSeed(db dbm.DB) (bool, error) {
	Encryptedseed, err := db.Get(WalletSeed)
	if len(Encryptedseed) == 0 || err != nil {
		return false, types.ErrSeedNotExist
	}
	return true, nil
}

//通过seed生成子私钥十六进制字符串
func GetPrivkeyBySeed(db dbm.DB, seed string) (string, error) {
	var backupindex uint32
	var Hexsubprivkey string
	var err error

	//通过主私钥随机生成child私钥十六进制字符串
	backuppubkeyindex, err := db.Get([]byte(BACKUPKEYINDEX))
	if backuppubkeyindex == nil || err != nil {
		backupindex = 0
	} else {
		if err = json.Unmarshal(backuppubkeyindex, &backupindex); err != nil {
			return "", err
		}
	}
	index := backupindex + 1

	//secp256k1
	if SignType == 1 {
		pkx := btc.MasterKey([]byte(seed), false)
		masterprivkey := pkx.String() //主私钥字符串
		xpubkey := pkx.Pub().String() //主公钥字符串

		//生成子私钥和子公钥字符串，并校验是否相同
		subprivkey := btc.StringChild(masterprivkey, index)
		subpubkey := btc.StringChild(xpubkey, index)
		//seedlog.Info("GetPrivkeyBySeed", "subprivkey", subprivkey, "subpubkey", subpubkey)

		//通过子私钥字符串生成对应的hex字符串
		wallet, _ := btc.StringWallet(subprivkey)
		rec := btc.NewPrivateAddr(wallet.Key, 0x80, true)
		Hexsubprivkey = common.ToHex(rec.Key[1:])

		//对生成的子公钥做校验
		creatpubkey := wallet.Pub().String()
		if subpubkey != creatpubkey {
			seedlog.Error("GetPrivkeyBySeed subpubkey != creatpubkeybypriv")
			return "", types.ErrSubPubKeyVerifyFail
		}
	} else if SignType == 2 { //ed25519

		//通过助记词形式的seed生成私钥和公钥,一个seed根据不同的index可以生成许多组密钥
		//字符串形式的助记词(英语单词)通过计算一次hash转成字节形式的seed

		var Seed modules.Seed
		hash := common.Sha256([]byte(seed))

		copy(Seed[:], hash)
		sk, _ := sccrypto.GenerateKeyPairDeterministic(sccrypto.HashAll(Seed, index))
		secretKey := fmt.Sprintf("%x", sk)
		//publicKey := fmt.Sprintf("%x", pk)
		//seedlog.Error("GetPrivkeyBySeed", "index", index, "secretKey", secretKey, "publicKey", publicKey)

		Hexsubprivkey = secretKey
	} else if SignType == 3 { //sm2
		return "", types.ErrNotSupport
	} else {
		return "", types.ErrNotSupport
	}
	// back up index in db
	var pubkeyindex []byte
	pubkeyindex, err = json.Marshal(index)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed", "Marshal err ", err)
		return "", types.ErrMarshal
	}

	db.SetSync([]byte(BACKUPKEYINDEX), pubkeyindex)
	//seedlog.Info("GetPrivkeyBySeed", "Hexsubprivkey", Hexsubprivkey, "index", index)
	return Hexsubprivkey, nil
}

//通过私钥生成对应的公钥地址，传入的私钥是十六进制字符串，输出addr
func GetAddrByPrivkey(HexPrivkey string) (string, error) {
	if len(HexPrivkey) == 0 {
		return "", types.ErrInputPara
	}
	//解码hex格式的私钥
	privkeybyte, err := common.FromHex(HexPrivkey)
	if err != nil {
		return "", err
	}
	//通过privkey生成一个pubkey然后换算成对应的addr
	cr, err := crypto.New(types.GetSignatureTypeName(SignType))
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

	Encrypted := aesgcm.Seal(nil, key[:12], seed, nil)
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
