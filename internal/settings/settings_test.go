package settings

import (
	"errors"
	"testing"
)

func TestDefaultIsValid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("Default().Validate() = %v, want nil", err)
	}
}

func TestDefaultMatchesSpec(t *testing.T) {
	d := Default()
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"version", d.Version, 1},
		{"mode", d.Mode, ModeLocal},
		{"style", d.Style, StyleClean},
		{"openai transcribe", d.Models.OpenAITranscribe, "gpt-transcribe"},
		{"openai cleanup", d.Models.OpenAICleanup, "gpt-6-luna"},
		{"anthropic cleanup", d.Models.AnthropicCleanup, "claude-haiku-5-5"},
		{"push to talk", d.Shortcuts.PushToTalk, "Ctrl+Alt"},
		{"hands-free", d.Shortcuts.HandsFree, "Ctrl+Alt+Space"},
		{"paste last", d.Shortcuts.PasteLast, "Alt+Shift+Z"},
		{"copy last", d.Shortcuts.CopyLast, ""},
		{"microphone", d.MicrophoneID, ""},
		{"sounds", d.Sounds, true},
		{"launch at startup", d.LaunchAtStartup, true},
		{"always show bar", d.AlwaysShowBar, false},
		{"history retention", d.HistoryRetentionDays, 30},
		{"telemetry", d.Telemetry, true},
		{"onboarding done", d.OnboardingDone, false},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Errorf("got %v, want %v", c.got, c.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Settings)
		wantErr bool
	}{
		{"defaults", func(*Settings) {}, false},
		{"openai mode", func(s *Settings) { s.Mode = ModeOpenAI }, false},
		{"claude mode", func(s *Settings) { s.Mode = ModeClaude }, false},
		{"raw style", func(s *Settings) { s.Style = StyleRaw }, false},
		{"formal style", func(s *Settings) { s.Style = StyleFormal }, false},
		{"retention of one day", func(s *Settings) { s.HistoryRetentionDays = 1 }, false},
		{"unknown mode", func(s *Settings) { s.Mode = "gemini" }, true},
		{"empty mode", func(s *Settings) { s.Mode = "" }, true},
		{"unknown style", func(s *Settings) { s.Style = "casual" }, true},
		{"zero retention", func(s *Settings) { s.HistoryRetentionDays = 0 }, true},
		{"negative retention", func(s *Settings) { s.HistoryRetentionDays = -5 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Default()
			tt.modify(&s)
			err := s.Validate()
			if tt.wantErr != (err != nil) {
				t.Fatalf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrInvalid) {
				t.Errorf("Validate() = %v, want it to wrap ErrInvalid", err)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Settings)
		want   func(*Settings)
	}{
		{"valid values are kept", func(s *Settings) { s.Mode = ModeClaude; s.Style = StyleFormal }, func(s *Settings) { s.Mode = ModeClaude; s.Style = StyleFormal }},
		{"unknown mode becomes local", func(s *Settings) { s.Mode = "gemini" }, func(*Settings) {}},
		{"unknown style becomes clean", func(s *Settings) { s.Style = "casual" }, func(*Settings) {}},
		{"bad retention becomes 30", func(s *Settings) { s.HistoryRetentionDays = 0 }, func(*Settings) {}},
		{"version becomes current", func(s *Settings) { s.Version = 7 }, func(*Settings) {}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := Default()
			tt.modify(&in)
			want := Default()
			tt.want(&want)
			if got := in.normalize(); got != want {
				t.Errorf("normalize() = %+v, want %+v", got, want)
			}
		})
	}
}
