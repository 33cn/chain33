// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package strategy 实现开发者工具实现不同策略的功能
package strategy

import (
	"fmt"

	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/pkg/errors"
)

var (
	mlog = log15.New("module", "strategy")
)

//Strategy 策略
type Strategy interface {
	SetParam(key string, value string)
	Run() error
}

//New new
func New(name string) Strategy {
	switch name {
	case types.KeyImportPackage:
		return &importPackageStrategy{
			strategyBasic: strategyBasic{
				params: make(map[string]string),
			},
		}
	case types.KeyUpdateInit:
		return &updateInitStrategy{
			strategyBasic: strategyBasic{
				params: make(map[string]string),
			},
		}
	case types.KeyCreatePlugin:
		return &createPluginStrategy{
			strategyBasic: strategyBasic{
				params: make(map[string]string),
			},
		}
	case types.KeyGenDapp:
		return &genDappStrategy{
			strategyBasic: strategyBasic{
				params: make(map[string]string),
			},
		}
	}
	return nil
}

type strategyBasic struct {
	params map[string]string
}

//SetParam 设置参数
func (s *strategyBasic) SetParam(key string, value string) {
	s.params[key] = value
}

func (s *strategyBasic) getParam(key string) (string, error) {
	if v, ok := s.params[key]; ok {
		return v, nil
	}
	return "", fmt.Errorf("Key:%v not exist", key)
}

//Run 运行
func (s *strategyBasic) Run() error {
	return errors.New("NotSupport")
}
