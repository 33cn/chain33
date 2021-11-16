// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"fmt"

	mty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
)

func managerIDKey(id string) []byte {
	return []byte(fmt.Sprintf("%s-%s", types.ManagePrefix+mty.ManageX+"-id", id))
}
