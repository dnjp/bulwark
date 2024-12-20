package bulwark

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/bradenaw/backpressure"
	"github.com/deixis/faults"
	"golang.org/x/time/rate"
)

func TestAdaptiveThrottleBasic(t *testing.T) {
	duration := 28 * time.Second
	start := time.Now()

	demandRates := []int{5, 10, 20}
	supplyRate := 20.0
	acceptedErrorPct := 0.1

	serverLimiter := backpressure.NewRateLimiter(len(demandRates), supplyRate, supplyRate)

	priorities := NewPriorityRange(MustPriorityFromValue(len(demandRates)))
	clientThrottle := NewAdaptiveThrottle(priorities, WithAdaptiveThrottleWindow(3*time.Second))

	var wg sync.WaitGroup
	requestsByPriority := make([]int, len(demandRates))
	sentByPriority := make([]int, len(demandRates))
	for i, r := range demandRates {
		i := i
		p := MustPriorityFromValue(i)
		l := rate.NewLimiter(rate.Limit(r), r)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Since(start) < duration {
				if err := l.Wait(context.Background()); err != nil {
					t.Error("Wait returned an error", err)

					return
				}

				requestsByPriority[i]++
				_, _ = WithAdaptiveThrottle(clientThrottle, p, func() (struct{}, error) {
					sentByPriority[i]++
					err := serverLimiter.Wait(context.Background(), backpressure.Priority(p.Value()), 1)
					if err != nil {
						return struct{}{}, err
					}
					if rand.Float64() < acceptedErrorPct {
						return struct{}{}, faults.WithNotFound(errors.New("not really an error"))
					}
					return struct{}{}, nil
				})
			}
		}()
	}
	wg.Wait()
	realDuration := time.Since(start)

	totalSent := 0
	for _, sent := range sentByPriority {
		totalSent += sent
	}
	sendRate := float64(totalSent) / realDuration.Seconds()

	t.Logf("total supply:        %.2f", supplyRate)
	t.Logf("aggregate sent rate: %.2f", sendRate)
	var sb strings.Builder
	tw := tabwriter.NewWriter(&sb, 0, 4, 2, ' ', 0)
	fmt.Fprint(tw, "priority\trequest rate\tsend rate\treject %\n")
	for i := range demandRates {
		fmt.Fprintf(
			tw,
			"%d\t%.2f/sec\t%.2f/sec\t%.2f%%\n",
			i,
			float64(requestsByPriority[i])/realDuration.Seconds(),
			float64(sentByPriority[i])/realDuration.Seconds(),
			float64(requestsByPriority[i]-sentByPriority[i])/float64(requestsByPriority[i])*100,
		)
	}
	tw.Flush()
	t.Log("\n" + sb.String())
}

// TestFallback ensures the fallback function is called when an execution is
// rejected by the throttle.
func TestFallback(t *testing.T) {
	ctx := context.Background()
	throttle := NewAdaptiveThrottle(StandardPriorities, WithAdaptiveThrottleRatio(1))
	for i := 0; i < 100; i++ {
		throttle.Throttle(ctx, High, func(ctx context.Context) error {
			return faults.Unavailable(0)
		})
	}

	throttledFnCalls := 0
	fallbackFnCalls := 0
	throttle.Throttle(ctx, High, func(ctx context.Context) error {
		throttledFnCalls++

		return nil
	}, func(ctx context.Context, err error, local bool) error {
		fallbackFnCalls++

		return err
	})

	if throttledFnCalls != 0 {
		t.Errorf("expected throttled function to not be called, got %d", throttledFnCalls)
	}
	if fallbackFnCalls != 1 {
		t.Errorf("expected fallback function to be called once, got %d", fallbackFnCalls)
	}
}

// This test ensures that no errors returned by the throttled function can
// trigger the fallback function.
func TestInvalidFallback(t *testing.T) {
	stdError := errors.New("standard error")

	table := []struct {
		name   string
		err    error
		expect error
	}{
		{
			name:   "No errors",
			err:    nil,
			expect: nil,
		},
		{
			name:   "Error",
			err:    DefaultClientSideRejectionError,
			expect: DefaultClientSideRejectionError,
		},
		{
			name:   "Wrapped error",
			err:    RejectedError(faults.ResourceExhausted()),
			expect: faults.ResourceExhausted(),
		},
		{
			name:   "Standard error",
			err:    stdError,
			expect: stdError,
		},
	}

	ctx := context.Background()

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			throttle := NewAdaptiveThrottle(StandardPriorities)
			err := throttle.Throttle(ctx, High, func(ctx context.Context) error {
				return tt.err
			}, func(ctx context.Context, err error, local bool) error {
				return err
			})
			if !errors.Is(err, tt.expect) {
				t.Errorf("expected %v, got %v", tt.expect, err)
			}
		})
	}
}

func TestPriorityValidation(t *testing.T) {
	testCases := []struct {
		name        string
		r           PriorityRange
		p           Priority
		expectedP   Priority
		expectError bool
		expectPanic bool
	}{
		{
			name:        "StandardPriorities",
			r:           StandardPriorities,
			p:           Low,
			expectedP:   Low,
			expectError: false,
			expectPanic: false,
		},
		{
			name:        "Adjusted",
			r:           NewPriorityRange(Medium, WithRangeValidator(OnInvalidPriorityAdjust)),
			p:           Low,
			expectedP:   Medium,
			expectError: false,
			expectPanic: false,
		},
		{
			name:        "Panic",
			r:           NewPriorityRange(Medium, WithRangeValidator(OnInvalidPriorityPanic)),
			p:           Low,
			expectedP:   Medium,
			expectError: false,
			expectPanic: true,
		},
		{
			name:        "Error",
			r:           NewPriorityRange(Medium, WithRangeValidator(OnInvalidPriorityError)),
			p:           Low,
			expectedP:   Medium,
			expectError: true,
			expectPanic: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runFn := func() (panicked bool, p Priority, err error) {
				defer func() {
					r := recover()
					panicked = r != nil
				}()

				p, err = tc.r.Validate(tc.p)
				if tc.expectError && err == nil {
					t.Errorf("expected error, got nil")
				}

				return panicked, p, err
			}

			panicked, p, err := runFn()
			if tc.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if tc.expectPanic {
				if !panicked {
					t.Errorf("expected panic, got nil")
				}
				return
			}

			if tc.expectedP.Value() != p.Value() {
				t.Errorf("expected %d, got %d", tc.expectedP.Value(), p.Value())
			}
		})
	}
}
