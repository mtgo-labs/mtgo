package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestSearchStickers(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetStickersTypeID, &tg.MessagesStickers{
		Stickers: []tg.DocumentClass{
			&tg.Document{ID: 1, MimeType: "image/webp"},
			&tg.Document{ID: 2, MimeType: "image/webp"},
		},
	})

	stickers, err := c.SearchStickers(context.Background(), "😀", 0)
	if err != nil {
		t.Fatalf("SearchStickers() error: %v", err)
	}
	if len(stickers) != 2 || stickers[0].FileUniqueID != "1" {
		t.Fatalf("stickers = %#v, want two parsed stickers", stickers)
	}
	req, ok := inv.lastCall().(*tg.MessagesGetStickersRequest)
	if !ok || req.Emoticon != "😀" {
		t.Fatalf("request = %#v, want emoticon 😀", inv.lastCall())
	}
}

func TestSearchStickers_NotModified(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetStickersTypeID, &tg.MessagesStickersNotModified{})

	stickers, err := c.SearchStickers(context.Background(), "😀", 42)
	if err != nil {
		t.Fatalf("SearchStickers() error: %v", err)
	}
	if stickers != nil {
		t.Fatalf("stickers = %#v, want nil when not modified", stickers)
	}
}

func TestGetAllStickerSets(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetAllStickersTypeID, &tg.MessagesAllStickers{
		Sets: []tg.StickerSetClass{
			&tg.StickerSet{ID: 7, Title: "Packs", ShortName: "packs"},
			&tg.StickerSet{ID: 8, Title: "More", ShortName: "more"},
		},
	})

	sets, err := c.GetAllStickerSets(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetAllStickerSets() error: %v", err)
	}
	if len(sets) != 2 || sets[0].ShortName != "packs" {
		t.Fatalf("sets = %#v, want two parsed sets", sets)
	}
}

func TestGetStickerSet(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetStickerSetTypeID, &tg.MessagesStickerSet{
		Set:       &tg.StickerSet{ID: 7, Title: "Packs"},
		Documents: []tg.DocumentClass{&tg.Document{ID: 1}},
	})

	set, err := c.GetStickerSet(context.Background(), &tg.InputStickerSetID{ID: 7}, 0)
	if err != nil {
		t.Fatalf("GetStickerSet() error: %v", err)
	}
	if set.Set == nil || len(set.Documents) != 1 {
		t.Fatalf("set = %#v, want full sticker set", set)
	}
}

func TestStickerSetShortNames(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StickersCheckShortNameTypeID, &tg.BoolTrue{})
	inv.setResult(tg.StickersSuggestShortNameTypeID, &tg.StickersSuggestedShortName{ShortName: "MyPack"})

	ok, err := c.CheckStickerSetShortName(context.Background(), "mypack")
	if err != nil || !ok {
		t.Fatalf("CheckStickerSetShortName() = %v err=%v", ok, err)
	}
	suggested, err := c.SuggestStickerSetShortName(context.Background(), "My Pack")
	if err != nil || suggested != "MyPack" {
		t.Fatalf("SuggestStickerSetShortName() = %q err=%v", suggested, err)
	}
}

func TestCreateStickerSet(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.StickersCreateStickerSetTypeID, &tg.MessagesStickerSet{
		Set: &tg.StickerSet{ID: 9, Title: "Mine"},
	})

	items := []*tg.InputStickerSetItem{
		{Document: &tg.InputDocument{ID: 1}, Emoji: "😀"},
	}
	set, err := c.CreateStickerSet(context.Background(), 0, "Mine", "mine", items,
		&CreateStickerSetOption{Software: "mtgo"})
	if err != nil {
		t.Fatalf("CreateStickerSet() error: %v", err)
	}
	if inner, ok := set.Set.(*tg.StickerSet); !ok || inner.Title != "Mine" {
		t.Fatalf("set = %#v, want created set", set.Set)
	}
	req, ok := inv.lastCall().(*tg.StickersCreateStickerSetRequest)
	if !ok {
		t.Fatalf("expected StickersCreateStickerSetRequest, got %T", inv.lastCall())
	}
	if _, self := req.UserID.(*tg.InputUserSelf); !self {
		t.Errorf("user = %#v, want self for userID 0", req.UserID)
	}
	if req.Software != "mtgo" || len(req.Stickers) != 1 {
		t.Errorf("request = %#v, want software tag and one sticker", req)
	}
}

func TestStickerSetLifecycle_UnwrapsFullSet(t *testing.T) {
	c, inv := newClientWithMock(t)
	full := func() *tg.MessagesStickerSet {
		return &tg.MessagesStickerSet{Set: &tg.StickerSet{ID: 7, Title: "Packs"}}
	}
	inv.setResult(tg.StickersAddStickerToSetTypeID, full())
	inv.setResult(tg.StickersRemoveStickerFromSetTypeID, full())
	inv.setResult(tg.StickersChangeStickerPositionTypeID, full())
	inv.setResult(tg.StickersRenameStickerSetTypeID, full())
	inv.setResult(tg.StickersSetStickerSetThumbTypeID, full())

	set := &tg.InputStickerSetShortName{ShortName: "packs"}
	doc := &tg.InputDocument{ID: 1}
	item := &tg.InputStickerSetItem{Document: doc, Emoji: "😀"}

	if _, err := c.AddStickerToSet(context.Background(), set, item); err != nil {
		t.Fatalf("AddStickerToSet() error: %v", err)
	}
	if _, err := c.RemoveStickerFromSet(context.Background(), doc); err != nil {
		t.Fatalf("RemoveStickerFromSet() error: %v", err)
	}
	if _, err := c.ChangeStickerPosition(context.Background(), doc, 0); err != nil {
		t.Fatalf("ChangeStickerPosition() error: %v", err)
	}
	if _, err := c.RenameStickerSet(context.Background(), set, "Renamed"); err != nil {
		t.Fatalf("RenameStickerSet() error: %v", err)
	}
	if _, err := c.SetStickerSetThumbnail(context.Background(), set, doc); err != nil {
		t.Fatalf("SetStickerSetThumbnail() error: %v", err)
	}
	if _, ok := inv.lastCall().(*tg.StickersSetStickerSetThumbRequest); !ok {
		t.Fatalf("expected thumbnail request, got %T", inv.lastCall())
	}
}

func TestInstallUninstallStickerSet(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesInstallStickerSetTypeID, &tg.MessagesStickerSetInstallResultSuccess{})
	inv.setResult(tg.MessagesUninstallStickerSetTypeID, &tg.BoolTrue{})

	set := &tg.InputStickerSetShortName{ShortName: "packs"}
	if err := c.InstallStickerSet(context.Background(), set, false); err != nil {
		t.Fatalf("InstallStickerSet() error: %v", err)
	}
	if err := c.UninstallStickerSet(context.Background(), set); err != nil {
		t.Fatalf("UninstallStickerSet() error: %v", err)
	}
}

func TestSavedGIFs(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetSavedGifsTypeID, &tg.MessagesSavedGifs{
		Gifs: []tg.DocumentClass{&tg.Document{ID: 5, MimeType: "video/mp4"}},
	})
	inv.setResult(tg.MessagesSaveGIFTypeID, &tg.BoolTrue{})

	gifs, err := c.GetSavedGIFs(context.Background(), 0)
	if err != nil {
		t.Fatalf("GetSavedGIFs() error: %v", err)
	}
	if len(gifs) != 1 || gifs[0].ID != 5 {
		t.Fatalf("gifs = %#v, want one parsed gif", gifs)
	}
	doc := &tg.InputDocument{ID: 5}
	if err := c.SaveGIF(context.Background(), doc); err != nil {
		t.Fatalf("SaveGIF() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.MessagesSaveGIFRequest)
	if !ok {
		t.Fatalf("expected MessagesSaveGIFRequest, got %T", inv.lastCall())
	}
	if req.Unsave {
		t.Error("SaveGIF must not set unsave")
	}
	if err := c.RemoveSavedGIF(context.Background(), doc); err != nil {
		t.Fatalf("RemoveSavedGIF() error: %v", err)
	}
	if req, ok := inv.lastCall().(*tg.MessagesSaveGIFRequest); !ok || !req.Unsave {
		t.Fatalf("request = %#v, want unsave set", inv.lastCall())
	}
}

func TestRecentAndFavoriteStickers(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.MessagesGetRecentStickersTypeID, &tg.MessagesRecentStickers{
		Stickers: []tg.DocumentClass{&tg.Document{ID: 3}},
	})
	inv.setResult(tg.MessagesSaveRecentStickerTypeID, &tg.BoolTrue{})
	inv.setResult(tg.MessagesClearRecentStickersTypeID, &tg.BoolTrue{})
	inv.setResult(tg.MessagesFaveStickerTypeID, &tg.BoolTrue{})

	stickers, err := c.GetRecentStickers(context.Background(), false, 0)
	if err != nil {
		t.Fatalf("GetRecentStickers() error: %v", err)
	}
	if len(stickers) != 1 {
		t.Fatalf("stickers = %#v, want one", stickers)
	}
	doc := &tg.InputDocument{ID: 3}
	if err := c.SaveRecentSticker(context.Background(), doc, false, false); err != nil {
		t.Fatalf("SaveRecentSticker() error: %v", err)
	}
	if err := c.ClearRecentStickers(context.Background(), false); err != nil {
		t.Fatalf("ClearRecentStickers() error: %v", err)
	}
	if err := c.FavoriteSticker(context.Background(), doc); err != nil {
		t.Fatalf("FavoriteSticker() error: %v", err)
	}
	if err := c.UnfavoriteSticker(context.Background(), doc); err != nil {
		t.Fatalf("UnfavoriteSticker() error: %v", err)
	}
	if req, ok := inv.lastCall().(*tg.MessagesFaveStickerRequest); !ok || !req.Unfave {
		t.Fatalf("request = %#v, want unfave set", inv.lastCall())
	}
}
