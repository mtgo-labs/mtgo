package types

// GiftType enumerates the kinds of Telegram gifts. Use this to distinguish
// between regular star gifts, unique collectibles, premium subscriptions,
// and other gift categories when processing gift-related events.
type GiftType string

const (
	// GiftTypeStarGift represents a regular star gift that can be purchased
	// with Telegram Stars.
	GiftTypeStarGift GiftType = "star_gift"
	// GiftTypeStarGiftUnique represents a unique (collectible) star gift with
	// a limited supply and individual serial number.
	GiftTypeStarGiftUnique GiftType = "star_gift_unique"
	// GiftTypePremiumSubscription represents a gifted Telegram Premium
	// subscription for a specified number of months.
	GiftTypePremiumSubscription GiftType = "premium_subscription"
	// GiftTypeTonGift represents a gift denominated in TON cryptocurrency.
	GiftTypeTonGift GiftType = "ton_gift"
	// GiftTypeStarGiftForResale represents a star gift listed for resale by
	// its current owner.
	GiftTypeStarGiftForResale GiftType = "star_gift_for_resale"
	// GiftTypeUpgradedStarGift represents a star gift that has been upgraded
	// to a unique collectible version.
	GiftTypeUpgradedStarGift GiftType = "upgraded_star_gift"
)

// String returns the string representation of the GiftType.
func (g GiftType) String() string { return string(g) }
