// +build go1.8

package main
import (
	"fmt"
	"os"
	"path/filepath"

	mavl "gitlab.33.cn/chain33/chain33/system/store/mavl/db"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"os/signal"
	"syscall"
	log "github.com/inconshreveable/log15"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/types"
	"bufio"
)

func main() {
	log1 := &types.Log{
		Loglevel: "debug",
		LogConsoleLevel: "error",
		LogFile: "logs/syc.log",
		MaxFileSize: 400,
		MaxBackups: 100,
		MaxAge: 28,
		LocalTime: true,
		Compress: false,
	}
	clog.SetFileLog(log1)

	stdin := bufio.NewReader(os.Stdin)

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(dir)
	log.Info("test", dir)
	db := dbm.NewDB("store", "leveldb", dir, 100)

	a := 0
	fmt.Println("是否需要查询..mk..计数")
	fmt.Fscan(stdin, &a)
	stdin.ReadString('\n')
	if a > 0 {
		mavl.PruningTreePrint(db, []byte("..mk.."))
	}
	b := 0
	fmt.Println("是否需要查询_mh_计数")
	fmt.Fscan(stdin, &b)
	stdin.ReadString('\n')
	if b > 0 {
		mavl.PruningTreePrint(db, []byte("_mh_"))
	}
	c := 0
	fmt.Println("是否需要查询_mb_计数")
	fmt.Fscan(stdin, &c)
	stdin.ReadString('\n')
	if c > 0 {
		mavl.PruningTreePrint(db, []byte("_mb_"))
	}
	exit := make(chan os.Signal,10) //初始化一个channel
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM) //notify方法用来监听收到的信号
	sig := <-exit
	fmt.Println(sig.String())
}

