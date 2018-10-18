package types

import "gitlab.33.cn/chain33/chain33/types"

// IsExpire 检查FTXO是否过期，过期返回true
func (ftxos *FTXOsSTXOsInOneTx) IsExpire(blockheight, blocktime int64) bool {
	valid := ftxos.GetExpire()
	if valid == 0 {
		// Expire为0，返回false
		return false
	}
	// 小于expireBound的都是认为是区块高度差
	if valid <= types.ExpireBound {
		return valid <= blockheight
	}
	return valid <= blocktime
}

// SetExpire 设定过期
func (ftxos *FTXOsSTXOsInOneTx) SetExpire(tx *types.Transaction) {
	expire := tx.Expire
	if expire > types.ExpireBound {
		// FTXO的超时为时间时，则用Tx的过期时间加上12秒后认为超时
		ftxos.Expire = expire + 12
	} else {
		// FTXO的过期时间为区块高度时，则用Tx的高度+1
		ftxos.Expire = expire + 1
	}
}
