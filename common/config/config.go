package config

import (
	"encoding/json"

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

func InitCfg(path string) (*types.Config, *types.ConfigSubModule) {
	cfg, err := Init(path)
	if err != nil {
		panic(err)
	}
	sub, err := InitSubModule(path)
	if err != nil {
		panic(err)
	}
	return cfg, sub
}

func InitString(cfgstring string) (*types.Config, error) {
	var cfg types.Config
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func InitCfgString(cfgstring string) (*types.Config, *types.ConfigSubModule) {
	cfg, err := InitString(cfgstring)
	if err != nil {
		panic(err)
	}
	sub, err := InitSubModuleString(cfgstring)
	if err != nil {
		panic(err)
	}
	return cfg, sub
}

type subModule struct {
	Store     map[string]interface{}
	Exec      map[string]interface{}
	Consensus map[string]interface{}
	Wallet    map[string]interface{}
}

func InitSubModuleString(cfgstring string) (*types.ConfigSubModule, error) {
	var cfg subModule
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		return nil, err
	}
	return parseSubModule(&cfg)
}

func InitSubModule(path string) (*types.ConfigSubModule, error) {
	var cfg subModule
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		return nil, err
	}
	return parseSubModule(&cfg)
}

func parseSubModule(cfg *subModule) (*types.ConfigSubModule, error) {
	var subcfg types.ConfigSubModule
	subcfg.Store = parseItem(cfg.Store)
	subcfg.Exec = parseItem(cfg.Exec)
	subcfg.Consensus = parseItem(cfg.Consensus)
	subcfg.Wallet = parseItem(cfg.Wallet)
	return &subcfg, nil
}

func parseItem(data map[string]interface{}) map[string][]byte {
	subconfig := make(map[string][]byte)
	if len(data) == 0 {
		return subconfig
	}
	for key := range data {
		if key == "sub" {
			subcfg := data[key].(map[string]interface{})
			for k := range subcfg {
				subconfig[k], _ = json.Marshal(subcfg[k])
			}
		}
	}
	return subconfig
}
