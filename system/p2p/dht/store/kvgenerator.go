package store

import (
	"encoding/hex"
	"fmt"
)

func MakeBlockHashAsKey(hash []byte) string {
	return fmt.Sprintf("/%s/%s", DhtStoreNamespace, hex.EncodeToString(hash))
}
