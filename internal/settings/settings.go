package settings

import (
	"errors"
	"fmt"
)

// CurrentVersion is the settings file version written by this build.
const CurrentVersion = 1

// Mode selects the provider mode. Exactly one is active at a time.
type Mode string

// Provider modes.
const (
	ModeOpenAI Mode = "openai"
	ModeClaude Mode = "claude"
	ModeLocal  Mode = "local"
)

// Style selects how dictated text is cleaned up.
type Style string

// Cleanup styles.
const (
	StyleRaw    Style = "raw"
	StyleClean  Style = "clean"
	StyleFormal Style = "formal"
)

// Models holds the model IDs used by each provider.
type Models struct {
	OpenAITranscribe string `json:"openaiTranscribe"`
	OpenAICleanup    string `json:"openaiCleanup"`
	AnthropicCleanup string `json:"anthropicCleanup"`
}

// Shortcuts holds the hotkey bindings. An empty string means unbound.
type Shortcuts struct {
	PushToTalk string `json:"pushToTalk"`
	HandsFree  string `json:"handsFree"`
	PasteLast  string `json:"pasteLast"`
	CopyLast   string `json:"copyLast"`
}

// Settings is the content of the settings file.
type Settings struct {
	Version              int       `json:"version"`
	Mode                 Mode      `json:"mode"`
	Style                Style     `json:"style"`
	Models               Models    `json:"models"`
	Shortcuts            Shortcuts `json:"shortcuts"`
	MicrophoneID         string    `json:"microphoneId"`
	Sounds               bool      `json:"sounds"`
	LaunchAtStartup      bool      `json:"launchAtStartup"`
	AlwaysShowBar        bool      `json:"alwaysShowBar"`
	HistoryRetentionDays int       `json:"historyRetentionDays"`
	Telemetry            bool      `json:"telemetry"`
	OnboardingDone       bool      `json:"onboardingDone"`
}

// ErrInvalid is returned when settings contain a value outside the allowed set.
var ErrInvalid = errors.New("invalid settings")

// Default returns the default settings (docs/SPEC.md §10).
func Default() Settings {
	return Settings{
		Version: CurrentVersion,
		Mode:    ModeLocal,
		Style:   StyleClean,
		Models: Models{
			OpenAITranscribe: "gpt-transcribe",
			OpenAICleanup:    "gpt-6-luna",
			AnthropicCleanup: "claude-haiku-5-5",
		},
		Shortcuts: Shortcuts{
			PushToTalk: "Ctrl+Alt",
			HandsFree:  "Ctrl+Alt+Space",
			PasteLast:  "Alt+Shift+Z",
			CopyLast:   "",
		},
		Sounds:               true,
		LaunchAtStartup:      true,
		AlwaysShowBar:        false,
		HistoryRetentionDays: 30,
		Telemetry:            true,
		OnboardingDone:       false,
	}
}

// Validate reports whether s contains only allowed values.
func (s Settings) Validate() error {
	switch s.Mode {
	case ModeOpenAI, ModeClaude, ModeLocal:
	default:
		return fmt.Errorf("%w: unknown mode %q", ErrInvalid, s.Mode)
	}
	switch s.Style {
	case StyleRaw, StyleClean, StyleFormal:
	default:
		return fmt.Errorf("%w: unknown style %q", ErrInvalid, s.Style)
	}
	if s.HistoryRetentionDays < 1 {
		return fmt.Errorf("%w: history retention must be at least 1 day, got %d", ErrInvalid, s.HistoryRetentionDays)
	}
	return nil
}

// normalize replaces invalid values with their defaults, so a hand-edited or
// damaged field never stops the app from starting.
func (s Settings) normalize() Settings {
	d := Default()
	switch s.Mode {
	case ModeOpenAI, ModeClaude, ModeLocal:
	default:
		s.Mode = d.Mode
	}
	switch s.Style {
	case StyleRaw, StyleClean, StyleFormal:
	default:
		s.Style = d.Style
	}
	if s.HistoryRetentionDays < 1 {
		s.HistoryRetentionDays = d.HistoryRetentionDays
	}
	s.Version = CurrentVersion
	return s
}
