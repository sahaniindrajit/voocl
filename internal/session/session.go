package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sahaniindrajit/voocl/internal/audio"
	"github.com/sahaniindrajit/voocl/internal/clock"
	"github.com/sahaniindrajit/voocl/internal/settings"
)

// inputBuffer is how many inputs can queue before Send drops one. Hotkey
// hooks must never block, so Send doesn't wait.
const inputBuffer = 64

// ErrAlreadyRunning is returned when Run is called more than once.
var ErrAlreadyRunning = errors.New("session already running")

// Options configures a Session.
type Options struct {
	Recorder    Recorder
	Transcriber Transcriber
	// Cleaner is required unless Style is settings.StyleRaw.
	Cleaner Cleaner
	Paster  Paster
	Style   settings.Style
	Clock   clock.Clock
	// Logger is optional. Nil discards logs.
	Logger *slog.Logger
	// OnUpdate is called on the session goroutine after every change. It
	// must return quickly and must not call back into the session.
	OnUpdate func(Update)
}

// Session runs the dictation state machine. Create it with New, start it
// with Run, and feed it inputs with Send.
type Session struct {
	opts    Options
	log     *slog.Logger
	events  chan any
	done    chan struct{}
	running atomic.Bool
	workers sync.WaitGroup

	// Owned by the Run goroutine.
	state         State
	handsFree     bool
	started       time.Time
	gen           uint64 // increments for every dictation; stale results are ignored
	capTimer      clock.Timer
	cancelWork    context.CancelFunc
	cleanupFailed bool
	kept          *audio.Clip // last failed recording, for Retry
	lastText      string      // last transcript, for paste/copy last
}

// Internal events delivered to the Run goroutine.
type (
	capReached  struct{ gen uint64 }
	transcribed struct {
		gen  uint64
		clip audio.Clip
		res  Transcript
		err  error
	}
	cleaned struct {
		gen uint64
		raw string
		res CleanResult
		err error
	}
	pasted struct {
		gen uint64
		res PasteResult
		err error
	}
	// flush closes done once every earlier event has been handled.
	flush struct{ done chan struct{} }
)

// New returns a Session in the Idle state.
func New(opts Options) (*Session, error) {
	switch {
	case opts.Recorder == nil:
		return nil, errors.New("new session: recorder is required")
	case opts.Transcriber == nil:
		return nil, errors.New("new session: transcriber is required")
	case opts.Paster == nil:
		return nil, errors.New("new session: paster is required")
	case opts.Clock == nil:
		return nil, errors.New("new session: clock is required")
	}
	switch opts.Style {
	case settings.StyleRaw:
	case settings.StyleClean, settings.StyleFormal:
		if opts.Cleaner == nil {
			return nil, fmt.Errorf("new session: cleaner is required for style %q", opts.Style)
		}
	default:
		return nil, fmt.Errorf("new session: unknown style %q", opts.Style)
	}
	log := opts.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Session{
		opts:   opts,
		log:    log,
		events: make(chan any, inputBuffer),
		done:   make(chan struct{}),
		state:  StateIdle,
	}, nil
}

// Send queues an input. It never blocks: if the queue is full or the session
// has stopped, the input is dropped and Send returns false.
func (s *Session) Send(in Input) bool {
	select {
	case <-s.done:
		return false
	default:
	}
	select {
	case s.events <- in:
		return true
	default:
		s.log.Warn("session input dropped, queue full", "input", in)
		return false
	}
}

// Run handles inputs until ctx is canceled. It stops any recording, cancels
// work in progress, waits for workers to finish and drops the kept
// recording before returning.
func (s *Session) Run(ctx context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return ErrAlreadyRunning
	}
	defer s.shutdown(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-s.events:
			s.handle(ctx, ev)
		}
	}
}

func (s *Session) shutdown(ctx context.Context) {
	if s.state == StateRecording {
		s.stopCapTimer()
		// Run's context is already canceled; stopping the microphone still has to happen.
		if _, err := s.opts.Recorder.Stop(context.WithoutCancel(ctx)); err != nil {
			s.log.Warn("stop recording on shutdown", "err", err)
		}
	}
	s.endWork()
	close(s.done)
	s.workers.Wait()
	s.kept = nil
	s.lastText = ""
}

func (s *Session) handle(ctx context.Context, ev any) {
	switch e := ev.(type) {
	case Input:
		s.handleInput(ctx, e)
	case capReached:
		if e.gen == s.gen && s.state == StateRecording {
			s.log.Info("hands-free cap reached")
			s.finishRecording(ctx)
		}
	case transcribed:
		if e.gen == s.gen && s.state == StateTranscribing {
			s.onTranscribed(ctx, e)
		}
	case cleaned:
		if e.gen == s.gen && s.state == StateCleaning {
			s.onCleaned(ctx, e)
		}
	case pasted:
		if e.gen == s.gen && s.state == StatePasting {
			s.onPasted(e)
		}
	case flush:
		close(e.done)
	}
}

func (s *Session) handleInput(ctx context.Context, in Input) {
	recording := s.state == StateRecording
	switch in {
	case InputPushToTalkDown:
		if s.state == StateIdle {
			s.startRecording(ctx, false)
		}
	case InputPushToTalkUp:
		if recording && !s.handsFree {
			s.finishRecording(ctx)
		}
	case InputLock:
		if recording && !s.handsFree {
			s.lock()
		}
	case InputHandsFree:
		switch {
		case s.state == StateIdle:
			s.startRecording(ctx, true)
		case recording && !s.handsFree:
			s.lock()
		case recording:
			s.finishRecording(ctx)
		}
	case InputStop:
		if recording {
			s.finishRecording(ctx)
		}
	case InputCancel:
		s.cancel(ctx)
	case InputRetry:
		if s.state == StateIdle && s.kept != nil {
			clip := *s.kept
			s.kept = nil
			s.gen++
			s.transcribe(ctx, clip)
		}
	case InputDismissRetry:
		if s.state == StateIdle && s.kept != nil {
			s.kept = nil
			s.toIdle(Update{})
		}
	case InputPasteLast:
		if s.state == StateIdle && s.lastText != "" {
			s.gen++
			s.paste(ctx, s.lastText, false)
		}
	case InputCopyLast:
		if s.state == StateIdle && s.lastText != "" {
			s.copyLast(ctx)
		}
	default:
		s.log.Warn("unknown session input", "input", in)
	}
}

// startRecording publishes the Recording state before starting the
// microphone, so the floating bar appears as fast as possible.
func (s *Session) startRecording(ctx context.Context, handsFree bool) {
	s.gen++
	s.state = StateRecording
	s.handsFree = handsFree
	s.started = s.opts.Clock.Now()
	s.publish(Update{})
	if err := s.opts.Recorder.Start(ctx); err != nil {
		s.fail(fmt.Errorf("start recording: %w", err))
		return
	}
	if handsFree {
		s.startCapTimer()
	}
}

// lock switches a push-to-talk recording to hands-free. The 5-minute cap
// counts from when the recording started.
func (s *Session) lock() {
	s.handsFree = true
	s.startCapTimer()
	s.publish(Update{})
}

func (s *Session) finishRecording(ctx context.Context) {
	held := s.opts.Clock.Now().Sub(s.started)
	pushToTalk := !s.handsFree
	s.stopCapTimer()
	clip, err := s.opts.Recorder.Stop(ctx)
	if err != nil {
		s.fail(fmt.Errorf("stop recording: %w", err))
		return
	}
	if (pushToTalk && held < MinRecording) || clip.Duration() < MinRecording {
		s.toIdle(Update{Outcome: OutcomeDiscarded})
		return
	}
	s.transcribe(ctx, clip)
}

// cancel discards the recording or dictation in progress. A paste in
// progress can't be canceled: it takes a moment and stopping it halfway
// could leave the user's clipboard unrestored.
func (s *Session) cancel(ctx context.Context) {
	switch s.state {
	case StateRecording:
		s.stopCapTimer()
		if _, err := s.opts.Recorder.Stop(ctx); err != nil {
			s.log.Warn("stop recording on cancel", "err", err)
		}
	case StateTranscribing, StateCleaning:
		s.endWork()
	default:
		return
	}
	s.toIdle(Update{Outcome: OutcomeCanceled})
}

func (s *Session) copyLast(ctx context.Context) {
	if err := s.opts.Paster.Copy(ctx, s.lastText); err != nil {
		s.fail(fmt.Errorf("copy last transcript: %w", err))
		return
	}
	s.toIdle(Update{Outcome: OutcomeCopied})
}

func (s *Session) startCapTimer() {
	s.stopCapTimer()
	remaining := max(HandsFreeCap-s.opts.Clock.Now().Sub(s.started), 0)
	gen := s.gen
	s.capTimer = s.opts.Clock.AfterFunc(remaining, func() {
		s.deliver(capReached{gen: gen})
	})
}

func (s *Session) stopCapTimer() {
	if s.capTimer != nil {
		s.capTimer.Stop()
		s.capTimer = nil
	}
}

// deliver sends an event to the Run goroutine, or drops it if the session
// has stopped.
func (s *Session) deliver(ev any) {
	select {
	case s.events <- ev:
	case <-s.done:
	}
}

func (s *Session) fail(err error) {
	s.log.Warn("dictation failed", "state", s.state, "err", err)
	s.toIdle(Update{Outcome: OutcomeFailed, Err: err})
}

// toIdle returns to Idle and publishes u, filling in the retry and cleanup
// flags.
func (s *Session) toIdle(u Update) {
	s.stopCapTimer()
	s.endWork()
	s.state = StateIdle
	s.handsFree = false
	s.started = time.Time{}
	u.CleanupFailed = s.cleanupFailed
	s.cleanupFailed = false
	s.publish(u)
}

// publish reports the current state, with u's outcome fields, to OnUpdate.
func (s *Session) publish(u Update) {
	u.State = s.state
	u.HandsFree = s.handsFree
	u.RecordingStarted = s.started
	u.CanRetry = s.kept != nil
	s.log.Debug("session update", "state", u.State, "outcome", u.Outcome)
	if s.opts.OnUpdate != nil {
		s.opts.OnUpdate(u)
	}
}
