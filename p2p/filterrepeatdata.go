package p2p

import (
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
)

var Filter = NewFilter()

func NewFilter() *Filterdata {
	filter := new(Filterdata)
	filter.regRData, _ = lru.New(P2pCacheTxSize)
	return filter
}

type Filterdata struct {
	isclose  int32
	regRData *lru.Cache
}

func (f *Filterdata) RegRecvData(key string) bool {
	f.regRData.Add(key, time.Duration(time.Now().Unix()))
	return true
}

func (f *Filterdata) QueryRecvData(key string) bool {
	ok := f.regRData.Contains(key)
	return ok

}

func (f *Filterdata) RemoveRecvData(key string) {
	f.regRData.Remove(key)
}

func (f *Filterdata) Close() {
	atomic.StoreInt32(&f.isclose, 1)
}

func (f *Filterdata) isClose() bool {
	return atomic.LoadInt32(&f.isclose) == 1
}

func (f *Filterdata) ManageRecvFilter() {
	ticker := time.NewTicker(time.Second * 30)
	var timeout int64 = 60
	defer ticker.Stop()
	for {
		<-ticker.C
		now := time.Now().Unix()
		for _, key := range f.regRData.Keys() {
			regtime, _ := f.regRData.Get(key)
			if now-int64(regtime.(time.Duration)) < timeout {
				break
			}
			f.regRData.Remove(key)
		}

		if !f.isClose() {
			return
		}
	}
}
