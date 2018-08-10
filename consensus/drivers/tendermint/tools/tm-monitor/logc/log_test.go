package logc

import (
	"fmt"
	"testing"
)

const (
	// 后期将各种常用正则补充完整
	IP_REGEX     = "[0-9]{3}\\.[0-9]{3}\\.[0-9]{1}\\.[0-9]{3}"
	DOMAIN_REGEX = ""
)

func TestLogMonitor(t *testing.T) {
	m := NewLogMonitor("ExecBlock", false)
	ch, err := m.GetCurrentKeyInfo()
	if err != nil {
		fmt.Println("Get current info failed.")
		panic(err)
	}

	for {
		select {
		case a := <-ch:
			fmt.Println(a.text)
			// output:
			// t=2018-08-10T16:02:30+0800 lvl=info msg=ExecBlock module=util height=1232 ntx=20 writebatchsync=true cost=12.657497ms
			// t=2018-08-10T16:02:31+0800 lvl=dbug msg=ExecBlock module=util height------->=1233 ntx=20
			// t=2018-08-10T16:02:31+0800 lvl=dbug msg=ExecBlock module=util prevtx=20 newtx=20
			// t=2018-08-10T16:02:31+0800 lvl=info msg=ExecBlock module=util height=1233 ntx=20 writebatchsync=true cost=10.980444ms
			// t=2018-08-10T16:02:32+0800 lvl=dbug msg=ExecBlock module=util height------->=1234 ntx=20
			// t=2018-08-10T16:02:32+0800 lvl=dbug msg=ExecBlock module=util prevtx=20 newtx=20

		}
	}
}

func TestLogMonitorRegex(t *testing.T) {
	m := NewLogMonitor(IP_REGEX, true)
	ch, err := m.GetCurrentKeyInfo()
	if err != nil {
		fmt.Println("Get current info failed.")
		panic(err)
	}

	for {
		select {
		case a := <-ch:
			fmt.Println(a.text)
			// output:
			// t=2018-08-10T15:28:49+0800 lvl=dbug msg=192.168.0.108 module=p2p
			// t=2018-08-10T15:28:49+0800 lvl=info msg=DetectNodeAddr module=p2p addr:=192.168.0.108
			// t=2018-08-10T15:28:49+0800 lvl=info msg=detectNodeAddr module=p2p LocalAddr=192.168.0.108
			// t=2018-08-10T15:28:49+0800 lvl=dbug msg=DetectionNodeAddr module=p2p AddBlackList=192.168.0.108:23802
			// t=2018-08-10T15:28:49+0800 lvl=dbug msg="Add our address to book" module=p2p addr=192.168.0.108:23802
			// t=2018-08-10T15:28:49+0800 lvl=dbug msg="Add our address to book" module=p2p addr=192.168.0.108:13802
		}
	}
}
