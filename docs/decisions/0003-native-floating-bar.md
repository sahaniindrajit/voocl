# 0003: Native floating bar instead of a webview

- **Status:** Accepted
- **Date:** 2026-11-11

## Context

The floating bar must appear the moment the hotkey is pressed, and the app must stay under 40 MB of RAM while idle. Options considered:

- **A.** A native overlay window drawn by the Go core.
- **B.** A hidden webview window kept alive (adds 40–80 MB permanently).
- **C.** A webview window created on each press (200–500 ms delay before it appears).

## Decision

Option **A**: a native, layered, top-most, non-activating overlay window in `internal/overlay`, implemented per platform.

## Consequences

- No extra memory while idle, and the bar appears instantly.
- The bar never steals focus from the app the user is typing in.
- Its look is implemented separately from the React UI and per platform, so it stays deliberately simple: a pill with a status, a level indicator, and stop/cancel buttons.
