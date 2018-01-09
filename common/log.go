package common

import "github.com/inconshreveable/log15"
import "os"

func init() {
	//resetWithLogLevel("error")
}

// 设置log输出级别
func SetLogLevel(logLevel string) {
	resetWithLogLevel(logLevel)
}

func SetFileLog(file, logLevel string, logConsoleLevel string) {
	if file == "" {
		resetWithLogLevel(logLevel)
	} else {
		resetWithLogLevel2(file, logLevel, logConsoleLevel)
	}
}

/*
srvlog.SetHandler(log.MultiHandler(
    log.StreamHandler(os.Stderr, log.LogfmtFormat()),
    log.LvlFilterHandler(
        log.LvlError,
        log.Must.FileHandler("errors.json", log.JsonFormat()))))
*/
func resetWithLogLevel(logLevel string) {
	mainHandler := log15.LvlFilterHandler(
		getLevel(logLevel),
		log15.StreamHandler(os.Stdout, log15.TerminalFormat()),
	)
	log15.Root().SetHandler(mainHandler)
}

func resetWithLogLevel2(file, logLevel string, logConsoleLevel string) {
	stdouth := log15.LvlFilterHandler(
		getLevel(logConsoleLevel),
		log15.StreamHandler(os.Stdout, log15.TerminalFormat()),
	)

	fileh := log15.LvlFilterHandler(
		getLevel(logLevel),
		log15.Must.FileHandler(file, log15.LogfmtFormat()),
	)
	log15.Root().SetHandler(log15.MultiHandler(stdouth, fileh))
}

func getLevel(lvlString string) log15.Lvl {
	lvl, err := log15.LvlFromString(lvlString)
	if err != nil {
		return 5
	}
	return lvl
}

func New(ctx ...interface{}) log15.Logger {
	return NewMain(ctx...)
}

func NewMain(ctx ...interface{}) log15.Logger {
	return log15.Root().New(ctx...)
}
