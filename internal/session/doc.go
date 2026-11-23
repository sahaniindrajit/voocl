// Package session owns the dictation lifecycle.
//
// A Session is a state machine running in one goroutine (Run). Hotkeys and
// the floating bar send it Inputs; it records audio, then transcribes,
// cleans up and pastes the text, and reports every change as an Update.
//
//	Idle → Recording → Transcribing → Cleaning → Pasting → Idle
//
// Slow steps (transcription, cleanup, paste) run in worker goroutines owned
// by the session. Their results come back to the session goroutine as
// events, so session state is never shared between goroutines.
//
// The session depends on small interfaces (Recorder, Transcriber, Cleaner,
// Paster) so it can be tested with fakes and doesn't know which provider is
// in use. See docs/ARCHITECTURE.md ("Session controller").
package session
