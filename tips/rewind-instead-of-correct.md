---
id: rewind-instead-of-correct
trigger: tool_failures_recent >= 3 && turns > 5
priority: 85
cooldown_turns: 10
---

several tool errors — try `/rewind` (esc esc) to a point after the relevant reads instead of correcting in place.
