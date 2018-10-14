package types

//Lottery op
const (
	LotteryActionCreate = 1 + iota
	LotteryActionBuy
	LotteryActionShow
	LotteryActionDraw
	LotteryActionClose

	//log for lottery
	TyLogLotteryCreate = 801
	TyLogLotteryBuy    = 802
	TyLogLotteryDraw   = 803
	TyLogLotteryClose  = 804
)

const (
	LotteryX = "lottery"
)

//Lottery status
const (
	LotteryCreated = 1 + iota
	LotteryPurchase
	LotteryDrawed
	LotteryClosed
)
