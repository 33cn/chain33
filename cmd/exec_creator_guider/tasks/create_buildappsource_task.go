package tasks

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/authority/utils"
	"regexp"
	"strings"
	"unicode"
)

type actionInfoItem struct {
	memberName string
	memberType string
}

// CreateBuildAppSourceTask 通过生成好的pb.go和预先设计的模板，生成反射程序源码
type CreateBuildAppSourceTask struct {
	TaskBase
	TemplatePath string // 生成最终源码时的模板路径
	ClsName      string // 生成源码的类名
	ActionName   string // 生成源码的Action类名
	ProtoFile    string // 推导的原始proto文件

	actionInfos []*actionInfoItem // Action中的成员变量名称PB格式
}

func (this *CreateBuildAppSourceTask) Execute() error {
	mlog.Info("Execute create build app source task.")
	if err := this.readActionMemberNames(); err != nil {
		return err
	}
	return nil
}

/**
通过正则获取Action的成员变量名和类型，其具体操作步骤如下：
1. 读取需要解析的proto文件
2. 通过搜索，定位到指定Action的起始为止
3. 使用正则获取该Action中的oneof Value的内容
4. 使用正则解析oneof Value中的内容，获取变量名和类型名
5. 将获取到的变量名去除空格，并将首字母大写
*/
func (this *CreateBuildAppSourceTask) readActionMemberNames() error {
	pbContext, err := utils.ReadFile(this.ProtoFile)
	if err != nil {
		return err
	}
	context := string(pbContext)
	index := strings.Index(context, this.ActionName)
	if index < 0 {
		return errors.New(fmt.Sprintf("Action %s Not Existed", this.ActionName))
	}
	expr := fmt.Sprintf(`\s*oneof\s+value\s*{\s+([\w\s=;]*)\}`)
	reg := regexp.MustCompile(expr)
	oneOfValueStrs := reg.FindAllStringSubmatch(string(pbContext), index)

	expr = fmt.Sprintf(`\s+(\w+)([\s\w]+)=\s+(\d+);`)
	reg = regexp.MustCompile(expr)
	members := reg.FindAllStringSubmatch(oneOfValueStrs[0][0], -1)

	this.actionInfos = make([]*actionInfoItem, 0)
	for _, member := range members {
		memberType := strings.Replace(member[1], " ", "", -1)
		memberName := strings.Replace(member[2], " ", "", -1)
		tmp := []rune(memberName)
		// 根据proto生成pb.go的规则，成员变量首字母必须大写
		tmp[0] = unicode.ToUpper(tmp[0])
		memberName = string(tmp)
		this.actionInfos = append(this.actionInfos, &actionInfoItem{
			memberName: memberName,
			memberType: memberType,
		})
	}
	if len(this.actionInfos) == 0 {
		return errors.New(fmt.Sprintf("Can Not Find %s Member Info", this.ActionName))
	}
	return nil
}
