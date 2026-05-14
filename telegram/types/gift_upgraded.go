package types

import "github.com/mtgo-labs/mtgo/tg"

// UpgradedGiftValueInfo holds market and valuation data for a unique upgraded star gift.
type UpgradedGiftValueInfo struct {
	// GiftID is the unique identifier of the upgraded gift.
	GiftID int64
	// Stars is the estimated value in Telegram Stars.
	Stars int64
}

// ParseUpgradedGiftValueInfo converts a tg.PaymentsUniqueStarGiftValueInfo into an UpgradedGiftValueInfo.
// Returns nil if raw is nil.
func ParseUpgradedGiftValueInfo(raw *tg.PaymentsUniqueStarGiftValueInfo) *UpgradedGiftValueInfo {
	if raw == nil {
		return nil
	}
	return &UpgradedGiftValueInfo{
		Stars: raw.InitialSaleStars,
	}
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
	// SenderID is the user ID of the original sender, or zero if hidden.
	SenderID int64
	// RecipientID is the user ID of the original recipient.
	RecipientID int64
	// Date is the Unix timestamp when the gift was originally sent.
	Date int32
}

// ParseUpgradedGiftOriginalDetails converts a tg.StarGiftAttributeOriginalDetails into
// an UpgradedGiftOriginalDetails. Returns nil if raw is nil.
func ParseUpgradedGiftOriginalDetails(raw *tg.StarGiftAttributeOriginalDetails) *UpgradedGiftOriginalDetails {
	if raw == nil {
		return nil
	}
	d := &UpgradedGiftOriginalDetails{
		Date: raw.Date,
	}
	if raw.SenderID != nil {
		d.SenderID = peerUserID(raw.SenderID)
	}
	if raw.RecipientID != nil {
		d.RecipientID = peerUserID(raw.RecipientID)
	}
	return d
}

// UpgradedGiftPurchaseOffer describes an offer to purchase an upgraded gift.
type UpgradedGiftPurchaseOffer struct {
	// GiftID is the unique identifier of the upgraded gift being offered.
	GiftID int64
	// Stars is the asking price in Telegram Stars.
	Stars int64
	// UntilDate is the Unix timestamp when the offer expires, or zero.
	UntilDate int32
}

// peerUserID extracts a user ID from a PeerClass, returning 0 if not a user peer.
func peerUserID(peer tg.PeerClass) int64 {
	if peer == nil {
		return 0
	}
	if p, ok := peer.(*tg.PeerUser); ok {
		return p.UserID
	}
	return 0
}
