package main

import (
	"time"

	"github.com/libp2p/go-libp2p-kad-dht/crawler"
	"github.com/qianlnk/pgbar"
)

var finished bool
var barDone = make(chan struct{})

// 模拟显示进度条,并非真实进度
// rate 100% (100/100) [================================================] 1.31  ps 00:01:15 in: 00:00:00
func barRun(handleSuccess crawler.HandleQueryResult) {
	pgbar.Println("\ndht_crawler rate bar")
	bar := pgbar.NewBar(0, "rate", 100)
	go func() {
		for i := 0; i < 100; i++ {
			bar.Add(1)
			if finished {
				time.Sleep(time.Millisecond * 300)
				continue
			}
			if i < 80 {
				time.Sleep(time.Millisecond * 900)
			} else {
				time.Sleep(time.Millisecond * 1500) //超过80%进度以后，降低进度条的速率
			}
		}

		barDone <- struct{}{}

	}()
}
