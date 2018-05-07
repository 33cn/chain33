package testdata

import (
	"testing"
	"io/ioutil"
	"fmt"
	"html/template"
	"os"
)

type ariTestData struct {
	code string
	out string
}

func TestGenTest(t *testing.T)  {
	data,err := ioutil.ReadFile("testdata/VMTests/vmArithmeticTest/tpl0.json")
	if err != nil{
		fmt.Println(err)
		return
	}

	tpl := template.New(string(data))

	tpl.Execute(os.Stdout, ariTestData{"aaaaaa", "bbbbb"})
}