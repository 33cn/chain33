package blackwhite

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/types"
	"github.com/stretchr/testify/require"
)

func Test_getWinnerAndLoser(t *testing.T) {

	a := action{}

	var addrRes []*types.AddressResult

	addres := &types.AddressResult{
		Addr:    "1",
		IsBlack: []bool{true, false, true, false},
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr:    "2",
		IsBlack: []bool{true, false, true, false},
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr:    "3",
		IsBlack: []bool{true, false, true, true},
	}
	addrRes = append(addrRes, addres)

	winers := a.getWinner(addrRes)
	require.Equal(t, "3", winers[0])
	losers := a.getLoser(addrRes)
	require.Equal(t, "1", losers[0])
	require.Equal(t, "2", losers[1])
	t.Logf("winers1 is %v", winers)
	t.Logf("losers1 is %v", losers)

	addres = &types.AddressResult{
		Addr:    "4",
		IsBlack: []bool{true, false, true, true},
	}
	addrRes = append(addrRes, addres)

	addres = &types.AddressResult{
		Addr:    "5",
		IsBlack: []bool{true, false, true, false},
	}
	addrRes = append(addrRes, addres)

	winers = a.getWinner(addrRes)
	require.Equal(t, 0, len(winers))
	losers = a.getLoser(addrRes)
	require.Equal(t, 5, len(losers))
	t.Logf("winers2 is %v", winers)
	t.Logf("losers2 is %v", losers)

	addres = &types.AddressResult{
		Addr:    "6",
		IsBlack: []bool{true, false, true, false, true, false, true, false, true, false},
	}
	addrRes = append(addrRes, addres)

	winers = a.getWinner(addrRes)
	require.Equal(t, "6", winers[0])
	losers = a.getLoser(addrRes)
	require.Equal(t, 5, len(losers))
	t.Logf("winers3 is %v", winers)
	t.Logf("losers3 is %v", losers)

}
