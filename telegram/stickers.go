package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SearchStickers returns stickers matching an emoji, such as "😀". Pass the
// previous hash to receive an empty result when nothing changed.
func (c *StickerMethods) SearchStickers(ctx context.Context, emoticon string, hash int64) ([]*types.Sticker, error) {
	c.Log.Debugf("SearchStickers %q", emoticon)
	result, err := c.Raw().MessagesGetStickers(ctx, &tg.MessagesGetStickersRequest{
		Emoticon: emoticon,
		Hash:     hash,
	})
	if err != nil {
		return nil, fmt.Errorf("search stickers: %w", err)
	}
	var docs []tg.DocumentClass
	switch v := result.(type) {
	case *tg.MessagesStickers:
		docs = v.Stickers
	case *tg.MessagesStickersNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("search stickers: unexpected result %T", result)
	}
	return parseStickerDocuments(docs), nil
}

// GetAllStickerSets returns every installed sticker set. Pass the previous
// hash to receive an empty result when nothing changed.
func (c *StickerMethods) GetAllStickerSets(ctx context.Context, hash int64) ([]*tg.StickerSet, error) {
	c.Log.Debug("GetAllStickerSets")
	result, err := c.Raw().MessagesGetAllStickers(ctx, &tg.MessagesGetAllStickersRequest{
		Hash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get all sticker sets: %w", err)
	}
	var raw []tg.StickerSetClass
	switch v := result.(type) {
	case *tg.MessagesAllStickers:
		raw = v.Sets
	case *tg.MessagesAllStickersNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("get all sticker sets: unexpected result %T", result)
	}
	sets := make([]*tg.StickerSet, 0, len(raw))
	for _, sc := range raw {
		if set, ok := sc.(*tg.StickerSet); ok {
			sets = append(sets, set)
		}
	}
	return sets, nil
}

// SearchStickerSets searches installed and featured sticker sets by title or
// short name. excludeFeatured limits results to installed sets. Results are
// covered sets: the set description plus cover stickers.
func (c *StickerMethods) SearchStickerSets(ctx context.Context, query string, excludeFeatured bool, hash int64) ([]tg.StickerSetCoveredClass, error) {
	c.Log.Debugf("SearchStickerSets %q", query)
	result, err := c.Raw().MessagesSearchStickerSets(ctx, &tg.MessagesSearchStickerSetsRequest{
		ExcludeFeatured: excludeFeatured,
		Q:               query,
		Hash:            hash,
	})
	if err != nil {
		return nil, fmt.Errorf("search sticker sets: %w", err)
	}
	var sets []tg.StickerSetCoveredClass
	switch v := result.(type) {
	case *tg.MessagesFoundStickerSets:
		sets = v.Sets
	case *tg.MessagesFoundStickerSetsNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("search sticker sets: unexpected result %T", result)
	}
	return sets, nil
}

// GetFeaturedStickerSets returns the sticker sets featured by Telegram, with
// cover stickers for previews.
func (c *StickerMethods) GetFeaturedStickerSets(ctx context.Context, hash int64) ([]tg.StickerSetCoveredClass, error) {
	c.Log.Debug("GetFeaturedStickerSets")
	result, err := c.Raw().MessagesGetFeaturedStickers(ctx, &tg.MessagesGetFeaturedStickersRequest{
		Hash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get featured sticker sets: %w", err)
	}
	var raw []tg.StickerSetCoveredClass
	switch v := result.(type) {
	case *tg.MessagesFeaturedStickers:
		raw = v.Sets
	case *tg.MessagesFeaturedStickersNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("get featured sticker sets: unexpected result %T", result)
	}
	return raw, nil
}

// GetStickerSet fetches one sticker set with all of its stickers and emoji
// packs. Pass the previous hash to receive the cached-empty short form.
func (c *StickerMethods) GetStickerSet(ctx context.Context, set tg.InputStickerSetClass, hash int32) (*tg.MessagesStickerSet, error) {
	c.Log.Debug("GetStickerSet")
	result, err := c.Raw().MessagesGetStickerSet(ctx, &tg.MessagesGetStickerSetRequest{
		Stickerset: set,
		Hash:       hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get sticker set: %w", err)
	}
	return unwrapStickerSet(result)
}

// CheckStickerSetShortName reports whether the short name is available for a
// new sticker set.
func (c *StickerMethods) CheckStickerSetShortName(ctx context.Context, shortName string) (bool, error) {
	c.Log.Debugf("CheckStickerSetShortName %q", shortName)
	ok, err := c.Raw().StickersCheckShortName(ctx, &tg.StickersCheckShortNameRequest{
		ShortName: shortName,
	})
	if err != nil {
		return false, fmt.Errorf("check sticker set short name: %w", err)
	}
	return ok, nil
}

// SuggestStickerSetShortName suggests an available short name derived from a
// sticker set title.
func (c *StickerMethods) SuggestStickerSetShortName(ctx context.Context, title string) (string, error) {
	c.Log.Debugf("SuggestStickerSetShortName %q", title)
	result, err := c.Raw().StickersSuggestShortName(ctx, &tg.StickersSuggestShortNameRequest{
		Title: title,
	})
	if err != nil {
		return "", fmt.Errorf("suggest sticker set short name: %w", err)
	}
	return result.ShortName, nil
}

// CreateStickerSetOption configures optional parameters when creating a
// sticker set with [Client.CreateStickerSet].
type CreateStickerSetOption struct {
	// Masks marks the set as a mask pack instead of a sticker pack.
	Masks bool
	// Emojis marks the set as a custom-emoji pack (bot accounts only).
	Emojis bool
	// TextColor allows arbitrary text colors on custom-emoji packs.
	TextColor bool
	// Thumb is the document used as the set thumbnail.
	Thumb *tg.InputDocument
	// Software names the client software in use.
	Software string
}

// CreateStickerSet creates a new sticker set owned by the given user (0 means
// the logged-in user) from already-uploaded documents.
func (c *StickerMethods) CreateStickerSet(ctx context.Context, userID int64, title, shortName string, stickers []*tg.InputStickerSetItem, opts ...*CreateStickerSetOption) (*tg.MessagesStickerSet, error) {
	c.Log.Debugf("CreateStickerSet user_id=%d title=%q", userID, title)
	user := tg.InputUserClass(&tg.InputUserSelf{})
	if userID != 0 {
		u, err := resolveUserID(ctx, c.Client, userID)
		if err != nil {
			return nil, fmt.Errorf("resolve user: %w", err)
		}
		user = u
	}
	opt := getOptDef(&CreateStickerSetOption{}, opts...)
	result, err := c.Raw().StickersCreateStickerSet(ctx, &tg.StickersCreateStickerSetRequest{
		Masks:     opt.Masks,
		Emojis:    opt.Emojis,
		TextColor: opt.TextColor,
		UserID:    user,
		Title:     title,
		ShortName: shortName,
		Thumb:     opt.Thumb,
		Stickers:  stickers,
		Software:  opt.Software,
	})
	if err != nil {
		return nil, fmt.Errorf("create sticker set: %w", err)
	}
	return unwrapStickerSet(result)
}

// AddStickerToSet appends a sticker to an existing set owned by the logged-in
// user and returns the updated set.
func (c *StickerMethods) AddStickerToSet(ctx context.Context, set tg.InputStickerSetClass, sticker *tg.InputStickerSetItem) (*tg.MessagesStickerSet, error) {
	c.Log.Debug("AddStickerToSet")
	result, err := c.Raw().StickersAddStickerToSet(ctx, &tg.StickersAddStickerToSetRequest{
		Stickerset: set,
		Sticker:    sticker,
	})
	if err != nil {
		return nil, fmt.Errorf("add sticker to set: %w", err)
	}
	return unwrapStickerSet(result)
}

// RemoveStickerFromSet removes a sticker from the set that contains it and
// returns the updated set.
func (c *StickerMethods) RemoveStickerFromSet(ctx context.Context, sticker *tg.InputDocument) (*tg.MessagesStickerSet, error) {
	c.Log.Debug("RemoveStickerFromSet")
	result, err := c.Raw().StickersRemoveStickerFromSet(ctx, &tg.StickersRemoveStickerFromSetRequest{
		Sticker: sticker,
	})
	if err != nil {
		return nil, fmt.Errorf("remove sticker from set: %w", err)
	}
	return unwrapStickerSet(result)
}

// ChangeStickerPosition moves a sticker within its set and returns the
// updated set.
func (c *StickerMethods) ChangeStickerPosition(ctx context.Context, sticker *tg.InputDocument, position int32) (*tg.MessagesStickerSet, error) {
	c.Log.Debugf("ChangeStickerPosition position=%d", position)
	result, err := c.Raw().StickersChangeStickerPosition(ctx, &tg.StickersChangeStickerPositionRequest{
		Sticker:  sticker,
		Position: position,
	})
	if err != nil {
		return nil, fmt.Errorf("change sticker position: %w", err)
	}
	return unwrapStickerSet(result)
}

// ChangeSticker updates the emoji list and search keywords of a sticker and
// returns the updated set. Empty values keep the current ones.
func (c *StickerMethods) ChangeSticker(ctx context.Context, sticker *tg.InputDocument, emoji, keywords string) (*tg.MessagesStickerSet, error) {
	c.Log.Debugf("ChangeSticker emoji=%q", emoji)
	req := &tg.StickersChangeStickerRequest{
		Sticker: sticker,
	}
	if emoji != "" {
		req.Emoji = emoji
	}
	if keywords != "" {
		req.Keywords = keywords
	}
	result, err := c.Raw().StickersChangeSticker(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("change sticker: %w", err)
	}
	return unwrapStickerSet(result)
}

// ReplaceSticker replaces a sticker's document in place and returns the
// updated set.
func (c *StickerMethods) ReplaceSticker(ctx context.Context, sticker *tg.InputDocument, newSticker *tg.InputStickerSetItem) (*tg.MessagesStickerSet, error) {
	c.Log.Debug("ReplaceSticker")
	result, err := c.Raw().StickersReplaceSticker(ctx, &tg.StickersReplaceStickerRequest{
		Sticker:    sticker,
		NewSticker: newSticker,
	})
	if err != nil {
		return nil, fmt.Errorf("replace sticker: %w", err)
	}
	return unwrapStickerSet(result)
}

// RenameStickerSet changes a sticker set's title and returns the updated set.
func (c *StickerMethods) RenameStickerSet(ctx context.Context, set tg.InputStickerSetClass, title string) (*tg.MessagesStickerSet, error) {
	c.Log.Debugf("RenameStickerSet title=%q", title)
	result, err := c.Raw().StickersRenameStickerSet(ctx, &tg.StickersRenameStickerSetRequest{
		Stickerset: set,
		Title:      title,
	})
	if err != nil {
		return nil, fmt.Errorf("rename sticker set: %w", err)
	}
	return unwrapStickerSet(result)
}

// DeleteStickerSet permanently deletes a sticker set owned by the logged-in
// user.
func (c *StickerMethods) DeleteStickerSet(ctx context.Context, set tg.InputStickerSetClass) error {
	c.Log.Debug("DeleteStickerSet")
	_, err := c.Raw().StickersDeleteStickerSet(ctx, &tg.StickersDeleteStickerSetRequest{
		Stickerset: set,
	})
	if err != nil {
		return fmt.Errorf("delete sticker set: %w", err)
	}
	return nil
}

// SetStickerSetThumbnail sets the thumbnail document of a sticker set and
// returns the updated set.
func (c *StickerMethods) SetStickerSetThumbnail(ctx context.Context, set tg.InputStickerSetClass, thumb *tg.InputDocument) (*tg.MessagesStickerSet, error) {
	c.Log.Debug("SetStickerSetThumbnail")
	result, err := c.Raw().StickersSetStickerSetThumb(ctx, &tg.StickersSetStickerSetThumbRequest{
		Stickerset: set,
		Thumb:      thumb,
	})
	if err != nil {
		return nil, fmt.Errorf("set sticker set thumbnail: %w", err)
	}
	return unwrapStickerSet(result)
}

// InstallStickerSet installs a sticker set for the logged-in user. When
// archive is true, the set is installed directly into the archive.
func (c *StickerMethods) InstallStickerSet(ctx context.Context, set tg.InputStickerSetClass, archive bool) error {
	c.Log.Debugf("InstallStickerSet archive=%v", archive)
	_, err := c.Raw().MessagesInstallStickerSet(ctx, &tg.MessagesInstallStickerSetRequest{
		Stickerset: set,
		Archived:   archive,
	})
	if err != nil {
		return fmt.Errorf("install sticker set: %w", err)
	}
	return nil
}

// UninstallStickerSet removes a sticker set from the logged-in user's
// installed sets.
func (c *StickerMethods) UninstallStickerSet(ctx context.Context, set tg.InputStickerSetClass) error {
	c.Log.Debug("UninstallStickerSet")
	_, err := c.Raw().MessagesUninstallStickerSet(ctx, &tg.MessagesUninstallStickerSetRequest{
		Stickerset: set,
	})
	if err != nil {
		return fmt.Errorf("uninstall sticker set: %w", err)
	}
	return nil
}

// GetSavedGIFs returns the GIFs saved by the logged-in user.
func (c *StickerMethods) GetSavedGIFs(ctx context.Context, hash int64) ([]*tg.Document, error) {
	c.Log.Debug("GetSavedGIFs")
	result, err := c.Raw().MessagesGetSavedGifs(ctx, &tg.MessagesGetSavedGifsRequest{
		Hash: hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get saved gifs: %w", err)
	}
	var raw []tg.DocumentClass
	switch v := result.(type) {
	case *tg.MessagesSavedGifs:
		raw = v.Gifs
	case *tg.MessagesSavedGifsNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("get saved gifs: unexpected result %T", result)
	}
	gifs := make([]*tg.Document, 0, len(raw))
	for _, d := range raw {
		if doc, ok := d.(*tg.Document); ok {
			gifs = append(gifs, doc)
		}
	}
	return gifs, nil
}

// SaveGIF saves a GIF for the logged-in user.
func (c *StickerMethods) SaveGIF(ctx context.Context, document *tg.InputDocument) error {
	c.Log.Debug("SaveGIF")
	_, err := c.Raw().MessagesSaveGIF(ctx, &tg.MessagesSaveGIFRequest{
		ID:     document,
		Unsave: false,
	})
	if err != nil {
		return fmt.Errorf("save gif: %w", err)
	}
	return nil
}

// RemoveSavedGIF removes a GIF from the logged-in user's saved GIFs.
func (c *StickerMethods) RemoveSavedGIF(ctx context.Context, document *tg.InputDocument) error {
	c.Log.Debug("RemoveSavedGIF")
	_, err := c.Raw().MessagesSaveGIF(ctx, &tg.MessagesSaveGIFRequest{
		ID:     document,
		Unsave: true,
	})
	if err != nil {
		return fmt.Errorf("remove saved gif: %w", err)
	}
	return nil
}

// GetRecentStickers returns the stickers recently used by the logged-in user;
// attached true lists recently used attached stickers instead.
func (c *StickerMethods) GetRecentStickers(ctx context.Context, attached bool, hash int64) ([]*types.Sticker, error) {
	c.Log.Debugf("GetRecentStickers attached=%v", attached)
	result, err := c.Raw().MessagesGetRecentStickers(ctx, &tg.MessagesGetRecentStickersRequest{
		Attached: attached,
		Hash:     hash,
	})
	if err != nil {
		return nil, fmt.Errorf("get recent stickers: %w", err)
	}
	var docs []tg.DocumentClass
	switch v := result.(type) {
	case *tg.MessagesRecentStickers:
		docs = v.Stickers
	case *tg.MessagesRecentStickersNotModified:
		return nil, nil
	default:
		return nil, fmt.Errorf("get recent stickers: unexpected result %T", result)
	}
	return parseStickerDocuments(docs), nil
}

// SaveRecentSticker adds a sticker to the recent list, or removes it when
// unsave is true; attached targets the attached-stickers list.
func (c *StickerMethods) SaveRecentSticker(ctx context.Context, document *tg.InputDocument, attached, unsave bool) error {
	c.Log.Debugf("SaveRecentSticker attached=%v unsave=%v", attached, unsave)
	_, err := c.Raw().MessagesSaveRecentSticker(ctx, &tg.MessagesSaveRecentStickerRequest{
		Attached: attached,
		ID:       document,
		Unsave:   unsave,
	})
	if err != nil {
		return fmt.Errorf("save recent sticker: %w", err)
	}
	return nil
}

// ClearRecentStickers clears the recent-stickers list; attached true clears
// the attached-stickers list.
func (c *StickerMethods) ClearRecentStickers(ctx context.Context, attached bool) error {
	c.Log.Debugf("ClearRecentStickers attached=%v", attached)
	_, err := c.Raw().MessagesClearRecentStickers(ctx, &tg.MessagesClearRecentStickersRequest{
		Attached: attached,
	})
	if err != nil {
		return fmt.Errorf("clear recent stickers: %w", err)
	}
	return nil
}

// FavoriteSticker adds a sticker to the logged-in user's favorites.
func (c *StickerMethods) FavoriteSticker(ctx context.Context, document *tg.InputDocument) error {
	c.Log.Debug("FavoriteSticker")
	_, err := c.Raw().MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{
		ID:     document,
		Unfave: false,
	})
	if err != nil {
		return fmt.Errorf("favorite sticker: %w", err)
	}
	return nil
}

// UnfavoriteSticker removes a sticker from the logged-in user's favorites.
func (c *StickerMethods) UnfavoriteSticker(ctx context.Context, document *tg.InputDocument) error {
	c.Log.Debug("UnfavoriteSticker")
	_, err := c.Raw().MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{
		ID:     document,
		Unfave: true,
	})
	if err != nil {
		return fmt.Errorf("unfavorite sticker: %w", err)
	}
	return nil
}

// unwrapStickerSet converts a sticker-set result into its full form.
func unwrapStickerSet(result tg.StickerSetClass) (*tg.MessagesStickerSet, error) {
	if set, ok := result.(*tg.MessagesStickerSet); ok {
		return set, nil
	}
	return nil, fmt.Errorf("unexpected sticker set result %T", result)
}

// parseStickerDocuments converts raw documents into parsed stickers, skipping
// non-sticker entries.
func parseStickerDocuments(docs []tg.DocumentClass) []*types.Sticker {
	stickers := make([]*types.Sticker, 0, len(docs))
	for _, d := range docs {
		if doc, ok := d.(*tg.Document); ok {
			if s := types.ParseSticker(doc); s != nil {
				stickers = append(stickers, s)
			}
		}
	}
	return stickers
}
