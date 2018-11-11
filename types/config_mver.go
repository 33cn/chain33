// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	fmt "fmt"
	"sort"
	"strings"

	tml "github.com/BurntSushi/toml"
)

//multi version config
type versionList struct {
	data     []int64
	key      string
	prefix   string
	suffix   string
	forkname map[int64]string
}

type mversion struct {
	title   string
	data    map[string]interface{}
	version map[string]*versionList
}

func newMversion(title, cfgstring string) *mversion {
	cfg := make(map[string]interface{})
	if _, err := tml.Decode(cfgstring, &cfg); err != nil {
		panic(err)
	}
	flat := FlatConfig(cfg)
	mver := &mversion{
		title:   title,
		data:    make(map[string]interface{}),
		version: make(map[string]*versionList),
	}
	for k, v := range flat {
		if !strings.HasPrefix(k, "mver.") {
			continue
		}
		mver.data[k] = v
	}
	return mver
}

func (m *mversion) Get(key string, height int64) (interface{}, error) {
	vlist, ok := m.version[key]
	if !ok {
		return m.get(key)
	}
	key = vlist.GetForkName(height)
	return m.get(key)
}

func (m *mversion) get(key string) (interface{}, error) {
	if data, ok := m.data[key]; ok {
		return data, nil
	}
	tlog.Error("mver config " + key + " not found")
	return nil, ErrNotFound
}

//根据title 计算fork 信息, 保证这个函数调用在title 的fork 初始化完成之后
func (m *mversion) UpdateFork() {
	for k := range m.data {
		//global fork
		//mver.forkname.name
		//mver.consensus.forkname.name
		items := strings.Split(k, ".")
		if len(items) < 3 {
			continue
		}
		forkname := items[len(items)-2]
		if !systemFork.HasFork(m.title, forkname) {
			//maybe sub forl
			//mver.exec.sub.token.forkname
			forkname = items[len(items)-3] + "." + items[len(items)-2]
			if !systemFork.HasFork(m.title, forkname) {
				continue
			}
		}
		id := systemFork.GetFork(title, forkname)
		items[len(items)-2] = items[len(items)-1]
		suffix := items[len(items)-1]
		prefix := strings.Join(items[0:len(items)-2], ".")
		items = items[0 : len(items)-1]
		key := strings.Join(items, ".")
		if _, ok := m.version[key]; !ok {
			m.version[key] = &versionList{key: key, prefix: prefix, suffix: suffix}
		}
		err := m.version[key].addItem(id, key, forkname)
		if err != nil {
			panic(err)
		}
	}
	//sort all []int data
	for k, v := range m.version {
		sort.Slice(v.data, func(i, j int) bool { return v.data[i] < v.data[j] })
		m.version[k] = v
	}
}

func (v *versionList) addItem(forkid int64, key, forkname string) error {
	if v.key != key {
		return fmt.Errorf("version list key not the same")
	}
	if len(v.forkname) == 0 {
		v.forkname = make(map[int64]string)
	}
	if _, ok := v.forkname[forkid]; ok {
		return fmt.Errorf(key + " same fork height is set: old name is '" + v.forkname[forkid] + "' new name is '" + forkname + "'")
	}
	v.forkname[forkid] = forkname
	v.data = append(v.data, forkid)
	return nil
}

func (v *versionList) GetForkName(height int64) string {
	if len(v.data) == 0 {
		return v.key
	}
	//find first big than [0, 10, 20] 11
	for i := len(v.data) - 1; i >= 0; i-- {
		if height >= v.data[i] {
			//fork find
			forkname := v.forkname[v.data[i]]
			if strings.Contains(forkname, ".") {
				items := strings.Split(forkname, ".")
				forkname = items[1]
			}
			s := v.prefix + "." + forkname + "." + v.suffix
			return s
		}
	}
	return v.key
}
