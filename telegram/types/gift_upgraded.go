package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// UpgradedGiftValueInfo holds market and valuation data for a unique upgraded star gift.
type UpgradedGiftValueInfo struct {
	Currency                string
	Value                   int64
	IsValueAverage          bool
	InitialSaleDate         time.Time
	InitialSaleStarCount    int64
	InitialSalePrice        int64
	LastSaleDate            time.Time
	LastSalePrice           int64
	IsLastSaleOnFragment    bool
	MinimumPrice            int64
	AverageSalePrice        int64
	TelegramListedGiftCount int32
	FragmentListedGiftCount int32
	FragmentURL             string
}

// ParseUpgradedGiftValueInfo converts a tg.PaymentsUniqueStarGiftValueInfo into an UpgradedGiftValueInfo.
// Returns nil if raw is nil.
func ParseUpgradedGiftValueInfo(raw *tg.PaymentsUniqueStarGiftValueInfo) *UpgradedGiftValueInfo {
	if raw == nil {
		return nil
	}
	info := &UpgradedGiftValueInfo{
		Currency:                raw.Currency,
		Value:                   raw.Value,
		IsValueAverage:          raw.ValueIsAverage,
		InitialSaleStarCount:    raw.InitialSaleStars,
		InitialSalePrice:        raw.InitialSalePrice,
		IsLastSaleOnFragment:    raw.LastSaleOnFragment,
		MinimumPrice:            raw.FloorPrice,
		AverageSalePrice:        raw.AveragePrice,
		TelegramListedGiftCount: raw.ListedCount,
		FragmentListedGiftCount: raw.FragmentListedCount,
		FragmentURL:             raw.FragmentListedURL,
	}
	if raw.InitialSaleDate != 0 {
		info.InitialSaleDate = time.Unix(int64(raw.InitialSaleDate), 0)
	}
	if raw.LastSaleDate != 0 {
		info.LastSaleDate = time.Unix(int64(raw.LastSaleDate), 0)
	}
	return info
}

// UpgradedGiftAttribute describes a single attribute on an upgraded star gift
// such as a model, pattern, or backdrop.
type UpgradedGiftAttribute struct {
	// ID is the numeric identifier of the attribute.
	ID int64
	// Type classifies the attribute as model, pattern, backdrop, etc.
	Type GiftAttributeType
	// Name is the human-readable name of the attribute.
	Name string
	// IconDocID is the document ID of the attribute's icon sticker, or zero.
	IconDocID int64
	// RarityPermille is the rarity of the attribute expressed in permille (‰).
	RarityPermille int32
}

// ParseUpgradedGiftAttribute converts a tg.StarGiftAttributeClass into an UpgradedGiftAttribute.
// Returns nil if attr is nil.
func ParseUpgradedGiftAttribute(attr tg.StarGiftAttributeClass) *UpgradedGiftAttribute {
	if attr == nil {
		return nil
	}
	switch a := attr.(type) {
	case *tg.StarGiftAttributeModel:
		ua := &UpgradedGiftAttribute{
			Type: GiftAttributeTypeModel,
			Name: a.Name,
		}
		if doc, ok := a.Document.(*tg.Document); ok {
			ua.IconDocID = doc.ID
		}
		if r, ok := a.Rarity.(*tg.StarGiftAttributeRarity); ok {
			ua.RarityPermille = r.Permille
		}
		return ua
	case *tg.StarGiftAttributePattern:
		ua := &UpgradedGiftAttribute{
			Type: GiftAttributeTypePattern,
			Name: a.Name,
		}
		if doc, ok := a.Document.(*tg.Document); ok {
			ua.IconDocID = doc.ID
		}
		if r, ok := a.Rarity.(*tg.StarGiftAttributeRarity); ok {
			ua.RarityPermille = r.Permille
		}
		return ua
	case *tg.StarGiftAttributeBackdrop:
		ua := &UpgradedGiftAttribute{
			Type: GiftAttributeTypeBackdrop,
			Name: a.Name,
		}
		if r, ok := a.Rarity.(*tg.StarGiftAttributeRarity); ok {
			ua.RarityPermille = r.Permille
		}
		return ua
	}
	return nil
}

// UpgradedGiftOriginalDetails records the original sender and recipient of an upgraded gift.
type UpgradedGiftOriginalDetails struct {
	Sender   *Chat
	Receiver *Chat
	Text     *FormattedText
	Date     time.Time
}

// ParseUpgradedGiftOriginalDetails converts a tg.StarGiftAttributeOriginalDetails into
// an UpgradedGiftOriginalDetails. Returns nil if raw is nil.
func ParseUpgradedGiftOriginalDetails(raw *tg.StarGiftAttributeOriginalDetails) *UpgradedGiftOriginalDetails {
	if raw == nil {
		return nil
	}
	d := &UpgradedGiftOriginalDetails{}
	if raw.Date != 0 {
		d.Date = time.Unix(int64(raw.Date), 0)
	}
	if raw.SenderID != nil {
		if p, ok := raw.SenderID.(*tg.PeerUser); ok {
			d.Sender = &Chat{ID: p.UserID}
		}
	}
	if raw.RecipientID != nil {
		if p, ok := raw.RecipientID.(*tg.PeerUser); ok {
			d.Receiver = &Chat{ID: p.UserID}
		}
	}
	if raw.Message != nil {
		d.Text = &FormattedText{Text: raw.Message.Text}
		for _, e := range raw.Message.Entities {
			if me := ParseMessageEntity(e); me != nil {
				d.Text.Entities = append(d.Text.Entities, me)
			}
		}
	}
	return d
}

// UpgradedGiftPurchaseOffer describes an offer to purchase an upgraded gift.
type UpgradedGiftPurchaseOffer struct {
	Gift           *Gift
	State          GiftPurchaseOfferState
	Price          *GiftResalePrice
	ExpirationDate time.Time
}
