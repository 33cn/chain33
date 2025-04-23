// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wallet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/33cn/chain33/common/crypto"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet/bipwallet"
)

var (
	// SeedLong 随机种子的长度
	SeedLong = 15
	// SaveSeedLong 保存的随机种子个数
	SaveSeedLong = 12

	// WalletSeed 钱包种子前缀
	WalletSeed = []byte("walletseed")
	seedlog    = log.New("module", "wallet")

	// ChineseSeedCache 中文种子缓存映射
	ChineseSeedCache = make(map[string]string)
	// EnglishSeedCache 英文种子缓存映射
	EnglishSeedCache = make(map[string]string)
)

// BACKUPKEYINDEX 备份索引Key值
const BACKUPKEYINDEX = "backupkeyindex"

// CreateSeed 通过指定语言类型生成seed种子，传入语言类型以及
// lang = 0 通过英语单词生成种子
// lang = 1 通过中文生成种子
// bitsize=128 返回12个单词或者汉子，bitsize+32=160  返回15个单词或者汉子，bitszie=256 返回24个单词或者汉子
func CreateSeed(folderpath string, lang int32) (string, error) {
	mnem, err := bipwallet.NewMnemonicString(int(lang), 160)
	if err != nil {
		seedlog.Error("CreateSeed", "NewMnemonicString err", err)
		return "", err
	}
	return mnem, nil
}

// InitSeedLibrary 初始化seed标准库的单词到map中，方便seed单词的校验
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

// VerifySeed 校验输入的seed字符串数是否合法，通过助记词能否生成钱包来判断合法性
func VerifySeed(seed string, signType int, coinType uint32) (bool, error) {

	_, err := bipwallet.NewWalletFromMnemonic(coinType, uint32(signType), seed)
	if err != nil {
		seedlog.Error("VerifySeed NewWalletFromMnemonic", "err", err)
		return false, err
	}
	return true, nil
}

// SaveSeedInBatch 保存种子数据到数据库
func SaveSeedInBatch(db dbm.DB, seed string, password string, batch dbm.Batch) (bool, error) {
	if len(seed) == 0 || len(password) == 0 {
		return false, types.ErrInvalidParam
	}

	Encrypted, err := AesgcmEncrypter([]byte(password), []byte(seed))
	if err != nil {
		seedlog.Error("SaveSeed", "AesgcmEncrypter err", err)
		return false, err
	}
	batch.Set(WalletSeed, Encrypted)
	//seedlog.Info("SaveSeed ok", "Encryptedseed", Encryptedseed)
	return true, nil
}

// GetSeed 使用password解密seed上报给上层
func GetSeed(db dbm.DB, password string) (string, error) {
	if len(password) == 0 {
		return "", types.ErrInvalidParam
	}
	Encryptedseed, err := db.Get(WalletSeed)
	if err != nil {
		return "", err
	}
	if len(Encryptedseed) == 0 {
		return "", types.ErrSeedNotExist
	}
	seed, err := AesgcmDecrypter([]byte(password), Encryptedseed)
	if err != nil {
		seedlog.Error("GetSeed", "AesgcmDecrypter err", err)
		return "", types.ErrInputPassword
	}
	return string(seed), nil
}

// GetPrivkeyBySeed 通过seed生成子私钥十六进制字符串
func GetPrivkeyBySeed(db dbm.DB, seed string, specificIndex uint32, SignType int, coinType uint32) (string, error) {
	var backupindex uint32
	var Hexsubprivkey string
	var err error
	var index uint32
	signType := uint32(SignType)
	//通过主私钥随机生成child私钥十六进制字符串
	if specificIndex == 0 {
		backuppubkeyindex, err := db.Get([]byte(BACKUPKEYINDEX))
		if backuppubkeyindex == nil || err != nil {
			index = 0
		} else {
			if err = json.Unmarshal(backuppubkeyindex, &backupindex); err != nil {
				return "", err
			}
			index = backupindex + 1
		}
	} else {
		index = specificIndex
	}
	cryptoName := crypto.GetName(SignType)
	if cryptoName == "unknown" {
		return "", types.ErrNotSupport
	}

	wallet, err := bipwallet.NewWalletFromMnemonic(coinType, signType, seed)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed NewWalletFromMnemonic", "err", err)
		wallet, err = bipwallet.NewWalletFromSeed(coinType, signType, []byte(seed))
		if err != nil {
			seedlog.Error("GetPrivkeyBySeed NewWalletFromSeed", "err", err)
			return "", types.ErrNewWalletFromSeed
		}
	}

	//通过索引生成Key pair
	priv, pub, err := wallet.NewKeyPair(index)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed NewKeyPair", "err", err)
		return "", types.ErrNewKeyPair
	}

	Hexsubprivkey = hex.EncodeToString(priv)

	public, err := bipwallet.PrivkeyToPub(coinType, signType, priv)
	if err != nil {
		seedlog.Error("GetPrivkeyBySeed PrivkeyToPub", "err", err)
		return "", types.ErrPrivkeyToPub
	}
	if !bytes.Equal(pub, public) {
		seedlog.Error("GetPrivkeyBySeed NewKeyPair pub  != PrivkeyToPub", "err", err)
		return "", types.ErrSubPubKeyVerifyFail
	}

	// back up index in db
	if specificIndex == 0 {
		var pubkeyindex []byte
		pubkeyindex, err = json.Marshal(index)
		if err != nil {
			seedlog.Error("GetPrivkeyBySeed", "Marshal err ", err)
			return "", types.ErrMarshal
		}

		err = db.SetSync([]byte(BACKUPKEYINDEX), pubkeyindex)
		if err != nil {
			seedlog.Error("GetPrivkeyBySeed", "SetSync err ", err)
			return "", err
		}
	}
	return Hexsubprivkey, nil
}

// AesgcmEncrypter 使用钱包的password对seed进行aesgcm加密,返回加密后的seed
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

// AesgcmDecrypter 使用钱包的password对seed进行aesgcm解密,返回解密后的seed
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
