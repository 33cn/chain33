package address

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatAddrKey(t *testing.T) {

	addr1 := "0x6c0d7be0d2c8350042890a77393158181716b0d6"
	addr2 := "0x6c0d7BE0d2C8350042890a77393158181716b0d6"

	require.True(t, IsEthAddress(addr1))
	require.False(t, IsEthAddress("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"))
	addrKey1 := fmt.Sprintf("%s:%s", "addrKey:", FormatAddrKey(addr1))
	addrKey2 := fmt.Sprintf("%s:%s", "addrKey:", FormatAddrKey(addr2))

	expect := fmt.Sprintf("%s:%s", "addrKey:", string(FormatAddrKey(ToLower(addr1))))

	require.Equal(t, expect, addrKey1)
	require.Equal(t, expect, addrKey2)
}
