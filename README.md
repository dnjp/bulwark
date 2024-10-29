# Bulwark (Work in progress)

Go library that helps making services more resilient.

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
	throttler := bulwark.NewAdaptiveThrottle(
		bulwark.StandardPriorities,
		// Other options can be set here to customise the throttler
	)

	// Any function that needs to be throttled can be wrapped with this function.
	// Each function call has a priority level set, which will be used to determine
	// how the throttler should prioritse the call when the system is under load.
	msg, err := bulwark.WithAdaptiveThrottle(throttler, bulwark.Medium, func() (string, error) {
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
	msg, err = bulwark.WithAdaptiveThrottle(throttler, priority, func() (string, error) {
		return "World", nil
	})
	if err != nil {
		// handle the error
	}
	fmt.Println(msg)
}

```

## Inspirations

1. https://github.com/bradenaw/backpressure
2. https://github.com/slok/goresilience
3. https://sre.google/sre-book/handling-overload/