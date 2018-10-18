package wallet

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

func Test_checkAmountValid(t *testing.T) {
	testCases := []struct {
		amount int64
		actual bool
	}{
		{amount: 0, actual: false},
		{amount: -1, actual: false},
		{amount: 1*types.Coin + 100, actual: false},
		{amount: 5 * types.Coin, actual: true},
	}
	for _, test := range testCases {
		ret := checkAmountValid(test.amount)
		require.Equal(t, ret, test.actual)
	}
}

func Test_decomAmount2Nature(t *testing.T) {
	testCase := []struct {
		amount int64
		order  int64
		actual []int64
	}{
		{
			amount: 0,
			order:  0,
			actual: []int64{},
		},
		{
			amount: -1,
			order:  0,
			actual: []int64{},
		},
		{
			amount: 2,
			order:  0,
			actual: []int64{},
		},
		{
			amount: 2,
			order:  1,
			actual: []int64{2},
		},
		{
			amount: 3,
			order:  1,
			actual: []int64{1, 2},
		},
		{
			amount: 4,
			order:  1,
			actual: []int64{2, 2},
		},
		{
			amount: 5,
			order:  1,
			actual: []int64{5},
		},
		{
			amount: 6,
			order:  1,
			actual: []int64{5, 1},
		},
		{
			amount: 7,
			order:  1,
			actual: []int64{5, 2},
		},
		{
			amount: 8,
			order:  1,
			actual: []int64{5, 2, 1},
		},
		{
			amount: 9,
			order:  1,
			actual: []int64{5, 2, 2},
		},
	}
	for index, test := range testCase {
		res := decomAmount2Nature(test.amount, test.order)
		require.Equalf(t, res, test.actual, "testcase index %d", index)
	}
}

func Test_decomposeAmount2digits(t *testing.T) {
	testCase := []struct {
		amount         int64
		dust_threshold int64
		actual         []int64
	}{
		{
			amount:         0,
			dust_threshold: 0,
			actual:         []int64{},
		},
		{
			amount:         -1,
			dust_threshold: 0,
			actual:         []int64{},
		},
		{
			amount:         2,
			dust_threshold: 1,
			actual:         []int64{2},
		},
		{
			amount:         62387455827,
			dust_threshold: types.BTYDustThreshold,
			actual:         []int64{87455827, 1e8, 2e8, 2e9, 5e10, 1e10},
		},
	}
	for _, test := range testCase {
		res := decomposeAmount2digits(test.amount, test.dust_threshold)
		require.Equal(t, res, test.actual)
	}
}

func Test_generateOuts(t *testing.T) {
	var viewPubTo, spendpubto, viewpubChangeto, spendpubChangeto [32]byte
	tmp, _ := common.FromHex("0x92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea858")
	copy(viewPubTo[:], tmp)
	tmp, _ = common.FromHex("0x86b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389")
	copy(spendpubto[:], tmp)
	tmp, _ = common.FromHex("0x92fe6cfec2e19cd15f203f83b5d440ddb63d0cb71559f96dc81208d819fea858")
	copy(viewpubChangeto[:], tmp)
	tmp, _ = common.FromHex("0x86b08f6e874fca15108d244b40f9086d8c03260d4b954a40dfb3cbe41ebc7389")
	copy(spendpubChangeto[:], tmp)

	testCase := []struct {
		viewPubTo        *[32]byte
		spendpubto       *[32]byte
		viewpubChangeto  *[32]byte
		spendpubChangeto *[32]byte
		transAmount      int64
		selectedAmount   int64
		fee              int64
		actualErr        error
	}{
		{
			viewPubTo:        &viewPubTo,
			spendpubto:       &spendpubto,
			viewpubChangeto:  &viewpubChangeto,
			spendpubChangeto: &spendpubChangeto,
			transAmount:      10 * types.Coin,
			selectedAmount:   100 * types.Coin,
		},
		{
			viewPubTo:        &viewPubTo,
			spendpubto:       &spendpubto,
			viewpubChangeto:  &viewpubChangeto,
			spendpubChangeto: &spendpubChangeto,
			transAmount:      10 * types.Coin,
			selectedAmount:   100 * types.Coin,
			fee:              1 * types.Coin,
		},
		{
			viewPubTo:        &viewPubTo,
			spendpubto:       &spendpubto,
			viewpubChangeto:  &viewpubChangeto,
			spendpubChangeto: &spendpubChangeto,
			transAmount:      3e8,
		},
	}

	for _, test := range testCase {
		_, err := generateOuts(test.viewPubTo, test.spendpubto, test.viewpubChangeto, test.spendpubChangeto, test.transAmount, test.selectedAmount, test.fee)
		require.Equal(t, err, test.actualErr)
	}
}
