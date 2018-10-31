package executor

import (
	"fmt"
	"math/rand"
	"sort"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
)

var POKER_CARD_NUM = 52 //4 * 13 不带大小王
var COLOR_OFFSET uint32 = 8
var COLOR_BIT_MAST = 0xFF
var COLOR_NUM = 4
var CARD_NUM_PER_COLOR = 13
var CARD_NUM_PER_GAME = 5

const (
	POKERBULL_RESULT_X1    = 1
	POKERBULL_RESULT_X2    = 2
	POKERBULL_RESULT_X3    = 3
	POKERBULL_RESULT_X4    = 4
	POKERBULL_RESULT_X5    = 5
	POKERBULL_LEVERAGE_MAX = POKERBULL_RESULT_X1
)

func NewPoker() *types.PBPoker {
	poker := new(types.PBPoker)
	poker.Cards = make([]int32, POKER_CARD_NUM)
	poker.Pointer = int32(POKER_CARD_NUM - 1)

	for i := 0; i < POKER_CARD_NUM; i++ {
		color := i / CARD_NUM_PER_COLOR
		num := i%CARD_NUM_PER_COLOR + 1
		poker.Cards[i] = int32(color<<COLOR_OFFSET + num)
	}
	return poker
}

// 洗牌
func Shuffle(poker *types.PBPoker, rng int64) {
	rndn := rand.New(rand.NewSource(rng))

	for i := 0; i < POKER_CARD_NUM; i++ {
		idx := rndn.Intn(POKER_CARD_NUM - 1)
		tmpV := poker.Cards[idx]
		poker.Cards[idx] = poker.Cards[POKER_CARD_NUM-i-1]
		poker.Cards[POKER_CARD_NUM-i-1] = tmpV
	}
	poker.Pointer = int32(POKER_CARD_NUM - 1)
}

// 发牌
func Deal(poker *types.PBPoker, rng int64) []int32 {
	if poker.Pointer < int32(CARD_NUM_PER_GAME) {
		logger.Error(fmt.Sprintf("Wait to be shuffled: deal cards [%d], left [%d]", CARD_NUM_PER_GAME, poker.Pointer+1))
		Shuffle(poker, rng+int64(poker.Cards[0]))
	}

	rndn := rand.New(rand.NewSource(rng))
	res := make([]int32, CARD_NUM_PER_GAME)
	for i := 0; i < CARD_NUM_PER_GAME; i++ {
		idx := rndn.Intn(int(poker.Pointer))
		tmpV := poker.Cards[poker.Pointer]
		res[i] = poker.Cards[idx]
		poker.Cards[idx] = tmpV
		poker.Cards[poker.Pointer] = res[i]
		poker.Pointer--
	}

	return res
}

// 计算斗牛结果
func Result(cards []int32) int32 {
	temp := 0
	r := -1 //是否有牛标志

	pk := newcolorCard(cards)

	//花牌等于10
	cardsC := make([]int, len(cards))
	for i := 0; i < len(pk); i++ {
		if pk[i].num > 10 {
			cardsC[i] = 10
		} else {
			cardsC[i] = pk[i].num
		}
	}

	//斗牛算法
	result := make([]int, 10)
	var offset = 0
	for x := 0; x < 3; x++ {
		for y := x + 1; y < 4; y++ {
			for z := y + 1; z < 5; z++ {
				if (cardsC[x]+cardsC[y]+cardsC[z])%10 == 0 {
					for j := 0; j < len(cardsC); j++ {
						if j != x && j != y && j != z {
							temp += cardsC[j]
						}
					}

					if temp%10 == 0 {
						r = 10 //若有牛，且剩下的两个数也是牛十
					} else {
						r = temp % 10 //若有牛，剩下的不是牛十
					}
					result[offset] = r
					offset++
				}
			}
		}
	}

	//没有牛
	if r == -1 {
		return -1
	}

	return int32(result[0])
}

// 计算结果倍数
func Leverage(hand *types.PBHand) int32 {
	result := hand.Result

	// 小牛 [1, 6]
	if result < 7 {
		return POKERBULL_RESULT_X1
	}

	// 大牛 [7, 9]
	if result >= 7 && result < 10 {
		return POKERBULL_RESULT_X2
	}

	var flowers = 0
	if result == 10 {
		for _, card := range hand.Cards {
			if (int(card) & COLOR_BIT_MAST) > 10 {
				flowers++
			}
		}

		// 牛牛
		if flowers < 4 {
			return POKERBULL_RESULT_X3
		}

		// 四花
		if flowers == 4 {
			return POKERBULL_RESULT_X4
		}

		// 五花
		if flowers == 5 {
			return POKERBULL_RESULT_X5
		}
	}

	return POKERBULL_RESULT_X1
}

type pokerCard struct {
	num   int
	color int
}

type colorCardSlice []*pokerCard

func (p colorCardSlice) Len() int {
	return len(p)
}
func (p colorCardSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p colorCardSlice) Less(i, j int) bool {
	if i >= p.Len() || j >= p.Len() {
		logger.Error("length error. slice length:", p.Len(), " compare lenth: ", i, " ", j)
	}

	if p[i] == nil || p[j] == nil {
		logger.Error("nil pointer at ", i, " ", j)
	}
	return p[i].num < p[j].num
}

func newcolorCard(a []int32) colorCardSlice {
	var cardS []*pokerCard
	for i := 0; i < len(a); i++ {
		num := int(a[i]) & COLOR_BIT_MAST
		color := int(a[i]) >> COLOR_OFFSET
		cardS = append(cardS, &pokerCard{num, color})
	}

	return cardS
}

func CompareResult(i, j *types.PBHand) bool {
	if i.Result < j.Result {
		return true
	}

	if i.Result == j.Result {
		return Compare(i.Cards, j.Cards)
	}

	return false
}

func Compare(a []int32, b []int32) bool {
	cardA := newcolorCard(a)
	cardB := newcolorCard(b)

	if !sort.IsSorted(cardA) {
		sort.Sort(cardA)
	}
	if !sort.IsSorted(cardB) {
		sort.Sort(cardB)
	}

	maxA := cardA[len(a)-1]
	maxB := cardB[len(b)-1]
	if maxA.num != maxB.num {
		return maxA.num < maxB.num
	}

	return maxA.color < maxB.color
}
