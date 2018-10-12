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

// token status
const (
	TokenStatusPreCreated = iota
	TokenStatusCreated
	TokenStatusCreateRevoked
)

var (
//TokenX          = "token"
)
