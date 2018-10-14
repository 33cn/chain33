package types

import "gitlab.33.cn/chain33/chain33/types"

const (
	//action for token
	ActionTransfer            = 4
	ActionGenesis             = 5
	ActionWithdraw            = 6
	TokenActionPreCreate      = 7
	TokenActionFinishCreate   = 8
	TokenActionRevokeCreate   = 9
	TokenActionTransferToExec = 11
)

// token status
const (
	TokenStatusPreCreated = iota
	TokenStatusCreated
	TokenStatusCreateRevoked
)

var (
	TokenX = types.TokenX
)

const (
	TyLogPreCreateToken       = 211
	TyLogFinishCreateToken    = 212
	TyLogRevokeCreateToken    = 213
	TyLogTokenTransfer        = 313
	TyLogTokenGenesis         = 314
	TyLogTokenDeposit         = 315
	TyLogTokenExecTransfer    = 316
	TyLogTokenExecWithdraw    = 317
	TyLogTokenExecDeposit     = 318
	TyLogTokenExecFrozen      = 319
	TyLogTokenExecActive      = 320
	TyLogTokenGenesisTransfer = 321
	TyLogTokenGenesisDeposit  = 322
)
