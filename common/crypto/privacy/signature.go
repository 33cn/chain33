package privacy

import (
	"bytes"
	"fmt"

	. "gitlab.33.cn/chain33/chain33/common/crypto"
)

// Signature
type SignatureOnetime [64]byte

type SignatureS struct {
	Signature
}

func (sig SignatureOnetime) Bytes() []byte {
	s := make([]byte, 64)
	copy(s, sig[:])
	return s
}

func (sig SignatureOnetime) IsZero() bool { return len(sig) == 0 }

func (sig SignatureOnetime) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)
}

func (sig SignatureOnetime) Equals(other Signature) bool {
	if otherEd, ok := other.(SignatureOnetime); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}
