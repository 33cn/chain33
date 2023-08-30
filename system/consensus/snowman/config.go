// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package snowman

import (
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/consensus/snowball"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/validators"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/prometheus/client_golang/prometheus"
)

// Config wraps all the parameters needed for a snowman engine
type Config struct {
	common.AllGetsServer

	//ctx  *consensus.Context
	chainCfg *types.Chain33Config

	ctx        *snow.ConsensusContext
	log        logging.Logger
	registerer prometheus.Registerer
	//VM         block.ChainVM
	//Sender     common.Sender
	validators validators.Set
	params     snowball.Parameters
	consensus  snowman.Consensus
}
