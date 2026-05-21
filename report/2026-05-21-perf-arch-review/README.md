# Performance Architecture Review — 2026-05-21

**Scope:** `internal/` (crypto, session, transport, storage)
**Mode:** Architecture review — 3 parallel agents by concern area
**Status:** Findings identified, no changes applied

## Reports

| File | Concern |
|------|---------|
| [01-allocations.md](01-allocations.md) | Allocation hot paths, memory layout, sync.Pool opportunities |
| [02-io-concurrency.md](02-io-concurrency.md) | I/O buffering, goroutine lifecycle, connection handling |
| [03-algorithms-caching.md](03-algorithms-caching.md) | Data structure choices, repeated computation, caching |
| [04-priority-fix-list.md](04-priority-fix-list.md) | Deduplicated, ordered fix list with estimated impact |

## Summary

- **29 findings** across 3 concern areas
- **7 high-severity** per-message allocation issues in crypto (items 1-7)
- **1 correctness bug** — ws.go bufio.Reader discard causes data corruption
- **3 goroutine lifecycle** issues in session package
- Recommended fix order: stack-allocate crypto temps → fix ws.go bufio → reduce transport allocations → address session lifecycle
