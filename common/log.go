package common

import "github.com/inconshreveable/log15"
import "os"

var mainHandler log15.Handler

func init() {
	resetWithLogLevel("error")
}

// 设置log输出级别
func SetLogLevel(logLevel string) {
	resetWithLogLevel(logLevel)
}

func resetWithLogLevel(logLevel string) {
	mainHandler = log15.LvlFilterHandler(
		getLevel(logLevel),
		log15.StreamHandler(os.Stdout, log15.TerminalFormat()),
	)
	log15.Root().SetHandler(mainHandler)
}

func getLevel(lvlString string) log15.Lvl {
	lvl, err := log15.LvlFromString(lvlString)
	if err != nil {
		return 5
	}
	return lvl
}

func MainHandler() log15.Handler {
	return mainHandler
}

func New(ctx ...interface{}) log15.Logger {
	return NewMain(ctx...)
}

func NewMain(ctx ...interface{}) log15.Logger {
	return log15.Root().New(ctx...)
}
