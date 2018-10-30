package strategy

import (
	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/cmd/tools/types"
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
		return &importPackageStrategy{}
	}
	return nil
}

type strategyBasic struct {
	params map[string]string
}

func (this *strategyBasic) SetParam(key string, value string) {
	this.params[key] = value
}

func (this *strategyBasic) Run() error {
	return errors.New("NotSupport")
}
