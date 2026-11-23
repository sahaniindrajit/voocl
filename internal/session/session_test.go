package session

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/sahaniindrajit/voocl/internal/clock"
	"github.com/sahaniindrajit/voocl/internal/settings"
)

var errBoom = errors.New("boom")

func TestNewValidatesOptions(t *testing.T) {
	valid := func() Options {
		return Options{
			Recorder:    &fakeRecorder{},
			Transcriber: &fakeTranscriber{},
			Cleaner:     &fakeCleaner{},
			Paster:      &fakePaster{},
			Style:       settings.StyleClean,
			Clock:       clock.NewFake(time.Time{}),
		}
	}
	tests := []struct {
		name    string
		modify  func(*Options)
		wantErr bool
	}{
		{"valid", func(*Options) {}, false},
		{"raw style without cleaner", func(o *Options) { o.Style = settings.StyleRaw; o.Cleaner = nil }, false},
		{"no logger or OnUpdate", func(o *Options) { o.Logger = nil; o.OnUpdate = nil }, false},
		{"no recorder", func(o *Options) { o.Recorder = nil }, true},
		{"no transcriber", func(o *Options) { o.Transcriber = nil }, true},
		{"no paster", func(o *Options) { o.Paster = nil }, true},
		{"no clock", func(o *Options) { o.Clock = nil }, true},
		{"clean style without cleaner", func(o *Options) { o.Cleaner = nil }, true},
		{"formal style without cleaner", func(o *Options) { o.Style = settings.StyleFormal; o.Cleaner = nil }, true},
		{"unknown style", func(o *Options) { o.Style = "casual" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := valid()
			tt.modify(&opts)
			_, err := New(opts)
			if tt.wantErr != (err != nil) {
				t.Errorf("New() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPushToTalkRawStyle(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputPushToTalkDown)
	u := h.expect(StateRecording, OutcomeNone)
	if u.HandsFree {
		t.Error("HandsFree = true for push-to-talk")
	}
	if !u.RecordingStarted.Equal(h.clock.Now()) {
		t.Errorf("RecordingStarted = %v, want %v", u.RecordingStarted, h.clock.Now())
	}
	h.clock.Advance(time.Second)
	h.send(InputPushToTalkUp)
	h.expect(StateTranscribing, OutcomeNone)
	h.expect(StatePasting, OutcomeNone)
	u = h.expect(StateIdle, OutcomePasted)
	if u.CanRetry || u.CleanupFailed || u.Err != nil || !u.RecordingStarted.IsZero() {
		t.Errorf("final update = %+v, want a clean Idle", u)
	}

	if got := h.paster.pastedTexts(); !slices.Equal(got, []string{"hello world"}) {
		t.Errorf("pasted %q, want the trimmed transcript", got)
	}
	if starts, stops := h.rec.counts(); starts != 1 || stops != 1 {
		t.Errorf("recorder starts/stops = %d/%d, want 1/1", starts, stops)
	}
}

func TestPushToTalkCleanStyle(t *testing.T) {
	for _, style := range []settings.Style{settings.StyleClean, settings.StyleFormal} {
		t.Run(string(style), func(t *testing.T) {
			h := newHarness(t, style)
			h.record(time.Second)
			h.expect(StateCleaning, OutcomeNone)
			h.expect(StatePasting, OutcomeNone)
			h.expect(StateIdle, OutcomePasted)

			h.cl.set(func(f *fakeCleaner) {
				if !slices.Equal(f.inputs, []string{"hello world"}) {
					t.Errorf("cleaner got %q, want the trimmed transcript", f.inputs)
				}
				if !slices.Equal(f.styles, []settings.Style{style}) {
					t.Errorf("cleaner styles = %v, want %v", f.styles, style)
				}
			})
			if got := h.paster.pastedTexts(); !slices.Equal(got, []string{"Hello world."}) {
				t.Errorf("pasted %q, want the cleaned text", got)
			}
		})
	}
}

func TestShortRecordingsAreDiscarded(t *testing.T) {
	tests := []struct {
		name string
		held time.Duration
		clip time.Duration
	}{
		{"short press", 499 * time.Millisecond, 2 * time.Second},
		{"short clip", time.Second, 499 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, settings.StyleRaw)
			h.rec.set(func(r *fakeRecorder) { r.clip = clipOf(tt.clip, 1) })
			h.send(InputPushToTalkDown)
			h.expect(StateRecording, OutcomeNone)
			h.clock.Advance(tt.held)
			h.send(InputPushToTalkUp)
			u := h.expect(StateIdle, OutcomeDiscarded)
			if u.Err != nil {
				t.Errorf("Err = %v, want nil (no error is shown)", u.Err)
			}
			if _, stops := h.rec.counts(); stops != 1 {
				t.Errorf("recorder stops = %d, want 1", stops)
			}
			if n := len(h.tr.calls()); n != 0 {
				t.Errorf("transcriber called %d times, want 0", n)
			}
		})
	}
}

func TestMinimumLengthIsAllowed(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.rec.set(func(r *fakeRecorder) { r.clip = clipOf(MinRecording, 1) })
	h.record(MinRecording)
	h.expect(StatePasting, OutcomeNone)
	h.expect(StateIdle, OutcomePasted)
}

func TestHandsFree(t *testing.T) {
	stops := []struct {
		name string
		in   Input
	}{
		{"stopped with the hands-free shortcut", InputHandsFree},
		{"stopped with the stop button", InputStop},
	}
	for _, tt := range stops {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t, settings.StyleRaw)
			h.send(InputHandsFree)
			u := h.expect(StateRecording, OutcomeNone)
			if !u.HandsFree {
				t.Error("HandsFree = false, want true")
			}
			// Push-to-talk keys don't affect a hands-free recording.
			h.send(InputPushToTalkUp)
			h.send(InputPushToTalkDown)
			h.send(InputLock)
			h.expectNoUpdate()

			h.clock.Advance(time.Second)
			h.send(tt.in)
			h.expect(StateTranscribing, OutcomeNone)
			h.expect(StatePasting, OutcomeNone)
			h.expect(StateIdle, OutcomePasted)
		})
	}
}

func TestHandsFreeShortStopIsDiscardedByClipLength(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.rec.set(func(r *fakeRecorder) { r.clip = clipOf(200*time.Millisecond, 1) })
	h.send(InputHandsFree)
	h.expect(StateRecording, OutcomeNone)
	h.send(InputHandsFree)
	h.expect(StateIdle, OutcomeDiscarded)
}

func TestLockToHandsFree(t *testing.T) {
	for _, in := range []Input{InputLock, InputHandsFree} {
		t.Run(string(in), func(t *testing.T) {
			h := newHarness(t, settings.StyleRaw)
			h.send(InputPushToTalkDown)
			first := h.expect(StateRecording, OutcomeNone)
			h.clock.Advance(200 * time.Millisecond)
			h.send(in)
			u := h.expect(StateRecording, OutcomeNone)
			if !u.HandsFree {
				t.Error("HandsFree = false after lock, want true")
			}
			if !u.RecordingStarted.Equal(first.RecordingStarted) {
				t.Error("RecordingStarted changed on lock, want the original start")
			}

			// Releasing push-to-talk no longer stops the recording.
			h.send(InputPushToTalkUp)
			h.expectNoUpdate()

			h.clock.Advance(time.Second)
			h.send(InputHandsFree)
			h.expect(StateTranscribing, OutcomeNone)
			h.expect(StatePasting, OutcomeNone)
			h.expect(StateIdle, OutcomePasted)
		})
	}
}

func TestHandsFreeCap(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputHandsFree)
	h.expect(StateRecording, OutcomeNone)
	h.clock.Advance(HandsFreeCap - time.Millisecond)
	h.expectNoUpdate()
	h.clock.Advance(time.Millisecond)
	h.expect(StateTranscribing, OutcomeNone)
	h.expect(StatePasting, OutcomeNone)
	h.expect(StateIdle, OutcomePasted)
}

func TestHandsFreeCapCountsFromRecordingStartAfterLock(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputPushToTalkDown)
	h.expect(StateRecording, OutcomeNone)
	h.clock.Advance(time.Minute)
	h.send(InputLock)
	h.expect(StateRecording, OutcomeNone)
	h.clock.Advance(HandsFreeCap - time.Minute)
	h.expect(StateTranscribing, OutcomeNone)
}

func TestHandsFreeCapTimerStopsWithRecording(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputHandsFree)
	h.expect(StateRecording, OutcomeNone)
	h.clock.Advance(time.Second)
	h.send(InputHandsFree)
	h.expect(StateTranscribing, OutcomeNone)
	h.expect(StatePasting, OutcomeNone)
	h.expect(StateIdle, OutcomePasted)

	// A new recording isn't stopped by the old cap.
	h.send(InputHandsFree)
	h.expect(StateRecording, OutcomeNone)
	// Past the old recording's cap, before the new one's.
	h.clock.Advance(HandsFreeCap - 500*time.Millisecond)
	h.expectNoUpdate()
	if _, stops := h.rec.counts(); stops != 1 {
		t.Errorf("recorder stops = %d, want 1", stops)
	}
}

func TestPushToTalkHasNoCap(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputPushToTalkDown)
	h.expect(StateRecording, OutcomeNone)
	h.clock.Advance(2 * HandsFreeCap)
	h.expectNoUpdate()
}

func TestCancelWhileRecording(t *testing.T) {
	for _, start := range []Input{InputPushToTalkDown, InputHandsFree} {
		t.Run(string(start), func(t *testing.T) {
			h := newHarness(t, settings.StyleRaw)
			h.send(start)
			h.expect(StateRecording, OutcomeNone)
			h.clock.Advance(time.Second)
			h.send(InputCancel)
			h.expect(StateIdle, OutcomeCanceled)
			if _, stops := h.rec.counts(); stops != 1 {
				t.Errorf("recorder stops = %d, want 1", stops)
			}
			// Releasing the key and the cap afterwards do nothing.
			h.send(InputPushToTalkUp)
			h.clock.Advance(HandsFreeCap)
			h.expectNoUpdate()
			if n := len(h.tr.calls()); n != 0 {
				t.Errorf("transcriber called %d times, want 0", n)
			}
		})
	}
}

func TestCancelWhileTranscribing(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	g := newGate()
	h.tr.set(func(f *fakeTranscriber) { f.gate = g })
	h.record(time.Second)
	g.waitEntered(t)
	h.send(InputCancel)
	h.expect(StateIdle, OutcomeCanceled)
	h.flush()

	h.shutdown() // waits for the worker, so its result is in
	if !errors.Is(h.tr.cancelErr(), context.Canceled) {
		t.Errorf("transcriber context error = %v, want context.Canceled", h.tr.cancelErr())
	}
	if n := len(h.paster.pastedTexts()); n != 0 {
		t.Errorf("pasted %d times after cancel, want 0", n)
	}
}

func TestCancelWhileCleaning(t *testing.T) {
	h := newHarness(t, settings.StyleClean)
	g := newGate()
	h.cl.set(func(f *fakeCleaner) { f.gate = g })
	h.record(time.Second)
	h.expect(StateCleaning, OutcomeNone)
	g.waitEntered(t)
	h.send(InputCancel)
	u := h.expect(StateIdle, OutcomeCanceled)
	if u.CanRetry {
		t.Error("CanRetry = true after cancel, want false")
	}
	h.shutdown()
	if n := len(h.paster.pastedTexts()); n != 0 {
		t.Errorf("pasted %d times after cancel, want 0", n)
	}
}

func TestCancelIsIgnoredWhilePasting(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	g := newGate()
	h.paster.set(func(f *fakePaster) { f.gate = g })
	h.record(time.Second)
	h.expect(StatePasting, OutcomeNone)
	g.waitEntered(t)
	h.send(InputCancel)
	h.expectNoUpdate()
	g.release()
	h.expect(StateIdle, OutcomePasted)
}

func TestCancelIsIgnoredWhenIdle(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputCancel)
	h.expectNoUpdate()
}

func TestNewRecordingCantStartWhileProcessing(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	g := newGate()
	h.tr.set(func(f *fakeTranscriber) { f.gate = g })
	h.record(time.Second)
	g.waitEntered(t)
	h.send(InputPushToTalkDown)
	h.send(InputHandsFree)
	h.send(InputPasteLast)
	h.send(InputRetry)
	h.expectNoUpdate()
	if starts, _ := h.rec.counts(); starts != 1 {
		t.Errorf("recorder starts = %d, want 1", starts)
	}
	g.release()
	h.expect(StatePasting, OutcomeNone)
	h.expect(StateIdle, OutcomePasted)
}

func TestRepeatedKeyDownWhileRecordingIsIgnored(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.send(InputPushToTalkDown)
	h.expect(StateRecording, OutcomeNone)
	h.send(InputPushToTalkDown) // key repeat
	h.expectNoUpdate()
	if starts, _ := h.rec.counts(); starts != 1 {
		t.Errorf("recorder starts = %d, want 1", starts)
	}
}

func TestRecorderErrors(t *testing.T) {
	t.Run("start fails", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.rec.set(func(r *fakeRecorder) { r.startErr = errBoom })
		h.send(InputHandsFree)
		h.expect(StateRecording, OutcomeNone)
		u := h.expect(StateIdle, OutcomeFailed)
		if !errors.Is(u.Err, errBoom) {
			t.Errorf("Err = %v, want it to wrap the recorder error", u.Err)
		}
		if u.CanRetry {
			t.Error("CanRetry = true, want false (nothing was recorded)")
		}
		// No cap timer is left running.
		h.clock.Advance(HandsFreeCap)
		h.expectNoUpdate()
	})
	t.Run("stop fails", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.rec.set(func(r *fakeRecorder) { r.stopErr = errBoom })
		h.send(InputPushToTalkDown)
		h.expect(StateRecording, OutcomeNone)
		h.clock.Advance(time.Second)
		h.send(InputPushToTalkUp)
		u := h.expect(StateIdle, OutcomeFailed)
		if !errors.Is(u.Err, errBoom) {
			t.Errorf("Err = %v, want it to wrap the recorder error", u.Err)
		}
	})
}

func TestTranscriptionFailureKeepsRecordingForRetry(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.tr.set(func(f *fakeTranscriber) { f.err = errBoom })
	h.record(time.Second)
	u := h.expect(StateIdle, OutcomeFailed)
	if !errors.Is(u.Err, errBoom) || !u.CanRetry {
		t.Fatalf("update = %+v, want the transcription error with CanRetry", u)
	}

	// Retry fails again: the recording is still kept.
	h.send(InputRetry)
	h.expect(StateTranscribing, OutcomeNone)
	u = h.expect(StateIdle, OutcomeFailed)
	if !u.CanRetry {
		t.Fatal("CanRetry = false after a second failure, want true")
	}

	// Retry succeeds: the same recording is used, then dropped.
	h.tr.set(func(f *fakeTranscriber) { f.err = nil })
	h.send(InputRetry)
	h.expect(StateTranscribing, OutcomeNone)
	h.expect(StatePasting, OutcomeNone)
	u = h.expect(StateIdle, OutcomePasted)
	if u.CanRetry {
		t.Error("CanRetry = true after a successful retry, want false")
	}
	clips := h.tr.calls()
	if len(clips) != 3 {
		t.Fatalf("transcriber called %d times, want 3", len(clips))
	}
	for i, c := range clips {
		if c.Samples[0] != 1 {
			t.Errorf("call %d transcribed recording %d, want recording 1", i, c.Samples[0])
		}
	}
	h.send(InputRetry)
	h.expectNoUpdate()
}

func TestDismissRetry(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.tr.set(func(f *fakeTranscriber) { f.err = errBoom })
	h.record(time.Second)
	h.expect(StateIdle, OutcomeFailed)
	h.send(InputDismissRetry)
	u := h.expect(StateIdle, OutcomeNone)
	if u.CanRetry {
		t.Error("CanRetry = true after dismiss, want false")
	}
	h.send(InputRetry)
	h.send(InputDismissRetry)
	h.expectNoUpdate()
}

func TestKeptRecordingAfterNewRecordings(t *testing.T) {
	t.Run("a successful recording drops it", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.tr.set(func(f *fakeTranscriber) { f.err = errBoom })
		h.record(time.Second)
		h.expect(StateIdle, OutcomeFailed)

		h.tr.set(func(f *fakeTranscriber) { f.err = nil })
		h.dictateRaw()
		h.send(InputRetry)
		h.expectNoUpdate()
	})
	t.Run("a failed recording replaces it", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.tr.set(func(f *fakeTranscriber) { f.err = errBoom })
		h.record(time.Second)
		h.expect(StateIdle, OutcomeFailed)

		h.rec.set(func(r *fakeRecorder) { r.clip = clipOf(2*time.Second, 2) })
		h.record(time.Second)
		h.expect(StateIdle, OutcomeFailed)

		h.send(InputRetry)
		h.expect(StateTranscribing, OutcomeNone)
		clips := h.tr.calls()
		if got := clips[len(clips)-1].Samples[0]; got != 2 {
			t.Errorf("retry used recording %d, want recording 2", got)
		}
	})
	t.Run("a canceled or discarded recording keeps it", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.tr.set(func(f *fakeTranscriber) { f.err = errBoom })
		h.record(time.Second)
		h.expect(StateIdle, OutcomeFailed)

		h.send(InputPushToTalkDown)
		if u := h.expect(StateRecording, OutcomeNone); !u.CanRetry {
			t.Error("CanRetry = false while recording, want true")
		}
		h.send(InputCancel)
		if u := h.expect(StateIdle, OutcomeCanceled); !u.CanRetry {
			t.Error("CanRetry = false after canceling a new recording, want true")
		}
		h.send(InputPushToTalkDown)
		h.expect(StateRecording, OutcomeNone)
		h.send(InputPushToTalkUp) // short tap
		if u := h.expect(StateIdle, OutcomeDiscarded); !u.CanRetry {
			t.Error("CanRetry = false after a short tap, want true")
		}
	})
	t.Run("canceling a retry drops it", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.tr.set(func(f *fakeTranscriber) { f.err = errBoom })
		h.record(time.Second)
		h.expect(StateIdle, OutcomeFailed)

		g := newGate()
		h.tr.set(func(f *fakeTranscriber) { f.gate = g })
		h.send(InputRetry)
		h.expect(StateTranscribing, OutcomeNone)
		g.waitEntered(t)
		h.send(InputCancel)
		if u := h.expect(StateIdle, OutcomeCanceled); u.CanRetry {
			t.Error("CanRetry = true after canceling the retry, want false")
		}
	})
}

func TestEmptyResultsAreDiscarded(t *testing.T) {
	t.Run("empty transcript", func(t *testing.T) {
		h := newHarness(t, settings.StyleClean)
		h.tr.set(func(f *fakeTranscriber) { f.text = "   " })
		h.record(time.Second)
		h.expect(StateIdle, OutcomeDiscarded)
		h.cl.set(func(f *fakeCleaner) {
			if len(f.inputs) != 0 {
				t.Error("cleaner was called for an empty transcript")
			}
		})
	})
	t.Run("empty cleanup", func(t *testing.T) {
		h := newHarness(t, settings.StyleClean)
		h.cl.set(func(f *fakeCleaner) { f.text = "\n" })
		h.record(time.Second)
		h.expect(StateCleaning, OutcomeNone)
		h.expect(StateIdle, OutcomeDiscarded)
		if n := len(h.paster.pastedTexts()); n != 0 {
			t.Errorf("pasted %d times, want 0", n)
		}
	})
}

func TestCleanupFailurePastesRawTranscript(t *testing.T) {
	h := newHarness(t, settings.StyleClean)
	h.cl.set(func(f *fakeCleaner) { f.err = errBoom })
	h.record(time.Second)
	h.expect(StateCleaning, OutcomeNone)
	h.expect(StatePasting, OutcomeNone)
	u := h.expect(StateIdle, OutcomePasted)
	if !u.CleanupFailed {
		t.Error("CleanupFailed = false, want true")
	}
	if got := h.paster.pastedTexts(); !slices.Equal(got, []string{"hello world"}) {
		t.Errorf("pasted %q, want the raw transcript", got)
	}

	// The flag only applies to that dictation.
	h.cl.set(func(f *fakeCleaner) { f.err = nil })
	h.record(time.Second)
	h.expect(StateCleaning, OutcomeNone)
	h.expect(StatePasting, OutcomeNone)
	if u := h.expect(StateIdle, OutcomePasted); u.CleanupFailed {
		t.Error("CleanupFailed = true on the next dictation, want false")
	}
}

func TestPasteOutcomes(t *testing.T) {
	t.Run("blocked paste leaves text on clipboard", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.paster.set(func(f *fakePaster) { f.copiedOnly = true })
		h.record(time.Second)
		h.expect(StatePasting, OutcomeNone)
		h.expect(StateIdle, OutcomeCopied)
	})
	t.Run("failed paste keeps text for paste last", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.paster.set(func(f *fakePaster) { f.err = errBoom })
		h.record(time.Second)
		h.expect(StatePasting, OutcomeNone)
		u := h.expect(StateIdle, OutcomeFailed)
		if !errors.Is(u.Err, errBoom) {
			t.Errorf("Err = %v, want the paste error", u.Err)
		}
		h.paster.set(func(f *fakePaster) { f.err = nil })
		h.send(InputPasteLast)
		h.expect(StatePasting, OutcomeNone)
		h.expect(StateIdle, OutcomePasted)
		if got := h.paster.pastedTexts(); !slices.Equal(got, []string{"hello world", "hello world"}) {
			t.Errorf("pasted %q, want the transcript twice", got)
		}
	})
}

func TestPasteAndCopyLast(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	// Nothing to paste or copy before the first dictation.
	h.send(InputPasteLast)
	h.send(InputCopyLast)
	h.expectNoUpdate()

	h.dictateRaw()
	h.send(InputPasteLast)
	h.expect(StatePasting, OutcomeNone)
	h.expect(StateIdle, OutcomePasted)
	h.send(InputCopyLast)
	h.expect(StateIdle, OutcomeCopied)

	if got := h.paster.pastedTexts(); !slices.Equal(got, []string{"hello world", "hello world"}) {
		t.Errorf("pasted %q, want the transcript twice", got)
	}
	if got := h.paster.copiedTexts(); !slices.Equal(got, []string{"hello world"}) {
		t.Errorf("copied %q, want the transcript", got)
	}

	h.paster.set(func(f *fakePaster) { f.copyErr = errBoom })
	h.send(InputCopyLast)
	if u := h.expect(StateIdle, OutcomeFailed); !errors.Is(u.Err, errBoom) {
		t.Errorf("Err = %v, want the copy error", u.Err)
	}
}

func TestRunTwice(t *testing.T) {
	h := newHarness(t, settings.StyleRaw)
	h.flush() // Run has started
	if err := h.s.Run(context.Background()); !errors.Is(err, ErrAlreadyRunning) {
		t.Errorf("second Run() = %v, want ErrAlreadyRunning", err)
	}
}

func TestShutdown(t *testing.T) {
	t.Run("while recording stops the microphone", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		h.send(InputHandsFree)
		h.expect(StateRecording, OutcomeNone)
		h.shutdown()
		if _, stops := h.rec.counts(); stops != 1 {
			t.Errorf("recorder stops = %d, want 1", stops)
		}
		if h.s.Send(InputPushToTalkDown) {
			t.Error("Send() after shutdown = true, want false")
		}
		// The cap firing after shutdown is harmless.
		h.clock.Advance(HandsFreeCap)
	})
	t.Run("while transcribing cancels the work", func(t *testing.T) {
		h := newHarness(t, settings.StyleRaw)
		g := newGate()
		h.tr.set(func(f *fakeTranscriber) { f.gate = g })
		h.record(time.Second)
		g.waitEntered(t)
		h.shutdown()
		if !errors.Is(h.tr.cancelErr(), context.Canceled) {
			t.Errorf("transcriber context error = %v, want context.Canceled", h.tr.cancelErr())
		}
	})
}

func TestSendDropsWhenQueueIsFull(t *testing.T) {
	s, err := New(Options{
		Recorder:    &fakeRecorder{},
		Transcriber: &fakeTranscriber{},
		Paster:      &fakePaster{},
		Style:       settings.StyleRaw,
		Clock:       clock.NewFake(time.Time{}),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Run isn't started, so nothing drains the queue.
	for i := range inputBuffer {
		if !s.Send(InputCancel) {
			t.Fatalf("Send() #%d = false, want true", i)
		}
	}
	if s.Send(InputCancel) {
		t.Error("Send() on a full queue = true, want false")
	}
}
