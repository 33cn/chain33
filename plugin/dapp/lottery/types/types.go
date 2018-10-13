package types

//Lottery op
const (
	LotteryActionCreate = 1 + iota
	LotteryActionBuy
	LotteryActionShow
	LotteryActionDraw
	LotteryActionClose
)

const (
	LotteryX = "lottery"
)
