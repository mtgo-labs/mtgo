# mtgo bug report: update dispatch dies with the session — re-created session answers pings but never receives updates

> **RESOLVED upstream** by `f01cd02` — *fix(telegram): restore update
> subscription after server-side session replacement* (mtgo PR #92, merged
> to `main` @ `abc7c0a`, 2026-11-04; spybot pinned there). Root cause: the
> server pushes updates only to connections that ran `initConnection` plus
> an API method; after a server-side session replacement (`new_session_created`)
> mtgo's `apiInit` flag stayed set, so the new session was never re-identified
> and never received another update. The fix resets `apiInit` and re-identifies
> via an initConnection-wrapped `updates.getState` on `new_session_created`,
> after every reconnect, and on a new 15-min update-silence watchdog
> (`Config.UpdateSilenceTimeout`). spybot keeps its own `SPYBOT_WATCHDOG`
> (20 m, exit-for-restart) as the second line of defense.

**Project:** github.com/mtgo-labs/mtgo
**Versions:** `v0.21.1-0.20261003144258-a00ac798a03c` and `v0.24.0` (spybot
production pins)
**Area:** session lifecycle × update dispatch (client reconnect path)
**Impact:** the client stops receiving every update (business messages,
commands, connection updates) while the process, TCP connection and ping/pong
loop stay healthy. Observed deaf windows lasted 15–52 hours in production
because nothing failed loudly; a manual restart (sometimes two) revived it.

---

## Summary

Every time the main session died in production, the re-created session never
received another update. The connection answers pings (no follow-up pong
timeouts for the whole 52 h window), TCP stays ESTABLISHED to DC1, RPC-level
errors are absent, and other mtgo clients on the same host keep receiving
updates throughout — the wedge is specific to the client whose session died.
Restarting the process usually revives intake within seconds; on 2026-11-04 a
fresh boot on v0.24.0 stayed deaf for ≥9 minutes *after* the session death
that preceded it, then a second restart revived it in <30 s.

## Evidence (spybot production host, 29 business connections, ~500 updates/h)

| Window (local) | Session death | Deaf for | Revived by |
|---|---|---|---|
| 2026-10-26 16:59 → 10-27 10:15 | `readLoop: read deadline exceeded` (during Telegram `-500 No workers running` DC trouble) | ~17 h | manual restart |
| 2026-10-28 19:10 → 10-29 10:53 | `pingLoop: pong timeout` (age 32 h) | ~15.7 h | manual restart |
| 2026-11-01 12:54 → 11-03 16:41 | `pingLoop: pong timeout` (age 9 h) | ~52 h | manual restart |
| 2026-11-04 16:37 → 17:00 | silent `new_session_created` (no shutdown-cause ERROR logged) | ≥24 min | two restarts (first boot on v0.24.0 deaf ≥9 min, second alive <30 s) |

Per-window facts:

- Zero dispatched updates for the whole window (spybot writes a snapshot row
  per business message; hourly counts drop from hundreds to exactly 0, across
  all 29 connections simultaneously).
- **No further `pong timeout`s** during the window — the ping loop is being
  answered, so the transport and session are alive server-side.
- `ss -tnp`: ESTABLISHED to `149.154.175.53:443` (DC1) throughout.
- Host clock within ~1 s of real time; `systemd-timesyncd` synchronized —
  rules out the v0.23.0 msg_id-validator clock wedge.
- Another mtgo client on the same host (user-account bot, v0.23.0) received
  and processed updates continuously through the 2026-11-04 window.
- Every reconnect logs `new_session_created … — triggering gap recovery`, but
  the deaf hours never backfill — updates missed while deaf are not recovered
  by gap recovery either.

## Suspicion

After a session death the client rebuilds the MTProto session, but something
in the update pipeline (dispatcher registration, `updates.getState` /
difference seeding, or the push subscription implied by it) does not follow
the new session, so the server never (re)starts pushing updates to it. The
wedge survives a client restart only when the new process inherits the same
wedged server-side state (same auth key + persisted session id/salt), which
matches the 2026-11-04 sequence where the first restart stayed deaf and a
second (new session id) revived.

## Reproduction sketch

Hard to reproduce deterministically — needs a server-side session death. In
production it fired 4 times in 9 days (~1 per 2 days, always after
`pong timeout` / `read deadline` / a silent `new_session_created`). Killing
the TCP connection (e.g. `ss -K` to the DC endpoint) to force the reconnect
path while watching whether update dispatch resumes is the closest controlled
trigger we can offer from the client side.
