package clock

import (
	"testing"
	"time"
)

var start = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func TestFakeNowAndAdvance(t *testing.T) {
	c := NewFake(start)
	if got := c.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}
	c.Advance(90 * time.Second)
	if got, want := c.Now(), start.Add(90*time.Second); !got.Equal(want) {
		t.Errorf("Now() = %v, want %v", got, want)
	}
}

func TestFakeAfterFunc(t *testing.T) {
	tests := []struct {
		name    string
		delays  []time.Duration
		advance []time.Duration
		stop    []int // indexes of timers to stop before advancing
		want    []int // indexes of timers that fire, in order
	}{
		{"fires when due", []time.Duration{time.Second}, []time.Duration{time.Second}, nil, []int{0}},
		{"doesn't fire early", []time.Duration{time.Second}, []time.Duration{999 * time.Millisecond}, nil, nil},
		{"fires across several advances", []time.Duration{time.Second}, []time.Duration{500 * time.Millisecond, 500 * time.Millisecond}, nil, []int{0}},
		{"fires in due order", []time.Duration{3 * time.Second, time.Second, 2 * time.Second}, []time.Duration{5 * time.Second}, nil, []int{1, 2, 0}},
		{"stopped timer doesn't fire", []time.Duration{time.Second, time.Second}, []time.Duration{time.Second}, []int{0}, []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewFake(start)
			var fired []int
			timers := make([]Timer, len(tt.delays))
			for i, d := range tt.delays {
				timers[i] = c.AfterFunc(d, func() { fired = append(fired, i) })
			}
			for _, i := range tt.stop {
				if !timers[i].Stop() {
					t.Errorf("Stop() on pending timer %d = false, want true", i)
				}
			}
			for _, d := range tt.advance {
				c.Advance(d)
			}
			if len(fired) != len(tt.want) {
				t.Fatalf("fired %v, want %v", fired, tt.want)
			}
			for i := range fired {
				if fired[i] != tt.want[i] {
					t.Fatalf("fired %v, want %v", fired, tt.want)
				}
			}
		})
	}
}

func TestFakeStopAfterFire(t *testing.T) {
	c := NewFake(start)
	timer := c.AfterFunc(time.Second, func() {})
	c.Advance(time.Second)
	if timer.Stop() {
		t.Error("Stop() after the timer fired = true, want false")
	}
}

func TestFakeCallbackCanScheduleTimer(t *testing.T) {
	c := NewFake(start)
	fired := false
	c.AfterFunc(time.Second, func() {
		c.AfterFunc(time.Second, func() { fired = true })
	})
	c.Advance(time.Second)
	if fired {
		t.Fatal("nested timer fired too early")
	}
	c.Advance(time.Second)
	if !fired {
		t.Error("nested timer didn't fire")
	}
}

func TestRealClock(t *testing.T) {
	c := Real()
	if c.Now().IsZero() {
		t.Error("Now() returned the zero time")
	}
	done := make(chan struct{})
	c.AfterFunc(time.Millisecond, func() { close(done) })
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("AfterFunc callback didn't run")
	}
}
