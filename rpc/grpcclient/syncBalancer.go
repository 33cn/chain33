package grpcclient

import (
	"errors"
	"math/rand"
	"sync"

	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
)

func init() {
	balancer.Register(newBuilder())
}

// SyncLbName is the name of sync balancer.
const SyncLbName = "sync"

// attributeKey is the type used as the key to store AddrInfo in the Attributes
// field of resolver.Address.
type attributeKey struct{}

// AddrInfo will be stored inside Address metadata in order to use weighted balancer.
type AddrInfo struct {
	State int
}

// SetAddrInfo returns a copy of addr in which the Attributes field is updated
// with addrInfo.
func SetAddrInfo(addr resolver.Address, addrInfo AddrInfo) resolver.Address {
	addr.Attributes = attributes.New(attributeKey{}, addrInfo)
	return addr
}

// GetAddrInfo returns the AddrInfo stored in the Attributes fields of addr.
func GetAddrInfo(addr resolver.Address) AddrInfo {
	v := addr.Attributes.Value(attributeKey{})
	ai, _ := v.(AddrInfo)
	return ai
}

// NewBuilder creates a new weight balancer builder.
func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(SyncLbName, &syncPickerBuilder{}, base.Config{HealthCheck: false})
}

type syncPickerBuilder struct{}

// Build return picker
func (*syncPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}
	var scs []balancer.SubConn
	for subConn, addr := range info.ReadySCs {
		node := GetAddrInfo(addr.Address)
		if node.State == SYNC {
			scs = append(scs, subConn)
		}
	}
	return &syncPicker{
		subConns: scs,
	}
}

type syncPicker struct {
	// subConns is the snapshot of the roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns []balancer.SubConn

	mu sync.Mutex
}

// Pick return subConn
func (p *syncPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.subConns) == 0 {
		return balancer.PickResult{}, errors.New("no SubConn is sync")
	}
	index := rand.Intn(len(p.subConns))
	sc := p.subConns[index]
	return balancer.PickResult{SubConn: sc}, nil
}
