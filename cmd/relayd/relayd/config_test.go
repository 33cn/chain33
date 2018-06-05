package relayd_test

import (
	"testing"

	. "gitlab.33.cn/chain33/chain33/cmd/relayd/relayd"
)

func TestConfig(t *testing.T) {
	config := NewConfig("./../relayd.toml")
	t.Logf("%#v", *config)
	t.Log(*config)
}
