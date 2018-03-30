package main
import (
	"fmt"
	"os"

	"code.aliyun.com/chain33/chain33/types"
	tml "github.com/BurntSushi/toml"
)
func main() {

}
func InitCfg(path string) *types.Config {
	var cfg types.Config
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(0)

	}
	return &cfg
}