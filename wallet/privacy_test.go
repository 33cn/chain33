package wallet

import (
	"testing"
)

func TestDecomAmount2Nature(t *testing.T) {
	order := int64(1000)
	for i := 1; i<= 10; i++ {
		res := decomAmount2Nature(order*int64(i), order)
		t.Logf("The Result for %d is %d\n", i, res)
	}
}
