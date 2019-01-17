package grpcclient

import (
	"fmt"

	log "github.com/33cn/chain33/common/log/log15"
	"golang.org/x/net/context"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

var glog = log.New("module", "grpcclient")

const SwitchBalancer = "switch"

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(SwitchBalancer, &swPickerBuilder{})
}

func init() {
	balancer.Register(newBuilder())
}

type swPickerBuilder struct {
	masterAddr string
}

func (sw *swPickerBuilder) Build(readySCs map[resolver.Address]balancer.SubConn) balancer.Picker {
	glog.Debug(fmt.Sprintf("switchPicker: newPicker called with readySCs: %v", readySCs))
	var offset, step int
	var scs []balancer.SubConn
	var addrs []string
	for addr, sc := range readySCs {
		scs = append(scs, sc)
		addrs = append(addrs, addr.Addr)

		if sw.masterAddr != "" && sw.masterAddr == addr.Addr {
			offset = step
		}
		step++
	}
	if len(addrs) > 0 {
		sw.masterAddr = addrs[offset]
	}

	glog.Debug(fmt.Sprintf("master address:%s offset:%d", sw.masterAddr, offset))
	return &swPicker{
		subConns: scs,
		offset:   offset,
	}
}

type swPicker struct {
	subConns []balancer.SubConn
	offset   int
}

func (p *swPicker) Pick(ctx context.Context, opts balancer.PickOptions) (balancer.SubConn, func(balancer.DoneInfo), error) {
	if len(p.subConns) <= 0 {
		return nil, nil, balancer.ErrNoSubConnAvailable
	}

	glog.Debug(fmt.Sprintf("offset:%d", p.offset))
	return p.subConns[p.offset], nil, nil
}
