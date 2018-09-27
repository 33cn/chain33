package executor

import "fmt"

func calcLotteryBuyPrefix(lotteryId string, addr string) []byte {
	key := fmt.Sprintf("lottery-buy:%s:%s", lotteryId, addr)
	return []byte(key)
}

func calcLotteryBuyRoundPrefix(lotteryId string, addr string, round int64) []byte {
	key := fmt.Sprintf("lottery-buy:%s:%s:%10d", lotteryId, addr, round)
	return []byte(key)
}

func calcLotteryBuyKey(lotteryId string, addr string, round int64, txId string) []byte {
	key := fmt.Sprintf("lottery-buy:%s:%s:%10d:%s", lotteryId, addr, round, txId)
	return []byte(key)
}

func calcLotteryDrawPrefix(lotteryId string) []byte {
	key := fmt.Sprintf("lottery-draw:%s", lotteryId)
	return []byte(key)
}

func calcLotteryDrawKey(lotteryId string, round int64) []byte {
	key := fmt.Sprintf("lottery-draw:%s:%10d", lotteryId, round)
	return []byte(key)
}

func calcLotteryKey(lotteryId string, status int32) []byte {
	key := fmt.Sprintf("lottery:%d:%s", status, lotteryId)
	return []byte(key)
}
