# AGENTS.md

Guidelines for anyone (people or tools) writing code in this repository. Follow them unless a task explicitly says otherwise.

## Read first

| Document | Use it for |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | How the app is structured and where code belongs |
| [docs/CODING_GUIDELINES.md](docs/CODING_GUIDELINES.md) | Full Go and TypeScript conventions |
| [docs/FEATURES.md](docs/FEATURES.md) | Feature scope and targets (v1, Win+, Later, Skip) |
| [ROADMAP.md](ROADMAP.md) | Milestones and what to work on next |
| [docs/decisions/](docs/decisions/) | Why key design choices were made |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Setup, branches, commits, pull requests |

This file is the summary. When it and the documents above disagree, the documents above win; fix this file.

## Project

Voocl is an open-source, bring-your-own-key voice dictation app for desktop. The user holds a hotkey, speaks, and cleaned-up text is pasted into the focused text field of any app.

- **Platforms:** Windows first, then Linux, then macOS.
- **Provider modes (exactly one active at a time):**
  - **OpenAI:** OpenAI transcription + OpenAI cleanup
  - **Claude:** local Parakeet transcription + Claude Haiku cleanup
  - **Local:** local Parakeet transcription, no AI cleanup
- **Recording modes:** push-to-talk (default `Ctrl+Alt`) and hands-free (default `Ctrl+Alt+Space`, 5-minute cap).
- **Cleanup styles:** Raw, Clean (default), Formal.

## Stack

| Layer | Technology |
|---|---|
| Core | Go 1.26+ |
| Desktop shell | Wails v3.0.0 (pinned; do not upgrade without a dedicated change) |
| UI | React + TypeScript + Vite, shadcn/ui + Tailwind |
| Local model | NVIDIA Parakeet (INT8 ONNX) |
| Storage | SQLite for history, JSON for settings, OS keyring for API keys |

## Commands

```bash
wails3 dev                      # run with hot reload
wails3 build                    # production build into bin/
wails3 generate bindings -ts    # regenerate Go → TS bindings

go vet ./...
golangci-lint run
go test -race ./...

cd frontend
npm run lint                    # ESLint
npm run format:check            # Prettier
npm run typecheck
npm test                        # vitest
npm run build
```

Before finishing any change, run all of the checks above and make sure they pass. CI runs the same checks.

## Product principles

These are non-negotiable and take priority over convenience.

1. **Low resource usage.** The app lives in the tray. No webview window stays open while idle. Target: under 40 MB RAM when idle.
2. **Fast.** Target: under 1 s from hotkey release to pasted text with a cloud provider, under 2 s locally.
3. **Private.** Never log, persist or send audio, transcripts or API keys anywhere except the provider the user chose. History stores text only, locally, for 30 days.
4. **Never lose a dictation.** If pasting fails, put the text on the clipboard and notify the user. If transcription fails, keep that one recording so the user can retry.

## Architecture rules

- `main.go` only wires things together: create the app, register services, start the tray. No business logic.
- Core logic lives in `internal/<package>`, one responsibility per package (e.g. `audio`, `hotkey`, `provider`, `cleanup`, `paste`, `history`, `settings`, `keystore`, `localmodel`).
- **Providers sit behind a Go interface.** Adding a provider must not require changes outside its own package plus one registration line.
- **OS-specific code** goes in files with OS suffixes (`paste_windows.go`, `paste_darwin.go`, `paste_linux.go`). Shared code must not call OS APIs directly.
- **Services exposed to the frontend are thin**: validate input, call `internal/` packages, return plain data. No logic in bindings.
- **The frontend only handles presentation.** It calls Go through the generated bindings in `frontend/bindings` and never reimplements core logic, talks to providers, or touches keys.
- The local model is loaded on first use and unloaded after a period of inactivity. Never load it at startup.

## Go conventions

- Format with `gofmt`/`goimports`. Code must pass `go vet` and `golangci-lint`.
- Wrap errors with context: `fmt.Errorf("transcribe audio: %w", err)`. Never ignore errors silently.
- No `panic` outside `main` for recoverable situations.
- Pass `context.Context` as the first argument to anything that does I/O or can be cancelled.
- No package-level mutable state. Pass dependencies explicitly.
- Keep interfaces small and define them where they're used.
- Don't start goroutines without a clear owner and a way to stop them.
- Tests are table-driven and live next to the code (`foo_test.go`). New logic needs tests. Network calls are mocked behind interfaces.

## TypeScript / React conventions

- `strict` mode. No `any`; use `unknown` and narrow it.
- Function components and hooks only.
- Build UI from shadcn/ui components. Don't add other component libraries.
- Keep components small. Side effects live in hooks, not in JSX.
- Use the generated bindings for every call into Go.

## Dependencies

- Prefer the standard library. Add a dependency only when it saves significant, non-trivial work, and note why in the change.
- No dependencies that bundle a browser engine, run background services, or phone home.

## Security

- API keys live only in the OS keyring (Windows Credential Manager, macOS Keychain, Linux Secret Service). Never write them to settings files, logs or the database.
- Never commit secrets, `.env` files or model files.
- Log messages contain no audio, transcript text or keys. Log events and error causes only.

## Do not edit by hand

- `frontend/bindings/`: generated by `wails3 generate bindings`.
- `frontend/dist/`, `bin/`: build output.
- `build/`: only edit when changing packaging, icons or installer behaviour.

## Commits and pull requests

- Small, focused changes. One concern per commit.
- Commit messages: short and direct, imperative mood (`Add hotkey service`). No extra explanation.
- PR descriptions: what changed, in a few lines. No co-author trailers and no tool attribution.
- Update `README.md` and this file when a change alters setup, commands or architecture.
