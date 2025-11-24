package snow

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
)

const (
	snowChitsID  = "/chain33/snowman/chits/1.0"
	snowGetBlock = "/chain33/snowman/get/1.0"
	//snowPutBlock  = "/chain33/snowman/put/1.0"
	snowPullQuery = "/chain33/snowman/pullq/1.0"
	snowPushQuery = "/chain33/snowman/pushq/1.0"
)

var log = log15.New("module", "dht.snowman")

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

type snowman struct {
	*protocol.P2PEnv
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	new(snowman).init(env)
}

func (s *snowman) init(env *protocol.P2PEnv) {

	s.P2PEnv = env
	protocol.RegisterEventHandler(types.EventSnowmanChits, s.handleEventChits)
	protocol.RegisterEventHandler(types.EventSnowmanGetBlock, s.handleEventGetBlock)
	//protocol.RegisterEventHandler(types.EventSnowmanPutBlock, s.handleEventPutBlock)
	protocol.RegisterEventHandler(types.EventSnowmanPullQuery, s.handleEventPullQuery)
	protocol.RegisterEventHandler(types.EventSnowmanPushQuery, s.handleEventPushQuery)

	protocol.RegisterStreamHandler(s.Host, snowChitsID, s.handleStreamChits)
	protocol.RegisterStreamHandler(s.Host, snowGetBlock, s.handleStreamGetBlock)
	protocol.RegisterStreamHandler(s.Host, snowPullQuery, s.handleStreamPullQuery)
	protocol.RegisterStreamHandler(s.Host, snowPushQuery, s.handleStreamPushQuery)
}
