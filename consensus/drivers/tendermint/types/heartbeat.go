package types

import (
	"io"

	"encoding/json"
	"gitlab.33.cn/chain33/chain33/types"
)

// Heartbeat is a simple vote-like structure so validators can
// alert others that they are alive and waiting for transactions.
// Note: We aren't adding ",omitempty" to Heartbeat's
// json field tags because we always want the JSON
// representation to be in its canonical form.
type Heartbeat struct {
	*types.Heartbeat
}

// WriteSignBytes writes the Heartbeat for signing.
// It panics if the Heartbeat is nil.
func (heartbeat *Heartbeat) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	if *err != nil {
		return
	}
	canonical := CanonicalJSONOnceHeartbeat{
		chainID,
		CanonicalHeartbeat(heartbeat),
	}
	byteHeartbeat, e := json.Marshal(&canonical)
	if e != nil {
		*err = e
		return
	}
	n_, err_ := w.Write(byteHeartbeat)
	*n = n_
	*err = err_
	return
}
