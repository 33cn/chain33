# StackImpact Go Agent

## Overview

StackImpact is a production-grade performance profiler built for both production and development environments. It gives developers continuous and historical view of application performance with line-of-code precision that is essential for locating CPU, memory allocation and blocking call hot spots as well as latency bottlenecks. Included runtime metric and error monitoring complement profiles for extensive performance analysis. Learn more at [stackimpact.com](https://stackimpact.com/).

![dashboard](https://stackimpact.com/wp-content/uploads/2017/06/hotspots-cpu-1.4.png)

#### Features

* Continuous hot spot profiling for CPU, memory allocations, network, system calls and lock contention.
* Continuous latency bottleneck tracing.
* Error and panic monitoring.
* Health monitoring including CPU, memory, garbage collection and other runtime metrics.
* Anomaly detection.
* Multiple account users for team collaboration.

Learn more on the [features](https://stackimpact.com/features/) page (with screenshots).


#### Documentation

See full [documentation](https://stackimpact.com/docs/) for reference.



## Requirements

Linux, OS X or Windows. Go version 1.5+.


## Getting started


#### Create StackImpact account

[Sign up](https://dashboard.stackimpact.com/#/signup) for a free account (also with GitHub login).


#### Installing the agent

Install the Go agent by running

```
go get github.com/stackimpact/stackimpact-go
```

And import the package `github.com/stackimpact/stackimpact-go` in your application.


#### Configuring the agent

Start the agent by specifying the agent key and application name. The agent key can be found in your account's Configuration section.

```go
agent := stackimpact.Start(stackimpact.Options{
	AgentKey: "agent key here",
	AppName: "MyGoApp",
})
```

All initialization options:

* `AgentKey` (Required) The access key for communication with the StackImpact servers.
* `AppName` (Required) A name to identify and group application data. Typically, a single codebase, deployable unit or executable module corresponds to one application. Sometimes also referred as a service.
* `AppVersion` (Optional) Sets application version, which can be used to associate profiling information with the source code release.
* `AppEnvironment` (Optional) Used to differentiate applications in different environments.
* `HostName` (Optional) By default, host name will be the OS hostname.
* `ProxyAddress` (Optional) Proxy server URL to use when connecting to the Dashboard servers.
* `HTTPClient` (Optional) An `http.Client` instance to be used instead of the default client for reporting data to Dashboard servers.
* `DisableAutoProfiling` (Optional) If set to `true`, disables the default automatic profiling and reporting. `agent.Profile()` and `agent.Report()` should be used instead. Useful for environments without support for timers or background tasks.
* `Debug` (Optional) Enables debug logging.
* `Logger` (Optional) A `log.Logger` instance to be used instead of default `STDOUT` logger.



Example:

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/stackimpact/stackimpact-go"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello world!")
}

func main() {
	agent := stackimpact.Start(stackimpact.Options{
		AgentKey: "agent key here",
		AppName: "Basic Go Server",
		AppVersion: "1.0.0",
		AppEnvironment: "production",
	})

	http.HandleFunc("/", handler) 
	http.ListenAndServe(":8080", nil)
}
```

#### Programmatic profiling

*The use of programmatic profiling is optional.*

Programmatic profiling is suitable for repeating code, such as request or event handlers. By default, the agent starts and stops profiling automatically. In order to make sure the agent profiles the most relevant execution intervals, the `agent.Profile()` method can be used.

```go
// Use this method to instruct the agent to start and stop 
// profiling. It does not guarantee that any profiler will be 
// started. The decision is made by the agent based on the 
// overhead constraints. The method returns Span object, on 
// which the Stop() method should be called. 
span := agent.Profile();
defer span.Stop();
```

```go
// A helper function to profile HTTP handler execution by wrapping 
// http.Handle method parameters.
// Usage example:
//   http.Handle(agent.ProfileHandler("/some-path", someHandler))
pattern, wrappedHandler := agent.ProfileHandler(pattern, handler)
```

```go
// A helper function to profile HTTP handler function execution 
// by wrapping http.HandleFunc method parameters.
// Usage example:
//   http.HandleFunc(agent.ProfileHandlerFunc("/some-path", someHandlerFunc))
pattern, wrappedHandlerFunc := agent.ProfileHandlerFunc(pattern, handlerFunc)
```

#### Measuring code segments

*The use of Segment API is optional.*

To measure the execution time of arbitrary parts of the application, the Segment API can be used.

```go
// Starts measurement of execution time of a code segment.
// To stop measurement, call Stop on returned Segment object.
// After calling Stop, the segment is recorded, aggregated and
// reported with regular intervals.
segment := agent.MeasureSegment("Segment1")
defer segment.Stop()
```

```go
// A helper function to measure HTTP handler execution by wrapping 
// http.Handle method parameters.
// Usage example:
//   http.Handle(agent.MeasureHandler("/some-path", someHandler))
pattern, wrappedHandler := agent.MeasureHandler(pattern, handler)
```

```go
// A helper function to measure HTTP handler function execution 
// by wrapping http.HandleFunc method parameters.
// Usage example:
//   http.HandleFunc(agent.MeasureHandlerFunc("/some-path", someHandlerFunc))
pattern, wrappedHandlerFunc := agent.MeasureHandlerFunc(pattern, handlerFunc)
```


#### Monitoring errors

*The use of Error API is optional.*


To monitor exceptions and panics with stack traces, the error recording API can be used.

Recording handled errors:

```go
// Aggregates and reports errors with regular intervals.
agent.RecordError(someError)
```

Recording panics without recovering:

```go
// Aggregates and reports panics with regular intervals.
defer agent.RecordPanic()
```

Recording and recovering from panics:

```go
// Aggregates and reports panics with regular intervals. This function also
// recovers from panics.
defer agent.RecordAndRecoverPanic()
```


#### Analyzing performance data in the Dashboard

Once your application is restarted, you can start observing continuously recorded CPU, memory, I/O, and other hot spot profiles, execution bottlenecks as well as process metrics in the [Dashboard](https://dashboard.stackimpact.com/).


#### Troubleshooting

To enable debug logging, add `Debug: true` to startup options. If the debug log doesn't give you any hints on how to fix a problem, please report it to our support team in your account's Support section.


## Overhead

The agent overhead is measured to be less than 1% for applications under high load. For applications that are horizontally scaled to multiple processes, StackImpact agents are only active on a small subset of the processes at any point of time, therefore the total overhead is much lower.
