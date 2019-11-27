package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_recommendSeqs(t *testing.T) {
	seqs := recommendSeqs(2, 10)
	assert.Equal(t, 3, len(seqs))

	seqs = recommendSeqs(20, 10)
	assert.Equal(t, 6, len(seqs))
	assert.Equal(t, []int64{20, 19, 18, 17, 16, 0}, seqs)

	seqs = recommendSeqs(10000, 6)
	assert.Equal(t, 6, len(seqs))
	assert.Equal(t, []int64{10000, 9999, 9998, 9898, 9798, 9698}, seqs)

	seqs = recommendSeqs(200, 6)
	assert.Equal(t, 5, len(seqs))
	assert.Equal(t, []int64{200, 199, 198, 98, 0}, seqs)
}
