# Voocl

**Open-source, bring-your-own-key voice dictation.** Hold a hotkey, speak, and clean text is pasted into whatever app you're using. No subscription, no word limits — you pay your AI provider directly, or run everything locally for free.

> **Status:** early development. Windows first, then Linux, then macOS.

## Why Voocl

Tools like Wispr Flow and Typeless are great, but they're subscriptions, and the ones that let you bring your own API key usually put it behind a paid plan. Voocl is free and open source: plug in your own key and pay only for what you actually use.

## Planned features

- **Push-to-talk** and **hands-free** dictation modes
- Works in any app: transcribed text is pasted at your cursor
- **Bring your own key:** OpenAI for transcription, Claude or OpenAI for cleanup
- **Local mode:** on-device transcription (NVIDIA Parakeet), no key or internet needed
- AI cleanup: removes filler words, fixes punctuation, handles self-corrections
- Local history, nothing stored in the cloud
- Lightweight: lives in the tray, no window running when idle

## Roadmap

Windows reaches full feature parity first, then Linux, then macOS. See [ROADMAP.md](ROADMAP.md) and the full feature list in [docs/FEATURES.md](docs/FEATURES.md).

## Tech stack

- [Go](https://go.dev) core
- [Wails v3](https://v3.wails.io) for the desktop shell
- React + TypeScript + [shadcn/ui](https://ui.shadcn.com) for the UI

## Development

**Prerequisites:** Go 1.26+, Node.js 22+, and the Wails CLI:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0
wails3 doctor   # checks your system for missing dependencies
```

**Run in development mode** (hot reload):

```bash
wails3 dev
```

**Build a production binary:**

```bash
wails3 build
```

The output goes to `bin/`.

Before contributing, read [CONTRIBUTING.md](CONTRIBUTING.md), [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and [docs/CODING_GUIDELINES.md](docs/CODING_GUIDELINES.md).

## Privacy

- Your API keys will be stored in your OS's secure credential store.
- Audio and text go only to the provider you choose, or nowhere at all in local mode.
- History is stored locally on your machine.

## License

[GPL-3.0](LICENSE) © Indrajit Sahani
