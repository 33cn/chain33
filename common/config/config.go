package config

import (
	tml "github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	RPCAddr  string
	ParaName string
)

func Init(path string) (*types.Config, error) {
	var cfg types.Config
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
func InitCfg(path string) *types.Config {
	cfg, err := Init(path)
	if err != nil {
		panic(err)
	}
	return cfg
}
