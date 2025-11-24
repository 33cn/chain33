package jsonpb

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/33cn/chain33/common"
)

// ErrBytesFormat 错误的bytes 类型
var ErrBytesFormat = errors.New("ErrBytesFormat")

func parseBytes(jsonstr string, enableUTF8BytesToString bool) ([]byte, error) {
	if jsonstr == "" {
		return []byte{}, nil
	}
	if strings.HasPrefix(jsonstr, "str://") {
		return []byte(jsonstr[len("str://"):]), nil
	}
	if strings.HasPrefix(jsonstr, "0x") || strings.HasPrefix(jsonstr, "0X") {
		return common.FromHex(jsonstr)
	}
	//字符串不是 hex 格式， 也不是 str:// 格式，但是是一个普通的utf8 字符串
	//那么强制转化为bytes, 注意这个选项默认不开启.
	if utf8.ValidString(jsonstr) && enableUTF8BytesToString {
		return []byte(jsonstr), nil
	}
	return nil, ErrBytesFormat
}
