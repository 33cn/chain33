// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package authority

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/executor/authority/core"
	"github.com/33cn/chain33/executor/authority/utils"

	"github.com/33cn/chain33/common/crypto"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var (
	alog = log.New("module", "authority")

	// Author 全局证书校验器
	Author = &Authority{}

	// IsAuthEnable 是否开启全局校验开关
	IsAuthEnable = false
)

// Authority 证书校验器主要结构
type Authority struct {
	// 证书文件路径
	cryptoPath string
	// certByte缓存
	authConfig *core.AuthConfig
	// 校验器
	validator core.Validator
	// 签名类型
	signType int
	// 有效证书缓存
	validCertCache [][]byte
	// 历史证书缓存
	HistoryCertCache *HistoryCertData
}

// HistoryCertData 历史变更记录
type HistoryCertData struct {
	CryptoCfg *core.AuthConfig
	CurHeight int64
	NxtHeight int64
}

// Init 初始化auth
func (auth *Authority) Init(conf *types.AuthorityCfg) error {
	if conf == nil || !conf.Enable {
		return nil
	}

	if len(conf.CryptoPath) == 0 {
		alog.Error("Crypto config path can not be null")
		return types.ErrInvalidParam
	}
	auth.cryptoPath = conf.CryptoPath

	sign := types.GetSignType("cert", conf.SignType)
	if sign == types.Invalid {
		alog.Error(fmt.Sprintf("Invalid sign type:%s", conf.SignType))
		return types.ErrInvalidParam
	}
	auth.signType = sign

	authConfig, err := core.GetAuthConfig(conf.CryptoPath)
	if err != nil {
		alog.Error("Get authority crypto config failed")
		return err
	}
	auth.authConfig = authConfig

	vldt, err := core.GetLocalValidator(authConfig, auth.signType)
	if err != nil {
		alog.Error(fmt.Sprintf("Get loacal validator failed. err:%s", err.Error()))
		return err
	}
	auth.validator = vldt

	auth.validCertCache = make([][]byte, 0)
	auth.HistoryCertCache = &HistoryCertData{authConfig, -1, -1}

	IsAuthEnable = true
	return nil
}

// newAuthConfig store数据转成authConfig数据
func newAuthConfig(store *types.HistoryCertStore) *core.AuthConfig {
	ret := &core.AuthConfig{}
	ret.RootCerts = make([][]byte, len(store.Rootcerts))
	for i, v := range store.Rootcerts {
		ret.RootCerts[i] = append(ret.RootCerts[i], v...)
	}

	ret.IntermediateCerts = make([][]byte, len(store.IntermediateCerts))
	for i, v := range store.IntermediateCerts {
		ret.IntermediateCerts[i] = append(ret.IntermediateCerts[i], v...)
	}

	ret.RevocationList = make([][]byte, len(store.RevocationList))
	for i, v := range store.RevocationList {
		ret.RevocationList[i] = append(ret.RevocationList[i], v...)
	}

	return ret
}

// ReloadCert 从数据库中的记录数据恢复证书，用于证书回滚
func (auth *Authority) ReloadCert(store *types.HistoryCertStore) error {
	if !IsAuthEnable {
		return nil
	}

	//判断是否回滚到无证书区块
	if len(store.Rootcerts) == 0 {
		auth.authConfig = nil
		auth.validator, _ = core.NewNoneValidator()
	} else {
		auth.authConfig = newAuthConfig(store)
		// 加载校验器
		vldt, err := core.GetLocalValidator(auth.authConfig, auth.signType)
		if err != nil {
			return err
		}
		auth.validator = vldt
	}

	// 清空有效证书缓存
	auth.validCertCache = auth.validCertCache[:0]

	// 更新最新历史数据
	auth.HistoryCertCache = &HistoryCertData{auth.authConfig, store.CurHeigth, store.NxtHeight}

	return nil
}

// ReloadCertByHeght 从新的authdir下的文件更新证书，用于证书更新
func (auth *Authority) ReloadCertByHeght(currentHeight int64) error {
	if !IsAuthEnable {
		return nil
	}

	authConfig, err := core.GetAuthConfig(auth.cryptoPath)
	if err != nil {
		alog.Error("Get authority crypto config failed")
		return err
	}
	auth.authConfig = authConfig

	// 加载校验器
	vldt, err := core.GetLocalValidator(auth.authConfig, auth.signType)
	if err != nil {
		return err
	}
	auth.validator = vldt

	// 清空有效证书缓存
	auth.validCertCache = auth.validCertCache[:0]

	// 更新最新历史数据
	auth.HistoryCertCache = &HistoryCertData{auth.authConfig, currentHeight, -1}

	return nil
}

// Validate 检验证书
func (auth *Authority) Validate(signature *types.Signature) error {
	// 从proto中解码signature
	cert, err := auth.validator.GetCertFromSignature(signature.Signature)
	if err != nil {
		return err
	}

	// 是否在有效证书缓存中
	for _, v := range auth.validCertCache {
		if bytes.Equal(v, cert) {
			return nil
		}
	}

	// 校验
	err = auth.validator.Validate(cert, signature.GetPubkey())
	if err != nil {
		alog.Error(fmt.Sprintf("validate cert failed. %s", err.Error()))
		return fmt.Errorf("validate cert failed. error:%s", err.Error())
	}
	auth.validCertCache = append(auth.validCertCache, cert)

	return nil
}

// User 用户关联的证书私钥信息
type User struct {
	ID   string
	Cert []byte
	Key  crypto.PrivKey
}

// UserLoader SKD加载user使用
type UserLoader struct {
	configPath string
	userMap    map[string]*User
	signType   int
}

// Init userloader初始化
func (loader *UserLoader) Init(configPath string, signType string) error {
	loader.configPath = configPath
	loader.userMap = make(map[string]*User)

	sign := types.GetSignType("cert", signType)
	if sign == types.Invalid {
		alog.Error(fmt.Sprintf("Invalid sign type:%s", signType))
		return types.ErrInvalidParam
	}
	loader.signType = sign

	return loader.loadUsers()
}

func (loader *UserLoader) loadUsers() error {
	certDir := path.Join(loader.configPath, "signcerts")
	dir, err := ioutil.ReadDir(certDir)
	if err != nil {
		return err
	}

	keyDir := path.Join(loader.configPath, "keystore")
	for _, file := range dir {
		filePath := path.Join(certDir, file.Name())
		certBytes, err := utils.ReadFile(filePath)
		if err != nil {
			continue
		}

		ski, err := utils.GetPublicKeySKIFromCert(certBytes, loader.signType)
		if err != nil {
			alog.Error(err.Error())
			continue
		}
		filePath = path.Join(keyDir, ski+"_sk")
		keyBytes, err := utils.ReadFile(filePath)
		if err != nil {
			continue
		}

		priv, err := loader.genCryptoPriv(keyBytes)
		if err != nil {
			alog.Error(fmt.Sprintf("Generate crypto private failed. error:%s", err.Error()))
			continue
		}

		loader.userMap[file.Name()] = &User{file.Name(), certBytes, priv}
	}

	return nil
}

func (loader *UserLoader) genCryptoPriv(keyBytes []byte) (crypto.PrivKey, error) {
	cr, err := crypto.New(types.GetSignName("cert", loader.signType))
	if err != nil {
		return nil, fmt.Errorf("create crypto %s failed, error:%s", types.GetSignName("cert", loader.signType), err)
	}
	//privKeyByte, err := utils.PrivKeyByteFromRaw(keyBytes, loader.signType)
	//if err != nil {
	//	return nil, err
	//}

	privkeyBytes, err := common.FromHex(string(keyBytes))
	if err != nil {
		return nil, err
	}

	priv, err := cr.PrivKeyFromBytes(privkeyBytes)
	if err != nil {
		return nil, fmt.Errorf("get private key failed, error:%s", err)
	}

	return priv, nil
}

// Get 根据用户名获取user结构
func (loader *UserLoader) Get(userName, orgName string) (*User, error) {
	keyvalue := fmt.Sprintf("%s@%s-cert.pem", userName, orgName)
	user, ok := loader.userMap[keyvalue]
	if !ok {
		return nil, types.ErrInvalidParam
	}

	resp := &User{}
	resp.Cert = append(resp.Cert, user.Cert...)
	resp.Key = user.Key

	return resp, nil
}
