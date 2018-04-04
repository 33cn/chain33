package p2p

import (
	"sync"
	"sync/atomic"
	"time"
)

var Filter = NewFilter()

func NewFilter() *Filterdata {
	filter := new(Filterdata)
	filter.regRData = make(map[string]time.Duration)
	return filter
}

type Filterdata struct {
	rmtx     sync.Mutex
	isclose  int32
	regRData map[string]time.Duration
}

func (f *Filterdata) RegRecvData(key string) bool {
	f.rmtx.Lock()
	defer f.rmtx.Unlock()
	f.regRData[key] = time.Duration(time.Now().Unix())
	return true
}

func (f *Filterdata) QueryRecvData(key string) bool {
	f.rmtx.Lock()
	defer f.rmtx.Unlock()
	_, ok := f.regRData[key]
	return ok

}

func (f *Filterdata) RemoveRecvData(key string) {
	f.rmtx.Lock()
	defer f.rmtx.Unlock()
	delete(f.regRData, key)
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

		select {

		case <-ticker.C:
			f.rmtx.Lock()
			now := time.Now().Unix()
			for key, regtime := range f.regRData {
				if now-int64(regtime) > timeout {
					delete(f.regRData, key)
				}
			}
			f.rmtx.Unlock()
		}
		if f.isClose() == false {
			return
		}
	}
}
