package internal

import (
	"sync"
)

type Config struct {
	agent             *Agent
	configLock        *sync.RWMutex
	agentEnabled      bool
	profilingDisabled bool
}

func newConfig(agent *Agent) *Config {
	c := &Config{
		agent:             agent,
		configLock:        &sync.RWMutex{},
		agentEnabled:      false,
		profilingDisabled: false,
	}

	return c
}

func (c *Config) setAgentEnabled(val bool) {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	c.agentEnabled = val
}

func (c *Config) isAgentEnabled() bool {
	c.configLock.RLock()
	defer c.configLock.RUnlock()

	return c.agentEnabled
}

func (c *Config) setProfilingDisabled(val bool) {
	c.configLock.Lock()
	defer c.configLock.Unlock()

	c.profilingDisabled = val
}

func (c *Config) isProfilingDisabled() bool {
	c.configLock.RLock()
	defer c.configLock.RUnlock()

	return c.profilingDisabled
}
