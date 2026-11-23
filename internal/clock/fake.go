package clock

import (
	"sort"
	"sync"
	"time"
)

// Fake is a Clock for tests. Time only moves when Advance is called.
type Fake struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

// NewFake returns a Fake clock set to start.
func NewFake(start time.Time) *Fake {
	return &Fake{now: start}
}

// Now returns the fake current time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// AfterFunc schedules fn to run when Advance moves time past d from now.
func (f *Fake) AfterFunc(d time.Duration, fn func()) Timer {
	f.mu.Lock()
	defer f.mu.Unlock()
	t := &fakeTimer{clock: f, at: f.now.Add(d), fn: fn}
	f.timers = append(f.timers, t)
	return t
}

// Advance moves time forward by d and runs every timer that is now due, in
// the order they're due. Callbacks run synchronously on the caller's
// goroutine, after the clock's lock is released.
func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	var due, pending []*fakeTimer
	for _, t := range f.timers {
		if t.at.After(f.now) {
			pending = append(pending, t)
		} else {
			due = append(due, t)
		}
	}
	f.timers = pending
	f.mu.Unlock()

	sort.SliceStable(due, func(i, j int) bool { return due[i].at.Before(due[j].at) })
	for _, t := range due {
		t.fn()
	}
}

// remove drops t from the pending timers and reports whether it was pending.
func (f *Fake) remove(t *fakeTimer) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, p := range f.timers {
		if p == t {
			f.timers = append(f.timers[:i], f.timers[i+1:]...)
			return true
		}
	}
	return false
}

type fakeTimer struct {
	clock *Fake
	at    time.Time
	fn    func()
}

func (t *fakeTimer) Stop() bool {
	return t.clock.remove(t)
}
