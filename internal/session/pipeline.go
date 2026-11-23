package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/sahaniindrajit/voocl/internal/audio"
	"github.com/sahaniindrajit/voocl/internal/settings"
)

// This file holds the processing steps after recording: transcribe, clean
// and paste. Each step runs in a worker goroutine and reports back with an
// event, so the Run goroutine stays free to handle Cancel.

func (s *Session) transcribe(ctx context.Context, clip audio.Clip) {
	s.state = StateTranscribing
	s.publish(Update{})
	gen := s.gen
	s.work(ctx, func(ctx context.Context) {
		res, err := s.opts.Transcriber.Transcribe(ctx, clip, TranscribeOptions{})
		s.deliver(transcribed{gen: gen, clip: clip, res: res, err: err})
	})
}

func (s *Session) onTranscribed(ctx context.Context, e transcribed) {
	s.endWork()
	if e.err != nil {
		s.kept = &e.clip
		s.fail(fmt.Errorf("transcribe: %w", e.err))
		return
	}
	s.kept = nil
	text := strings.TrimSpace(e.res.Text)
	if text == "" {
		s.toIdle(Update{Outcome: OutcomeDiscarded})
		return
	}
	if s.opts.Style == settings.StyleRaw {
		s.paste(ctx, text, false)
		return
	}
	s.clean(ctx, text)
}

func (s *Session) clean(ctx context.Context, raw string) {
	s.state = StateCleaning
	s.publish(Update{})
	gen := s.gen
	opts := CleanOptions{Style: s.opts.Style}
	s.work(ctx, func(ctx context.Context) {
		res, err := s.opts.Cleaner.Clean(ctx, raw, opts)
		s.deliver(cleaned{gen: gen, raw: raw, res: res, err: err})
	})
}

// onCleaned pastes the cleaned text. If cleanup failed, it pastes the raw
// transcript instead so the dictation isn't lost.
func (s *Session) onCleaned(ctx context.Context, e cleaned) {
	s.endWork()
	if e.err != nil {
		s.log.Warn("cleanup failed, pasting raw transcript", "err", e.err)
		s.paste(ctx, e.raw, true)
		return
	}
	text := strings.TrimSpace(e.res.Text)
	if text == "" {
		s.toIdle(Update{Outcome: OutcomeDiscarded})
		return
	}
	s.paste(ctx, text, false)
}

// paste remembers text for paste/copy last before pasting, so it's
// available even if pasting fails.
func (s *Session) paste(ctx context.Context, text string, cleanupFailed bool) {
	s.lastText = text
	s.cleanupFailed = cleanupFailed
	s.state = StatePasting
	s.publish(Update{})
	gen := s.gen
	s.work(ctx, func(ctx context.Context) {
		res, err := s.opts.Paster.Paste(ctx, text)
		s.deliver(pasted{gen: gen, res: res, err: err})
	})
}

func (s *Session) onPasted(e pasted) {
	s.endWork()
	switch {
	case e.err != nil:
		s.fail(fmt.Errorf("paste: %w", e.err))
	case e.res.CopiedOnly:
		s.toIdle(Update{Outcome: OutcomeCopied})
	default:
		s.toIdle(Update{Outcome: OutcomePasted})
	}
}

// work runs f in a worker goroutine with a context that endWork cancels.
func (s *Session) work(ctx context.Context, f func(ctx context.Context)) {
	workCtx, cancel := context.WithCancel(ctx)
	s.cancelWork = cancel
	s.workers.Go(func() { f(workCtx) })
}

// endWork cancels the current worker's context, if any.
func (s *Session) endWork() {
	if s.cancelWork != nil {
		s.cancelWork()
		s.cancelWork = nil
	}
}
