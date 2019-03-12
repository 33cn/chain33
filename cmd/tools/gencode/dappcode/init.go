// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dappcode

import (
	_ "github.com/33cn/chain33/cmd/tools/gencode/dappcode/cmd"      //init cmd
	_ "github.com/33cn/chain33/cmd/tools/gencode/dappcode/commands" // init command
	_ "github.com/33cn/chain33/cmd/tools/gencode/dappcode/executor" // init executor
	_ "github.com/33cn/chain33/cmd/tools/gencode/dappcode/proto"    // init proto
	_ "github.com/33cn/chain33/cmd/tools/gencode/dappcode/rpc"      // init rpc
	_ "github.com/33cn/chain33/cmd/tools/gencode/dappcode/types"    // init types
)
