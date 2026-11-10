# 0001: Go core with Wails v3 and a React UI

- **Status:** Accepted
- **Date:** 2026-11-11

## Context

Voocl must run on Windows, Linux and macOS from one codebase, use little memory, and be maintainable by a small team. The main options were Electron (Node.js), Tauri (Rust), Wails (Go), and fully native apps per platform.

## Decision

Use **Go** for the core and **Wails v3** as the desktop shell, with a **React + TypeScript** UI rendered by the operating system's webview. Wails is pinned to an exact version.

## Consequences

- One codebase for all platforms. OS-specific code is limited to small `_windows.go` / `_linux.go` / `_darwin.go` files.
- Much lower memory than Electron, because no browser engine is bundled. Go's runtime adds a little over Rust, which is small compared with the webview and the local model.
- Audio and local inference use C libraries through cgo, so each platform is built on its own CI runner.
- Wails has a smaller community than Electron or Tauri, so some problems need to be solved without ready-made examples.
