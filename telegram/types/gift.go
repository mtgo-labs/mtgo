package types

import "github.com/mtgo-labs/mtgo/tg"

type GiftBinder interface {
	BoundShowGift(msgID int32) error
	BoundHideGift(msgID int32) error
	BoundConvertGift(msgID int32) error
	BoundUpgradeGift(msgID int32, keepOriginalDetails bool) error
	BoundTransferGift(msgID int32, toID int64) error
}

type Gift struct {
	ID      int64
	MsgID   int32
	Stars   int64
	Title   string
	Limited bool
	SoldOut bool
	Sticker *DocumentMedia
	binder  GiftBinder
}

func (g *Gift) SetBinder(b GiftBinder) {
	g.binder = b
}

func (g *Gift) Show() error {
	if g.binder == nil {
		return ErrNoBinder
	}
	return g.binder.BoundShowGift(g.MsgID)
}

func (g *Gift) Hide() error {
	if g.binder == nil {
		return ErrNoBinder
	}
	return g.binder.BoundHideGift(g.MsgID)
}

func (g *Gift) Convert() error {
	if g.binder == nil {
		return ErrNoBinder
	}
	return g.binder.BoundConvertGift(g.MsgID)
}

func (g *Gift) Upgrade(keepOriginalDetails bool) error {
	if g.binder == nil {
		return ErrNoBinder
	}
	return g.binder.BoundUpgradeGift(g.MsgID, keepOriginalDetails)
}

func (g *Gift) Transfer(toID int64) error {
	if g.binder == nil {
		return ErrNoBinder
	}
	return g.binder.BoundTransferGift(g.MsgID, toID)
}

func ParseGift(raw tg.StarGiftClass) *Gift {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StarGift:
		g := &Gift{
			ID:      v.ID,
			Stars:   v.Stars,
			Limited: v.Limited,
			SoldOut: v.SoldOut,
		}
		if v.Title != "" {
			g.Title = v.Title
		}
		return g
	}
	return nil
}
