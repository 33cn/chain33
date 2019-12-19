package p2pnext

import (
	"fmt"

	proto "github.com/libp2p/go-libp2p-core/protocol"
)

var protoId []proto.ID

func init() {
	fmt.Println("xxxxxxxxxxxxxxxxxxxprotos")
	logger.Info("xxxxxxxxxxxxxxxxxxxprotos", "init register", "")
	Register(PeerInfo, &PeerInfoProtol{})
	Register(Header, &HeaderInfoProtol{})
	Register(Download, &DownloadBlockProtol{})
	protoId = append(protoId, peerInfoReq, peerInfoResp)
	protoId = append(protoId, headerInfoReq, headerInfoResp)
	protoId = append(protoId, downloadBlockReq, downloadBlockResp)
}
