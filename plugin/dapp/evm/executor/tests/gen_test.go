package tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"text/template"
)

type TestData struct {
	Name string
	Code string
	Out  string
	Err  string
}

func clearTestCase(basePath string) {
	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if path == basePath || info == nil {
			return nil
		}

		if info.IsDir() {
			clearTestCase(path)
			return nil
		}

		baseName := info.Name()
		if filepath.Ext(path) == ".json" && baseName[:4] != "tpl_" && baseName[:5] != "data_" {
			os.Remove(path)
		}
		return nil
	})
}

func genTestCase(basePath string) {
	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}
		if path == basePath {
			return nil
		}
		scanTestData(path)
		return nil
	})
}

func scanTestData(basePath string) {
	var testmap = make(map[string]string)

	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		basename := info.Name()
		// 生成模板文件和数据文件之间的关系
		if filepath.Ext(path) == ".json" && basename[:5] == "data_" {
			re := regexp.MustCompile("([0-9]+)")
			match := re.FindStringSubmatch(basename)
			num := match[1]
			number, _ := strconv.Atoi(num)
			if !testCaseFilter.filter(number) {
				return nil
			}

			key := fmt.Sprintf("tpl_%s.json", num)
			value := fmt.Sprintf("data_%s.json", num)
			keyFile := basePath + string(filepath.Separator) + key
			dataFile := basePath + string(filepath.Separator) + value

			// 检查两个文件是否都存在
			if _, err := os.Stat(keyFile); os.IsNotExist(err) {
				fmt.Errorf("test template file:%s, not exists!", keyFile)
				return nil
			}
			if _, err := os.Stat(dataFile); os.IsNotExist(err) {
				fmt.Errorf("test data file:%s, not exists!", dataFile)
				return nil
			}
			testmap[keyFile] = dataFile
		}
		return nil
	})

	genTestFile(basePath, testmap)
}

func genTestFile(basePath string, datas map[string]string) {
	for k, v := range datas {

		tpldata, err := ioutil.ReadFile(k)
		if err != nil {
			fmt.Println(err)
			continue
		}
		testdata, err := ioutil.ReadFile(v)
		if err != nil {
			fmt.Println(err)
			continue
		}

		txt := string(tpldata)
		tpl := template.New("gen test data")
		tpl.Parse(txt)

		var datas []TestData
		json.Unmarshal(testdata, &datas)

		for _, v := range datas {
			if !testCaseFilter.filterCaseName(v.Name) {
				continue
			}
			fp, err := os.Create(basePath + string(filepath.Separator) + "generated_" + v.Name + ".json")
			if err != nil {
				fmt.Println(err)
				continue
			}
			tpl.Execute(fp, v)
		}
	}
}
