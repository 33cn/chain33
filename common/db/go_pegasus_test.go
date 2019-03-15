// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"context"
	"testing"
	"time"

	"github.com/XiaoMi/pegasus-go-client/pegasus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPegasusDB_Get(t *testing.T) {
	key := []byte("my_key")
	key1 := []byte("my_key1")
	val := []byte("my_value")

	db := new(PegasusDB)
	tbl := new(MyTable)
	tbl.On("Get", context.Background(), getHashKey(key), key).Return(val, nil)
	tbl.On("Get", context.Background(), getHashKey(key1), key1).Return(nil, nil)
	db.table = tbl

	data, err := db.Get(key)
	assert.Nil(t, err)
	assert.Equal(t, data, val)

	data, err = db.Get([]byte(key1))
	assert.Nil(t, data)
	assert.EqualError(t, err, ErrNotFoundInDb.Error())
	tbl.AssertExpectations(t)
}

type MyTable struct {
	pegasus.TableConnector
	mock.Mock
}

func (tbl *MyTable) Get(ctx context.Context, hashKey []byte, sortKey []byte) ([]byte, error) {
	args := tbl.Called(ctx, hashKey, sortKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}
func (tbl *MyTable) SetTTL(ctx context.Context, hashKey []byte, sortKey []byte, value []byte, ttl time.Duration) error {
	return nil
}
func (tbl *MyTable) Del(ctx context.Context, hashKey []byte, sortKey []byte) error { return nil }

func (tbl *MyTable) MultiGet(ctx context.Context, hashKey []byte, sortKeys [][]byte) ([]*pegasus.KeyValue, bool, error) {
	args := tbl.Called(ctx, hashKey, sortKeys)
	return args.Get(0).([]*pegasus.KeyValue), args.Bool(1), args.Error(2)
}
func (tbl *MyTable) MultiGetOpt(ctx context.Context, hashKey []byte, sortKeys [][]byte, options *pegasus.MultiGetOptions) ([]*pegasus.KeyValue, bool, error) {
	return nil, false, nil
}
func (tbl *MyTable) MultiGetRange(ctx context.Context, hashKey []byte, startSortKey []byte, stopSortKey []byte) ([]*pegasus.KeyValue, bool, error) {
	return nil, false, nil
}
func (tbl *MyTable) MultiGetRangeOpt(ctx context.Context, hashKey []byte, startSortKey []byte, stopSortKey []byte, options *pegasus.MultiGetOptions) ([]*pegasus.KeyValue, bool, error) {
	return nil, false, nil
}
func (tbl *MyTable) MultiSet(ctx context.Context, hashKey []byte, sortKeys [][]byte, values [][]byte) error {
	return nil
}
func (tbl *MyTable) MultiSetOpt(ctx context.Context, hashKey []byte, sortKeys [][]byte, values [][]byte, ttl time.Duration) error {
	return nil
}
func (tbl *MyTable) MultiDel(ctx context.Context, hashKey []byte, sortKeys [][]byte) error {
	return nil
}
func (tbl *MyTable) TTL(ctx context.Context, hashKey []byte, sortKey []byte) (int, error) {
	return 0, nil
}
func (tbl *MyTable) Exist(ctx context.Context, hashKey []byte, sortKey []byte) (bool, error) {
	return false, nil
}
func (tbl *MyTable) GetScanner(ctx context.Context, hashKey []byte, startSortKey []byte, stopSortKey []byte, options *pegasus.ScannerOptions) (pegasus.Scanner, error) {
	return nil, nil
}
func (tbl *MyTable) GetUnorderedScanners(ctx context.Context, maxSplitCount int, options *pegasus.ScannerOptions) ([]pegasus.Scanner, error) {
	return nil, nil
}
func (tbl *MyTable) CheckAndSet(ctx context.Context, hashKey []byte, checkSortKey []byte, checkType pegasus.CheckType,
	checkOperand []byte, setSortKey []byte, setValue []byte, options *pegasus.CheckAndSetOptions) (*pegasus.CheckAndSetResult, error) {
	return nil, nil
}
func (tbl *MyTable) SortKeyCount(ctx context.Context, hashKey []byte) (int64, error) { return 0, nil }
func (tbl *MyTable) Close() error                                                    { return nil }
