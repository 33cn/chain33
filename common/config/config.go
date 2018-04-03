package config

import (
	"fmt"
	"os"

	tml "github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/types"
)

func InitCfg(path string) *types.Config {
	var cfg types.Config
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(0)

	}
	return &cfg
}
