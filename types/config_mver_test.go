package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigFlat(t *testing.T) {
	conf := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"key21": "value21",
		},
	}
	flat := FlatConfig(conf)
	assert.Equal(t, "value1", flat["key1"])
	assert.Equal(t, "value21", flat["key2.key21"])
}

func TestConfigMverInit(t *testing.T) {
	cfg, _ := InitCfg("testdata/local.mvertest.toml")
	Init(cfg.Title, cfg)
	assert.Equal(t, MGStr("mver.consensus.name2", 0), "ticket-bityuan")
	assert.Equal(t, MGStr("mver.consensus.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, MGStr("mver.hello", 0), "world")
	assert.Equal(t, MGStr("mver.hello", 11), "forkworld")
	assert.Equal(t, MGStr("mver.nofork", 0), "nofork")
	assert.Equal(t, MGStr("mver.nofork", 9), "nofork")
	assert.Equal(t, MGStr("mver.nofork", 11), "nofork")

	assert.Equal(t, MGStr("mver.exec.sub.token.name2", -1), "ticket-bityuan")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 0), "ticket-bityuanv5-enable")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 9), "ticket-bityuanv5-enable")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, MGStr("mver.exec.sub.token.name2", 11), "ticket-bityuanv5")
}
