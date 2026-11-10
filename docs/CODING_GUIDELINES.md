# Coding guidelines

Conventions for all code in this repository. Tooling enforces what it can (`golangci-lint`, ESLint, Prettier, TypeScript); the rest is checked in review. See [ARCHITECTURE.md](ARCHITECTURE.md) for where code belongs.

## General

- **Clarity over cleverness.** Code is read far more often than it is written.
- **Small units.** Aim for files under ~300 lines and functions under ~50. Split when a unit does more than one thing.
- **Return early** instead of nesting.
- **No dead code.** Delete unused code instead of commenting it out.
- **No TODOs without an issue.** Write `// TODO(#123): …` or don't write it.
- **Measure before optimizing,** but never do work while idle (no polling loops, no timers without a purpose).

## Go

### Structure

- One responsibility per package. Every package has a `doc.go` with a package comment explaining its purpose.
- Constructors take an options struct and return an error when setup can fail:

  ```go
  type Options struct {
      Logger *slog.Logger
      Clock  clock.Clock
  }

  func New(opts Options) (*Recorder, error)
  ```

- No `init()` with side effects. No package-level mutable state. No global logger.
- Dependencies (logger, clock, HTTP client, stores) are passed in, never created deep inside a package. This keeps code testable.
- Define interfaces where they're used, keep them small (1–3 methods), and return concrete types.

### Naming

- Packages: short, lowercase, singular, no underscores (`history`, not `historyStore`).
- No stutter: `history.Store`, not `history.HistoryStore`.
- Errors: `ErrNoMicrophone` for sentinel errors, `…Error` for error types.
- Receivers: one or two letters, consistent within a type.
- Booleans read as statements: `enabled`, `hasKey`, `isRecording`.

### Errors

- Always handle errors. Ignoring one requires a comment explaining why.
- Wrap with context using `%w`: `fmt.Errorf("load settings: %w", err)`. Lowercase, no trailing punctuation.
- Check with `errors.Is` / `errors.As`, never by comparing strings.
- No `panic` for recoverable situations. `main` is the only place allowed to exit on startup failure.
- Errors shown to the user are written for the user ("Your OpenAI key was rejected") and mapped in `services/`, not in core packages.

### Context, concurrency and I/O

- `context.Context` is the first parameter of every function that does I/O, waits, or can be cancelled.
- Every goroutine has an owner that can stop it and waits for it to finish. No fire-and-forget goroutines.
- Prefer passing messages over channels to sharing memory. When a mutex is needed, keep it next to the data it protects.
- Every HTTP request has a timeout and uses the shared client. Retry only on 429 and 5xx, with exponential backoff and a small retry limit. Never retry 4xx.
- Close everything you open (`defer resp.Body.Close()` right after the error check).

### Logging

- Use the injected `*slog.Logger` with structured fields: `logger.Info("transcription done", "provider", p, "ms", d.Milliseconds())`.
- **Never log** audio, transcript or cleaned text, dictionary entries, snippets, or API keys.
- `Debug` for development detail, `Info` for lifecycle events, `Warn` for recoverable problems, `Error` for failures the user will notice.

### Platform code

- OS-specific code goes in `_windows.go`, `_linux.go`, `_darwin.go` files with the same exported API in each.
- Platforms not implemented yet return `ErrUnsupported`.
- Keep raw syscalls in the smallest possible functions, with a comment naming the Win32 API used.

### Testing

- Tests live next to the code (`foo_test.go`) and use table-driven cases with `t.Run`.
- New logic needs tests. Bug fixes start with a failing test.
- Use fakes behind interfaces for OS layers, and `httptest.Server` for providers. Tests never call real APIs or need a microphone.
- Inject `clock.Clock` instead of calling `time.Now()` or `time.Sleep()` in logic that depends on time.
- The session state machine has a test for every state transition.
- Tests must pass with `-race`.

### Formatting and linting

- `gofmt` and `goimports` (local imports grouped last).
- `golangci-lint run` must pass with the repository's `.golangci.yml`.

## TypeScript and React

### Language

- `strict` mode with `noUncheckedIndexedAccess`. No `any`; use `unknown` and narrow it.
- Prefer `type` for object shapes and unions. Use string literal unions instead of `enum`.
- No non-null assertions (`!`) without a comment explaining why the value is guaranteed.

### Files and naming

- Files: `kebab-case.tsx` / `kebab-case.ts`. Components: `PascalCase`. Hooks: `useSomething`.
- One component per file. Pages in `pages/`, shared components in `components/`, shadcn components in `components/ui/` (generated, edited only when customizing them).

### Components

- Function components and hooks only.
- Pages compose components. Components receive data through props and stay presentational where possible.
- **All calls into Go go through hooks in `hooks/`.** Pages and components never import from `bindings/` directly.
- Side effects live in hooks (`useEffect`, event subscriptions), with cleanup for every subscription.
- Show loading and error states for every async operation. Errors go to a `sonner` toast or inline message, never `console` only.

### Styling

- Tailwind utility classes only. No CSS files other than the global stylesheet, no inline `style` props.
- Use shadcn/ui components and theme tokens. Don't add other component libraries.

### Accessibility

- Every input has a label. Every icon-only button has an `aria-label`.
- Everything is reachable and usable with the keyboard, with visible focus.

### Testing and linting

- Unit tests with `vitest` for hooks and utilities, named `*.test.ts(x)` next to the code.
- ESLint and Prettier must pass. `npm run build` must succeed with no type errors.

## Dependencies

- Prefer the Go standard library and existing dependencies.
- A new dependency needs a reason in the PR description: what it saves, its maintenance status, its license (must be GPL-3.0 compatible), and its size.
- No dependencies that bundle a browser engine, run background services, or send data anywhere.
- Pin versions. Wails is pinned to an exact version and upgraded only in a dedicated change.

## Documentation

- Exported Go identifiers have doc comments starting with the identifier name.
- Update [ARCHITECTURE.md](ARCHITECTURE.md) when adding a package or changing how parts connect.
- Record significant design decisions in `docs/decisions/` using the existing format.
- Update [FEATURES.md](FEATURES.md) and [ROADMAP.md](../ROADMAP.md) when scope changes.
