package session

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sahaniindrajit/voocl/internal/audio"
	"github.com/sahaniindrajit/voocl/internal/clock"
	"github.com/sahaniindrajit/voocl/internal/settings"
)

const waitTimeout = 5 * time.Second

// gate makes a fake call wait until the test releases it or the call's
// context is canceled. A nil gate doesn't wait.
type gate struct {
	ch      chan struct{}
	entered chan struct{}
}

func newGate() *gate {
	return &gate{ch: make(chan struct{}), entered: make(chan struct{}, 10)}
}

// wait blocks until release or ctx is done, and returns ctx's error if the
// context ended first.
func (g *gate) wait(ctx context.Context) error {
	if g == nil {
		return nil
	}
	g.entered <- struct{}{}
	select {
	case <-g.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *gate) release() { close(g.ch) }

// waitEntered waits until a call is blocked on the gate.
func (g *gate) waitEntered(t *testing.T) {
	t.Helper()
	select {
	case <-g.entered:
	case <-time.After(waitTimeout):
		t.Fatal("call never reached the gate")
	}
}

// clipOf returns a recording of length d whose first sample is id, so tests
// can tell recordings apart.
func clipOf(d time.Duration, id int16) audio.Clip {
	n := max(int(d*audio.SampleRate/time.Second), 1)
	samples := make([]int16, n)
	samples[0] = id
	return audio.Clip{Samples: samples, SampleRate: audio.SampleRate}
}

type fakeRecorder struct {
	mu       sync.Mutex
	starts   int
	stops    int
	startErr error
	stopErr  error
	clip     audio.Clip
}

func (r *fakeRecorder) Start(context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.starts++
	return r.startErr
}

func (r *fakeRecorder) Stop(context.Context) (audio.Clip, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops++
	return r.clip, r.stopErr
}

func (r *fakeRecorder) set(f func(r *fakeRecorder)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	f(r)
}

func (r *fakeRecorder) counts() (starts, stops int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.starts, r.stops
}

type fakeTranscriber struct {
	mu     sync.Mutex
	gate   *gate
	text   string
	err    error
	clips  []audio.Clip
	ctxErr error
}

func (f *fakeTranscriber) Transcribe(ctx context.Context, clip audio.Clip, _ TranscribeOptions) (Transcript, error) {
	f.mu.Lock()
	f.clips = append(f.clips, clip)
	g := f.gate
	f.mu.Unlock()
	if err := g.wait(ctx); err != nil {
		f.mu.Lock()
		f.ctxErr = err
		f.mu.Unlock()
		return Transcript{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return Transcript{Text: f.text, Model: "fake-transcribe"}, f.err
}

func (f *fakeTranscriber) set(fn func(f *fakeTranscriber)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

func (f *fakeTranscriber) calls() []audio.Clip {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]audio.Clip(nil), f.clips...)
}

func (f *fakeTranscriber) cancelErr() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ctxErr
}

type fakeCleaner struct {
	mu     sync.Mutex
	gate   *gate
	text   string
	err    error
	inputs []string
	styles []settings.Style
	ctxErr error
}

func (f *fakeCleaner) Clean(ctx context.Context, text string, opts CleanOptions) (CleanResult, error) {
	f.mu.Lock()
	f.inputs = append(f.inputs, text)
	f.styles = append(f.styles, opts.Style)
	g := f.gate
	f.mu.Unlock()
	if err := g.wait(ctx); err != nil {
		f.mu.Lock()
		f.ctxErr = err
		f.mu.Unlock()
		return CleanResult{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return CleanResult{Text: f.text, Model: "fake-clean"}, f.err
}

func (f *fakeCleaner) set(fn func(f *fakeCleaner)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

type fakePaster struct {
	mu         sync.Mutex
	gate       *gate
	copiedOnly bool
	err        error
	copyErr    error
	pasted     []string
	copied     []string
	ctxErr     error
}

func (f *fakePaster) Paste(ctx context.Context, text string) (PasteResult, error) {
	f.mu.Lock()
	f.pasted = append(f.pasted, text)
	g := f.gate
	f.mu.Unlock()
	if err := g.wait(ctx); err != nil {
		f.mu.Lock()
		f.ctxErr = err
		f.mu.Unlock()
		return PasteResult{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return PasteResult{CopiedOnly: f.copiedOnly}, f.err
}

func (f *fakePaster) Copy(_ context.Context, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.copied = append(f.copied, text)
	return f.copyErr
}

func (f *fakePaster) set(fn func(f *fakePaster)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f)
}

func (f *fakePaster) pastedTexts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.pasted...)
}

func (f *fakePaster) copiedTexts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.copied...)
}

// harness runs a Session with fakes and collects its updates.
type harness struct {
	t       *testing.T
	s       *Session
	clock   *clock.Fake
	rec     *fakeRecorder
	tr      *fakeTranscriber
	cl      *fakeCleaner
	paster  *fakePaster
	updates chan Update
	stop    context.CancelFunc
	runErr  chan error
}

func newHarness(t *testing.T, style settings.Style) *harness {
	t.Helper()
	h := &harness{
		t:       t,
		clock:   clock.NewFake(time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)),
		rec:     &fakeRecorder{clip: clipOf(2*time.Second, 1)},
		tr:      &fakeTranscriber{text: "  hello world  "},
		cl:      &fakeCleaner{text: "Hello world."},
		paster:  &fakePaster{},
		updates: make(chan Update, 100),
		runErr:  make(chan error, 1),
	}
	s, err := New(Options{
		Recorder:    h.rec,
		Transcriber: h.tr,
		Cleaner:     h.cl,
		Paster:      h.paster,
		Style:       style,
		Clock:       h.clock,
		OnUpdate:    func(u Update) { h.updates <- u },
	})
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	h.s = s
	ctx, cancel := context.WithCancel(context.Background())
	h.stop = cancel
	go func() { h.runErr <- s.Run(ctx) }()
	t.Cleanup(h.shutdown)
	return h
}

// shutdown stops Run and waits for it to return.
func (h *harness) shutdown() {
	h.t.Helper()
	h.stop()
	select {
	case err := <-h.runErr:
		if err != nil {
			h.t.Errorf("Run() = %v", err)
		}
		h.runErr <- nil // let later calls return immediately
	case <-time.After(waitTimeout):
		h.t.Fatal("Run didn't return after cancel")
	}
}

func (h *harness) send(in Input) {
	h.t.Helper()
	if !h.s.Send(in) {
		h.t.Fatalf("Send(%s) = false", in)
	}
}

// next returns the next update.
func (h *harness) next() Update {
	h.t.Helper()
	select {
	case u := <-h.updates:
		return u
	case <-time.After(waitTimeout):
		h.t.Fatal("no update")
		return Update{}
	}
}

// expect returns the next update and checks its state and outcome.
func (h *harness) expect(state State, outcome Outcome) Update {
	h.t.Helper()
	u := h.next()
	if u.State != state || u.Outcome != outcome {
		h.t.Fatalf("update = {State: %s, Outcome: %q, Err: %v}, want {State: %s, Outcome: %q}",
			u.State, u.Outcome, u.Err, state, outcome)
	}
	return u
}

// flush waits until the session has handled every event sent so far.
func (h *harness) flush() {
	h.t.Helper()
	done := make(chan struct{})
	h.s.events <- flush{done: done}
	select {
	case <-done:
	case <-time.After(waitTimeout):
		h.t.Fatal("flush timed out")
	}
}

// expectNoUpdate checks that nothing was published after the session handled
// every event sent so far.
func (h *harness) expectNoUpdate() {
	h.t.Helper()
	h.flush()
	select {
	case u := <-h.updates:
		h.t.Fatalf("unexpected update {State: %s, Outcome: %q, Err: %v}", u.State, u.Outcome, u.Err)
	default:
	}
}

// record does a push-to-talk recording held for d and waits for Transcribing.
func (h *harness) record(d time.Duration) {
	h.t.Helper()
	h.send(InputPushToTalkDown)
	h.expect(StateRecording, OutcomeNone)
	h.clock.Advance(d)
	h.send(InputPushToTalkUp)
	h.expect(StateTranscribing, OutcomeNone)
}

// dictateRaw does a full raw-style dictation that ends with a paste.
func (h *harness) dictateRaw() {
	h.t.Helper()
	h.record(time.Second)
	h.expect(StatePasting, OutcomeNone)
	h.expect(StateIdle, OutcomePasted)
}
