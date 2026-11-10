# Features

The full feature set Voocl aims for, based on what Wispr Flow and Typeless offer for dictation, so people can switch without losing anything they rely on.

**Target legend**

| Target | Meaning |
|---|---|
| **v1** | Ships in the first public Windows release (M1–M4) |
| **Win+** | Windows, after v1 and before Linux/macOS work starts (M5) |
| **Later** | Wanted, not scheduled yet |
| **Skip** | Out of scope |

**Provider modes:** features marked 🧠 need an LLM, so they work in **OpenAI** and **Claude** modes. **Local** mode always outputs raw transcription.

## 1. Recording and hotkeys

| # | Feature | Target | Notes |
|---|---|---|---|
| 1 | Push-to-talk: hold, speak, release | **v1** | Default `Ctrl+Alt` |
| 2 | Hands-free: press to start, press to stop | **v1** | Default `Ctrl+Alt+Space`, 5-minute cap |
| 3 | Lock into hands-free by pressing `Space` while holding push-to-talk | **v1** | |
| 4 | Cancel with `Esc`, nothing gets pasted | **v1** | |
| 5 | Ignore accidental taps shorter than 0.5 s | **v1** | |
| 6 | Customizable shortcuts | **v1** | Warn when a combination is already taken |
| 7 | "Wispr Flow" shortcut preset | **v1** | `Ctrl+Win`, `Ctrl+Win+Space`, `Alt+Shift+Z` |
| 8 | Multiple shortcuts per action, mouse buttons as triggers | Win+ | |
| 9 | Say "press enter" / remap Enter to send after pasting | Win+ | |
| 10 | Whisper mode (quiet speech) | Later | Depends on model and mic |
| 11 | Wake word | Skip | |

## 2. Inserting text

| # | Feature | Target | Notes |
|---|---|---|---|
| 12 | Paste automatically into the focused field of any app | **v1** | Clipboard is restored afterwards |
| 13 | Paste last transcript shortcut | **v1** | For terminals, WSL, remote sessions, admin windows |
| 14 | Copy last transcript | **v1** | |
| 15 | Clipboard fallback + notice when pasting fails | **v1** | A dictation is never lost |

## 3. Auto-editing (cleanup) 🧠

| # | Feature | Target | Notes |
|---|---|---|---|
| 16 | Remove filler words (um, uh, you know) | **v1** | |
| 17 | Remove repeated words | **v1** | |
| 18 | Apply self-corrections ("at 3, no, 4" → "at 4") | **v1** | |
| 19 | Automatic punctuation + spoken punctuation ("comma") | **v1** | |
| 20 | Spoken lists → formatted lists | **v1** | |
| 21 | Smart formatting of numbers, emails, phone numbers | **v1** | |
| 22 | Developer formatting: camelCase, snake_case, CLI commands, code terms | Win+ | |
| 23 | File tagging in AI code editors | Later | |

## 4. Personalization

| # | Feature | Target | Notes |
|---|---|---|---|
| 24 | Styles: Raw / Clean / Formal, one setting for all apps | **v1** | Default Clean 🧠 |
| 25 | Different style per app (casual in chat, formal in email) | Win+ | Needs active-app detection 🧠 |
| 26 | Personal dictionary: custom words | **v1** | Fed to transcription and cleanup |
| 27 | Snippets / text replacements | **v1** | Applied exactly, not by the LLM |
| 28 | Learn dictionary words from the user's corrections | Later | |
| 29 | Spell names using on-screen context | Later | Privacy trade-off |
| 30 | Learn the user's writing style | Later | |

## 5. AI actions 🧠

| # | Feature | Target | Notes |
|---|---|---|---|
| 31 | Edit selected text by voice ("make this shorter") | Win+ | Wispr's Command Mode / Typeless's Speak to edit |
| 32 | Translation mode: speak in one language, paste in another | Win+ | |
| 33 | Ask questions / web search / open websites | Skip | Outside dictation |
| 34 | Scratchpad / notes | Skip | |

## 6. App, history and settings

| # | Feature | Target | Notes |
|---|---|---|---|
| 35 | Floating bar: status, stop and cancel buttons | **v1** | Native overlay, no webview |
| 36 | Tray menu: open, paste last, microphone, mode, quit | **v1** | |
| 37 | History: search, copy, paste again | **v1** | Text only |
| 38 | History retention setting | **v1** | Default 30 days |
| 39 | Retry a failed transcription | **v1** | Last failed recording kept in memory |
| 40 | Usage: words, words per minute, time saved, cost per dictation | **v1** | Cost shown in the app, never as a popup |
| 41 | Microphone selection | **v1** | |
| 42 | Start/stop sounds | **v1** | Can be switched off |
| 43 | Launch at startup, show/hide floating bar | **v1** | |
| 44 | First-run setup with a test dictation | **v1** | |
| 45 | Anonymous install count | **v1** | Can be switched off, documented |
| 46 | Mute music or media while dictating | Win+ | |
| 47 | Notification settings by category | Later | |
| 48 | Languages beyond English, auto-detect | Later | v1 is English only |

## 7. Not relevant to Voocl

Cross-device sync, mobile apps, team dictionaries/snippets/dashboards, accounts and SSO, meeting notetaker, enterprise compliance plans.
