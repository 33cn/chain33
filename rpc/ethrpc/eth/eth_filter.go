package eth

import (
	"context"
	"fmt"
	"time"

	"github.com/33cn/chain33/rpc/ethrpc/types"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/rpc"
)

// Type ...
type Type byte

const (
	// UnknownSubscription indicates an unknown subscription type
	UnknownSubscription Type = iota
	// LogsSubscription queries for new or removed (chain reorg) logs
	LogsSubscription
)

type filter struct {
	typ      Type
	deadline *time.Timer // filter is inactiv when deadline triggers
	crit     *types.FilterQuery
	logs     []*types.EvmLog
	done     chan struct{}
	timeout  time.Duration
}

// NewFilter eth_newFilter
func (e *ethHandler) NewFilter(options *types.FilterQuery) (*rpc.ID, error) {
	logs := make(chan []*types.EvmLog)
	id := rpc.NewID()
	err := e.subscribeLogs(options, logs, id)
	if err != nil {
		return nil, err
	}
	e.filtersMu.Lock()
	e.filters[id] = &filter{typ: LogsSubscription, crit: options, timeout: e.filterTimeout,
		deadline: time.NewTimer(e.filterTimeout), logs: make([]*types.EvmLog, 0), done: make(chan struct{})}
	e.filtersMu.Unlock()
	go func() {
		for {
			select {
			case l := <-logs:
				e.filtersMu.Lock()
				if f, found := e.filters[id]; found {
					f.logs = append(f.logs, l...)
				}
				e.filtersMu.Unlock()
			case <-e.filters[id].done:
				e.grpcCli.UnSubEvent(context.Background(), &ctypes.ReqString{Data: string(id)})
				return
			}
		}
	}()
	return &id, nil
}

// UninstallFilter 取消filter eth_uninstallFilter
func (e *ethHandler) UninstallFilter(id rpc.ID) bool {
	e.filtersMu.Lock()
	f, found := e.filters[id]
	if found {
		delete(e.filters, id)
	}
	e.filtersMu.Unlock()
	if found {
		close(f.done)
	}

	return found
}

// GetFilterLogs eth_getfilterlogs returns the logs for the filter with the given id.
// If the filter could not be found an empty array of logs is returned.
// https://eth.wiki/json-rpc/API#eth_getfilterlogs
func (e *ethHandler) GetFilterLogs(ctx context.Context, id rpc.ID) ([]*types.EvmLog, error) {
	e.filtersMu.Lock()
	f, found := e.filters[id]
	e.filtersMu.Unlock()

	if !found || f.typ != LogsSubscription {
		return nil, fmt.Errorf("filter not found")
	}

	logs, err := e.GetLogs(f.crit)
	if err == nil {
		return returnLogs(logs), nil
	}
	return nil, err

}

// GetFilterChanges eth_getFilterChanges
func (e *ethHandler) GetFilterChanges(id rpc.ID) (interface{}, error) {
	e.filtersMu.Lock()
	defer e.filtersMu.Unlock()
	//5分钟内调用此函数有效，然后重新计时5分钟，
	if f, found := e.filters[id]; found {
		if !f.deadline.Stop() { //停止计时，如果stop 返回false 说明计时器结束了
			// timer expired but filter is not yet removed in timeout loop
			// receive timer value and reset timer
			<-f.deadline.C
		}
		//重置计时器
		f.deadline.Reset(f.timeout)
		switch f.typ {
		case LogsSubscription:
			logs := f.logs
			f.logs = nil
			return returnLogs(logs), nil

		default:
			return nil, fmt.Errorf("no support")
		}
	}
	return []interface{}{}, fmt.Errorf("filter not found")
}

// returnLogs is a helper that will return an empty log array in case the given logs array is nil,
// otherwise the given logs array is returned.
func returnLogs(logs []*types.EvmLog) []*types.EvmLog {
	if logs == nil {
		return []*types.EvmLog{}
	}
	return logs
}

// timeoutLoop 定时器定时检测是否有超时的filter,超时则删除处理
func (e *ethHandler) timeoutLoop(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	for {
		<-ticker.C
		e.filtersMu.Lock()
		for id, f := range e.filters {
			select {
			case <-f.deadline.C:
				close(f.done)
				delete(e.filters, id)

			default:
				continue
			}
		}
		e.filtersMu.Unlock()
	}
}
