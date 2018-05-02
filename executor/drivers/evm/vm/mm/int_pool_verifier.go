// +build VERIFY_EVM_INTEGER_POOL

package mm

import "fmt"

const VerifyPool = true

func VerifyIntegerPool(ip *IntPool) {
	for i, item := range ip.pool.data {
		if item.Cmp(checkVal) != 0 {
			panic(fmt.Sprintf("%d'th item failed aggressive pool check. Value was modified", i))
		}
	}
}
