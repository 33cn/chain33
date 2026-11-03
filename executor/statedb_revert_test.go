// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

// newRevertStateDB builds the production StateDB shape used under
// ForkExecRollback: the concrete *StateDB (which has no Delete method) with the
// transaction opened by Begin. Pre-seeding via Set before Begin stands in for
// state already committed to the block cache.
func newRevertStateDB(t *testing.T) *StateDB {
	t.Helper()
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q := queue.New("channel")
	q.SetConfig(cfg)
	t.Cleanup(q.Close)

	height := cfg.GetFork("ForkExecRollback")
	db := NewStateDB(q.Client(), nil, nil, &StateDBOption{Height: height})
	st, ok := db.(*StateDB)
	require.True(t, ok)
	// The production shape has no Delete; RevertSet must not rely on one.
	_, hasDelete := interface{}(st).(interface{ Delete([]byte) error })
	require.False(t, hasDelete)
	return st
}

// receiptKVs models the receipt the dapp assembles from the surviving writes:
// one KV per key tracked by the StateDB. A key undone by RevertSet must not
// appear here, otherwise checkKV rejects the whole transaction.
func receiptKVs(t *testing.T, s *StateDB) []*types.KeyValue {
	t.Helper()
	seen := make(map[string]struct{})
	var kvs []*types.KeyValue
	for _, key := range s.GetSetKeys() {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		value, err := s.Get([]byte(key))
		if err != nil && err != types.ErrNotFound {
			t.Fatalf("read %q while building receipt: %v", key, err)
		}
		kvs = append(kvs, &types.KeyValue{Key: []byte(key), Value: value})
	}
	return kvs
}

func receiptKeys(t *testing.T, s *StateDB) map[string]struct{} {
	t.Helper()
	keys := make(map[string]struct{})
	for _, kv := range receiptKVs(t, s) {
		keys[string(kv.Key)] = struct{}{}
	}
	return keys
}

// A key created and then undone in the same transaction must disappear from the
// key list and from the receipt, and read back as not found.
func TestStateDBRevertSetNewKey(t *testing.T) {
	s := newRevertStateDB(t)
	key := []byte("mavl-coins-bty-exec-new-account")
	s.Begin()

	require.NoError(t, s.Set(key, []byte("created-in-tx")))
	require.Contains(t, s.GetSetKeys(), string(key))
	require.Contains(t, receiptKeys(t, s), string(key))

	require.NoError(t, s.RevertSet(key, nil))

	require.NotContains(t, s.GetSetKeys(), string(key))
	require.NotContains(t, receiptKeys(t, s), string(key), "the receipt must not emit the undone key")
	value, err := s.Get(key)
	require.ErrorIs(t, err, types.ErrNotFound)
	require.Nil(t, value)
}

// A key that existed before the transaction must read back its original value
// after the undo, and stop being tracked because no write survives.
func TestStateDBRevertSetExistingKey(t *testing.T) {
	s := newRevertStateDB(t)
	key := []byte("mavl-coins-bty-exec-existing")
	original := []byte("balance-before-tx")
	require.NoError(t, s.Set(key, original)) // committed state, before Begin

	s.Begin()
	require.NoError(t, s.Set(key, []byte("balance-in-tx")))
	require.NoError(t, s.RevertSet(key, original))

	value, err := s.Get(key)
	require.NoError(t, err)
	require.Equal(t, original, value)
	require.NotContains(t, s.GetSetKeys(), string(key))
	require.NotContains(t, receiptKeys(t, s), string(key))
}

// Undoing a write while an earlier write of the same key survives must restore
// the surviving value and keep the key tracked; undoing the last write drops it.
func TestStateDBRevertSetKeepsSurvivingWriteTracked(t *testing.T) {
	s := newRevertStateDB(t)
	key := []byte("mavl-coins-bty-exec-outer")
	before := []byte("value-before-tx")
	outer := []byte("value-outer-transfer")
	inner := []byte("value-inner-transfer")
	require.NoError(t, s.Set(key, before))

	s.Begin()
	require.NoError(t, s.Set(key, outer)) // outer transfer survives
	require.NoError(t, s.Set(key, inner)) // inner transfer is reverted
	require.NoError(t, s.RevertSet(key, outer))

	value, err := s.Get(key)
	require.NoError(t, err)
	require.Equal(t, outer, value)
	require.Contains(t, s.GetSetKeys(), string(key), "a surviving write must keep the key tracked")
	require.Contains(t, receiptKeys(t, s), string(key))

	require.NoError(t, s.RevertSet(key, before))
	value, err = s.Get(key)
	require.NoError(t, err)
	require.Equal(t, before, value)
	require.NotContains(t, s.GetSetKeys(), string(key))
}

// Production scenario: one transaction writes, reverts that write, then writes
// another key. The surviving receipt must satisfy checkKV.
func TestStateDBRevertSetCheckKVCombined(t *testing.T) {
	s := newRevertStateDB(t)
	newKey := []byte("mavl-coins-bty-exec-recipient")
	otherKey := []byte("mavl-coins-bty-exec-sender")
	s.Begin()

	require.NoError(t, s.Set(newKey, []byte("credit-reverted")))
	require.NoError(t, s.RevertSet(newKey, nil))
	require.NoError(t, s.Set(otherKey, []byte("debit-survives")))

	require.NotContains(t, receiptKeys(t, s), string(newKey))
	require.Contains(t, receiptKeys(t, s), string(otherKey))
	e := &executor{}
	require.NoError(t, e.checkKV(s.GetSetKeys(), receiptKVs(t, s)))
}

// Counter-case pinning why RevertSet is needed: shadowing a key with Set(nil)
// leaves it in GetSetKeys while the surviving receipt does not carry it, so
// checkKV rejects the transaction with ErrNotAllowMemSetKey.
func TestStateDBRevertSetCheckKVRejectsSetBasedUndo(t *testing.T) {
	s := newRevertStateDB(t)
	newKey := []byte("mavl-coins-bty-exec-recipient")
	otherKey := []byte("mavl-coins-bty-exec-sender")
	s.Begin()

	require.NoError(t, s.Set(newKey, []byte("credit-reverted")))
	require.NoError(t, s.Set(newKey, nil)) // old Set-based undo
	require.NoError(t, s.Set(otherKey, []byte("debit-survives")))

	require.Contains(t, s.GetSetKeys(), string(newKey))
	receipt := []*types.KeyValue{{Key: otherKey, Value: []byte("debit-survives")}}
	e := &executor{}
	require.ErrorIs(t, e.checkKV(s.GetSetKeys(), receipt), types.ErrNotAllowMemSetKey)
}

// RevertSet must reject invalid use without panicking, and must be a no-op for
// a key this transaction never wrote.
func TestStateDBRevertSetInputHandling(t *testing.T) {
	s := newRevertStateDB(t)
	require.NoError(t, s.Set([]byte("seeded"), []byte("v")))

	require.ErrorIs(t, s.RevertSet([]byte("seeded"), nil), ErrStateDBRevertWithoutTx)

	s.Begin()
	require.ErrorIs(t, s.RevertSet(nil, nil), ErrStateDBEmptyKey)
	require.ErrorIs(t, s.RevertSet([]byte{}, nil), ErrStateDBEmptyKey)

	require.NoError(t, s.Set([]byte("kept"), []byte("v-kept")))
	require.NoError(t, s.RevertSet([]byte("never-written"), nil), "a key without a recorded write is a no-op")
	require.Equal(t, []string{"kept"}, s.GetSetKeys())

	value, err := s.Get([]byte("kept"))
	require.NoError(t, err)
	require.Equal(t, []byte("v-kept"), value)
}
