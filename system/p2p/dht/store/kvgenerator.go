package store

import (
	"encoding/hex"
	"fmt"
)

func makeBlockHashAsKey(hash []byte) string {
	return fmt.Sprintf("/%s/%s", DhtStoreNamespace, hex.EncodeToString(hash))
}
