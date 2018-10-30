package strategy

import "fmt"

type importPackageStrategy struct {
	strategyBasic
}

func (this *importPackageStrategy) Run() error {
	fmt.Println("Begin run chain33 import packages.")
	return nil
}
