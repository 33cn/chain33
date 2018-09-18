package commands

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

const (
	defCfgFileName = "chain33.cpm.toml"
	dappFolderName = "dapp"
	consensusFolderName = "consensus"
	storeFolderName = "store"
)

func ImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:"import",
		Short:"import plugin package",
		Run:importPackage,
	}
	return cmd
}

func importPackage(cmd *cobra.Command, args []string) {
	type STEP func() error
	strategy := &importStrategy{}
	steps := []STEP{
		strategy.readConfig,
		strategy.initData,
		strategy.generateImportFile,
	}
	for s, step := range steps {
		err := step()
		if err != nil {
			fmt.Println("call", s+1, "step error", err)
			return
		}
	}
}

type pluginConfigItem struct {
	Type string
	Gitrepo string
}

type pluginItem struct {
	name string
	gitRepo string
}

type importStrategy struct {
	cfgFileName string
	cfgItems	map[string]*pluginConfigItem

	items		map[string][]*pluginItem
}

func (strategy *importStrategy) readConfig() error {
	if len(strategy.cfgFileName) == 0 {
		strategy.cfgFileName = defCfgFileName
	}
	_, err := toml.DecodeFile(strategy.cfgFileName, &strategy.cfgItems)
	return err
}

func (strategy *importStrategy) initData() error {
	if len(strategy.cfgItems) == 0 {
		return errors.New("Config is empty.")
	}
	strategy.items = make(map[string][]*pluginItem)
	dappItems := make([]*pluginItem, 0)
	consensusItems := make([]*pluginItem, 0)
	storeItems := make([]*pluginItem, 0)
	for name, cfgItem := range strategy.cfgItems {
		item := &pluginItem{
			name:name,
			gitRepo:cfgItem.Gitrepo,
		}
		switch cfgItem.Type {
		case dappFolderName:
			dappItems = append(dappItems, item)
		case consensusFolderName:
			consensusItems = append(consensusItems, item)
		case storeFolderName:
			storeItems = append(storeItems, item)
		default:
			fmt.Printf("type %s is not supported.\n", cfgItem.Type)
			return errors.New("Config error.")
		}
	}
	strategy.items[dappFolderName] = dappItems
	strategy.items[consensusFolderName] = consensusItems
	strategy.items[storeFolderName] = storeItems
	return nil
}

func (strategy *importStrategy) generateImportFile() error {
	importStrs := map[string]string{
		dappFolderName:"",
		consensusFolderName:"",
		storeFolderName:"",
	}
	for name, plugins := range strategy.items {
		for _, item := range plugins {
			importStrs[name] += fmt.Sprintf("\r\n_ \"%s\"", item.gitRepo)
		}
	}
	projRoot := filepath.Join(os.Getenv("GOPATH") + "/src/gitlab.33.cn/chain33/chain33/plugin")
	for key, value := range importStrs {
		content := fmt.Sprintf("package init\r\n\r\nimport(%s\r\n)", value)

	}

	return nil
}

