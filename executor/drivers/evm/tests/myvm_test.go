package tests

import (
	"testing"
	"io/ioutil"
	"fmt"
	"encoding/json"
)

func TestVM(t *testing.T) {
	raw,err := ioutil.ReadFile("testdata/VMTests/vmArithmeticTest/add0.json")
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	var data []VMJson

	json.Unmarshal(raw, &data)

	fmt.Println(data)
}