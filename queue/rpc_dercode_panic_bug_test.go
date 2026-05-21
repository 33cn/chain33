package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-RPC-002: rpc/ethrpc/types/tx.go:127-147 paraseDERCode panics on short/malformed input.
// No length checks before indexing sig[0], sig[2], sig[3], sig[int(sig[3])+4], etc.

func TestParseDERCodePanicBug(t *testing.T) {
	// Simulate the buggy paraseDERCode logic
	buggyParse := func(sig []byte) (r, s []byte, err error) {
		if sig[0] != 0x30 && sig[2] != 0x02 {
			return nil, nil, assert.AnError
		}
		if sig[0] == 0x30 && sig[2] == 0x02 {
			r = sig[4 : int(sig[3])+4]
		}
		return r, s, nil
	}

	// Fixed version with bounds checking
	fixedParse := func(sig []byte) (r, s []byte, err error) {
		if len(sig) < 4 {
			return nil, nil, assert.AnError
		}
		if sig[0] != 0x30 || sig[2] != 0x02 {
			return nil, nil, assert.AnError
		}
		rLen := int(sig[3])
		if len(sig) < 4+rLen {
			return nil, nil, assert.AnError
		}
		r = sig[4 : 4+rLen]
		if r[0] == 0x0 && len(r) > 1 {
			r = r[1:]
		}
		sOffset := 4 + rLen
		if len(sig) < sOffset+2 {
			return nil, nil, assert.AnError
		}
		if sig[sOffset] != 0x02 {
			return nil, nil, assert.AnError
		}
		sLen := int(sig[sOffset+1])
		if len(sig) < sOffset+2+sLen {
			return nil, nil, assert.AnError
		}
		s = sig[sOffset+2 : sOffset+2+sLen]
		return r, s, nil
	}

	testCases := []struct {
		name string
		sig  []byte
	}{
		{"empty slice", []byte{}},
		{"one byte", []byte{0x30}},
		{"two bytes", []byte{0x30, 0x44}},
		{"three bytes", []byte{0x30, 0x44, 0x02}},
		{"sig[3] too large", []byte{0x30, 0x44, 0x02, 0xFF}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// BUG: buggy version panics on short input
			assert.Panics(t, func() { buggyParse(tc.sig) },
				"buggy paraseDERCode panics on: %s", tc.name)

			// FIXED: fixed version returns error
			_, _, err := fixedParse(tc.sig)
			assert.Error(t, err,
				"fixed paraseDERCode returns error on: %s", tc.name)
		})
	}
}
