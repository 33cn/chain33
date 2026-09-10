// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"fmt"
	"sync"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

// TestLoadAccountConcurrentRace hammers LoadAccount from many goroutines on a
// single *DB. Before the fix, accountReadKey reused a shared backing array
// (DB.accountKeyBuffer), so concurrent loads overwrote each other's key while
// db.Get was still reading it — a data race (go test -race) and a potential
// wrong-account read. After the fix every call builds a fresh key slice.
func TestLoadAccountConcurrentRace(t *testing.T) {
	acc := newTestCoinsDB(t)

	const nAccounts = 8
	addrs := make([]string, nAccounts)
	for i := 0; i < nAccounts; i++ {
		addrs[i] = fmt.Sprintf("%s%02d", addr1[:len(addr1)-2], i)
		balance := int64(100+i) * types.DefaultCoinPrecision
		acc.SaveAccount(&types.Account{Addr: addrs[i], Balance: balance})
	}

	var wg sync.WaitGroup
	errCh := make(chan error, nAccounts*4)
	for g := 0; g < nAccounts*2; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			addr := addrs[g%nAccounts]
			want := int64(100+g%nAccounts) * types.DefaultCoinPrecision
			for i := 0; i < 200; i++ {
				a := acc.LoadAccount(addr)
				if a.GetBalance() != want {
					errCh <- fmt.Errorf("addr %s: got balance %d, want %d", addr, a.GetBalance(), want)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
}

// TestAccountReadKeyIndependent verifies the returned key slice does not
// change when a subsequent key is built (no shared backing array).
func TestAccountReadKeyIndependent(t *testing.T) {
	acc := newTestCoinsDB(t)
	key1 := acc.accountReadKey(addr1)
	key1Copy := append([]byte(nil), key1...)
	_ = acc.accountReadKey(addr2)
	require.Equal(t, key1Copy, key1, "previously returned key must not be mutated by later calls")
	require.NotEqual(t, acc.accountReadKey(addr1), acc.accountReadKey(addr2))
}
