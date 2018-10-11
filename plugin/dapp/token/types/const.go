package types

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

//const (
//	TyLogPreCreateToken         = 210
//	TyLogFinishCreateToken      = 211
//	TyLogRevokeCreateToken      = 212
//	TyLogTokenTransfer          = 213
//	TyLogTokenGenesis           = 214
//	TyLogTokenDeposit           = 215
//	TyLogTokenExecTransfer      = 216
//	TyLogTokenExecWithdraw      = 217
//	TyLogTokenExecDeposit       = 218
//	TyLogTokenExecFrozen        = 219
//	TyLogTokenExecActive        = 220
//	TyLogTokenGenesisTransfer   = 221
//	TyLogTokenGenesisDeposit    = 222
//)

// token status
const (
	TokenStatusPreCreated = iota
	TokenStatusCreated
	TokenStatusCreateRevoked
)

var (
	//TokenX          = "token"
)

