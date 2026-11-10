# Roadmap

Windows comes first and must reach full feature parity (M1–M5) before Linux and macOS work starts. Feature numbers refer to [docs/FEATURES.md](docs/FEATURES.md), behaviour details to [docs/SPEC.md](docs/SPEC.md).

## Open inputs

Things only the maintainer can provide. Work that needs one of these stops until it's available.

| Needed by | Input | Status |
|---|---|---|
| M1 end (and every milestone) | Manual testing on Windows, with feedback as issues | Ongoing |
| M1 | OpenAI API key for the maintainer's own testing (never committed) | Pending |
| M1 | Approval of the floating bar look (screenshot or GIF from the maintainer's test) | Pending |
| M2 | Anthropic API key for testing | Pending |
| M2 | Hugging Face account and mirror repo for the Parakeet model (name, revision) | Pending |
| M4 | App icon and logo (replacing the Wails template icons) | Pending |
| M4 | Domain for the install script and landing page (`voocl.…`) | Pending |
| M4 | Install-ping endpoint: Cloudflare account and Worker URL | Pending |
| M4 | Scoop bucket repository and winget submission | Pending |
| M4 | Decision: landing page now or README only | Pending |
| Later | Code signing (SignPath Foundation application) | Not started |

## M1: Core loop (Windows)

Hold a key, speak, get text pasted.

- [ ] Remove the Wails template demo and its lint/format exclusions (`frontend/eslint.config.js`, `frontend/.prettierignore`)
- [ ] Settings file with defaults and versioning
- [ ] Global hotkeys via low-level keyboard hook: push-to-talk, hands-free, lock, cancel, short-tap ignore (#1–6)
- [ ] Microphone capture, 16 kHz mono, in memory
- [ ] Session state machine with tests
- [ ] OpenAI transcription
- [ ] Paste with clipboard save/restore, paste last, copy last, clipboard fallback (#12–15)
- [ ] Native floating bar overlay (#35)
- [ ] Tray icon and menu (#36)
- [ ] API key stored in Windows Credential Manager

## M2: Cleanup and provider modes

- [ ] Cleanup prompt builder with Raw / Clean / Formal styles (#16–21, #24)
- [ ] OpenAI cleanup
- [ ] Claude mode: Parakeet transcription + Claude Haiku cleanup
- [ ] Local mode: Parakeet transcription, raw output
- [ ] Local model download from the Hugging Face mirror, checksum, lazy load and idle unload
- [ ] Mode switching in settings and tray

## M3: Personalization and the app

- [ ] Personal dictionary (#26)
- [ ] Snippets / text replacements (#27)
- [ ] History with search, copy, paste again, retention (#37–38)
- [ ] Retry failed transcription (#39)
- [ ] Usage and cost screen (#40)
- [ ] Settings: microphone, sounds, launch at startup, floating bar, shortcuts, Wispr preset (#6–7, #41–43)
- [ ] First-run setup (#44)

## M4: Public v1.0 for Windows

- [ ] Install script (`install.ps1`) and uninstall
- [ ] Release workflow: build, checksums, GitHub Release on tag
- [ ] Auto-updater
- [ ] Anonymous install ping, documented, can be switched off (#45)
- [ ] Scoop bucket, then winget
- [ ] User documentation and landing page
- [ ] Performance check against targets: under 40 MB idle, under 1 s cloud, under 2 s local

## M5: Windows parity

- [ ] Edit selected text by voice (#31)
- [ ] Translation mode (#32)
- [ ] Per-app styles (#25)
- [ ] Developer formatting (#22)
- [ ] Multiple shortcuts, mouse buttons, "press enter" (#8–9)
- [ ] Mute media while dictating (#46)

## M6: Linux

- [ ] Hotkeys (X11 + Wayland portals), paste, overlay, tray
- [ ] Secret Service key storage
- [ ] AppImage, .deb, .rpm, Flathub, install script

## M7: macOS

- [ ] Hotkeys, paste via Accessibility, overlay, menu bar
- [ ] Keychain key storage
- [ ] Homebrew tap, install script

## Later

Whisper mode, languages beyond English, auto-learned dictionary, on-screen name context, writing-style learning, notification categories, file tagging, paid cloud provider.
