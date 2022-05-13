package address

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// IsEthAddress verifies whether a string can represent
// a valid hex-encoded eth address
func IsEthAddress(addr string) bool {
	return common.IsHexAddress(addr)
}

// ToLower to lower case string
func ToLower(addr string) string {
	return strings.ToLower(addr)
}

// FormatAddrKey format addr as db key
func FormatAddrKey(addr string) []byte {
	// eth地址有大小写区分, 统一采用小写格式key存储(#1245)
	if IsEthAddress(addr) {
		addr = ToLower(addr)
	}

	return []byte(addr)
}
