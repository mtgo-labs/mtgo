package types

// GiftPurchaseOfferState enumerates the possible states of a gift purchase
// offer. Use this to determine whether an offer is still available, has been
// sold, upgraded, or refunded.
type GiftPurchaseOfferState string

const (
	GiftPurchaseOfferStateActive   GiftPurchaseOfferState = "active"
	GiftPurchaseOfferStateSoldOut  GiftPurchaseOfferState = "sold_out"
	GiftPurchaseOfferStateUpgraded GiftPurchaseOfferState = "upgraded"
	GiftPurchaseOfferStateRefunded GiftPurchaseOfferState = "refunded"
	GiftPurchaseOfferStatePending  GiftPurchaseOfferState = "pending"
	GiftPurchaseOfferStateAccepted GiftPurchaseOfferState = "accepted"
	GiftPurchaseOfferStateRejected GiftPurchaseOfferState = "rejected"
)

// String returns the string representation of the GiftPurchaseOfferState.
func (g GiftPurchaseOfferState) String() string { return string(g) }
