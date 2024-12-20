package bulwark

import (
	"fmt"
	"log/slog"

	"github.com/deixis/faults"
)

// ValidatorFunc should return the validated priority value. If the priority is
// invalid, the function should return an error.
type ValidatorFunc func(p Priority, priorities PriorityRange) (Priority, error)

var (
	// OnInvalidPriorityPanic panics when a priority is out of range.
	// A priority is out of range when it is less than 0 or greater than or equal
	// to priorities-1.
	OnInvalidPriorityPanic = func(p Priority, priorities PriorityRange) (Priority, error) {
		if p.Value() >= priorities.Lowest().Value() {
			panic(fmt.Sprintf("bulwark: priority must be in the range [0, %d), but got %d",
				priorities.Lowest().Value(), p.Value()))
		}

		return p, nil
	}

	// OnInvalidPriorityAdjust adjusts the priority to the nearest valid value.
	// A negative priority will be set to the lowest priority.
	// A priority is out of range when it is less than 0 or greater than or equal
	// to priorities-1.
	OnInvalidPriorityAdjust = func(p Priority, priorities PriorityRange) (Priority, error) {
		if p.Value() <= priorities.Lowest().Value() {
			return p, nil
		}
		slog.Warn("bulwark: priority is out of range", "max", priorities.Lowest(), "priority", p)

		// Receiving an invalid value is likely due to an input that was not properly
		// validated, so this prevents abuse of the system.
		return priorities.Lowest(), nil
	}

	// OnInvalidPriorityError returns an error when a priority is out of range.
	// A priority is out of range when it is less than 0 or greater than or equal
	// to priorities-1.
	OnInvalidPriorityError = func(p Priority, priorities PriorityRange) (Priority, error) {
		if p.Value() > priorities.Lowest().Value() {
			return p, faults.Bad(&faults.FieldViolation{
				Field: "priority",
				Description: fmt.Sprintf("priority must be in the range [0, %d), but got %d",
					priorities.Lowest().Value(), p.Value()),
			})
		}

		return p, nil
	}
)
