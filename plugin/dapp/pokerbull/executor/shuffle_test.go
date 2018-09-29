package executor

import (
	"testing"
	"math/rand"
	"fmt"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
)

func shuffle(attr []int, src int64) []int {
	length := len(attr)
	if length == 0 {
		fmt.Println("Invalid Parameter.")
		return nil
	}

	var attrV = make([]int, length)
	copy(attrV, attr)

	rng := rand.New(rand.NewSource(src))
	for i := 0; i < length; i++ {
		idx := rng.Intn(length-1)
		tmpV := attrV[idx]
		attrV[idx] = attrV[length-i-1]
		attrV[length-i-1] = tmpV
	}

	return attrV
}

func TestShuffle(t *testing.T) {
	var attr = []int{0,1,2,3,4,5,6,7,8,9}
	var res = []int{0,0,0,0,0,0,0,0,0,0}

	for j := 0; j < 10; j++ {
		for i := 0; i < 100; i++ {
			attrV := shuffle(attr, int64(i))
			copy(res, attrV)
			fmt.Println(res)
		}
		//fmt.Println(res)
	}
}

var cardColor map[int]string
//func TestPoker_Shuffle(t *testing.T) {
//	poker := NewPoker()
//
//	cardColor = make(map[int]string)
//	cardColor[0] = "Spade"
//	cardColor[1] = "Diamond"
//	cardColor[2] = "Club"
//	cardColor[3] = "Heart"
//
//	for j := 0; j < 10; j++ {
//		poker.Shuffle(12345432)
//		fmt.Println(poker.card)
//		//for i := 0; i < len(poker.card); i++ {
//		//	fmt.Printf("%s%d\n", cardColor[poker.card[i]&0xff00>>8], poker.card[i]&0xff)
//		//}
//	}
//}
//
//func TestPoker_Deal(t *testing.T) {
//	poker := NewPoker()
//	poker.Shuffle(12345432)
//
//	for j := 0; j < 11; j++ {
//		fmt.Println(poker.Deal(j))
//	}
//}

func TestBlue(t *testing.T) {
	data := []int32{12,8,6,8,11}

	fmt.Println(Result(data))
}

func TestPointer(t *testing.T) {
	var dataS []*types.PBGameQuit
	var tmp *types.PBGameQuit
	for i:=0; i < 5; i++ {
		data := &types.PBGameQuit{}
		data.GameId = fmt.Sprintf("%d", i)
		if data.GameId == "3" {
			tmp = data
		}
		dataS = append(dataS, data)
	}

	fmt.Println(tmp.GameId)
	fmt.Println(dataS)
}