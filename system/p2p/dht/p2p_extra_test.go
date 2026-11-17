package dht

import (
	"encoding/json"
	"testing"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNatPortMapConfigDefault walks the real config path (TOML -> subModule -> raw JSON ->
// P2PSubConfig) and checks that a config written before isNatPortMap existed still maps ports.
func TestNatPortMapConfigDefault(t *testing.T) {
	const dhtSection = "\n[p2p.sub.dht]\nport=13803\n"
	load := func(cfgString string) *p2pty.P2PSubConfig {
		cfg := types.NewChain33Config(cfgString)
		mcfg := &p2pty.P2PSubConfig{}
		types.MustDecode(cfg.GetSubConfig().P2P[p2pty.DHTTypeName], mcfg)
		return mcfg
	}

	// Old config shape: the [p2p.sub.dht] section exists but has no isNatPortMap key.
	old := load(types.GetDefaultCfgstring() + dhtSection)
	t.Logf("old config without isNatPortMap: IsNatPortMap=%v NatPortMapEnabled=%v", old.IsNatPortMap, old.NatPortMapEnabled())
	require.Nil(t, old.IsNatPortMap, "the key must really be absent")
	require.True(t, old.NatPortMapEnabled(), "absent isNatPortMap must default to enabled")

	// The config shipped in the repo must keep mapping ports too.
	shipped := load(types.ReadFile("../../../cmd/chain33/chain33.toml"))
	t.Logf("shipped chain33.toml: IsNatPortMap=%v NatPortMapEnabled=%v", shipped.IsNatPortMap, shipped.NatPortMapEnabled())
	require.True(t, shipped.NatPortMapEnabled())

	explicitFalse := load(types.GetDefaultCfgstring() + dhtSection + "isNatPortMap=false\n")
	require.NotNil(t, explicitFalse.IsNatPortMap)
	t.Logf("explicit false: IsNatPortMap=%v NatPortMapEnabled=%v", *explicitFalse.IsNatPortMap, explicitFalse.NatPortMapEnabled())
	require.False(t, explicitFalse.NatPortMapEnabled())

	explicitTrue := load(types.GetDefaultCfgstring() + dhtSection + "isNatPortMap=true\n")
	require.NotNil(t, explicitTrue.IsNatPortMap)
	t.Logf("explicit true: IsNatPortMap=%v NatPortMapEnabled=%v", *explicitTrue.IsNatPortMap, explicitTrue.NatPortMapEnabled())
	require.True(t, explicitTrue.NatPortMapEnabled())

	// The cggmp integration test encodes a P2PSubConfig and then injects the key into the JSON
	// directly; that path must decode as an explicit false.
	jcfg, err := json.Marshal(&p2pty.P2PSubConfig{Port: 13803})
	require.NoError(t, err)
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(jcfg, &raw))
	raw["isNatPortMap"] = json.RawMessage("false")
	jcfg, err = json.Marshal(raw)
	require.NoError(t, err)
	injected := &p2pty.P2PSubConfig{}
	types.MustDecode(jcfg, injected)
	require.False(t, injected.NatPortMapEnabled(), "the cggmp-style JSON injection must be honoured")
}

func TestIsRestartFalse(t *testing.T) {
	p := &P2P{restart: 0}
	assert.False(t, p.isRestart())
}

func TestIsRestartTrue(t *testing.T) {
	p := &P2P{restart: 1}
	assert.True(t, p.isRestart())
}

func TestNewDBDefaults(t *testing.T) {
	dir := t.TempDir()
	db := newDB("", "", dir, 0)
	assert.NotNil(t, db)
	db.Close()
}

func TestAirDropIndexInRange(t *testing.T) {
	assert.Equal(t, int32(types.AirDropMaxIndex), int32(types.AirDropMinIndex)+airDropSpan()-1)
	seen := map[int32]struct{}{}
	for seed := int64(0); seed < 64; seed++ {
		idx := airDropIndex(seed)
		assert.GreaterOrEqual(t, idx, int32(types.AirDropMinIndex))
		assert.LessOrEqual(t, idx, int32(types.AirDropMaxIndex))
		assert.Equal(t, idx, airDropIndex(seed))
		seen[idx] = struct{}{}
	}
	assert.GreaterOrEqual(t, len(seen), 60)
}

func TestNewDBCustomName(t *testing.T) {
	dir := t.TempDir()
	db := newDB("testdb", "leveldb", dir, 128)
	assert.NotNil(t, db)
	db.Close()
}
