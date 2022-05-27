package address

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ForkFormatAddressKey 地址key格式化分叉名称,主要针对eth地址
const ForkFormatAddressKey = "ForkFormatAddressKey"

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
		ethDriver, _ := LoadDriver(2, -1)
		addr = ethDriver.FormatAddr(addr)
	}

	return []byte(addr)
}
