package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeConfig(t *testing.T) {
	conf := map[string]interface{}{
		"key1": "value1",
	}
	def := map[string]interface{}{
		"key2": "value2",
	}
	errstr := MergeConfig(conf, def)
	assert.Equal(t, errstr, "")
	assert.Equal(t, conf["key1"], "value1")
	assert.Equal(t, conf["key2"], "value2")

	conf = map[string]interface{}{
		"key1": "value1",
	}
	def = map[string]interface{}{
		"key1": "value2",
	}
	errstr = MergeConfig(conf, def)
	assert.Equal(t, errstr, "rewrite defalut key key1\n")
	assert.Equal(t, conf["key1"], "value1")

	//level2
	conf1 := map[string]interface{}{
		"key1": "value1",
	}
	def1 := map[string]interface{}{
		"key2": "value2",
	}
	conf = map[string]interface{}{
		"key1": conf1,
	}
	def = map[string]interface{}{
		"key2": def1,
	}
	errstr = MergeConfig(conf, def)
	assert.Equal(t, errstr, "")
	assert.Equal(t, conf["key1"].(map[string]interface{})["key1"], "value1")
	assert.Equal(t, conf["key2"].(map[string]interface{})["key2"], "value2")
}

func TestMergeLevel2Error(t *testing.T) {
	//level2
	conf1 := map[string]interface{}{
		"key1": "value1",
	}
	def1 := map[string]interface{}{
		"key1": "value2",
	}
	conf := map[string]interface{}{
		"key1": conf1,
	}
	def := map[string]interface{}{
		"key1": def1,
	}
	errstr := MergeConfig(conf, def)
	assert.Equal(t, errstr, "rewrite defalut key key1.key1\n")
	assert.Equal(t, conf["key1"].(map[string]interface{})["key1"], "value1")
}

func TestMergeToml(t *testing.T) {
	newcfg := mergeCfg(readFile("../cmd/chain33/bityuan.toml"))
	cfg1, err := initCfgString(newcfg)
	assert.Nil(t, err)
	cfg2, err := initCfgString(readFile("testdata/bityuan.toml"))
	assert.Nil(t, err)
	assert.Equal(t, cfg1, cfg2)
}
