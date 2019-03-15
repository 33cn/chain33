// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"fmt"
	"strings"

	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/33cn/chain33/cmd/tools/util"
)

// CreateDappSourceTask 通过生成好的pb.go和预先设计的模板，生成反射程序源码
type CreateDappSourceTask struct {
	TaskBase
	TemplatePath       string // 生成最终源码时的模板路径
	OutputPath         string
	ProjectName        string
	ClsName            string // 生成源码的类名
	ActionName         string // 生成源码的Action类名
	TypeName           string
	ExecuteName        string
	ProtoFile          string // 推导的原始proto文件
	ExecHeaderTempFile string
	TypeTempFile       string
	TypeOutputFile     string

	actionInfos           []*actionInfoItem // Action中的成员变量名称PB格式
	execHeaderTempContent string
}

//GetName 获取name
func (c *CreateDappSourceTask) GetName() string {
	return "CreateDappSourceTask"
}

//Execute 执行
func (c *CreateDappSourceTask) Execute() error {
	mlog.Info("Execute create build app source task.")
	if err := c.init(); err != nil {
		return err
	}
	if err := c.readActionMemberNames(); err != nil {
		return err
	}
	if err := c.createExecFile(); err != nil {
		return err
	}
	if err := c.createExecLocalFile(); err != nil {
		return err
	}
	if err := c.createExecDelLocalFile(); err != nil {
		return err
	}
	if err := c.createTypeExecuteFile(); err != nil {
		return err
	}
	return nil
}

func (c *CreateDappSourceTask) init() error {
	if !util.CheckFileIsExist(c.ExecHeaderTempFile) {
		return fmt.Errorf("file %s not exist", c.ExecHeaderTempFile)
	}
	contentbt, err := util.ReadFile(c.ExecHeaderTempFile)
	if err != nil {
		return fmt.Errorf("read file %s failed. error %q", c.ExecHeaderTempFile, err)
	}
	content := strings.Replace(string(contentbt), types.TagClassName, c.ClsName, -1)
	content = strings.Replace(content, types.TagExecName, c.ExecuteName, -1)
	c.execHeaderTempContent = content
	return nil
}

func (c *CreateDappSourceTask) readActionMemberNames() error {
	var err error
	pbContext, err := util.ReadFile(c.ProtoFile)
	if err != nil {
		return err
	}
	c.actionInfos, err = readDappActionFromProto(string(pbContext), c.ActionName)
	return err
}

func (c *CreateDappSourceTask) createExecFile() error {

	content := c.execHeaderTempContent
	content += formatExecContent(c.actionInfos, c.ClsName)
	fileName := fmt.Sprintf("%s/executor/exec.go", c.OutputPath)
	_, err := util.WriteStringToFile(fileName, content)
	if err != nil {
		mlog.Error(fmt.Sprintf("Write to file %s failed. error %q", fileName, err))
		return err
	}
	return nil
}

func (c *CreateDappSourceTask) createExecLocalFile() error {

	content := c.execHeaderTempContent
	content += formatExecLocalContent(c.actionInfos, c.ClsName)
	fileName := fmt.Sprintf("%s/executor/exec_local.go", c.OutputPath)
	_, err := util.WriteStringToFile(fileName, content)
	if err != nil {
		mlog.Error(fmt.Sprintf("Write to file %s failed. error %q", fileName, err))
		return err
	}
	return nil
}

func (c *CreateDappSourceTask) createExecDelLocalFile() error {

	content := c.execHeaderTempContent
	content += formatExecDelLocalContent(c.actionInfos, c.ClsName)
	fileName := fmt.Sprintf("%s/executor/exec_del_local.go", c.OutputPath)
	_, err := util.WriteStringToFile(fileName, content)
	if err != nil {
		mlog.Error(fmt.Sprintf("Write to file %s failed. error %q", fileName, err))
		return err
	}
	return nil
}

/**
createTypeExecuteFile 根据自己的需求，创建一个types中与执行器同名的Type对照关系
需要处理的内容：
1. 定义TyLogXXXX的常量，规则是 TyLog + 变量名称
2. 定义类型常量，规则是 ActionName + 变量名称
3. 实现GetLogMap()
4. 实现GetTypeMap()
*/
func (c *CreateDappSourceTask) createTypeExecuteFile() error {
	logText := buildActionLogTypeText(c.actionInfos, c.ExecuteName) // ${TYLOGACTIONTYPE}

	actionIDText := buildActionIDText(c.actionInfos, c.ExecuteName) // ${ACTIONIDTEXT}

	logMapText := buildLogMapText() // ${LOGMAPTEXT}

	typeMapText := buildTypeMapText(c.actionInfos, c.ExecuteName) // ${TYPEMAPTEXT}

	replacePairs := []struct {
		src string
		dst string
	}{
		{src: types.TagTyLogActionType, dst: logText},
		{src: types.TagActionIDText, dst: actionIDText},
		{src: types.TagLogMapText, dst: logMapText},
		{src: types.TagTypeMapText, dst: typeMapText},
		{src: types.TagTypeName, dst: c.TypeName},
		{src: types.TagExecName, dst: c.ExecuteName},
		{src: types.TagActionName, dst: c.ActionName},
	}
	bcontent, err := util.ReadFile(c.TypeTempFile)
	if err != nil {
		return err
	}
	content := string(bcontent)
	for _, pair := range replacePairs {
		content = strings.Replace(content, pair.src, pair.dst, -1)
	}
	fileName := fmt.Sprintf("%s%s.go", c.TypeOutputFile, c.ClsName)
	util.DeleteFile(fileName)
	_, err = util.WriteStringToFile(fileName, content)
	return err
}
