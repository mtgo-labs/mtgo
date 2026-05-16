package types

import (
	"fmt"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// GiftBinder defines the interface for gift-related bound methods that control
// visibility, conversion, upgrading, and transferring of gifts.
//
// Example:
//
//	type MyBinder struct{}
//	func (b *MyBinder) BoundShowGift(msgID int32) error { return nil }
//	// ... implement remaining methods
//	gift.SetBinder(&MyBinder{})
type GiftBinder interface {
	BoundShowGift(msgID int32) error
	BoundHideGift(msgID int32) error
	BoundConvertGift(msgID int32) error
	BoundUpgradeGift(msgID int32, keepOriginalDetails bool) error
	BoundTransferGift(msgID int32, toID int64) error
}

// Gift represents a Telegram star gift with its metadata including unique
// ID, rarity, model, and upgrade information.
//
// Example:
//
//	gifts, _ := client.GetGifts(ctx, nil)
//	for _, g := range gifts {
//	    fmt.Printf("Gift #%d: %s (limit: %d)\n", g.ID, g.Title, g.Limit)
//	}
type Gift struct {
	ID                           int64
	Type                         GiftType
	Origin                       *UpgradedGiftOrigin
	ReceivedGiftID               string
	RegularGiftID                int64
	PublisherChat                *Chat
	Sticker                      *Sticker
	Text                         *FormattedText
	Date                         time.Time
	FirstSaleDate                time.Time
	LastSaleDate                 time.Time
	LockedUntilDate              time.Time
	CraftDate                    time.Time
	Sender                       *Chat
	Receiver                     *Chat
	Host                         *Chat
	Owner                        *Chat
	OwnerAddress                 string
	OwnerName                    string
	GiftAddress                  string
	Title                        string
	Name                         string
	Model                        *GiftAttribute
	Symbol                       *GiftAttribute
	Backdrop                     *GiftAttribute
	OriginalDetails              *UpgradedGiftOriginalDetails
	TotalUpgradedCount           int32
	MaxUpgradedCount             int32
	AvailableResaleCount         int32
	UniqueGiftNumber             int32
	UniqueGiftVariantCount       int32
	StarCount                    int64
	DefaultSellStarCount         int64
	ConvertStarCount             int64
	UpgradeStarCount             int64
	TransferStarCount            int64
	DropOriginalDetailsStarCount int64
	MinimumResellStarCount       int64
	MinimumOfferStarCount        int64
	PrepaidUpgradeStarCount      int64
	PrepaidUpgradeHash           string
	AuctionInfo                  *GiftAuction
	ResaleParameters             *GiftResaleParameters
	UserLimits                   *GiftPurchaseLimit
	OverallLimits                *GiftPurchaseLimit
	ValueCurrency                string
	ValueAmount                  int64
	ValueUSDAmount               int64
	LastResaleCurrency           string
	LastResaleAmount             int64
	NextSendDate                 time.Time
	NextTransferDate             time.Time
	NextResaleDate               time.Time
	ExportDate                   time.Time
	CollectionIDs                []int32
	UsedThemeChatID              int64
	CraftProbabilityPerMille     int32
	HasColors                    bool
	IsAuction                    bool
	IsPrivate                    bool
	IsSaved                      bool
	IsPinned                     bool
	IsLimited                    bool
	IsLimitedPerUser             bool
	IsSoldOut                    bool
	IsBurned                     bool
	IsCrafted                    bool
	IsPremium                    bool
	IsForBirthday                bool
	IsThemeAvailable             bool
	IsUpgradeSeparate            bool
	IsNameHidden                 bool
	CanBeUpgraded                bool
	CanBeTransferred             bool
	CanSendPurchaseOffer         bool
	WasConverted                 bool
	WasUpgraded                  bool
	WasRefunded                  bool
	MsgID                        int32
	Raw                          tg.StarGiftClass
	binder                       GiftBinder
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

// ParseGift parses a star gift from a TL StarGiftClass, handling both regular
// and unique gift types.
//
// Example:
//
//	raw := result.(*tg.StarGift)
//	gift := types.ParseGift(raw)
//	fmt.Println(gift.Title)
func ParseGift(raw tg.StarGiftClass) *Gift {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StarGift:
		g := &Gift{
			ID:                     v.ID,
			Type:                   GiftTypeStarGift,
			StarCount:              v.Stars,
			ConvertStarCount:       v.ConvertStars,
			UpgradeStarCount:       v.UpgradeStars,
			IsLimited:              v.Limited,
			IsSoldOut:              v.SoldOut,
			IsForBirthday:          v.Birthday,
			IsPremium:              v.RequirePremium,
			IsLimitedPerUser:       v.LimitedPerUser,
			HasColors:              v.PeerColorAvailable,
			IsAuction:              v.Auction,
			MaxUpgradedCount:       v.AvailabilityTotal,
			AvailableResaleCount:   int32(v.AvailabilityResale),
			UniqueGiftVariantCount: v.UpgradeVariants,
			Raw:                    raw,
		}
		if v.Title != "" {
			g.Title = v.Title
		}
		if v.FirstSaleDate != 0 {
			g.FirstSaleDate = time.Unix(int64(v.FirstSaleDate), 0)
		}
		if v.LastSaleDate != 0 {
			g.LastSaleDate = time.Unix(int64(v.LastSaleDate), 0)
		}
		if v.LockedUntilDate != 0 {
			g.LockedUntilDate = time.Unix(int64(v.LockedUntilDate), 0)
		}
		if v.ResellMinStars != 0 {
			g.MinimumResellStarCount = v.ResellMinStars
		}
		if v.PerUserTotal != 0 || v.PerUserRemains != 0 {
			g.UserLimits = &GiftPurchaseLimit{
				TotalCount:     v.PerUserTotal,
				RemainingCount: v.PerUserRemains,
			}
		}
		return g
	case *tg.StarGiftUnique:
		return parseUniqueGift(v, raw)
	}
	return nil
}

func parseUniqueGift(v *tg.StarGiftUnique, raw tg.StarGiftClass) *Gift {
	g := &Gift{
		ID:                       v.ID,
		RegularGiftID:            v.GiftID,
		Type:                     GiftTypeStarGiftUnique,
		Title:                    v.Title,
		Name:                     v.Slug,
		UniqueGiftNumber:         v.Num,
		OwnerAddress:             v.OwnerAddress,
		OwnerName:                v.OwnerName,
		GiftAddress:              v.GiftAddress,
		TotalUpgradedCount:       v.AvailabilityIssued,
		MaxUpgradedCount:         v.AvailabilityTotal,
		ValueAmount:              v.ValueAmount,
		ValueCurrency:            v.ValueCurrency,
		ValueUSDAmount:           v.ValueUsdAmount,
		IsPremium:                v.RequirePremium,
		IsThemeAvailable:         v.ThemeAvailable,
		IsBurned:                 v.Burned,
		IsCrafted:                v.Crafted,
		CraftProbabilityPerMille: v.CraftChancePermille,
		MinimumOfferStarCount:    int64(v.OfferMinStars),
		Raw:                      raw,
	}
	for _, attr := range v.Attributes {
		a := ParseGiftAttribute(attr)
		if a == nil {
			continue
		}
		switch a.Type {
		case GiftAttributeTypeModel:
			g.Model = a
		case GiftAttributeTypeSymbol:
			g.Symbol = a
		case GiftAttributeTypeBackdrop:
			g.Backdrop = a
		}
	}
	return g
}

// ParseGiftFromSaved parses a gift from a saved star gift TL type, populating
// message, sender, and date fields using the provided peer map for resolving
// users and chats.
//
// Example:
//
//	saved := result.(*tg.SavedStarGift)
//	gift := types.ParseGiftFromSaved(saved, peerMap)
//	fmt.Printf("From: %v\n", gift.Sender)
func ParseGiftFromSaved(raw *tg.SavedStarGift, pm *PeerMap) *Gift {
	if raw == nil {
		return nil
	}
	g := ParseGift(raw.Gift)
	if g == nil {
		g = &Gift{Raw: raw.Gift}
	}
	g.MsgID = raw.MsgID
	g.IsNameHidden = raw.NameHidden
	g.IsSaved = !raw.Unsaved
	g.WasRefunded = raw.Refunded
	g.CanBeUpgraded = raw.CanUpgrade
	g.IsPinned = raw.PinnedToTop
	g.IsUpgradeSeparate = raw.UpgradeSeparate
	g.ConvertStarCount = raw.ConvertStars
	g.UpgradeStarCount = raw.UpgradeStars
	g.TransferStarCount = raw.TransferStars
	g.PrepaidUpgradeHash = raw.PrepaidUpgradeHash
	g.DropOriginalDetailsStarCount = raw.DropOriginalDetailsStars
	g.UniqueGiftNumber = raw.GiftNum
	g.ReceivedGiftID = fmt.Sprintf("%d", raw.SavedID)
	if raw.Date != 0 {
		g.Date = time.Unix(int64(raw.Date), 0)
	}
	if raw.CanExportAt != 0 {
		g.ExportDate = time.Unix(int64(raw.CanExportAt), 0)
	}
	if raw.CanTransferAt != 0 {
		g.NextTransferDate = time.Unix(int64(raw.CanTransferAt), 0)
	}
	if raw.CanResellAt != 0 {
		g.NextResaleDate = time.Unix(int64(raw.CanResellAt), 0)
	}
	if raw.CanCraftAt != 0 {
		g.CraftDate = time.Unix(int64(raw.CanCraftAt), 0)
	}
	if raw.Message != nil {
		g.Text = &FormattedText{Text: raw.Message.Text}
		for _, e := range raw.Message.Entities {
			if me := ParseMessageEntity(e); me != nil {
				g.Text.Entities = append(g.Text.Entities, me)
			}
		}
	}
	if len(raw.CollectionID) > 0 {
		g.CollectionIDs = raw.CollectionID
	}
	if raw.FromID != nil {
		g.Sender = ParseChatFromPeer(raw.FromID, pm)
	}
	return g
}

// GiftAttribute represents an attribute of a gift such as its model, symbol,
// backdrop, or original details including rarity and visual properties.
//
// Example:
//
//	if gift.Model != nil {
//	    fmt.Printf("Model: %s (rarity: %s)\n", gift.Model.Name, gift.Model.Rarity.Type)
//	}
type GiftAttribute struct {
	Type            GiftAttributeType
	Name            string
	BackdropID      int32
	Rarity          GiftAttributeRarity
	Date            time.Time
	Caption         string
	CaptionEntities []*MessageEntity
	FromUser        *User
	ToUser          *User
	CenterColor     int32
	EdgeColor       int32
	PatternColor    int32
	TextColor       int32
	Sticker         *Sticker
	Crafted         bool
}

// GiftAttributeRarity represents the rarity level of a gift attribute, with
// either a per-mille value or a named rarity tier.
type GiftAttributeRarity struct {
	Type     GiftAttributeRarityType
	Permille int32
}

// GiftAttributeRarityType is the named rarity tier for a gift attribute.
// Valid values include "permille", "uncommon", "rare", "epic", and "legendary".
type GiftAttributeRarityType string

const (
	RarityPerMille  GiftAttributeRarityType = "permille"
	RarityUncommon  GiftAttributeRarityType = "uncommon"
	RarityRare      GiftAttributeRarityType = "rare"
	RarityEpic      GiftAttributeRarityType = "epic"
	RarityLegendary GiftAttributeRarityType = "legendary"
)

// ParseGiftAttribute parses a gift attribute from a TL StarGiftAttributeClass,
// returning model, backdrop, symbol, or original details attributes.
//
// Example:
//
//	for _, raw := range uniqueGift.Attributes {
//	    attr := types.ParseGiftAttribute(raw)
//	    fmt.Println(attr.Name, attr.Type)
//	}
func ParseGiftAttribute(raw tg.StarGiftAttributeClass) *GiftAttribute {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.StarGiftAttributeModel:
		a := &GiftAttribute{
			Type:    GiftAttributeTypeModel,
			Name:    v.Name,
			Crafted: v.Crafted,
		}
		if v.Rarity != nil {
			a.Rarity = parseRarity(v.Rarity)
		}
		if v.Document != nil {
			if doc, ok := v.Document.(*tg.Document); ok {
				a.Sticker = ParseSticker(doc)
			}
		}
		return a
	case *tg.StarGiftAttributeBackdrop:
		a := &GiftAttribute{
			Type:         GiftAttributeTypeBackdrop,
			Name:         v.Name,
			BackdropID:   v.BackdropID,
			CenterColor:  v.CenterColor,
			EdgeColor:    v.EdgeColor,
			PatternColor: v.PatternColor,
			TextColor:    v.TextColor,
		}
		if v.Rarity != nil {
			a.Rarity = parseRarity(v.Rarity)
		}
		return a
	case *tg.StarGiftAttributePattern:
		a := &GiftAttribute{
			Type: GiftAttributeTypeSymbol,
			Name: v.Name,
		}
		if v.Rarity != nil {
			a.Rarity = parseRarity(v.Rarity)
		}
		if v.Document != nil {
			if doc, ok := v.Document.(*tg.Document); ok {
				a.Sticker = ParseSticker(doc)
			}
		}
		return a
	case *tg.StarGiftAttributeOriginalDetails:
		a := &GiftAttribute{
			Type: GiftAttributeTypeOriginalDetails,
		}
		if v.Date != 0 {
			a.Date = time.Unix(int64(v.Date), 0)
		}
		if v.Message != nil {
			a.Caption = v.Message.Text
			a.CaptionEntities = ParseMessageEntities(v.Message.Entities)
		}
		if v.SenderID != nil {
			if p, ok := v.SenderID.(*tg.PeerUser); ok {
				a.FromUser = &User{ID: p.UserID}
			}
		}
		if v.RecipientID != nil {
			if p, ok := v.RecipientID.(*tg.PeerUser); ok {
				a.ToUser = &User{ID: p.UserID}
			}
		}
		return a
	}
	return nil
}

func parseRarity(raw tg.StarGiftAttributeRarityClass) GiftAttributeRarity {
	if raw == nil {
		return GiftAttributeRarity{}
	}
	switch v := raw.(type) {
	case *tg.StarGiftAttributeRarity:
		return GiftAttributeRarity{Type: RarityPerMille, Permille: v.Permille}
	case *tg.StarGiftAttributeRarityUncommon:
		return GiftAttributeRarity{Type: RarityUncommon}
	case *tg.StarGiftAttributeRarityRare:
		return GiftAttributeRarity{Type: RarityRare}
	case *tg.StarGiftAttributeRarityEpic:
		return GiftAttributeRarity{Type: RarityEpic}
	case *tg.StarGiftAttributeRarityLegendary:
		return GiftAttributeRarity{Type: RarityLegendary}
	}
	return GiftAttributeRarity{}
}

// UpgradedGiftAttributeID identifies an attribute of an upgraded gift by its
// type and associated document or backdrop ID.
type UpgradedGiftAttributeID struct {
	Type       string
	DocumentID int64
	BackdropID int32
}

// UpgradedGiftPurchaseOfferRejected indicates that a purchase offer for an
// upgraded gift was rejected.
type UpgradedGiftPurchaseOfferRejected struct{}

// GiftCollection represents a named collection of star gifts with an icon and
// gift count.
//
// Example:
//
//	collections, _ := client.GetGiftCollections(ctx)
//	for _, c := range collections {
//	    fmt.Printf("Collection: %s (%d gifts)\n", c.Name, c.GiftCount)
//	}
type GiftCollection struct {
	ID        int32
	Name      string
	GiftCount int32
	Icon      *Sticker
	Hash      int64
}

// ParseGiftCollection parses a gift collection from a TL StarGiftCollection.
func ParseGiftCollection(raw *tg.StarGiftCollection) *GiftCollection {
	if raw == nil {
		return nil
	}
	c := &GiftCollection{
		ID:        raw.CollectionID,
		Name:      raw.Title,
		GiftCount: raw.GiftsCount,
		Hash:      raw.Hash,
	}
	return c
}

// GiftPurchaseLimit tracks the total and remaining purchase count for a gift,
// either per-user or overall.
type GiftPurchaseLimit struct {
	TotalCount     int32
	RemainingCount int32
}

// GiftResaleParameters holds the pricing parameters for reselling a gift,
// including star count and TON-cent amount.
type GiftResaleParameters struct {
	StarCount        int64
	ToncoinCentCount int64
	ToncoinOnly      bool
}

// GiftResalePrice represents the price of a gift on the resale market in both
// stars and TON cents.
type GiftResalePrice struct {
	StarCount        int64
	ToncoinCentCount int64
}

// GiftUpgradePreview contains sample attributes and pricing for upgrading a
// gift, including available models, symbols, and backdrops.
//
// Example:
//
//	preview := types.ParseGiftUpgradePreview(raw)
//	for _, m := range preview.Models {
//	    fmt.Println("Model:", m.Name)
//	}
type GiftUpgradePreview struct {
	Models     []GiftAttribute
	Symbols    []GiftAttribute
	Backdrops  []GiftAttribute
	Prices     []GiftUpgradePrice
	NextPrices []GiftUpgradePrice
}

// ParseGiftUpgradePreview parses a gift upgrade preview from a TL
// PaymentsStarGiftUpgradePreview, extracting sample attributes and tiered
// pricing.
func ParseGiftUpgradePreview(raw *tg.PaymentsStarGiftUpgradePreview) *GiftUpgradePreview {
	if raw == nil {
		return nil
	}
	p := &GiftUpgradePreview{}
	for _, a := range raw.SampleAttributes {
		if attr := ParseGiftAttribute(a); attr != nil {
			switch attr.Type {
			case GiftAttributeTypeModel:
				p.Models = append(p.Models, *attr)
			case GiftAttributeTypeSymbol:
				p.Symbols = append(p.Symbols, *attr)
			case GiftAttributeTypeBackdrop:
				p.Backdrops = append(p.Backdrops, *attr)
			}
		}
	}
	for _, price := range raw.Prices {
		p.Prices = append(p.Prices, parseUpgradePrice(price))
	}
	for _, price := range raw.NextPrices {
		p.NextPrices = append(p.NextPrices, parseUpgradePrice(price))
	}
	return p
}

func parseUpgradePrice(raw *tg.StarGiftUpgradePrice) GiftUpgradePrice {
	if raw == nil {
		return GiftUpgradePrice{}
	}
	return GiftUpgradePrice{
		Date:      time.Unix(int64(raw.Date), 0),
		StarCount: raw.UpgradeStars,
	}
}

// GiftUpgradePrice represents the star cost to upgrade a gift at a specific
// date.
type GiftUpgradePrice struct {
	Date      time.Time
	StarCount int64
}

// GiftUpgradeVariants holds the available model, symbol, and backdrop
// attribute variants for upgrading a gift.
type GiftUpgradeVariants struct {
	Models    []GiftAttribute
	Symbols   []GiftAttribute
	Backdrops []GiftAttribute
}

// CheckedGiftCode represents the result of checking a premium gift code,
// including duration, giveaway origin, and usage status.
//
// Example:
//
//	checked := types.ParseCheckedGiftCode(raw)
//	fmt.Printf("Days: %d, Via giveaway: %v\n", checked.DayCount, checked.ViaGiveaway)
type CheckedGiftCode struct {
	Date              time.Time
	MonthCount        int32
	DayCount          int32
	ViaGiveaway       bool
	FromChat          *Chat
	Winner            *User
	GiveawayMessageID int32
	UsedDate          time.Time
}

// ParseCheckedGiftCode parses a checked gift code result from a TL
// PaymentsCheckedGiftCode, resolving the sender chat and winner user from the
// embedded peer data.
func ParseCheckedGiftCode(raw *tg.PaymentsCheckedGiftCode) *CheckedGiftCode {
	if raw == nil {
		return nil
	}
	pm := NewPeerMapFromClasses(raw.Users, raw.Chats)
	c := &CheckedGiftCode{
		ViaGiveaway: raw.ViaGiveaway,
		Date:        time.Unix(int64(raw.Date), 0),
		DayCount:    raw.Days,
	}
	if raw.GiveawayMsgID != 0 {
		c.GiveawayMessageID = raw.GiveawayMsgID
	}
	if raw.UsedDate != 0 {
		c.UsedDate = time.Unix(int64(raw.UsedDate), 0)
	}
	if raw.FromID != nil {
		c.FromChat = ParseChatFromPeer(raw.FromID, pm)
	}
	if raw.ToID != 0 {
		c.Winner = getUserFromPM(pm, raw.ToID)
	}
	return c
}

// PremiumGiftCode represents a premium gift code with its creator, pricing,
// duration, and associated sticker.
type PremiumGiftCode struct {
	Creator              *Chat
	Text                 *FormattedText
	IsFromGiveaway       bool
	IsUnclaimed          bool
	Currency             string
	Amount               int64
	Cryptocurrency       string
	CryptocurrencyAmount int64
	MonthCount           int32
	DayCount             int32
	Sticker              *Sticker
	Code                 string
}

// CraftGiftResult holds the outcome of a gift crafting operation, indicating
// success and containing the resulting gift if successful.
type CraftGiftResult struct {
	Success bool
	Gift    *Gift
}

// CraftGiftResultSuccess represents a successful gift craft result containing
// the newly crafted gift.
type CraftGiftResultSuccess struct {
	Gift *Gift
}

// CraftGiftResultFail represents a failed gift craft attempt.
type CraftGiftResultFail struct{}

// GiftedPremium represents information about a gifted Telegram premium
// subscription, including gifter, receiver, pricing, and duration.
type GiftedPremium struct {
	Gifter               *User
	Receiver             *User
	Currency             string
	Amount               int64
	Cryptocurrency       string
	CryptocurrencyAmount int64
	MonthCount           int32
	DayCount             int32
	Sticker              *Sticker
	Caption              string
	CaptionEntities      []*MessageEntity
}

// GiftedStars represents information about gifted Telegram stars, including
// gifter, receiver, pricing, and transaction details.
type GiftedStars struct {
	Gifter               *User
	Receiver             *User
	Currency             string
	Amount               int64
	Cryptocurrency       string
	CryptocurrencyAmount int64
	StarCount            int64
	TransactionID        string
	Sticker              *Sticker
}

// GiftedTon represents information about gifted TON, including gifter,
// receiver, amount, and transaction details.
type GiftedTon struct {
	Gifter        *User
	Receiver      *User
	TonAmount     int64
	TransactionID string
	Sticker       *Sticker
}

// AuctionBid represents a single bid placed in a gift auction, including the
// bidder, amount, date, and round.
type AuctionBid struct {
	PeerID int64
	Amount int64
	Date   int32
	Round  int32
}

// AuctionStateActive represents an active auction state with bidding rounds,
// minimum bid, and timing information. It implements the AuctionStateVariant
// interface.
type AuctionStateActive struct {
	Version      int32
	StartDate    int32
	EndDate      int32
	MinBidAmount int64
	GiftsLeft    int32
	CurrentRound int32
	TotalRounds  int32
	Rounds       []AuctionRound
}

func (*AuctionStateActive) auctionState() {}

// AuctionStateFinished represents a completed auction state with start/end
// dates and the average sale price. It implements the AuctionStateVariant
// interface.
type AuctionStateFinished struct {
	StartDate    int32
	EndDate      int32
	AveragePrice int64
}

func (*AuctionStateFinished) auctionState() {}
