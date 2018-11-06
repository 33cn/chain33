package strategy

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"gitlab.33.cn/chain33/chain33/cmd/tools/types"

	"github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/util"
)

const (
	dappFolderName      = "dapp"
	consensusFolderName = "consensus"
	storeFolderName     = "store"
)

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

type importPackageStrategy struct {
	strategyBasic

	cfgFileName    string
	cfgItems       map[string]*pluginConfigItem
	projRootPath   string
	projPluginPath string
	items          map[string][]*pluginItem
}

func (this *importPackageStrategy) Run() error {
	mlog.Info("Begin run chain33 import packages.")
	defer mlog.Info("Run chain33 import packages finish.")
	return this.runImpl()
}

func (this *importPackageStrategy) runImpl() error {
	type STEP func() error
	steps := []STEP{
		this.readConfig,
		this.initData,
		this.generateImportFile,
		this.fetchPluginPackage,
	}

	for s, step := range steps {
		err := step()
		if err != nil {
			fmt.Println("call", s+1, "step error", err)
			return err
		}
	}
	return nil
}

func (this *importPackageStrategy) readConfig() error {
	mlog.Info("读取配置文件")
	if len(this.cfgFileName) == 0 {
		this.cfgFileName = fmt.Sprintf("config/%s", types.DEF_CPM_CONFIGFILE)
	}
	_, err := toml.DecodeFile(this.cfgFileName, &this.cfgItems)
	return err
}

func (this *importPackageStrategy) initData() error {
	mlog.Info("初始化数据")
	if len(this.cfgItems) == 0 {
		return errors.New("Config is empty.")
	}
	this.items = make(map[string][]*pluginItem)
	dappItems := make([]*pluginItem, 0)
	consensusItems := make([]*pluginItem, 0)
	storeItems := make([]*pluginItem, 0)
	for name, cfgItem := range this.cfgItems {
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
	this.items[dappFolderName] = dappItems
	this.items[consensusFolderName] = consensusItems
	this.items[storeFolderName] = storeItems
	this.projRootPath = filepath.Join(os.Getenv("GOPATH") + "/src/gitlab.33.cn/chain33/chain33")
	this.projPluginPath = filepath.Join(this.projRootPath + "/plugin")
	return nil
}

func (this *importPackageStrategy) generateImportFile() error {
	mlog.Info("生成引用文件")
	importStrs := map[string]string{}
	for name, plugins := range this.items {
		for _, item := range plugins {
			importStrs[name] += fmt.Sprintf("\r\n_ \"%s\"", item.gitRepo)
		}
	}
	for key, value := range importStrs {
		content := fmt.Sprintf("package init\r\n\r\nimport(%s\r\n)", value)
		initFile := fmt.Sprintf("%s/%s/init/init.go", this.projPluginPath, key)
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

func (this *importPackageStrategy) fetchPlugin(gitrepo, version string) error {
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
func (this *importPackageStrategy) fetchPluginPackage() error {
	mlog.Info("下载插件源码包")
	pwd := util.Pwd()
	os.Chdir(this.projRootPath)
	defer os.Chdir(pwd)
	for _, plugins := range this.items {
		for _, plugin := range plugins {
			mlog.Info("同步插件", "repo", plugin.gitRepo, "version", plugin.version)
			err := this.fetchPlugin(plugin.gitRepo, plugin.version)
			if err != nil {
				mlog.Info("同步插件包出错", "repo", plugin.gitRepo, "error", err.Error())
				return err
			}
		}
	}
	return nil
}
