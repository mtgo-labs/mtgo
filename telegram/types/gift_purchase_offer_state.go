package types

// GiftPurchaseOfferState enumerates the possible states of a gift purchase
// offer. Use this to determine whether an offer is still available, has been
// sold, upgraded, or refunded.
type GiftPurchaseOfferState string

const (
	// GiftPurchaseOfferStateActive indicates the offer is currently available
	// for purchase.
	GiftPurchaseOfferStateActive GiftPurchaseOfferState = "active"
	// GiftPurchaseOfferStateSoldOut indicates the gift has been purchased by
	// another buyer.
	GiftPurchaseOfferStateSoldOut GiftPurchaseOfferState = "sold_out"
	// GiftPurchaseOfferStateUpgraded indicates the gift has been upgraded by
	// the owner and is no longer available at the original offer price.
	GiftPurchaseOfferStateUpgraded GiftPurchaseOfferState = "upgraded"
	// GiftPurchaseOfferStateRefunded indicates the offer was cancelled and the
	// seller's listing fee was refunded.
	GiftPurchaseOfferStateRefunded GiftPurchaseOfferState = "refunded"
)

// String returns the string representation of the GiftPurchaseOfferState.
func (g GiftPurchaseOfferState) String() string { return string(g) }
