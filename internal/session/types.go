package session

import (
	"context"
	"time"

	"github.com/sahaniindrajit/voocl/internal/audio"
	"github.com/sahaniindrajit/voocl/internal/settings"
)

// Timing rules from docs/SPEC.md §2 and §3.
const (
	// MinRecording is the shortest recording that is sent for transcription.
	// Shorter push-to-talk presses and recordings are discarded silently.
	MinRecording = 500 * time.Millisecond
	// HandsFreeCap is the longest a hands-free recording can run before it
	// stops automatically.
	HandsFreeCap = 5 * time.Minute
)

// State is where the session is in the dictation lifecycle.
type State string

// Session states.
const (
	StateIdle         State = "idle"
	StateRecording    State = "recording"
	StateTranscribing State = "transcribing"
	StateCleaning     State = "cleaning"
	StatePasting      State = "pasting"
)

// Input is something the user did: a hotkey or a floating bar button.
type Input string

// Inputs the session understands. Inputs that don't apply to the current
// state are ignored.
const (
	// InputPushToTalkDown starts a push-to-talk recording.
	InputPushToTalkDown Input = "push-to-talk-down"
	// InputPushToTalkUp stops a push-to-talk recording.
	InputPushToTalkUp Input = "push-to-talk-up"
	// InputLock switches a push-to-talk recording to hands-free.
	InputLock Input = "lock"
	// InputHandsFree starts a hands-free recording, locks a push-to-talk
	// recording, or stops a hands-free recording.
	InputHandsFree Input = "hands-free"
	// InputStop stops the current recording (the floating bar's ■ button).
	InputStop Input = "stop"
	// InputCancel discards the current recording or dictation.
	InputCancel Input = "cancel"
	// InputRetry transcribes the last failed recording again.
	InputRetry Input = "retry"
	// InputDismissRetry drops the last failed recording.
	InputDismissRetry Input = "dismiss-retry"
	// InputPasteLast pastes the last transcript again.
	InputPasteLast Input = "paste-last"
	// InputCopyLast copies the last transcript to the clipboard.
	InputCopyLast Input = "copy-last"
)

// Outcome says how a dictation ended. It is set on the Update that returns
// the session to Idle.
type Outcome string

// Dictation outcomes.
const (
	// OutcomeNone means the update isn't the end of a dictation.
	OutcomeNone Outcome = ""
	// OutcomePasted means the text was pasted into the focused app.
	OutcomePasted Outcome = "pasted"
	// OutcomeCopied means pasting wasn't possible, so the text was left on
	// the clipboard for the user to paste ("Copied. Press Ctrl+V to paste.").
	OutcomeCopied Outcome = "copied"
	// OutcomeCanceled means the user canceled.
	OutcomeCanceled Outcome = "canceled"
	// OutcomeDiscarded means there was nothing to send or paste: the
	// recording was too short or the transcript was empty. No error is shown.
	OutcomeDiscarded Outcome = "discarded"
	// OutcomeFailed means a step failed. Err says which and why.
	OutcomeFailed Outcome = "failed"
)

// Update describes the session after a change. It never contains audio or
// text.
type Update struct {
	State State
	// HandsFree is true while recording in hands-free mode.
	HandsFree bool
	// RecordingStarted is when the current recording began, so the floating
	// bar can show elapsed time. Zero when not recording.
	RecordingStarted time.Time
	// Outcome is set when a dictation ends.
	Outcome Outcome
	// Err is set when Outcome is OutcomeFailed.
	Err error
	// CleanupFailed is true when cleanup failed and the raw transcript was
	// pasted instead ("Cleanup failed, pasted the raw transcript.").
	CleanupFailed bool
	// CanRetry is true while a failed recording is kept for Retry.
	CanRetry bool
}

// Recorder captures audio from the microphone.
type Recorder interface {
	// Start begins capturing.
	Start(ctx context.Context) error
	// Stop ends capturing and returns what was recorded.
	Stop(ctx context.Context) (audio.Clip, error)
}

// TranscribeOptions are passed to a Transcriber.
type TranscribeOptions struct {
	// Dictionary holds words to use as vocabulary hints.
	Dictionary []string
}

// Transcript is the result of transcription.
type Transcript struct {
	Text  string
	Model string
}

// Transcriber turns audio into text.
type Transcriber interface {
	Transcribe(ctx context.Context, clip audio.Clip, opts TranscribeOptions) (Transcript, error)
}

// CleanOptions are passed to a Cleaner.
type CleanOptions struct {
	Style settings.Style
	// Dictionary holds words to spell exactly as written.
	Dictionary []string
}

// CleanResult is the result of cleanup.
type CleanResult struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
}

// Cleaner cleans up a transcript in the selected style.
type Cleaner interface {
	Clean(ctx context.Context, text string, opts CleanOptions) (CleanResult, error)
}

// PasteResult reports how text reached the user.
type PasteResult struct {
	// CopiedOnly is true when pasting wasn't possible and the text was left
	// on the clipboard instead.
	CopiedOnly bool
}

// Paster puts text into the focused app.
type Paster interface {
	// Paste pastes text into the focused app, falling back to leaving it on
	// the clipboard when pasting isn't possible.
	Paste(ctx context.Context, text string) (PasteResult, error)
	// Copy puts text on the clipboard.
	Copy(ctx context.Context, text string) error
}
