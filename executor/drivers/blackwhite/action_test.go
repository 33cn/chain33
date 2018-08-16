package blackwhite

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
	"errors"
	"fmt"
	"strconv"
)


func newAddressResult(Addr string, blackwhite []int) *types.AddressResult{
	showSecret := "1234"
	var HashValues [][]byte
	for _, v:= range blackwhite {
		HashValues = append(HashValues, common.Sha256([]byte(showSecret + strconv.Itoa(v))))
	}

	return &types.AddressResult{
		Addr:       Addr,
		HashValues: HashValues,
		ShowSecret: showSecret,
	}
}

// 参数: 比对次数， 参加的数字， 那几个人赢了
func newGameRound(name string, loop int32, blackwhite [][]int, win ...int32) ([]*addrResult, error) {
	a := action{}
	var addrRes []*types.AddressResult
	round := &types.BlackwhiteRound{
		Loop: loop,
	}

	for i, v := range blackwhite {
		addrRes = append(addrRes, newAddressResult(fmt.Sprintf("%d", i), v))
	}

	round.AddrResult = addrRes
	winers, _ := a.getWinner(round)
	if len(win) != len(winers) {
		return winers, errors.New(fmt.Sprintln(name, " len err: ", len(win), " not equal", len(winers)))
	}

	if winers[0].addr != fmt.Sprintf("%d", win[0]) {
		return winers, errors.New(fmt.Sprintln(name, " win err: ", win[0], " not equal", winers[0].addr))
	}

	if len(winers) == 2 {
		if winers[1].addr != fmt.Sprintf("%d", win[1]) {
			return winers, errors.New(fmt.Sprintln(name, " win err: ", win[1], " not equal", winers[1].addr))
		}
	}
	return winers, nil
}

func Test_getWinnerLoser(t *testing.T) {
	test1(t)
	test2(t)
	test3(t)
	test4(t)
	test5(t)
}

func test1(t *testing.T) {
	var inputRet [][]int
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 1, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 1, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0}) // in 3 loop win
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1})

	_, err := newGameRound("第 1 次游戏", 7, inputRet, 5)
	if err != nil {
		fmt.Println(err)
		require.Error(t, err)
	}
}

func test2(t *testing.T) {
	var inputRet [][]int
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1}) // in 6 loop win
	inputRet = append(inputRet, []int{1, 1, 1, 0, 1, 0, 0}) // in 6 loop win
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 1, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{1, 0, 1, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1})

	_, err := newGameRound("第 2 次游戏", 7, inputRet, 0, 1)
	if err != nil {
		fmt.Println(err)
		require.Error(t, err)
	}
}

func test3(t *testing.T) {
	var inputRet [][]int
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 1, 1, 0, 1, 0, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 1, 0, 0, 1, 1}) // in 10 loop win
	inputRet = append(inputRet, []int{1, 1, 1, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 1, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 1, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})

	_, err := newGameRound("第 3 次游戏", 10, inputRet, 2)
	if err != nil {
		fmt.Println(err)
		require.Error(t, err)
	}
}

func test4(t *testing.T) {
	var inputRet [][]int
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 0, 0, 1, 0}) // in 10 loop win
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 0, 0, 1, 0}) // in 10 loop win
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 1, 1, 0, 1, 0, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 1, 0, 0, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 1, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 1, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 1, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{1, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})

	_, err := newGameRound("第 4 次游戏", 10, inputRet, 0, 1)
	if err != nil {
		fmt.Println(err)
		require.Error(t, err)
	}
}

func test5(t *testing.T) {
	var inputRet [][]int
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 0}) // in 10 loop win
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 0}) // in 10 loop win
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 0}) // in 10 loop win
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{1, 1, 0, 0, 1, 0, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 1, 0, 1, 1})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 1, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 1, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 0, 0, 1, 0})
	inputRet = append(inputRet, []int{0, 0, 0, 0, 1, 1, 0, 0, 1, 0})

	_, err := newGameRound("第 5 次游戏", 10, inputRet, 1, 2, 3)
	if err != nil {
		fmt.Println(err)
		require.Error(t, err)
	}
}

func Test_getWinnerAndLoser(t *testing.T) {

	a := action{}

	showSecret := "123456789012345678901234567890"

	var addrRes []*types.AddressResult

	round := &types.BlackwhiteRound{
		Loop: 4,
	}

	addres := &types.AddressResult{
		Addr: "1",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr: "2",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr: "3",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + black))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	round.AddrResult = addrRes

	winers, _ := a.getWinner(round)
	require.Equal(t, "3", winers[0].addr)
	losers := a.getLoser(round)
	require.Equal(t, "1", losers[0].addr)
	require.Equal(t, "2", losers[1].addr)
	//t.Logf("winers1 is %v", winers)
	//t.Logf("losers1 is %v", losers)

	addres = &types.AddressResult{
		Addr: "4",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + black))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr: "5",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	round.AddrResult = addrRes

	winers, _ = a.getWinner(round)
	require.Equal(t, 2, len(winers))
	losers = a.getLoser(round)
	require.Equal(t, 3, len(losers))
	//t.Logf("winers2 is %v", winers)
	//t.Logf("losers2 is %v", losers)

	addres = &types.AddressResult{
		Addr: "6",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white)),
			common.Sha256([]byte(showSecret + black)),
			common.Sha256([]byte(showSecret + white))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	round.AddrResult = addrRes

	winers, _ = a.getWinner(round)
	require.Equal(t, "3", winers[0].addr)
	losers = a.getLoser(round)
	require.Equal(t, 4, len(losers))
	//t.Logf("winers3 is %v", winers)
	//t.Logf("losers3 is %v", losers)

}

func Test_caclloopNumByPlayer(t *testing.T) {

	var player int32
	var loop int32

	player = 16
	loop = calcloopNumByPlayer(player)
	require.Equal(t, 5, int(loop))

	player = 70
	loop = calcloopNumByPlayer(player)
	require.Equal(t, 7, int(loop))

	player = 100
	loop = calcloopNumByPlayer(player)
	require.Equal(t, 8, int(loop))

	player = 100000
	loop = calcloopNumByPlayer(player)
	require.Equal(t, 18, int(loop))
}

