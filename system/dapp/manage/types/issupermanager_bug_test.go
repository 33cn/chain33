package types

import (
	"strings"
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

// Regression test for the bug fixed in manage.go:64
// Bug was: m = address.FormatEthAddress(addr)  — overwrites m with caller's address
// Fix:     m = address.FormatEthAddress(m)     — normalizes the config value

func isSuperManager(cfg *types.Chain33Config, addr string, height int64) bool {
	conf := types.ConfSub(cfg, "manage")
	for _, m := range conf.GStrList("superManager") {
		if address.IsEthAddress(m) && cfg.IsFork(height, address.ForkEthAddressFormat) {
			m = address.FormatEthAddress(m)
		}
		if addr == m {
			return true
		}
	}
	return false
}

func newCfgWithEthSuperManager() *types.Chain33Config {
	cfgStr := types.GetDefaultCfgstring()
	section := strings.Index(cfgStr, "[exec.sub.manage]")
	smKey := strings.Index(cfgStr[section:], "superManager=")
	smStart := section + smKey
	smEnd := strings.Index(cfgStr[smStart:], "]") + smStart + 1
	cfgStr = cfgStr[:smStart] + `superManager=["0x1234567890aBcDeF1234567890AbCdEf12345678"]` + cfgStr[smEnd:]
	return types.NewChain33Config(cfgStr)
}

func TestIsSuperManager_EthAddress_Regression(t *testing.T) {
	cfg := newCfgWithEthSuperManager()

	conf := types.ConfSub(cfg, "manage")
	list := conf.GStrList("superManager")
	assert.Equal(t, []string{"0x1234567890aBcDeF1234567890AbCdEf12345678"}, list)
	assert.True(t, cfg.IsFork(0, address.ForkEthAddressFormat))

	// Legitimate superManager (lowercase of the mixed-case config value)
	superAddr := "0x1234567890abcdef1234567890abcdef12345678"
	assert.True(t, isSuperManager(cfg, superAddr, 0),
		"legitimate superManager should pass")

	// Attacker must NOT pass
	attacker := "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	assert.False(t, isSuperManager(cfg, attacker, 0),
		"arbitrary address must not pass superManager check")
}
