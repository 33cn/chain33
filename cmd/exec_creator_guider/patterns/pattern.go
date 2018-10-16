package patterns

import "github.com/inconshreveable/log15"

type CreatePattern interface {
	Run(projName, clsName, actionName, propFile, templateFile string)
}

var (
	mlog = log15.New("pattern", "ecg")
)

func New(t string) CreatePattern {
	switch t {
	case "advance":
		return &advancePattern{}
	case "simple":
	}
	mlog.Error("Can not create pattern", "type", t)
	return nil
}
