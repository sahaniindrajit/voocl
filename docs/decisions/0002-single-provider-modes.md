# 0002: One provider mode at a time

- **Status:** Accepted
- **Date:** 2026-11-11

## Context

Users bring their own API key. Claude's API cannot transcribe audio, while OpenAI can do both transcription and cleanup. Mixing providers per step adds settings, failure cases and support burden.

## Decision

Exactly one mode is active at a time:

| Mode | Transcription | Cleanup |
|---|---|---|
| OpenAI | OpenAI | OpenAI |
| Claude | Parakeet (local) | Claude Haiku |
| Local | Parakeet (local) | None (raw output) |

The session depends on `Transcriber` and `Cleaner` interfaces, and modes are assembled in one place.

## Consequences

- Simple settings: pick a mode, paste one key (or none).
- Claude mode needs the local model download for transcription.
- Local mode has no Clean or Formal styles until a local cleanup model is added.
- New providers (or a paid cloud provider later) plug in as new implementations of the same interfaces.
