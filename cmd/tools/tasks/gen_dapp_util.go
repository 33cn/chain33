// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"fmt"
	"regexp"
	"strings"

	sysutil "github.com/33cn/chain33/util"
)

type actionInfoItem struct {
	memberName string
	memberType string
}

/**
通过正则获取Action的成员变量名和类型，其具体操作步骤如下：
1. 读取需要解析的proto文件
2. 通过搜索，定位到指定Action的起始为止
3. 使用正则获取该Action中的oneof Value的内容
4. 使用正则解析oneof Value中的内容，获取变量名和类型名
5. 将获取到的变量名去除空格，并将首字母大写
*/
func readDappActionFromProto(protoContent, actionName string) ([]*actionInfoItem, error) {

	// 如果文件中含有与ActionName部分匹配的文字，则会造成搜索到多个
	index := strings.Index(protoContent, actionName)
	if index < 0 {
		return nil, fmt.Errorf("action %s Not Existed", actionName)
	}
	expr := fmt.Sprintf(`\s*oneof\s+value\s*{\s+([\w\s=;]*)\}`)
	reg := regexp.MustCompile(expr)
	oneOfValueStrs := reg.FindAllStringSubmatch(protoContent, index)

	expr = fmt.Sprintf(`\s+(\w+)([\s\w]+)=\s+(\d+);`)
	reg = regexp.MustCompile(expr)
	members := reg.FindAllStringSubmatch(oneOfValueStrs[0][0], -1)

	actionInfos := make([]*actionInfoItem, 0)
	for _, member := range members {
		memberType := strings.Replace(member[1], " ", "", -1)
		memberName := strings.Replace(member[2], " ", "", -1)
		// 根据proto生成pb.go的规则，成员变量首字母必须大写
		memberName, _ = sysutil.MakeStringToUpper(memberName, 0, 1)
		actionInfos = append(actionInfos, &actionInfoItem{
			memberName: memberName,
			memberType: memberType,
		})
	}
	if len(actionInfos) == 0 {
		return nil, fmt.Errorf("can Not Find %s Member Info", actionName)
	}
	return actionInfos, nil
}

func formatExecContent(infos []*actionInfoItem, dappName string) string {

	fnFmtStr := `func (c *%s) Exec_%s(payload *ptypes.%s, tx *types.Transaction, index int) (*types.Receipt, error) {
	var receipt *types.Receipt
	//implement code
	return receipt, nil
}

`
	content := ""
	for _, info := range infos {
		content += fmt.Sprintf(fnFmtStr, dappName, info.memberName, info.memberType)
	}

	return content
}

func formatExecLocalContent(infos []*actionInfoItem, dappName string) string {

	fnFmtStr := `func (c *%s) ExecLocal_%s(payload *ptypes.%s, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var dbSet *types.LocalDBSet
	//implement code
	return dbSet, nil
}

`
	content := ""
	for _, info := range infos {
		content += fmt.Sprintf(fnFmtStr, dappName, info.memberName, info.memberType)
	}

	return content
}

func formatExecDelLocalContent(infos []*actionInfoItem, dappName string) string {

	fnFmtStr := `func (c *%s) ExecDelLocal_%s(payload *ptypes.%s, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var dbSet *types.LocalDBSet
	//implement code
	return dbSet, nil
}

`
	content := ""
	for _, info := range infos {
		content += fmt.Sprintf(fnFmtStr, dappName, info.memberName, info.memberType)
	}

	return content
}

// 组成规则是 TyLog+ActionName + ActionMemberName
func buildActionLogTypeText(infos []*actionInfoItem, className string) (text string) {
	items := fmt.Sprintf("TyUnknownLog = iota + 100\n")
	for _, info := range infos {
		items += fmt.Sprintf("Ty%sLog\n", info.memberName)
	}
	text = fmt.Sprintf("const (\n%s)\n", items)
	return
}

// 组成规则是 ActionName + ActionMemberName
func buildActionIDText(infos []*actionInfoItem, className string) (text string) {

	items := fmt.Sprintf("TyUnknowAction = iota + 100\n")
	for _, info := range infos {
		items += fmt.Sprintf("Ty%sAction\n", info.memberName)
	}
	items += "\n"
	for _, info := range infos {
		items += fmt.Sprintf("Name%sAction = \"%s\"\n", info.memberName, info.memberName)
	}

	text = fmt.Sprintf("const (\n%s)\n", items)
	return
}

// 返回 map[string]int32
func buildTypeMapText(infos []*actionInfoItem, className string) (text string) {
	var items string
	for _, info := range infos {
		items += fmt.Sprintf("Name%sAction: Ty%sAction,\n", info.memberName, info.memberName)
	}
	text = fmt.Sprintf("map[string]int32{\n%s}", items)
	return
}

// 返回 map[string]*types.LogInfo
func buildLogMapText() (text string) {
	text = fmt.Sprintf("map[int64]*types.LogInfo{\n\t//LogID:	{Ty: reflect.TypeOf(LogStruct), Name: LogName},\n}")
	return
}
