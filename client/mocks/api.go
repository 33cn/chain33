// Code generated by mockery v1.0.0. DO NOT EDIT.
package mocks

import mock "github.com/stretchr/testify/mock"
import types "gitlab.33.cn/chain33/chain33/types"

// QueueProtocolAPI is an autogenerated mock type for the QueueProtocolAPI type
type QueueProtocolAPI struct {
	mock.Mock
}

// Close provides a mock function with given fields:
func (_m *QueueProtocolAPI) Close() {
	_m.Called()
}

// CloseQueue provides a mock function with given fields:
func (_m *QueueProtocolAPI) CloseQueue() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CloseTickets provides a mock function with given fields:
func (_m *QueueProtocolAPI) CloseTickets() (*types.ReplyHashes, error) {
	ret := _m.Called()

	var r0 *types.ReplyHashes
	if rf, ok := ret.Get(0).(func() *types.ReplyHashes); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHashes)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateTrasaction provides a mock function with given fields: param
func (_m *QueueProtocolAPI) CreateTrasaction(param *types.ReqCreateTransaction) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqCreateTransaction) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqCreateTransaction) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateUTXOs provides a mock function with given fields: param
func (_m *QueueProtocolAPI) CreateUTXOs(param *types.ReqCreateUTXOs) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqCreateUTXOs) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqCreateUTXOs) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteCacheTransaction provides a mock function with given fields: param
func (_m *QueueProtocolAPI) DeleteCacheTransaction(param *types.ReqCreateCacheTxKey) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqCreateCacheTxKey) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqCreateCacheTxKey) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DumpPrivkey provides a mock function with given fields: param
func (_m *QueueProtocolAPI) DumpPrivkey(param *types.ReqStr) (*types.ReplyStr, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyStr
	if rf, ok := ret.Get(0).(func(*types.ReqStr) *types.ReplyStr); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyStr)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqStr) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GenSeed provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GenSeed(param *types.GenSeedLang) (*types.ReplySeed, error) {
	ret := _m.Called(param)

	var r0 *types.ReplySeed
	if rf, ok := ret.Get(0).(func(*types.GenSeedLang) *types.ReplySeed); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplySeed)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.GenSeedLang) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAddrOverview provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetAddrOverview(param *types.ReqAddr) (*types.AddrOverview, error) {
	ret := _m.Called(param)

	var r0 *types.AddrOverview
	if rf, ok := ret.Get(0).(func(*types.ReqAddr) *types.AddrOverview); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.AddrOverview)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqAddr) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockByHashes provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockByHashes(param *types.ReqHashes) (*types.BlockDetails, error) {
	ret := _m.Called(param)

	var r0 *types.BlockDetails
	if rf, ok := ret.Get(0).(func(*types.ReqHashes) *types.BlockDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHashes) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockHash provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockHash(param *types.ReqInt) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.ReqInt) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqInt) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockOverview provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockOverview(param *types.ReqHash) (*types.BlockOverview, error) {
	ret := _m.Called(param)

	var r0 *types.BlockOverview
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.BlockOverview); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockOverview)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockSequences provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlockSequences(param *types.ReqBlocks) (*types.BlockSequences, error) {
	ret := _m.Called(param)

	var r0 *types.BlockSequences
	if rf, ok := ret.Get(0).(func(*types.ReqBlocks) *types.BlockSequences); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockSequences)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqBlocks) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlocks provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetBlocks(param *types.ReqBlocks) (*types.BlockDetails, error) {
	ret := _m.Called(param)

	var r0 *types.BlockDetails
	if rf, ok := ret.Get(0).(func(*types.ReqBlocks) *types.BlockDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.BlockDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqBlocks) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFatalFailure provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetFatalFailure() (*types.Int32, error) {
	ret := _m.Called()

	var r0 *types.Int32
	if rf, ok := ret.Get(0).(func() *types.Int32); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetHeaders provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetHeaders(param *types.ReqBlocks) (*types.Headers, error) {
	ret := _m.Called(param)

	var r0 *types.Headers
	if rf, ok := ret.Get(0).(func(*types.ReqBlocks) *types.Headers); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Headers)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqBlocks) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastBlockSequence provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastBlockSequence() (*types.Int64, error) {
	ret := _m.Called()

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func() *types.Int64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastHeader provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastHeader() (*types.Header, error) {
	ret := _m.Called()

	var r0 *types.Header
	if rf, ok := ret.Get(0).(func() *types.Header); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Header)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLastMempool provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetLastMempool() (*types.ReplyTxList, error) {
	ret := _m.Called()

	var r0 *types.ReplyTxList
	if rf, ok := ret.Get(0).(func() *types.ReplyTxList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMempool provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetMempool() (*types.ReplyTxList, error) {
	ret := _m.Called()

	var r0 *types.ReplyTxList
	if rf, ok := ret.Get(0).(func() *types.ReplyTxList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNetInfo provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetNetInfo() (*types.NodeNetInfo, error) {
	ret := _m.Called()

	var r0 *types.NodeNetInfo
	if rf, ok := ret.Get(0).(func() *types.NodeNetInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.NodeNetInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSeed provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetSeed(param *types.GetSeedByPw) (*types.ReplySeed, error) {
	ret := _m.Called(param)

	var r0 *types.ReplySeed
	if rf, ok := ret.Get(0).(func(*types.GetSeedByPw) *types.ReplySeed); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplySeed)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.GetSeedByPw) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTicketCount provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetTicketCount() (*types.Int64, error) {
	ret := _m.Called()

	var r0 *types.Int64
	if rf, ok := ret.Get(0).(func() *types.Int64); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Int64)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTransactionByAddr provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetTransactionByAddr(param *types.ReqAddr) (*types.ReplyTxInfos, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyTxInfos
	if rf, ok := ret.Get(0).(func(*types.ReqAddr) *types.ReplyTxInfos); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxInfos)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqAddr) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTransactionByHash provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetTransactionByHash(param *types.ReqHashes) (*types.TransactionDetails, error) {
	ret := _m.Called(param)

	var r0 *types.TransactionDetails
	if rf, ok := ret.Get(0).(func(*types.ReqHashes) *types.TransactionDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.TransactionDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHashes) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTxList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) GetTxList(param *types.TxHashList) (*types.ReplyTxList, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyTxList
	if rf, ok := ret.Get(0).(func(*types.TxHashList) *types.ReplyTxList); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.TxHashList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetWalletStatus provides a mock function with given fields:
func (_m *QueueProtocolAPI) GetWalletStatus() (*types.WalletStatus, error) {
	ret := _m.Called()

	var r0 *types.WalletStatus
	if rf, ok := ret.Get(0).(func() *types.WalletStatus); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletStatus)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsNtpClockSync provides a mock function with given fields:
func (_m *QueueProtocolAPI) IsNtpClockSync() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsSync provides a mock function with given fields:
func (_m *QueueProtocolAPI) IsSync() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LocalGet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalGet(param *types.LocalDBGet) (*types.LocalReplyValue, error) {
	ret := _m.Called(param)

	var r0 *types.LocalReplyValue
	if rf, ok := ret.Get(0).(func(*types.LocalDBGet) *types.LocalReplyValue); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LocalReplyValue)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.LocalDBGet) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LocalList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) LocalList(param *types.LocalDBList) (*types.LocalReplyValue, error) {
	ret := _m.Called(param)

	var r0 *types.LocalReplyValue
	if rf, ok := ret.Get(0).(func(*types.LocalDBList) *types.LocalReplyValue); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.LocalReplyValue)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.LocalDBList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewAccount provides a mock function with given fields: param
func (_m *QueueProtocolAPI) NewAccount(param *types.ReqNewAccount) (*types.WalletAccount, error) {
	ret := _m.Called(param)

	var r0 *types.WalletAccount
	if rf, ok := ret.Get(0).(func(*types.ReqNewAccount) *types.WalletAccount); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletAccount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqNewAccount) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PeerInfo provides a mock function with given fields:
func (_m *QueueProtocolAPI) PeerInfo() (*types.PeerList, error) {
	ret := _m.Called()

	var r0 *types.PeerList
	if rf, ok := ret.Get(0).(func() *types.PeerList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.PeerList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Privacy2Privacy provides a mock function with given fields: param
func (_m *QueueProtocolAPI) Privacy2Privacy(param *types.ReqPri2Pri) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqPri2Pri) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqPri2Pri) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Privacy2Public provides a mock function with given fields: param
func (_m *QueueProtocolAPI) Privacy2Public(param *types.ReqPri2Pub) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqPri2Pub) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqPri2Pub) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PrivacyTransactionList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) PrivacyTransactionList(param *types.ReqPrivacyTransactionList) (*types.WalletTxDetails, error) {
	ret := _m.Called(param)

	var r0 *types.WalletTxDetails
	if rf, ok := ret.Get(0).(func(*types.ReqPrivacyTransactionList) *types.WalletTxDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletTxDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqPrivacyTransactionList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Publick2Privacy provides a mock function with given fields: param
func (_m *QueueProtocolAPI) Publick2Privacy(param *types.ReqPub2Pri) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqPub2Pri) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqPub2Pri) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Query provides a mock function with given fields: param
func (_m *QueueProtocolAPI) Query(param *types.Query) (*types.Message, error) {
	ret := _m.Called(param)

	var r0 *types.Message
	if rf, ok := ret.Get(0).(func(*types.Query) *types.Message); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Message)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.Query) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryCacheTransaction provides a mock function with given fields: param
func (_m *QueueProtocolAPI) QueryCacheTransaction(param *types.ReqCacheTxList) (*types.ReplyCacheTxList, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyCacheTxList
	if rf, ok := ret.Get(0).(func(*types.ReqCacheTxList) *types.ReplyCacheTxList); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyCacheTxList)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqCacheTxList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// QueryTx provides a mock function with given fields: param
func (_m *QueueProtocolAPI) QueryTx(param *types.ReqHash) (*types.TransactionDetail, error) {
	ret := _m.Called(param)

	var r0 *types.TransactionDetail
	if rf, ok := ret.Get(0).(func(*types.ReqHash) *types.TransactionDetail); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.TransactionDetail)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqHash) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveSeed provides a mock function with given fields: param
func (_m *QueueProtocolAPI) SaveSeed(param *types.SaveSeedByPw) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.SaveSeedByPw) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.SaveSeedByPw) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendTx provides a mock function with given fields: param
func (_m *QueueProtocolAPI) SendTx(param *types.Transaction) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.Transaction) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.Transaction) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SendTxHashToWallet provides a mock function with given fields: param
func (_m *QueueProtocolAPI) SendTxHashToWallet(param *types.ReqCreateCacheTxKey) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqCreateCacheTxKey) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqCreateCacheTxKey) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ShowPrivacyAccountInfo provides a mock function with given fields: param
func (_m *QueueProtocolAPI) ShowPrivacyAccountInfo(param *types.ReqPPrivacyAccount) (*types.ReplyPrivacyAccount, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyPrivacyAccount
	if rf, ok := ret.Get(0).(func(*types.ReqPPrivacyAccount) *types.ReplyPrivacyAccount); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyPrivacyAccount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqPPrivacyAccount) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ShowPrivacyAccountSpend provides a mock function with given fields: param
func (_m *QueueProtocolAPI) ShowPrivacyAccountSpend(param *types.ReqPrivBal4AddrToken) (*types.UTXOHaveTxHashs, error) {
	ret := _m.Called(param)

	var r0 *types.UTXOHaveTxHashs
	if rf, ok := ret.Get(0).(func(*types.ReqPrivBal4AddrToken) *types.UTXOHaveTxHashs); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.UTXOHaveTxHashs)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqPrivBal4AddrToken) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ShowPrivacyKey provides a mock function with given fields: param
func (_m *QueueProtocolAPI) ShowPrivacyKey(param *types.ReqStr) (*types.ReplyPrivacyPkPair, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyPrivacyPkPair
	if rf, ok := ret.Get(0).(func(*types.ReqStr) *types.ReplyPrivacyPkPair); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyPrivacyPkPair)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqStr) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SignRawTx provides a mock function with given fields: param
func (_m *QueueProtocolAPI) SignRawTx(param *types.ReqSignRawTx) (*types.ReplySignRawTx, error) {
	ret := _m.Called(param)

	var r0 *types.ReplySignRawTx
	if rf, ok := ret.Get(0).(func(*types.ReqSignRawTx) *types.ReplySignRawTx); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplySignRawTx)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqSignRawTx) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreGet provides a mock function with given fields: _a0
func (_m *QueueProtocolAPI) StoreGet(_a0 *types.StoreGet) (*types.StoreReplyValue, error) {
	ret := _m.Called(_a0)

	var r0 *types.StoreReplyValue
	if rf, ok := ret.Get(0).(func(*types.StoreGet) *types.StoreReplyValue); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.StoreReplyValue)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.StoreGet) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// StoreGetTotalCoins provides a mock function with given fields: _a0
func (_m *QueueProtocolAPI) StoreGetTotalCoins(_a0 *types.IterateRangeByStateHash) (*types.ReplyGetTotalCoins, error) {
	ret := _m.Called(_a0)

	var r0 *types.ReplyGetTotalCoins
	if rf, ok := ret.Get(0).(func(*types.IterateRangeByStateHash) *types.ReplyGetTotalCoins); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyGetTotalCoins)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.IterateRangeByStateHash) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Version provides a mock function with given fields:
func (_m *QueueProtocolAPI) Version() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletAutoMiner provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletAutoMiner(param *types.MinerFlag) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.MinerFlag) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.MinerFlag) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletGetAccountList provides a mock function with given fields:
func (_m *QueueProtocolAPI) WalletGetAccountList() (*types.WalletAccounts, error) {
	ret := _m.Called()

	var r0 *types.WalletAccounts
	if rf, ok := ret.Get(0).(func() *types.WalletAccounts); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletAccounts)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletImportprivkey provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletImportprivkey(param *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	ret := _m.Called(param)

	var r0 *types.WalletAccount
	if rf, ok := ret.Get(0).(func(*types.ReqWalletImportPrivKey) *types.WalletAccount); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletAccount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletImportPrivKey) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletLock provides a mock function with given fields:
func (_m *QueueProtocolAPI) WalletLock() (*types.Reply, error) {
	ret := _m.Called()

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func() *types.Reply); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletMergeBalance provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletMergeBalance(param *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHashes
	if rf, ok := ret.Get(0).(func(*types.ReqWalletMergeBalance) *types.ReplyHashes); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHashes)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletMergeBalance) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletSendToAddress provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletSendToAddress(param *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	ret := _m.Called(param)

	var r0 *types.ReplyHash
	if rf, ok := ret.Get(0).(func(*types.ReqWalletSendToAddress) *types.ReplyHash); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.ReplyHash)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletSendToAddress) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletSetFee provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletSetFee(param *types.ReqWalletSetFee) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqWalletSetFee) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletSetFee) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletSetLabel provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletSetLabel(param *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	ret := _m.Called(param)

	var r0 *types.WalletAccount
	if rf, ok := ret.Get(0).(func(*types.ReqWalletSetLabel) *types.WalletAccount); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletAccount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletSetLabel) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletSetPasswd provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletSetPasswd(param *types.ReqWalletSetPasswd) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.ReqWalletSetPasswd) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletSetPasswd) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletTransactionList provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletTransactionList(param *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	ret := _m.Called(param)

	var r0 *types.WalletTxDetails
	if rf, ok := ret.Get(0).(func(*types.ReqWalletTransactionList) *types.WalletTxDetails); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.WalletTxDetails)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.ReqWalletTransactionList) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WalletUnLock provides a mock function with given fields: param
func (_m *QueueProtocolAPI) WalletUnLock(param *types.WalletUnLock) (*types.Reply, error) {
	ret := _m.Called(param)

	var r0 *types.Reply
	if rf, ok := ret.Get(0).(func(*types.WalletUnLock) *types.Reply); ok {
		r0 = rf(param)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*types.Reply)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*types.WalletUnLock) error); ok {
		r1 = rf(param)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
