package model

import "errors"

var (
	ErrOutOfGas                 = errors.New("out of gas")
	ErrCodeStoreOutOfGas        = errors.New("contract creation code storage out of gas")
	ErrDepth                    = errors.New("max call depth exceeded")
	ErrTraceLimitReached        = errors.New("the number of logs reached the specified limit")
	ErrInsufficientBalance      = errors.New("insufficient balance for transfer")
	ErrContractAddressCollision = errors.New("contract address collision")
	ErrGasLimitReached          = errors.New("gas limit reached")
	ErrGasUintOverflow          = errors.New("gas uint64 overflow")
	ErrAddrNotExists            = errors.New("address not exists")
	ErrTransferBetweenContracts = errors.New("transferring between contracts not supports")
	ErrTransferBetweenEOA       = errors.New("transferring between external accounts not supports")
	ErrNoCreator                = errors.New("contract has no creator information")
	ErrDestruct                 = errors.New("contract has been destructed")

	ErrWriteProtection       = errors.New("evm: write protection")
	ErrReturnDataOutOfBounds = errors.New("evm: return data out of bounds")
	ErrExecutionReverted     = errors.New("evm: execution reverted")
	ErrMaxCodeSizeExceeded   = errors.New("evm: max code size exceeded")

	NoCoinsAccount = errors.New("no coins account in executor!")
)
