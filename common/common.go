// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package common contains various helper functions.
package common

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

var random *rand.Rand
var globalPointerMap = make(map[int64]interface{})
var globalPointerID int64
var gloabalMu sync.Mutex

//ErrPointerNotFound 指针没有找到
var ErrPointerNotFound = errors.New("ErrPointerNotFound")

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	go func() {
		for {
			time.Sleep(60 * time.Second)
			gloabalMu.Lock()
			pointerLen := len(globalPointerMap)
			gloabalMu.Unlock()
			if pointerLen > 10 {
				println("<==>global pointer count = ", len(globalPointerMap))
			}
		}
	}()
}

//StorePointer 保存指针返回int64
func StorePointer(p interface{}) int64 {
	gloabalMu.Lock()
	defer gloabalMu.Unlock()
	globalPointerID++
	for globalPointerMap[globalPointerID] != nil {
		globalPointerID++
	}
	globalPointerMap[globalPointerID] = p
	return globalPointerID
}

//RemovePointer 删除指针
func RemovePointer(id int64) {
	gloabalMu.Lock()
	defer gloabalMu.Unlock()
	delete(globalPointerMap, id)
}

//GetPointer 删除指针
func GetPointer(id int64) (interface{}, error) {
	gloabalMu.Lock()
	defer gloabalMu.Unlock()
	p, ok := globalPointerMap[id]
	if !ok {
		return nil, ErrPointerNotFound
	}
	return p, nil
}

//MinInt32 min
func MinInt32(left, right int32) int32 {
	if left > right {
		return right
	}
	return left
}

//MaxInt32 max
func MaxInt32(left, right int32) int32 {
	if left > right {
		return left
	}
	return right
}

//GetRandBytes 获取随机字节
func GetRandBytes(min, max int) []byte {
	length := max
	if min < max {
		length = min + random.Intn(max-min)
	}
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = byte(random.Intn(256))
	}
	return result
}

//GetRandString 获取随机字符串
func GetRandString(length int) string {
	return string(GetRandBytes(length, length))
}

var printString = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

//GetRandPrintString 获取随机可打印字符串
func GetRandPrintString(min, max int) string {
	l := max
	if min < max {
		l = min + random.Intn(max-min)
	}
	result := make([]byte, l)
	for i := 0; i < l; i++ {
		result[i] = printString[random.Intn(len(printString))]
	}
	return string(result)
}
