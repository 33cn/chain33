package internal

import (
	"time"
)

type ConfigLoader struct {
	LoadDelay    int64
	LoadInterval int64

	agent   *Agent
	started *Flag

	loadTimer *Timer

	lastLoadTimestamp int64
}

func newConfigLoader(agent *Agent) *ConfigLoader {
	cl := &ConfigLoader{
		LoadDelay:    2,
		LoadInterval: 120,

		agent:   agent,
		started: &Flag{},

		loadTimer: nil,

		lastLoadTimestamp: 0,
	}

	return cl
}

func (cl *ConfigLoader) start() {
	if !cl.started.SetIfUnset() {
		return
	}

	if cl.agent.AutoProfiling {
		cl.loadTimer = cl.agent.createTimer(time.Duration(cl.LoadDelay)*time.Second, time.Duration(cl.LoadInterval)*time.Second, func() {
			cl.load()
		})
	}
}

func (cl *ConfigLoader) stop() {
	if !cl.started.UnsetIfSet() {
		return
	}

	if cl.loadTimer != nil {
		cl.loadTimer.Stop()
	}
}

func (cl *ConfigLoader) load() {
	now := time.Now().Unix()
	if !cl.agent.AutoProfiling && cl.lastLoadTimestamp > now-cl.LoadInterval {
		return
	}
	cl.lastLoadTimestamp = now

	payload := map[string]interface{}{}
	if config, err := cl.agent.apiRequest.post("config", payload); err == nil {
		// agent_enabled yes|no
		if agentEnabled, exists := config["agent_enabled"]; exists {
			cl.agent.config.setAgentEnabled(agentEnabled.(string) == "yes")
		} else {
			cl.agent.config.setAgentEnabled(false)
		}

		// profiling_enabled yes|no
		if profilingDisabled, exists := config["profiling_disabled"]; exists {
			cl.agent.config.setProfilingDisabled(profilingDisabled.(string) == "yes")
		} else {
			cl.agent.config.setProfilingDisabled(false)
		}

		if cl.agent.config.isAgentEnabled() && !cl.agent.config.isProfilingDisabled() {
			cl.agent.cpuReporter.start()
			cl.agent.allocationReporter.start()
			cl.agent.blockReporter.start()
		} else {
			cl.agent.cpuReporter.stop()
			cl.agent.allocationReporter.stop()
			cl.agent.blockReporter.stop()
		}

		if cl.agent.config.isAgentEnabled() {
			cl.agent.segmentReporter.start()
			cl.agent.errorReporter.start()
			cl.agent.processReporter.start()
			cl.agent.log("Agent enabled")
		} else {
			cl.agent.segmentReporter.stop()
			cl.agent.errorReporter.stop()
			cl.agent.processReporter.stop()
			cl.agent.log("Agent disabled")
		}
	} else {
		cl.agent.log("Error loading config from Dashboard")
		cl.agent.error(err)
	}
}
