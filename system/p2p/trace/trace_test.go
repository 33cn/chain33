package trace

import (
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"testing"
)

func TestNew(t *testing.T) {
	var waitChan = make(chan struct{})
	var env protocol.P2PEnv
	env.SubConfig=&types2.P2PSubConfig{}
	env.SubConfig.Trace.ListenAddr="localhost:19331"
	New(&env)

	<-waitChan

}
