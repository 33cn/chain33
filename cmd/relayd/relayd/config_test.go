package relayd

import "testing"

func TestConfig(t *testing.T) {
	config := NewConfig("./relayd.toml")
	t.Logf("%#v", *config)
	t.Log(*config)
}
