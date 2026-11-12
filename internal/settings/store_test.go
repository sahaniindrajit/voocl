package settings

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(Options{Path: filepath.Join(t.TempDir(), "Voocl", "settings.json")})
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	return s
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content *string // nil means no file
		want    func(*Settings)
		wantErr error
	}{
		{
			name:    "missing file returns defaults",
			content: nil,
			want:    func(*Settings) {},
		},
		{
			name:    "empty object returns defaults",
			content: ptr(`{}`),
			want:    func(*Settings) {},
		},
		{
			name:    "missing fields take defaults",
			content: ptr(`{"version": 1, "mode": "openai", "models": {"openaiCleanup": "custom-model"}}`),
			want: func(s *Settings) {
				s.Mode = ModeOpenAI
				s.Models.OpenAICleanup = "custom-model"
			},
		},
		{
			name:    "false booleans are kept",
			content: ptr(`{"sounds": false, "telemetry": false, "launchAtStartup": false}`),
			want: func(s *Settings) {
				s.Sounds = false
				s.Telemetry = false
				s.LaunchAtStartup = false
			},
		},
		{
			name:    "unknown fields are ignored",
			content: ptr(`{"mode": "claude", "someFutureField": {"a": 1}}`),
			want:    func(s *Settings) { s.Mode = ModeClaude },
		},
		{
			name:    "invalid values become defaults",
			content: ptr(`{"mode": "gemini", "style": "casual", "historyRetentionDays": 0, "sounds": false}`),
			want:    func(s *Settings) { s.Sounds = false },
		},
		{
			name:    "file without version is read as version 1",
			content: ptr(`{"style": "raw"}`),
			want:    func(s *Settings) { s.Style = StyleRaw },
		},
		{
			name:    "file from a newer version keeps known fields",
			content: ptr(`{"version": 99, "style": "formal"}`),
			want:    func(s *Settings) { s.Style = StyleFormal },
		},
		{
			name:    "invalid JSON returns defaults and ErrCorrupt",
			content: ptr(`{"mode": `),
			want:    func(*Settings) {},
			wantErr: ErrCorrupt,
		},
		{
			name:    "wrong field type returns defaults and ErrCorrupt",
			content: ptr(`{"historyRetentionDays": "thirty"}`),
			want:    func(*Settings) {},
			wantErr: ErrCorrupt,
		},
		{
			name:    "JSON that isn't an object returns ErrCorrupt",
			content: ptr(`[1, 2, 3]`),
			want:    func(*Settings) {},
			wantErr: ErrCorrupt,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			if tt.content != nil {
				writeFile(t, s.Path(), *tt.content)
			}
			got, err := s.Load(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Load() error = %v, want %v", err, tt.wantErr)
			}
			want := Default()
			tt.want(&want)
			if got != want {
				t.Errorf("Load() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestSaveThenLoad(t *testing.T) {
	s := newTestStore(t)
	want := Default()
	want.Mode = ModeOpenAI
	want.Style = StyleFormal
	want.Shortcuts.PushToTalk = "Ctrl+Win"
	want.MicrophoneID = "mic-123"
	want.Sounds = false
	want.HistoryRetentionDays = 7
	want.OnboardingDone = true

	if err := s.Save(context.Background(), want); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestSaveWritesSpecFormat(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(context.Background(), Default()); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	data, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("saved file isn't valid JSON: %v", err)
	}
	keys := []string{
		"version", "mode", "style", "models", "shortcuts", "microphoneId", "sounds",
		"launchAtStartup", "alwaysShowBar", "historyRetentionDays", "telemetry", "onboardingDone",
	}
	for _, k := range keys {
		if _, ok := raw[k]; !ok {
			t.Errorf("saved file is missing %q", k)
		}
	}
	if len(raw) != len(keys) {
		t.Errorf("saved file has %d fields, want %d", len(raw), len(keys))
	}
	if raw["version"] != float64(CurrentVersion) {
		t.Errorf("version = %v, want %d", raw["version"], CurrentVersion)
	}
}

func TestSaveSetsCurrentVersion(t *testing.T) {
	s := newTestStore(t)
	st := Default()
	st.Version = 0
	if err := s.Save(context.Background(), st); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", got.Version, CurrentVersion)
	}
}

func TestSaveRejectsInvalidSettings(t *testing.T) {
	s := newTestStore(t)
	st := Default()
	st.Mode = "gemini"
	err := s.Save(context.Background(), st)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("Save() = %v, want ErrInvalid", err)
	}
	if _, err := os.Stat(s.Path()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("settings file was written despite invalid settings")
	}
}

func TestSaveReplacesFileAndLeavesNoTempFiles(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for _, mode := range []Mode{ModeOpenAI, ModeClaude} {
		st := Default()
		st.Mode = mode
		if err := s.Save(ctx, st); err != nil {
			t.Fatalf("Save(%s) = %v", mode, err)
		}
	}
	got, err := s.Load(ctx)
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}
	if got.Mode != ModeClaude {
		t.Errorf("Mode = %q, want %q", got.Mode, ModeClaude)
	}
	entries, err := os.ReadDir(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "settings.json" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("directory contains %v, want only settings.json", names)
	}
}

func TestSaveFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits don't apply on Windows")
	}
	s := newTestStore(t)
	if err := s.Save(context.Background(), Default()); err != nil {
		t.Fatalf("Save() = %v", err)
	}
	info, err := os.Stat(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 600", perm)
	}
}

func TestCancelledContext(t *testing.T) {
	s := newTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Save(ctx, Default()); !errors.Is(err, context.Canceled) {
		t.Errorf("Save() = %v, want context.Canceled", err)
	}
	if _, err := s.Load(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("Load() = %v, want context.Canceled", err)
	}
}

func TestNewUsesDefaultPath(t *testing.T) {
	want, err := DefaultPath()
	if err != nil {
		t.Skipf("no config directory in this environment: %v", err)
	}
	s, err := New(Options{})
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	if s.Path() != want {
		t.Errorf("Path() = %q, want %q", s.Path(), want)
	}
	if filepath.Base(want) != "settings.json" || filepath.Base(filepath.Dir(want)) != "Voocl" {
		t.Errorf("DefaultPath() = %q, want it to end in Voocl/settings.json", want)
	}
}

func TestMigrate(t *testing.T) {
	// addField returns a migration that records it ran by setting key.
	addField := func(key string) migration {
		return func(raw map[string]any) error {
			raw[key] = true
			return nil
		}
	}
	failing := func(map[string]any) error { return errors.New("boom") }

	tests := []struct {
		name       string
		raw        map[string]any
		migrations []migration
		target     int
		wantKeys   []string
		wantErr    bool
	}{
		{
			name:       "runs every step from version 1",
			raw:        map[string]any{"version": float64(1)},
			migrations: []migration{addField("v2"), addField("v3")},
			target:     3,
			wantKeys:   []string{"v2", "v3"},
		},
		{
			name:       "skips steps already applied",
			raw:        map[string]any{"version": float64(2)},
			migrations: []migration{addField("v2"), addField("v3")},
			target:     3,
			wantKeys:   []string{"v3"},
		},
		{
			name:       "missing version starts at 1",
			raw:        map[string]any{},
			migrations: []migration{addField("v2")},
			target:     2,
			wantKeys:   []string{"v2"},
		},
		{
			name:       "current version runs nothing",
			raw:        map[string]any{"version": float64(3)},
			migrations: []migration{addField("v2"), addField("v3")},
			target:     3,
		},
		{
			name:       "newer version runs nothing",
			raw:        map[string]any{"version": float64(9)},
			migrations: []migration{addField("v2")},
			target:     2,
		},
		{
			name:       "missing migration is an error",
			raw:        map[string]any{"version": float64(1)},
			migrations: []migration{addField("v2")},
			target:     3,
			wantErr:    true,
		},
		{
			name:       "failing migration is an error",
			raw:        map[string]any{"version": float64(1)},
			migrations: []migration{failing},
			target:     2,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrate(tt.raw, tt.migrations, tt.target)
			if tt.wantErr != (err != nil) {
				t.Fatalf("migrate() = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			for _, k := range tt.wantKeys {
				if tt.raw[k] != true {
					t.Errorf("migration for %q didn't run", k)
				}
			}
			ran := len(tt.raw) - 1 // minus "version"
			if ran != len(tt.wantKeys) {
				t.Errorf("%d migrations ran, want %d", ran, len(tt.wantKeys))
			}
		})
	}
}

func TestDefaultMigrationsCoverCurrentVersion(t *testing.T) {
	if got, want := len(defaultMigrations()), CurrentVersion-1; got != want {
		t.Errorf("len(defaultMigrations()) = %d, want %d (one per version step)", got, want)
	}
}

func ptr(s string) *string { return &s }
