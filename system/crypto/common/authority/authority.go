// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package authority

import (
	"errors"
	"fmt"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/crypto/common/authority/core"
)

var alog = log.New("module", "authority")

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
	// 初始化标记
	IsInit bool
}

// SubConfig 配置文件
type SubConfig struct {
	CertEnable bool   `json:"certEnable"`
	CertPath   string `json:"certPath"`
}

// Init 初始化auth
func (auth *Authority) Init(conf *SubConfig, sign int, lclValidator interface{}) error {
	if len(conf.CertPath) == 0 {
		alog.Error("Crypto config path can not be null")
		return errors.New("ErrInvalidParam")
	}
	auth.cryptoPath = conf.CertPath
	auth.signType = sign

	authConfig, err := core.GetAuthConfig(conf.CertPath)
	if err != nil {
		alog.Error("Get authority crypto config failed")
		return err
	}
	auth.authConfig = authConfig

	auth.validator = lclValidator.(core.Validator)
	auth.validator.Setup(authConfig)

	auth.IsInit = true

	return nil
}

// Validate 检验证书
func (auth *Authority) Validate(pub, signature []byte) error {
	// 从proto中解码signature
	cert, err := auth.validator.GetCertFromSignature(signature)
	if err != nil {
		return err
	}

	// 校验
	err = auth.validator.Validate(cert, pub)
	if err != nil {
		alog.Error(fmt.Sprintf("validate cert failed. %s", err.Error()))
		return fmt.Errorf("validate cert failed. error:%s", err.Error())
	}

	return nil
}
