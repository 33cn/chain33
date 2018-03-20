package p2p

import (
	"sync"
	"time"
)

var Filter *Filterdata

func NewFilter() *Filterdata {
	filter := new(Filterdata)
	filter.loopDone = make(chan struct{}, 1)
	filter.regData = make(map[interface{}]time.Duration)
	return filter
}

type Filterdata struct {
	mtx      sync.Mutex
	loopDone chan struct{}
	regData  map[interface{}]time.Duration
}

func (f *Filterdata) RegData(key interface{}) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	f.regData[key] = time.Duration(time.Now().Unix())
	return true
}

func (f *Filterdata) QueryData(key interface{}) bool {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	_, ok := f.regData[key]
	return ok

}
func (f *Filterdata) RemoveData(key interface{}) {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	delete(f.regData, key)
}
func (f *Filterdata) Close() {
	close(f.loopDone)
}
func (f *Filterdata) ManageFilter() {
	ticker := time.NewTicker(time.Second * 30)
	var timeout int64 = 60
	defer ticker.Stop()
	for {

		select {
		case <-f.loopDone:
			log.Debug("peer mangerFilterTask", "loop", "done")
			return
		case <-ticker.C:
			f.mtx.Lock()
			now := time.Now().Unix()
			for key, regtime := range f.regData {
				if now-int64(regtime) > timeout {
					delete(f.regData, key)
				}
			}
			f.mtx.Unlock()

		}
	}
}
