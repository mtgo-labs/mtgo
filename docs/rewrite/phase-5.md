# Phase 5 — Stickers, Stories

Status: complete

## Objective

Complete the sticker domain (set discovery, lifecycle, and personal
collections) and the story domain (archived/pinned listings, the contacts
feed, stealth mode, posting capability, viewer/reaction pages, reporting).

## What changed

### Commit `e429efc` — stickers

`StickerMethods` grows from one operation (`GetCustomEmojiStickers`) to the
full domain in `telegram/stickers.go`:

- Discovery: `SearchStickers` (by emoji), `GetAllStickerSets`,
  `SearchStickerSets` (covered sets), `GetFeaturedStickerSets`,
  `GetStickerSet` (full set with documents and packs). Every hash-gated
  listing returns `nil, nil` when the server reports NotModified.
- Short names: `CheckStickerSetShortName`, `SuggestStickerSetShortName`.
- Lifecycle: `CreateStickerSet` (option struct: masks, custom-emoji,
  text-color, thumbnail, software; userID 0 = self), `AddStickerToSet`,
  `RemoveStickerFromSet`, `ChangeStickerPosition`, `ChangeSticker` (emoji +
  keywords), `ReplaceSticker`, `RenameStickerSet`, `SetStickerSetThumbnail`,
  `DeleteStickerSet`, `InstallStickerSet` (with direct-archive flag),
  `UninstallStickerSet`. Lifecycle results unwrap the full
  `messages.StickerSet` form.
- Personal collections: `GetSavedGIFs` + `SaveGIF`/`RemoveSavedGIF`,
  `GetRecentStickers` + `SaveRecentSticker`/`ClearRecentStickers`
  (attached-sticker lists included), `FavoriteSticker`/`UnfavoriteSticker`.

### Commit `5981433` — stories

`StoryMethods` completes its canonical surface:

- Listings: `GetPinnedStories`, `GetArchivedStories` (both paginated by
  offset story ID).
- Feed: `GetAllStories` returning the parsed `StoriesFeed` — per-peer
  `StoriesFeedEntry` groups, state cursor, has-more, stealth state; the
  NotModified form returns nil.
- Privacy/stealth: `ActivateStealthMode(past, future)`,
  `ToggleStoriesHidden` (per peer), `ToggleAllStoriesHidden` (whole feed);
  `EditStoryPrivacy` already landed in Phase 1.
- Posting capability: `GetRemainingStoryCount`, `GetChatsToSendStories`.
- Sharing and moderation: `ExportStoryLink`, `ReportStory`.
- Engagement pages: `GetStoryViewers` (contacts filter, reaction/forward
  ordering, name search, counters) and `GetStoryReactions` (single-reaction
  filter, forward ordering) as typed paginated results.

Generic wrappers regenerated (172).

## Tests executed

| Gate | Result |
|---|---|
| `go build ./...` / `go vet ./...` | pass |
| `golangci-lint run` (via `make all`) | pass |
| `go test ./...` (full module, 0 failures) | pass |
| `TestNoPromotedMethodCollisions` | pass |
| `go mod tidy` (diff clean) | pass |

21 new constructor-ID-keyed tests across `stickers_test.go` and
`stories_gaps_test.go`, including NotModified paths, the lifecycle unwrap,
GIF save/unsave polarity, feed entry parsing, and viewer-page counters.

## Remaining gaps

- Story albums family (`stories.createAlbum`/`updateAlbum`/`reorderAlbums`/
  `deleteAlbum`/`getAlbums`/`getAlbumStories`), `stories.searchPosts`, and
  `stories.startLive` are schema-supported but deferred — they are new,
  low-traffic surfaces; promote from the raw RPC layer on demand.
- `messages.getAttachedStickers` (sticker sets attached to a media message)
  was left unwrapped; it requires a media-message input type that lands with
  the Phase 7 type normalization.
- Emoji keyword lookup (`messages.getEmojiKeywords*`) deferred with the
  same rationale.

## Commits

- `e429efc` feat(stickers): sticker-set lifecycle, discovery, saved GIFs, recents, favorites
- `5981433` feat(stories): archived and pinned listings, feed, stealth mode, viewers, reactions, reporting

All on `rewrite`; branch verified before every commit; no tags; no
downstream updates.
