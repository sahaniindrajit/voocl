# 0004: Local storage and OS keyring for API keys

- **Status:** Accepted
- **Date:** 2026-11-11

## Context

Voocl has no accounts and no cloud backend. Users need settings, history, a dictionary, snippets and usage data to persist, and API keys must be stored securely.

## Decision

- Settings in a versioned JSON file with migrations.
- History, dictionary, snippets and usage in a local SQLite database using a pure-Go driver, with migrations.
- API keys only in the OS keyring (Windows Credential Manager, macOS Keychain, Linux Secret Service).
- History stores text only, never audio, and is deleted after the retention period (default 30 days).

## Consequences

- Nothing leaves the user's machine except requests to the provider they chose and the optional anonymous install ping.
- No sync between devices.
- A pure-Go SQLite driver avoids another cgo dependency.
