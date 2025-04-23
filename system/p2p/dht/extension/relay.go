package extension

/*
   电路中继
   背景：
   电路中继是一种传输协议，它通过第三方“中继”对等点在两个对等点之间路由流量。

   在许多情况下，对等点将无法以公开访问的方式穿越其 NAT 和/或防火墙。或者它们可能不共享允许它们直接通信的通用传输协议。

   为了在面临 NAT 等连接障碍时启用对等架构，libp2p定义了一个名为 p2p-circuit 的协议。当对等点无法监听公共地址时，它可以拨出到中继对等点，
   这将保持长期连接打开。其他对等点将能够使用一个p2p-circuit地址通过中继对等点拨号，该地址会将流量转发到其目的地。
*/

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	circuitClient "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	circuit "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

// Relay p2p relay
type Relay struct {
	crelay *circuit.Relay
	//client *circuitClient.Client
}

func newRelay(host host.Host, opts []circuit.Option) (*circuit.Relay, error) {
	//配置节点提供中继服务
	r, err := circuit.New(host, opts...)
	if err != nil {
		return nil, err
	}

	return r, nil
}

// MakeNodeRelayService  provide relay services
func MakeNodeRelayService(host host.Host, opts []circuit.Option) *Relay {
	r := new(Relay)
	var err error
	r.crelay, err = newRelay(host, opts)
	if err != nil {
		return nil
	}

	return r
}

// ReserveRelaySlot reserve relay slot
func ReserveRelaySlot(ctx context.Context, host host.Host, relayInfo peer.AddrInfo, timeout time.Duration) {
	//向中继节点申请一个通信插槽

	var ticker = time.NewTicker(timeout)
	var err error
	var reserve *circuitClient.Reservation
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			if reserve == nil || reserve.Expiration.Before(time.Now()) {
				reserve, err = circuitClient.Reserve(context.Background(), host, relayInfo)
				if err != nil {
					log.Error("ReserveRelaySlot", "err", err)
					return
				}
			}
		}
	}

}

// Close close relay
func (r *Relay) Close() {
	r.crelay.Close()
}
