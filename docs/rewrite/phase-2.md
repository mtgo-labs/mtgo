# Phase 2 — Chats, Invite Links, Folders

Status: complete

## Objective

Complete the folder domain: introduce the `FolderMethods` group with canonical
folder CRUD and the shareable-folder (chatlist) invite-link suite, migrate the
`Folder` bound methods off the last `Bound*` bridge, and fill the
schema-supported chat-domain gaps (similar channels, accent colors, top chats,
notification settings).

## What changed

### Commit `2e85db7` — folder domain

- New `FolderMethods` group (struct, `Client` embedding, collision-map entry,
  `initMethodGroups` wiring) — the twentieth domain group.
- `telegram/folders.go` now carries the full folder domain:
  - `GetFolders` — parsed `*types.Folder` list, binder-wired, system entries
    (default All-Chats, chatlist filters) skipped.
  - `CreateFolder`, `EditFolder`, `DeleteFolder`, `ReorderFolders` — dialog
    filter CRUD.
  - `IncludeChat`, `ExcludeChat`, `UpdateFolderColor`, `PinChatInFolder`,
    `RemoveChatFromFolder` — folder membership, behavior-identical to the
    former bridge (fetch → mutate → save, `ErrFolderNotFound` preserved).
  - Shareable-folder invite links: `GetFolderInviteLinks`,
    `CreateFolderInviteLink`, `EditFolderInviteLink`,
    `DeleteFolderInviteLink`, `CheckFolderInviteLink`,
    `JoinFolderInviteLink`, `GetFolderUpdates`, `JoinFolderUpdates`,
    `HideFolderUpdates`, `LeaveFolder`.
- Peer-slice helpers (`removePeerFromSlice`, `peersEqual`) moved from the
  bridge file to the domain file ahead of the bridge deletion.
- 14 constructor-ID-keyed mock tests in `telegram/folders_test.go`.

### Commit `cca8d3f` — bridge deletion

- `FolderBinder` members renamed to the canonical operation names
  (`DeleteFolder`, `EditFolder`, `IncludeChat`, `ExcludeChat`,
  `PinChatInFolder`, `RemoveChatFromFolder`, `UpdateFolderColor`); the
  interface is now satisfied directly by `*FolderMethods` and by `*Client`
  through promotion.
- `telegram/bound_folders.go` deleted. **No `Bound*` surface remains anywhere
  in the module** (170 → 0); `telegram/bound.go` holds only two shared parse
  helpers.

### Commit `2fa0237` — chat-domain gaps

- `GetSimilarChannels(ctx, chatID)` — channel-scoped recommendations; chatID 0
  selects server-wide recommendations derived from the user's subscriptions.
- `SetChatAccentColor(ctx, chatID, color, backgroundEmojiID, forProfile)` —
  channel palette, background emoji, and profile variant.
- `GetTopChats(ctx, category, limit)` with a `TopChatCategory` enum (people,
  bots, inline bots, groups, channels, phone calls, forward recipients,
  forward chats, mini apps) over the server's peer ratings.
- `GetChatNotificationSettings` / `SetChatNotificationSettings` — the general
  notification-settings pair; `MuteChat`/`UnmuteChat` remain as the common
  convenience forms.
- 9 mock tests in `telegram/chats_gaps_test.go`.
- Generic wrappers regenerated after each feature commit (154 → 158).

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module) | pass |
| `TestNoPromotedMethodCollisions` (20 groups) | pass |
| `go mod tidy` (diff clean) | pass |

Bench gate not re-run: the phase adds new surface and swaps cold-path folder
calls; no existing hot path changed. Phase 0's bench comparison remains the
reference for the restructure.

## Remaining gaps

- Schema-gated (constructors absent from the pinned TL layer 232):
  `messages.getPersonalChannels` and `channels.updateParticipantSortableTag`
  are not in `compiler/source/api.tl`; implementing them requires a schema
  update and regeneration via `cmd/tlgen`.
- Compiler-gated: `chatlists.getLeaveChatlistSuggestions` returns a bare
  `Vector<Peer>`, which the generator types as `TLObject` and cannot decode;
  needs generator support for bare vector results before a high-level
  operation can wrap it.
- `Folder` bound set is complete for the manageable surface; shareable-folder
  bound methods (join/leave from a parsed `FolderInviteLink`) can be added
  once a chatlist-parsed folder type exists (deferred with Phase 7 types
  normalization).
- Pre-existing external-converter references (session-string interop tests,
  config and peers docs) predate the rewrite and are untouched by this phase
  (verified: zero occurrences in the phase diff).

## Commits

- `2e85db7` feat(folders): folder CRUD, membership, and shareable-folder invite links
- `cca8d3f` refactor(types,telegram)!: delete the folder Bound* bridge
- `2fa0237` feat(chats): similar channels, accent colors, top chats, notification settings

All on `rewrite`; branch verified before every commit; no tags created (38
pre-existing release tags, none pointing at rewrite commits); no downstream
repository updates.
