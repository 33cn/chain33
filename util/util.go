package util

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"unicode"
)

func GetRealExecName(paraName string, name string) string {
	if strings.HasPrefix(name, "user.p.") {
		return name
	}
	return paraName + name
}

// MakeStringToUpper 将给定的in字符串从pos开始一共count个转换为大写字母
func MakeStringToUpper(in string, pos, count int) (out string, err error) {
	l := len(in)
	if pos<0 || pos >= l || (pos+count)>=l {
		err = errors.New(fmt.Sprintf("Invalid params. in=%s pos=%d count=%d", in, pos, count))
		return
	}
	tmp := []rune(in)
	for n:=pos; n<pos+count; n++ {
		tmp[n] = unicode.ToUpper(tmp[n])
	}
	out = string(tmp)
	return
}

// MakeStringToLower 将给定的in字符串从pos开始一共count个转换为小写字母
func MakeStringToLower(in string, pos, count int) (out string, err error) {
	l := len(in)
	if pos<0 || pos >= l || (pos+count)>=l {
		err = errors.New(fmt.Sprintf("Invalid params. in=%s pos=%d count=%d", in, pos, count))
		return
	}
	tmp := []rune(in)
	for n:=pos; n<pos+count; n++ {
		tmp[n] = unicode.ToLower(tmp[n])
	}
	out = string(tmp)
	return
}