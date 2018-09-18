package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/util"
)

const (
	defCfgFileName      = "chain33.cpm.toml"
	dappFolderName      = "dapp"
	consensusFolderName = "consensus"
	storeFolderName     = "store"
)

func ImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "import plugin package",
		Run:   importPackage,
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
		strategy.fetchPluginPackage,
	}
	for s, step := range steps {
		err := step()
		if err != nil {
			fmt.Println("call", s+1, "step error", err)
			return
		}
	}
	fmt.Println("导入关联包操作成功结束")
}

type pluginConfigItem struct {
	Type    string
	Gitrepo string
	Version string
}

type pluginItem struct {
	name    string
	gitRepo string
	version string
}

type importStrategy struct {
	cfgFileName    string
	cfgItems       map[string]*pluginConfigItem
	projRootPath   string
	projPluginPath string
	items          map[string][]*pluginItem
}

func (strategy *importStrategy) readConfig() error {
	fmt.Println("读取配置文件")
	if len(strategy.cfgFileName) == 0 {
		strategy.cfgFileName = defCfgFileName
	}
	_, err := toml.DecodeFile(strategy.cfgFileName, &strategy.cfgItems)
	return err
}

func (strategy *importStrategy) initData() error {
	fmt.Println("初始化数据")
	if len(strategy.cfgItems) == 0 {
		return errors.New("Config is empty.")
	}
	strategy.items = make(map[string][]*pluginItem)
	dappItems := make([]*pluginItem, 0)
	consensusItems := make([]*pluginItem, 0)
	storeItems := make([]*pluginItem, 0)
	for name, cfgItem := range strategy.cfgItems {
		item := &pluginItem{
			name:    name,
			gitRepo: cfgItem.Gitrepo,
			version: cfgItem.Version,
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
	strategy.projRootPath = filepath.Join(os.Getenv("GOPATH") + "/src/gitlab.33.cn/chain33/chain33")
	strategy.projPluginPath = filepath.Join(strategy.projRootPath + "/plugin")
	return nil
}

func (strategy *importStrategy) generateImportFile() error {
	fmt.Println("生成引用文件")
	importStrs := map[string]string{}
	for name, plugins := range strategy.items {
		for _, item := range plugins {
			importStrs[name] += fmt.Sprintf("\r\n_ \"%s\"", item.gitRepo)
		}
	}
	for key, value := range importStrs {
		content := fmt.Sprintf("package init\r\n\r\nimport(%s\r\n)", value)
		initFile := fmt.Sprintf("%s/%s/init/init.go", strategy.projPluginPath, key)
		util.MakeDir(initFile)

		{ // 写入到文件中
			util.DeleteFile(initFile)
			file, err := util.OpenFile(initFile)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.WriteString(file, content)
			if err != nil {
				return err
			}
		}
		// 格式化生成的文件
		cmd := exec.Command("gofmt", "-l", "-s", "-w", initFile)
		err := cmd.Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (strategy *importStrategy) fetchPlugin(gitrepo, version string) error {
	var param string
	if len(version) > 0 {
		param = fmt.Sprintf("%s@%s", gitrepo, version)
	} else {
		param = gitrepo
	}
	cmd := exec.Command("govendor", "fetch", param)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// fetchPluginPackage 使用govendor来下载依赖包
func (strategy *importStrategy) fetchPluginPackage() error {
	fmt.Println("下载插件源码包")
	pwd := util.Pwd()
	os.Chdir(strategy.projRootPath)
	defer os.Chdir(pwd)
	for _, plugins := range strategy.items {
		for _, plugin := range plugins {
			fmt.Printf("同步插件包%s，版本%s\n", plugin.gitRepo, plugin.version)
			err := strategy.fetchPlugin(plugin.gitRepo, plugin.version)
			if err != nil {
				fmt.Printf("同步插件包%s出错 error:\n", plugin.gitRepo, err.Error())
				return err
			}
		}
	}
	return nil
}
