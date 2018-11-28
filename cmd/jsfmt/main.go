// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package main js转换 ->
// 1. 格式化
// 2. 默认int 类型除以 8,保留4位小数
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
)

var d = flag.Int("d", 8, "数字小数点位数")
var n = flag.Int("n", 4, "保留有效小数点位数")
var js = flag.String("js", "", "输入js字符串")
var keylist = flag.String("k", "", "key list")
var keymap = make(map[string]bool)

func main() {
	flag.Parse()
	list := strings.Split(*keylist, ",")
	for _, v := range list {
		keymap[v] = true
	}
	if *js == "" {
		reader := bufio.NewReader(os.Stdin)
		*js, _ = reader.ReadString('\n')
	}
	var data interface{}
	err := json.Unmarshal([]byte(*js), &data)
	if err != nil {
		panic(err)
	}
	s := parse("", data, *d, *n)
	jsf, _ := json.Marshal(s)
	fmt.Println(string(jsf))
}

/*
bool, for JSON booleans
float64, for JSON numbers
string, for JSON strings
[]interface{}, for JSON arrays
map[string]interface{}, for JSON objects
nil for JSON null
*/
func parse(key string, data interface{}, d, n int) interface{} {
	if data == nil {
		return data
	}
	switch data.(type) {
	case bool:
		return data
	case string:
		if !isValidNumber(data.(string)) {
			return data
		}
		numstr := data.(string)
		num := json.Number(numstr)
		if strings.Contains(numstr, ".") {
			dm, _ := num.Float64()
			return format64(key, dm, d, n)
		}
		dm, _ := num.Int64()
		return format64(key, float64(dm), d, n)
	case float64:
		num := data.(float64)
		return format64(key, num, d, n)
	case []interface{}:
		datas := data.([]interface{})
		for i := range datas {
			datas[i] = parse(key, datas[i], d, n)
		}
	case map[string]interface{}:
		datas := data.(map[string]interface{})
		for i := range datas {
			datas[i] = parse(i, datas[i], d, n)
		}
	}
	return data
}

func format64(key string, num float64, d, n int) interface{} {
	if len(keymap) > 0 && !keymap[key] {
		return num
	}
	div := math.Pow(10, float64(n))
	if div == 0 {
		return num
	}
	num = num / div
	return fmt.Sprintf("%."+fmt.Sprint(n)+"f", num)
}

func isValidNumber(s string) bool {
	if s == "" {
		return false
	}
	// Optional -
	if s[0] == '-' {
		s = s[1:]
		if s == "" {
			return false
		}
	}
	// Digits
	switch {
	default:
		return false

	case s[0] == '0':
		s = s[1:]

	case '1' <= s[0] && s[0] <= '9':
		s = s[1:]
		for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
			s = s[1:]
		}
	}
	// . followed by 1 or more digits.
	if len(s) >= 2 && s[0] == '.' && '0' <= s[1] && s[1] <= '9' {
		s = s[2:]
		for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
			s = s[1:]
		}
	}
	// e or E followed by an optional - or + and
	// 1 or more digits.
	if len(s) >= 2 && (s[0] == 'e' || s[0] == 'E') {
		s = s[1:]
		if s[0] == '+' || s[0] == '-' {
			s = s[1:]
			if s == "" {
				return false
			}
		}
		for len(s) > 0 && '0' <= s[0] && s[0] <= '9' {
			s = s[1:]
		}
	}
	// Make sure we are at the end.
	return s == ""
}
