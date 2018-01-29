package btc

const (
	COIN                     = 1e8
	MAX_MONEY                = 21000000 * COIN
	MAX_BLOCK_SIZE           = 1e6
	MessageMagic             = "Bitcoin Signed Message:\n"
	LOCKTIME_THRESHOLD       = 500000000
	MAX_SCRIPT_ELEMENT_SIZE  = 520
	MAX_BLOCK_SIGOPS_COST    = 80000
	MAX_PUBKEYS_PER_MULTISIG = 20
	WITNESS_SCALE_FACTOR     = 4
)
