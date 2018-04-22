package relayd

import "testing"

func TestConfig(t *testing.T) {
	config := config("./relayd.toml")
	t.Logf("%#v", *config)
	t.Log(*config)
}
