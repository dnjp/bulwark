package bulwark

import "fmt"

// StandardPriorities is the number of priority levels that are available.
// This value should be used when creating a new AdaptiveThrottle when the
// default Priority constants are used.
//
//		throttler := bulwark.NewAdaptiveThrottle(bulwark.Priorities)
//		_, err := bulwark.WithAdaptiveThrottle(throttler, bulwark.High, throttledFn)
//		if err != nil {
//			// handle the error
//	 }
var StandardPriorities = &correctiveRange{Low, High}

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

func PriorityFromInt(p int) Priority {
	// TODO: validate p is one of the supported priorities
	return Priority{p}
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

type PriorityRange interface {
	Range() int
	In(p Priority) bool
	Normalize(p Priority) (Priority, error)
}

type correctiveRange struct {
	min, max Priority
}

func (r *correctiveRange) Range() int {
	return r.max.Value()
}

func (r *correctiveRange) In(p Priority) bool {
	return p.Value() >= r.min.Value() && p.Value() <= r.max.Value()
}

func (r *correctiveRange) Normalize(p Priority) (Priority, error) {
	if !r.In(p) {
		return r.adapt(p), nil
	}
	return p, nil
}

func (r *correctiveRange) adapt(p Priority) Priority {
	if p.Value() > r.max.Value() {
		return r.max
	}
	if p.Value() < r.min.Value() {
		return r.min
	}

	return p
}

func NewCorrectiveRange(min, max Priority) (PriorityRange, error) {
	if min.Value() > max.Value() {
		return nil, fmt.Errorf("min priority %d is greater than max priority %d", min.Value(), max.Value())
	}
	return &correctiveRange{min, max}, nil
}
