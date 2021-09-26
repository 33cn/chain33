package statistical

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
)

const  (
	//p2p statistical information
	p2pStatisticalInfo ="/chain33/statistical/1.0.0"
)
func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

var log = log15.New("module", "p2p.statistical")

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	p:=Protocol{
		env,
	}
	protocol.RegisterStreamHandler(p.Host,p2pStatisticalInfo,p.handlerStreamStatistical)
}


type Protocol struct {
	*protocol.P2PEnv
}