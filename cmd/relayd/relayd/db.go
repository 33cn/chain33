package relayd

// Store hash and blockHeader
// SPV information

// Namespace keys
var (
	hashKey     = []byte("h")
	blockHeader = []byte("b")
)

type RelaydDB struct{}
