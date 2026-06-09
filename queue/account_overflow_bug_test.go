package queue

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-ACCT-001: account/account.go:135,161 uses plain addition without overflow check.
// Transfer: accTo.Balance = accTo.GetBalance() + amount
// depositBalance: acc1.Balance += amount
// Both can overflow int64, wrapping to negative. Mint() correctly uses safeAdd.

func TestBalanceOverflowBug(t *testing.T) {
	// safeAdd equivalent (from account/genesis.go)
	maxTokenBalance := int64(900 * 1e8 * 1e8) // 900 * 1e16

	safeAdd := func(balance, amount int64) (int64, error) {
		if balance+amount < amount || balance+amount > maxTokenBalance {
			return balance, assert.AnError
		}
		return balance + amount, nil
	}

	// Buggy addition: no overflow check
	buggyAdd := func(balance, amount int64) int64 {
		return balance + amount
	}

	// Case 1: overflow wraps to negative
	balance := int64(math.MaxInt64 - 100)
	amount := int64(200)

	buggyResult := buggyAdd(balance, amount)
	assert.True(t, buggyResult < 0,
		"buggy addition overflows int64, wrapping to negative: %d", buggyResult)

	// safeAdd catches it
	_, err := safeAdd(balance, amount)
	assert.Error(t, err, "safeAdd detects overflow")

	// Case 2: exceeds MaxTokenBalance but doesn't overflow int64
	balance2 := maxTokenBalance - 50
	amount2 := int64(100)

	buggyResult2 := buggyAdd(balance2, amount2)
	assert.True(t, buggyResult2 > maxTokenBalance,
		"buggy addition exceeds MaxTokenBalance without error")

	_, err2 := safeAdd(balance2, amount2)
	assert.Error(t, err2, "safeAdd rejects amount exceeding MaxTokenBalance")
}
