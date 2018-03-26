package p2p

import (
	"sync"
	"sync/atomic"
	"time"
)

var Filter *Filterdata

func NewFilter() *Filterdata {
	filter := new(Filterdata)
	filter.regSData = make(map[interface{}]time.Duration)
	filter.regRData = make(map[interface{}]time.Duration)
	return filter
}

type Filterdata struct {
	smtx     sync.Mutex
	rmtx     sync.Mutex
	isclose  int32
	regSData map[interface{}]time.Duration
	regRData map[interface{}]time.Duration
}

func (f *Filterdata) RegSendData(key interface{}) bool {
	f.smtx.Lock()
	defer f.smtx.Unlock()
	f.regSData[key] = time.Duration(time.Now().Unix())
	return true
}

func (f *Filterdata) RegRecvData(key interface{}) bool {
	f.rmtx.Lock()
	defer f.rmtx.Unlock()
	f.regRData[key] = time.Duration(time.Now().Unix())
	return true
}

func (f *Filterdata) QuerySendData(key interface{}) bool {
	f.smtx.Lock()
	defer f.smtx.Unlock()
	_, ok := f.regSData[key]
	return ok

}

func (f *Filterdata) QueryRecvData(key interface{}) bool {
	f.rmtx.Lock()
	defer f.rmtx.Unlock()
	_, ok := f.regRData[key]
	return ok

}

func (f *Filterdata) RemoveSendData(key interface{}) {
	f.smtx.Lock()
	defer f.smtx.Unlock()
	delete(f.regSData, key)
}

func (f *Filterdata) RemoveRecvData(key interface{}) {
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

func (f *Filterdata) ManageSendFilter() {
	ticker := time.NewTicker(time.Second * 30)
	var timeout int64 = 60
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			f.smtx.Lock()
			now := time.Now().Unix()
			for key, regtime := range f.regSData {
				if now-int64(regtime) > timeout {
					delete(f.regSData, key)
				}
			}
			f.smtx.Unlock()
		}

		if f.isClose() == false {
			return
		}
	}
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
