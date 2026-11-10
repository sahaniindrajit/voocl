# Contributing to Voocl

Thanks for helping out. This guide covers setup and the workflow for changes.

Before writing code, read:

- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): how the app is structured
- [docs/CODING_GUIDELINES.md](docs/CODING_GUIDELINES.md): conventions
- [docs/FEATURES.md](docs/FEATURES.md) and [ROADMAP.md](ROADMAP.md): what's planned

## Setup

**Requirements:** Windows 10 (1809+) or 11, Go 1.26+, Node.js 22+, Git.

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

git clone https://github.com/sahaniindrajit/voocl
cd voocl
wails3 doctor
cd frontend && npm ci && cd ..
wails3 dev
```

## Checks

Run these before opening a pull request. CI runs the same checks.

```bash
go vet ./...
golangci-lint run
go test -race ./...

cd frontend
npm run lint
npm run format:check
npm run typecheck
npm test
npm run build
```

## Workflow

1. Open or pick an issue. For anything bigger than a small fix, agree on the approach in the issue first.
2. Create a branch from `main`:
   - `feat/<short-name>` for features
   - `fix/<short-name>` for bug fixes
   - `chore/<short-name>` for tooling, docs and refactors
3. Keep changes small and focused. One concern per pull request.
4. Add or update tests for every change in behaviour.
5. Update docs when you change setup, commands, architecture or scope.

## Commits

- Short, direct, imperative subject line: `Add hotkey service`, `Fix clipboard restore delay`.
- No extra explanation in the message body unless something is genuinely non-obvious.
- No co-author trailers or tool attribution.

## Pull requests

- Describe what changed in a few lines and link the issue.
- Fill in the checklist in the pull request template.
- CI must pass before review.
- Changes that may affect idle RAM or dictation latency include before/after measurements.

## Reporting bugs

Include your Windows version, Voocl version, provider mode, and steps to reproduce. Attach logs from `%LOCALAPPDATA%\Voocl\logs\` if relevant. Logs never contain your dictated text or API keys.

## License

By contributing, you agree that your contributions are licensed under [GPL-3.0](LICENSE).
