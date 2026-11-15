# Phase 3 — Users, Contacts, Account, Auth+Password

Status: complete

## Objective

Complete the identity domains: profile-state operations (emoji status,
birthday, username ordering, contact notes), the terms-of-service pair, the
canonical phone-code flow (send, resend, phone-number change), cloud-password
reset, and the logout-surface dedupe.

## What changed

### Commit `faf8b53` — profile state and terms of service

- `UserMethods`: `SetEmojiStatus` (custom emoji with optional expiry; 0
  clears), `SetBirthday` (set or hide; year optional), `ReorderActiveUsernames`,
  `ToggleUsername`.
- `AccountMethods`: `GetTermsOfServiceUpdate` (nil when up to date) and
  `AcceptTermsOfService` (takes the pending revision object; nil-guarded).
- `ChatMethods`: `SetChatEmojiStatus` (channel variant of the status family).

### Commit `7d9346f` — canonical phone-code flow (breaking)

- `SendCode` renamed `SendPhoneNumberCode`; sent-code parsing factored into
  `parseSentCode` shared by all three code requests. `SendCodeResult` keeps its
  name (outcome type, not a method).
- New `ResendPhoneNumberCode(ctx, phone, hash, reason)`,
  `SendChangePhoneNumberCode(ctx, phone)`, and
  `ChangePhoneNumber(ctx, phone, hash, code)` returning the updated user.
- `PasswordMethods.ResetCloudPassword` with parsed
  `ResetCloudPasswordResult` (immediate OK / scheduled until / retry-limited).
- Logout dedupe: the local-disconnect alias `LogOut()` is removed —
  `SignOut(ctx)` revokes server-side, `Disconnect()` closes locally. Both test
  call sites migrated; the duplicated not-connected test merged into the
  existing `TestDisconnectNotConnected`.

### Commit `aa408e9` — contact notes

- `ContactsMethods.SetContactNote(ctx, userID, note)` over
  `contacts.updateContactNote` (initially misread as schema-absent; the
  constructor exists as `updateContactNote`, not `setContactNote`).

Generic wrappers regenerated (159); the new surface adds no peer-typed
wrappers beyond `SetChatEmojiStatus`.

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |
| `go test -race ./telegram/` (8 consecutive runs) | pass |

16 new constructor-ID-keyed tests across `profile_account_test.go` and
`auth_phone_test.go`; all rename-affected auth tests updated.

## Known issue (pre-existing, not introduced here)

An intermittent non-race test failure was observed in `./telegram/` at roughly
2-in-25 runs during this phase. It reproduced once on the stashed pre-phase
tree, is not detectable under `-race` (8 clean runs), and 10 verbose reruns
plus every gated `make all` run passed. Tracked as a flake to investigate
independently; no phase-3 test was ever implicated.

## Remaining gaps

- Schema-gated: `account.getBirthday` (own-birthday read) is absent from the
  pinned TL layer — only `contacts.getBirthdays` (listing contacts' birthdays)
  exists and remains unwrapped.
- `contacts.getContactNote` does not exist by design; notes are returned with
  contact info.
- Cloud-password verify-as-operation (SRP check without changing settings) is
  exercised through `CheckPassword` in the auth flow; a standalone
  `VerifyCloudPassword` was not added because every current caller wants either
  sign-in or a settings change.

## Commits

- `faf8b53` feat(account,user,chats): emoji status, birthday, username order, terms of service
- `7d9346f` refactor(auth,password)!: canonical phone-code flow and password reset
- `aa408e9` feat(contacts): contact notes

All on `rewrite`; branch verified before every commit; no tags; no downstream
updates.
