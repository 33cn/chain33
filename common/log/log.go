package log

import (
	"os"

	"code.aliyun.com/chain33/chain33/types"

	"github.com/inconshreveable/log15"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// 保存日志处理器的引用，方便后续调整日志信息，而不重新初始化
	fileHandler    *log15.Handler
	consoleHandler *log15.Handler
)

func init() {
	//resetWithLogLevel("error")
}

// 设置控制台日志输出级别
// [fixbug]原逻辑在已经设置文件日志的情况下会覆盖文件的Handler
// TODO 可以对接命令行接口，运行态修改日志级别
func SetLogLevel(logLevel string) {
	handler := getConsoleLogHandler(logLevel)
	(*handler).SetMaxLevel(int(getLevel(logLevel)))
}

// 设置文件日志和控制台日志信息
func SetFileLog(log *types.Log) {
	if log == nil {
		log = &types.Log{LogFile: "logs/chain33.log"}
	}
	if log.LogFile == "" {
		SetLogLevel(log.LogConsoleLevel)
	} else {
		resetLog(log)
	}
}

// 清空原来所有的日志Handler，根据配置文件信息重置文件和控制台日志
func resetLog(log *types.Log) {
	fillDefaultValue(log)
	log15.Root().SetHandler(log15.MultiHandler(*getConsoleLogHandler(log.LogConsoleLevel), *getFileLogHandler(log)))
}

// 保证默认性况下为info级别，防止debug打印太多日志
func fillDefaultValue(log *types.Log) {
	if log.Loglevel == "" {
		log.Loglevel = log15.LvlInfo.String()
	}
	if log.LogConsoleLevel == "" {
		log.LogConsoleLevel = log15.LvlInfo.String()
	}
}

func getConsoleLogHandler(logLevel string) *log15.Handler {
	if consoleHandler != nil {
		return consoleHandler
	}
	format := log15.TerminalFormat()

	stdouth := log15.LvlFilterHandler(
		getLevel(logLevel),
		log15.StreamHandler(os.Stdout, format),
	)

	consoleHandler = &stdouth

	return &stdouth
}

func getFileLogHandler(log *types.Log) *log15.Handler {
	if fileHandler != nil {
		return fileHandler
	}

	rotateLogger := &lumberjack.Logger{
		Filename:   log.LogFile,
		MaxSize:    int(log.MaxFileSize),
		MaxBackups: int(log.MaxBackups),
		MaxAge:     int(log.MaxAge),
		LocalTime:  log.LocalTime,
		Compress:   log.Compress,
	}

	fileh := log15.LvlFilterHandler(
		getLevel(log.Loglevel),
		log15.StreamHandler(rotateLogger, log15.LogfmtFormat()),
	)

	// 增加打印调用源文件、方法和代码行的判断
	if log.CallerFile {
		fileh = log15.CallerFileHandler(fileh)
	}
	if log.CallerFunction {
		fileh = log15.CallerFuncHandler(fileh)
	}

	fileHandler = &fileh

	return &fileh
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
