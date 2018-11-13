// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

type Strategy interface {
	SetParam(key string, value string)
	Run() error
}

func New(name string) Strategy {
	switch name {
	case types.KeyImportPackage:
		return &importPackageStrategy{
			strategyBasic: strategyBasic{
				params: make(map[string]string),
			},
		}
	case types.KeyCreateSimpleExecProject:
		return &simpleCreateExecProjStrategy{
			strategyBasic: strategyBasic{
				params: make(map[string]string),
			},
		}
	case types.KeyCreateAdvanceExecProject:
		return &advanceCreateExecProjStrategy{
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
	}
	return nil
}

type strategyBasic struct {
	params map[string]string
}

func (this *strategyBasic) SetParam(key string, value string) {
	this.params[key] = value
}

func (this *strategyBasic) getParam(key string) (string, error) {
	if v, ok := this.params[key]; ok {
		return v, nil
	}
	return "", errors.New(fmt.Sprintf("Key:%v not existed.", key))
}

func (this *strategyBasic) Run() error {
	return errors.New("NotSupport")
}
