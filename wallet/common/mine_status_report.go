package common

type MineStatusReport interface {
	IsAutoMining() bool
	// 返回挖矿买票锁的状态
	IsTicketLocked() bool
}
