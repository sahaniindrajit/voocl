// Package settings loads and saves the user's settings file.
//
// Settings live in a versioned JSON file (settings.json in the user's config
// directory). Missing fields take defaults, unknown fields are ignored, and
// the version field drives migrations. API keys are never stored here; they
// belong in the OS keyring (see internal/keystore).
package settings
