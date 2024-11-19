# Bulwark

- [Bulwark](#bulwark)
	- [Quick start](#quick-start)
	- [Priority](#priority)
	- [Configuration](#configuration)
		- [Throttle ratio](#throttle-ratio)
		- [Throttle minimum rate](#throttle-minimum-rate)
		- [Throttle window](#throttle-window)
		- [Accepted errors](#accepted-errors)
	- [Inspirations](#inspirations)
	- [Further reading](#further-reading)

**Bulwark** is a self-tuning adaptive throttle written in Go that enhances the resilience of distributed services.

Distributed services are vulnerable to cascading failures when parts of the system become overloaded. Gracefully managing overload conditions is essential for operating a reliable system, and this library addresses that need. When Bulwark detects that a significant portion of recent requests are being rejected due to "service unavailable" or "quota exhaustion" errors, it begins self-regulating by capping the amount of permitted outbound traffic. Requests exceeding the cap fail locally, without even reaching the network.

Under normal conditions, when available resources exceed demand, Bulwark operates passively and does not affect traffic. Additionally, Bulwark does not queue requests, ensuring that it introduces no added latency.

## Quick start

```go
package main

import (
	"context"
	"fmt"

	"github.com/deixis/bulwark"
)

func main() {
	// This creates an adaptive throttle with the default number of priorities
	// available priorities.
	throttle := bulwark.NewAdaptiveThrottle(
		bulwark.StandardPriorities,
		// Other options can be set here to customise the throttle
	)

	// Any function that needs to be throttled can be wrapped with this function.
	// Each function call has a priority level set, which will be used to determine
	// how the throttle should prioritse the call when the system is under load.
	msg, err := bulwark.WithAdaptiveThrottle(throttle, bulwark.Medium, func() (string, error) {
		// Call external service here...
		return "Hello", nil
	})
	if err != nil {
		// handle the error
	}
	fmt.Println(msg)

	// A priority level can be attached to a context. For example, requests coming
	// from users would be prioritised over background tasks.
	ctx := context.Background()
	ctx = bulwark.WithPriority(ctx, bulwark.High)

	// Same call, but with the priority taken from the context
	priority := bulwark.PriorityFromContext(ctx, bulwark.Medium)
	msg, err = bulwark.WithAdaptiveThrottle(throttle, priority, func() (string, error) {
		// Call external service here...
		return "World", nil
	})
	if err != nil {
		// handle the error
	}
	fmt.Println(msg)
}

```

## Priority

When the system reaches capacity, Bulwark dynamically determines the probability of successfully processing a request based on its priority. Higher-priority requests are given a greater chance of being forwarded to the backend, resulting in a lower error rate for higher-priority traffic during overload conditions. Since Bulwark operates using a probabilistic model, this prioritisation does not introduce additional delay to request handling.

> Important: Under normal operating conditions, the system is expected to have sufficient spare resources, ensuring that all request priorities are treated equally.

This example shows Bulwark determining the priority of a request from the request `context` and will use `bulwark.Medium` by default when the context does not provide a value.

```go
	priority := bulwark.PriorityFromContext(ctx, bulwark.Medium)
	msg, err = bulwark.WithAdaptiveThrottle(throttle, priority, func() (string, error) {
		// Call external service here...
		return "World", nil
	})
	if err != nil {
		// handle the error
	}
```

## Configuration

### Throttle ratio

The throttle ratio (a.k.a `k`) is a variable which determines the number of requests accepted based on the observed limit.

For example, when `k=2` the throttle **will allow twice as many requests to actually reach the backend as it believes will succeed.** Reducing the modifier to `k=1.1` means 110% of the observed limit will be allowed to reach the backend.

Higher values of `k` mean that the throttle will react more slowly when a backend becomes unhealthy, but react more quickly when it becomes healthy again, and will allow more load to an unhealthy backend. `k=2` is usually a good place to start, but backends that serve "cheap" requests (e.g. in-memory caches) may need a lower value.

> We generally prefer the 2x multiplier. By allowing more requests to reach the backend than are expected to actually be allowed, we waste more resources at the backend, but we also speed up the propagation of state from the backend to the clients. [Google SRE book]

```go
	throttle := bulwark.NewAdaptiveThrottle(
		bulwark.StandardPriorities,
		bulwark.WithAdaptivethrottleatio(1.1),
	)
```

### Throttle minimum rate

Configure the minimum number of requests per second that the adaptive throttle will allow (approximately) to reach the backend, **even if all requests are failing**. Sending a small number of requests to the backend is critical to continuously evaluate its health and tune the throttle.

```go
	throttle := bulwark.NewAdaptiveThrottle(
		bulwark.StandardPriorities,
		bulwark.WithAdaptiveThrottleMinimumRate(0.5),
	)
```

### Throttle window

Set the time window over which the throttle remembers requests for use in figuring out the success rate.

A larger window will make the throttle react more slowly to changes in the backend's health, but will also make it more resilient to short-term fluctuations in the backend's health. But a larger window will also increase the amount of memory used by the throttle.

By default, it uses a window of `1 * time.Minute`

```go
	throttle := bulwark.NewAdaptiveThrottle(
		bulwark.StandardPriorities,
		bulwark.WithAdaptiveThrottleWindow(5 * time.Minute),
	)
```

### Accepted errors

Set the function that determines whether an error should be considered for the throttling. When the call to `fn` returns true, the error is NOT counted towards the throttling.

```go
	isAcceptedErrors := func(err error) bool {
		return errors.Is(err, context.Canceled) // || other conditions
	}
	throttle := bulwark.NewAdaptiveThrottle(
		bulwark.StandardPriorities,
		bulwark.WithAcceptedErrors(isAcceptedErrors),
	)
```

> Errors unrelated to resource constraints or a service's inability to handle traffic should be allowed. For instance, errors caused by invalid user requests or authentication failures should be accepted.

## Inspirations

Most reliability libraries in Go are either lack robust support for `context` propagation and cancellation and very few of them provide support for Quality of Service (QoS) prioritisation when services are under heavy load.

This gap inspired me to create this library. I started by building upon the `AdaptiveThrottle` implementation from `bradenaw/backpressure`, which provided the foundational concepts I needed. A big thank you to Braden Walker for his excellent work!

The design of this library is also shaped by my own experiences managing failures in distributed systems, as well as insights drawn from the remarkable work done at Netflix and Google.

## Further reading

1. [Handling Overload - Google](https://sre.google/sre-book/handling-overload/)
2. [Performance Under Load - Netflix](https://netflixtechblog.medium.com/performance-under-load-3e6fa9a60581)