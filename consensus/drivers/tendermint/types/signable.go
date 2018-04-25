package types

import (
	"bytes"
	"io"

	cmn "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/common"
)

// Signable is an interface for all signable things.
// It typically removes signatures before serializing.
type Signable interface {
	WriteSignBytes(chainID string, w io.Writer, n *int, err *error)
}

// SignBytes is a convenience method for getting the bytes to sign of a Signable.
func SignBytes(chainID string, o Signable) []byte {
	buf, n, err := new(bytes.Buffer), new(int), new(error)
	o.WriteSignBytes(chainID, buf, n, err)
	if *err != nil {
		cmn.PanicCrisis(err)
	}
	return buf.Bytes()
}

// HashSignBytes is a convenience method for getting the hash of the bytes of a signable
func HashSignBytes(chainID string, o Signable) []byte {
	return cmn.SimpleHashFromBinary(SignBytes(chainID, o))
}
