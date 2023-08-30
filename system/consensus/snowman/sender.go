// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import "github.com/ava-labs/avalanchego/ids"

// 向外部模块发送请求, blockchain/p2p等
type sender struct {
}

func (s *sender) sendChits(nodeID ids.NodeID, requestID uint32, votes []ids.ID) {

}

func (s *sender) sendGet(nodeID ids.NodeID, requestID uint32, containerID ids.ID) {

}

func (s *sender) sendPullQuery(nodeIDs ids.NodeIDSet, requestID uint32, containerID ids.ID) {

}
