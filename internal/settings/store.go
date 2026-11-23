package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ErrCorrupt is returned by Load when the settings file isn't valid JSON.
// Load still returns usable default settings alongside it.
var ErrCorrupt = errors.New("settings file is corrupt")

// fileName is the settings file name inside the app's config directory.
const fileName = "settings.json"

// appDir is the app's folder inside the user's config directory.
const appDir = "Voocl"

// migration upgrades raw settings by exactly one version.
type migration func(raw map[string]any) error

// defaultMigrations returns the migrations for this build. Entry i upgrades a
// file from version i+1 to i+2, so the list grows with CurrentVersion.
func defaultMigrations() []migration {
	return []migration{}
}

// Options configures a Store.
type Options struct {
	// Path is the settings file path. Empty means DefaultPath().
	Path string
}

// Store reads and writes the settings file.
type Store struct {
	path       string
	migrations []migration
}

// New returns a Store for the settings file at opts.Path, or at DefaultPath()
// when opts.Path is empty.
func New(opts Options) (*Store, error) {
	path := opts.Path
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return nil, err
		}
		path = p
	}
	return &Store{path: path, migrations: defaultMigrations()}, nil
}

// DefaultPath returns the settings file path in the user's config directory
// (%APPDATA%\Voocl\settings.json on Windows).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find config directory: %w", err)
	}
	return filepath.Join(dir, appDir, fileName), nil
}

// Path returns the settings file path.
func (s *Store) Path() string {
	return s.path
}

// Load reads the settings file. A missing file returns the defaults with no
// error. A file that isn't valid JSON returns the defaults and an error
// wrapping ErrCorrupt. Invalid values are replaced with their defaults.
func (s *Store) Load(ctx context.Context) (Settings, error) {
	if err := ctx.Err(); err != nil {
		return Default(), fmt.Errorf("load settings: %w", err)
	}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("load settings: %w", err)
	}
	st, err := decode(data, s.migrations)
	if err != nil {
		return Default(), fmt.Errorf("load settings: %w", err)
	}
	return st, nil
}

// Save validates st and writes it to the settings file. The file is written
// to a temporary file first and then renamed, so a crash never leaves a
// half-written settings file.
func (s *Store) Save(ctx context.Context, st Settings) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	if err := st.Validate(); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	st.Version = CurrentVersion
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("save settings: encode: %w", err)
	}
	data = append(data, '\n')
	if err := writeFileAtomic(s.path, data); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}

// decode parses settings JSON, runs migrations for older versions, and fills
// missing fields with defaults.
func decode(data []byte, migrations []migration) (Settings, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Settings{}, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	if err := migrate(raw, migrations, CurrentVersion); err != nil {
		return Settings{}, err
	}
	migrated, err := json.Marshal(raw)
	if err != nil {
		return Settings{}, fmt.Errorf("re-encode settings: %w", err)
	}
	// Unmarshalling onto the defaults keeps the default for every missing field.
	st := Default()
	if err := json.Unmarshal(migrated, &st); err != nil {
		return Settings{}, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	return st.normalize(), nil
}

// migrate upgrades raw in place from its version to target. Files without a
// version are treated as version 1. Files from a newer build are left as they
// are, and their known fields are read as usual.
func migrate(raw map[string]any, migrations []migration, target int) error {
	version := 1
	if v, ok := raw["version"].(float64); ok && v >= 1 {
		version = int(v)
	}
	for version < target {
		i := version - 1
		if i >= len(migrations) {
			return fmt.Errorf("no migration from settings version %d", version)
		}
		if err := migrations[i](raw); err != nil {
			return fmt.Errorf("migrate settings from version %d: %w", version, err)
		}
		version++
	}
	raw["version"] = float64(version)
	return nil
}

// writeFileAtomic writes data to a temporary file next to path and renames it
// over path.
func writeFileAtomic(path string, data []byte) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, fileName+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		if err != nil {
			// Best effort: the temp file is useless once the write failed.
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() // The write error is the one worth reporting.
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close() // The sync error is the one worth reporting.
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replace settings file: %w", err)
	}
	return nil
}
