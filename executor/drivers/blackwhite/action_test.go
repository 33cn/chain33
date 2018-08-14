package blackwhite

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

func Test_getWinnerAndLoser(t *testing.T) {

	a := action{}

	showSecret := "123456789012345678901234567890"

	var addrRes []*types.AddressResult

	round := &types.BlackwhiteRound{
		Loop: 7,
	}

	addres := &types.AddressResult{
		Addr: "1",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white)))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr: "2",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white)))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr: "3",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(black)))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	round.AddrResult = addrRes

	winers := a.getWinner(round)
	require.Equal(t, "3", winers[0].addr)
	losers := a.getLoser(round)
	require.Equal(t, "1", losers[0].addr)
	require.Equal(t, "2", losers[1].addr)
	//t.Logf("winers1 is %v", winers)
	//t.Logf("losers1 is %v", losers)

	addres = &types.AddressResult{
		Addr: "4",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(black)))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr: "5",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white)))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	round.AddrResult = addrRes

	winers = a.getWinner(round)
	require.Equal(t, 0, len(winers))
	losers = a.getLoser(round)
	require.Equal(t, 5, len(losers))
	//t.Logf("winers2 is %v", winers)
	//t.Logf("losers2 is %v", losers)

	addres = &types.AddressResult{
		Addr: "6",
		HashValues: [][]byte{common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white))),
			common.Sha256([]byte(showSecret + string(black))),
			common.Sha256([]byte(showSecret + string(white)))},
		ShowSecret: showSecret,
	}
	addrRes = append(addrRes, addres)

	round.AddrResult = addrRes

	winers = a.getWinner(round)
	require.Equal(t, "6", winers[0].addr)
	losers = a.getLoser(round)
	require.Equal(t, 5, len(losers))
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

