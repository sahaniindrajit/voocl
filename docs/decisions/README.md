# Decision records

Short records of significant design decisions: what was decided, why, and what it costs. Add a new file for each new decision; don't rewrite old ones. If a decision is replaced, mark it **Superseded** and link the new record.

| # | Decision | Status |
|---|---|---|
| [0001](0001-go-and-wails.md) | Go core with Wails v3 and a React UI | Accepted |
| [0002](0002-single-provider-modes.md) | One provider mode at a time | Accepted |
| [0003](0003-native-floating-bar.md) | Native floating bar instead of a webview | Accepted |
| [0004](0004-local-storage-and-keys.md) | Local storage and OS keyring for API keys | Accepted |
| [0005](0005-gpl-license.md) | GPL-3.0 license | Accepted |

## Template

```markdown
# NNNN: Title

- **Status:** Proposed | Accepted | Superseded by NNNN
- **Date:** YYYY-MM-DD

## Context
What problem or constraint led to this decision.

## Decision
What we chose.

## Consequences
What we gain, what it costs, and what to watch for.
```
