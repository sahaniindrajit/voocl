# Architecture

This document describes how Voocl is structured and why. Read it before adding a package or moving code. Feature scope lives in [FEATURES.md](FEATURES.md), exact behaviour and defaults in [SPEC.md](SPEC.md), conventions in [CODING_GUIDELINES.md](CODING_GUIDELINES.md), and design decisions in [decisions/](decisions/).

## Overview

Voocl is a single desktop process:

- a **Go core** that does all the work (hotkeys, audio, providers, pasting, storage),
- a **native floating bar** drawn by the core (no webview),
- a **tray icon**, always present,
- a **React UI** in a Wails webview window, created only when the user opens it and destroyed when closed.

While idle, only the Go core and the tray are running.

## Dictation flow

```
 Hotkey                                 Floating bar (Listening… / Processing…)
   │                                                ▲
   ▼                                                │ state events
┌──────────────────────────── Session controller ─────────────────────────────┐
│  Idle → Recording → Transcribing → Cleaning → Pasting → Idle                 │
│  cancel · short-tap ignore · hands-free lock · 5-min cap · retry             │
└──────────────────────────────────────────────────────────────────────────────┘
   │ PCM           │ text             │ text             │ final text
   ▼               ▼                  ▼                  ▼
 Audio  ──►   Transcriber  ──►     Cleaner     ──►   Text rules  ──►  Paster
 capture      OpenAI or         OpenAI, Haiku       snippets and      clipboard save,
 (16 kHz)     Parakeet          or Raw              replacements      Ctrl+V, restore
                                                                         │
                                                                         ▼
                                                           History + usage (SQLite)
```

1. **Hotkey** events (down, up, lock, cancel) are sent to the session controller.
2. **Audio capture** records 16 kHz mono PCM into memory while recording.
3. **Transcriber** turns audio into text. The personal dictionary is passed as a vocabulary hint.
4. **Cleaner** applies the selected style (Raw / Clean / Formal) using the dictionary.
5. **Text rules** apply snippets and replacements deterministically.
6. **Paster** saves the clipboard, writes the text, sends `Ctrl+V`, and restores the clipboard. If pasting fails, the text stays on the clipboard and the user is notified.
7. **History and usage** record the final text, durations, token counts and estimated cost.

## Session controller

`internal/session` owns the dictation lifecycle. It is a state machine running in **one goroutine** that receives events over a channel, so state is never shared across goroutines and no locks are needed.

| State | Entered when | Leaves when |
|---|---|---|
| `Idle` | Startup, or after paste/cancel/error | Hotkey down / hands-free start |
| `Recording` | Hotkey down | Release (push-to-talk), stop (hands-free), `Esc`, 5-minute cap |
| `Transcribing` | Recording stopped with enough audio | Transcript ready, error, cancel |
| `Cleaning` | Transcript ready | Clean text ready, error, cancel |
| `Pasting` | Final text ready | Paste done or fallback used |

Rules:

- A push-to-talk press shorter than 0.5 s returns to `Idle` without sending anything.
- `Space` while holding push-to-talk switches to hands-free.
- `Esc` in any active state cancels and discards everything.
- On transcription failure, the last recording is kept in memory so the user can retry. It is dropped on the next recording or on exit.
- Each state change is published as an event for the overlay and the UI.

## Provider modes

The session depends on two small interfaces, defined in `internal/session`:

```go
type Transcriber interface {
    Transcribe(ctx context.Context, clip audio.Clip, opts TranscribeOptions) (Transcript, error)
}

type Cleaner interface {
    Clean(ctx context.Context, text string, opts CleanOptions) (CleanResult, error)
}
```

Exactly one mode is active at a time:

| Mode | Transcriber | Cleaner |
|---|---|---|
| OpenAI | `provider/openai` | `provider/openai` |
| Claude | `provider/parakeet` | `provider/anthropic` (Claude Haiku) |
| Local | `provider/parakeet` | `cleanup.Raw` (returns input unchanged) |

Modes are assembled in one place (`internal/session/modes.go`). Adding a provider means a new package under `internal/provider/` plus one registration there.

`Transcript` and `CleanResult` carry usage data (audio seconds, input/output tokens, model) so cost can be computed without provider-specific code elsewhere.

## Packages

```
voocl/
├── main.go                 # wiring only
├── internal/
│   ├── session/            # state machine, pipeline orchestration, mode assembly
│   ├── hotkey/             # global shortcuts (hotkey_windows.go: low-level hook)
│   ├── audio/              # microphone capture, device list, WAV encoding
│   ├── provider/
│   │   ├── openai/         # Transcriber + Cleaner
│   │   ├── anthropic/      # Cleaner
│   │   └── parakeet/       # Transcriber via sherpa-onnx
│   ├── cleanup/            # styles, prompt builder, Raw cleaner
│   ├── textrules/          # snippets and replacements
│   ├── paste/              # clipboard + key injection (paste_windows.go)
│   ├── overlay/            # native floating bar (overlay_windows.go)
│   ├── tray/               # tray icon and menu
│   ├── history/            # SQLite store, retention cleanup
│   ├── usage/              # price table, cost and stats
│   ├── settings/           # settings file, defaults, migrations
│   ├── keystore/           # API keys in the OS keyring
│   ├── localmodel/         # model download, checksum, load/unload lifecycle
│   ├── telemetry/          # anonymous install ping
│   └── logging/            # slog setup, rotating file
├── services/               # thin services bound to the UI by Wails
├── frontend/src/
│   ├── pages/              # home, dictionary, snippets, style, usage, settings, onboarding
│   ├── components/ui/      # shadcn/ui components (generated)
│   ├── components/         # app components
│   ├── hooks/              # one hook per service or event stream
│   └── lib/                # small utilities
├── docs/
├── scripts/                # install scripts (M4)
└── .github/workflows/
```

### Dependency rules

- Imports flow one way: `main → services → session → everything else in internal/`.
- `internal/` packages never import `services`, `main` or any Wails package. Core logic can be tested without the GUI.
- Providers never import each other or `session`.
- OS-specific code lives in `_windows.go`, `_linux.go`, `_darwin.go` files inside the package that needs it. Each package exposes the same API on every OS. Unsupported platforms return `ErrUnsupported`.

## Platform layer (Windows)

| Concern | Approach |
|---|---|
| Hotkeys | `SetWindowsHookEx(WH_KEYBOARD_LL)` on a dedicated goroutine with `runtime.LockOSThread` and a message loop. Needed for modifier-only shortcuts such as `Ctrl+Alt`. |
| Paste | Clipboard via Win32 clipboard APIs, `SendInput` for `Ctrl+V`. Previous clipboard content is restored after a short delay. |
| Overlay | A layered, top-most, non-activating window (`WS_EX_LAYERED \| WS_EX_TOPMOST \| WS_EX_NOACTIVATE \| WS_EX_TOOLWINDOW`) that never takes focus from the target app. |
| Elevated windows | Windows blocks input from a normal process into apps running as Administrator. Paste falls back to the clipboard and the limitation is documented. |

## Audio

- Capture with `malgo` (miniaudio) at 16 kHz, mono, 16-bit PCM.
- Audio is kept in memory only and never written to disk.
- Sent to OpenAI as WAV (about 2 MB per minute, under the 25 MB API limit at the 5-minute cap).
- Recordings shorter than 0.5 s or containing only silence are discarded.

## Local model

- NVIDIA Parakeet TDT 0.6B (English, INT8 ONNX), run through `sherpa-onnx`.
- Downloaded on demand from Voocl's Hugging Face mirror, pinned to a commit, verified with SHA-256, with GitHub Releases as a fallback.
- Loaded on first use, unloaded after 5 minutes of inactivity. Never loaded at startup.
- Stored in `%LOCALAPPDATA%\Voocl\models\`.

## Storage

| Data | Location | Format |
|---|---|---|
| Settings | `%APPDATA%\Voocl\settings.json` | JSON with a `version` field and migrations |
| History, dictionary, snippets, usage | `%APPDATA%\Voocl\voocl.db` | SQLite (pure-Go driver) with migrations |
| API keys | Windows Credential Manager | via `go-keyring` |
| Models | `%LOCALAPPDATA%\Voocl\models\` | ONNX files |
| Logs | `%LOCALAPPDATA%\Voocl\logs\` | Rotating text files |

History stores text only, never audio. Entries older than the retention setting (default 30 days) are deleted on startup and once a day.

## UI

- **Tray** menu: Open Voocl, Paste last transcript, Microphone, Mode, Quit.
- **Main window** pages: Home/History, Dictionary, Snippets, Style, Usage, Settings, plus first-run setup.
- The window is created when opened and destroyed when closed.
- **Go is the source of truth.** The UI reads data through generated bindings and refreshes on events from Go:

| Event | Payload |
|---|---|
| `session:state` | Current session state |
| `history:changed` | None (UI refetches) |
| `settings:changed` | None (UI refetches) |
| `model:download` | Download progress |

- Frontend stack: React, TypeScript, Tailwind v4, shadcn/ui, `wouter` for routing, `sonner` for toasts. No global state library; data lives in hooks backed by bindings and events.

## Usage and cost

- `internal/usage` holds a price table per model (per audio minute for transcription, per million tokens for cleanup).
- Each history entry stores audio seconds, tokens, model and estimated cost.
- The Usage page shows totals and per-dictation cost. Local transcription costs $0.
- Prices are estimates and are labeled as such.

## Telemetry

- One anonymous ping on first launch and once a day: random install ID, app version, OS version.
- Never includes audio, text, keys, usage or settings.
- Can be switched off in Settings. Documented in the README.

## Performance targets

| Metric | Target |
|---|---|
| Idle RAM (tray only) | under 40 MB |
| Hotkey release → pasted text (cloud) | under 1 s for a short dictation |
| Hotkey release → pasted text (local) | under 2 s for a short dictation |
| Hotkey press → overlay visible | under 50 ms |
| Startup to tray ready | under 1 s |

Changes that affect these numbers must be measured before merging.

## Security

- API keys only in the OS keyring, never in files, logs or the database.
- Logs record events and error causes, never audio, transcripts or keys.
- All provider calls use HTTPS with timeouts.
- Model files are verified by checksum before use.
