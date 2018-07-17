package types

import "time"

// IsExpire 检查FTXO是否过期，过期返回true
func (ftxos *FTXOsSTXOsInOneTx) IsExpire(blockheight, blocktime int64) bool {
	valid := ftxos.GetExpire()
	if valid == 0 {
		// Expire为0，返回false
		return false
	}
	// 小于expireBound的都是认为是区块高度差
	if valid <= expireBound {
		return valid <= blockheight
	}
	return valid <= blocktime
}

// SetExpire 设定过期
func (ftxos *FTXOsSTXOsInOneTx) SetExpire(tx *Transaction) {
	expire := tx.Expire
	if expire > expireBound {
		minTimeDuration := time.Second * 120
		timeexpire := time.Duration(expire)
		if timeexpire < minTimeDuration {
			timeexpire = minTimeDuration
		}
		// FTXO的超时为时间时，则用Tx的过期时间加上6秒后认为超时
		ftxos.Expire = Now().Unix() + int64(timeexpire/time.Second+6)
	} else {
		// FTXO的过期时间为区块高度时，则用Tx的高度+1
		ftxos.Expire = int64(expire + 1)
	}
}
