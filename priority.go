package bulwark

import (
	"fmt"
)

// StandardPriorities is the number of priority levels that are available.
// This value should be used when creating a new AdaptiveThrottle when the
// default Priority constants are used.
//
//		throttler := bulwark.NewAdaptiveThrottle(bulwark.Priorities)
//		_, err := bulwark.WithAdaptiveThrottle(throttler, bulwark.High, throttledFn)
//		if err != nil {
//			// handle the error
//	 }
var StandardPriorities = NewPriorityRange(Low)

// Priority determines the importance of a request in ascending order.
// e.g. priority 0 is more important than priority 1.
//
// When a system reaches its capacity, it will sort requests by their priority
// and process them. Lower-priority requests can either be delayed or dropped.
type Priority struct {
	p int
}

func (p Priority) Value() int {
	return p.p
}

// These are pre-defined priority levels that can be used, but any int value
// can be used as a priority.
var (
	// Use High when for requests that are critical to the overall experience.
	High Priority = Priority{0}
	// Use Important for requests that are important, but not critical.
	Important Priority = Priority{1}
	// Use Medium for noncritical requests where an elevated latency or
	// failure rate would not significantly impact the experience.
	Medium Priority = Priority{2}
	// Use Low for trivial requests and good for any system that can retry
	// later when the system has spare capacity.
	Low Priority = Priority{3}
)

var allPriorities = []Priority{Low, Medium, Important, High}

// MustPriorityFromValue returns a Priority from an int value. If the value is
// not found, it will panic.
func MustPriorityFromValue(p int) Priority {
	for _, priority := range allPriorities {
		if priority.Value() == p {
			return priority
		}
	}

	panic(fmt.Sprintf("bulwark: priority not found: %d", p))
}

// AdaptPriorityFromValue returns a Priority from an int value. If the value is
// not found, it will return the lowest priority.
func AdaptPriorityFromValue(p int) Priority {
	for _, priority := range allPriorities {
		if priority.Value() == p {
			return priority
		}
	}

	return Low
}

// PriorityRange is a range of priorities.
type PriorityRange struct {
	lowest, highest Priority
	validate        ValidatorFunc
}

// PriorityRangeOption is an option for a PriorityRange.
type PriorityRangeOption func(r *PriorityRange)

// WithRangeValidator sets the validator for the priority range.
func WithRangeValidator(v ValidatorFunc) PriorityRangeOption {
	return func(r *PriorityRange) {
		r.validate = v
	}
}

// NewPriorityRange creates a PriorityRange which represents the range
// [provided lowest, High].
func NewPriorityRange(lowest Priority, options ...PriorityRangeOption) PriorityRange {
	r := PriorityRange{
		lowest:   lowest,
		highest:  High,
		validate: OnInvalidPriorityAdjust,
	}

	for _, option := range options {
		option(&r)
	}

	return r
}

// Range returns the number of priorities in the range.
func (r PriorityRange) Range() int {
	// the lower the priority, the higher the value.
	return (r.lowest.Value() - r.highest.Value()) + 1
}

// Lowest returns the lowest priority in the range.
func (r PriorityRange) Lowest() Priority {
	return r.lowest
}

// Validate validates a priority. If the priority is not valid, it will return
// the lowest priority.
func (r PriorityRange) Validate(p Priority) (Priority, error) {
	return r.validate(p, r)
}
