package clock

import "time"

// Clock tells the time and schedules callbacks.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// AfterFunc calls f in its own goroutine once d has passed.
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is a callback scheduled with AfterFunc.
type Timer interface {
	// Stop prevents the callback from running. It reports whether the call
	// stopped the timer, false if it already ran or was stopped.
	Stop() bool
}

// Real returns a Clock backed by the time package.
func Real() Clock {
	return realClock{}
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) AfterFunc(d time.Duration, f func()) Timer {
	return time.AfterFunc(d, f)
}
