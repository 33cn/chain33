package address

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// ForkFormatAddressKey 地址key格式化分叉名称,主要针对eth地址
const ForkFormatAddressKey = "ForkFormatAddressKey"

// ForkEthAddressFormat eth地址统一格式化
const ForkEthAddressFormat = "ForkEthAddressFormat"

// IsEthAddress verifies whether a string can represent
// a valid hex-encoded eth address
func IsEthAddress(addr string) bool {
	return common.IsHexAddress(addr)
}

// FormatEthAddress eth地址格式化
func FormatEthAddress(addr string) string {
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
