package types

// GiftForResaleOrder enumerates the sort orders available when listing gifts
// for resale on the Telegram gift marketplace.
type GiftForResaleOrder string

const (
	// GiftForResaleOrderPriceAsc sorts resale listings from lowest to highest
	// price.
	GiftForResaleOrderPriceAsc GiftForResaleOrder = "price_asc"
	// GiftForResaleOrderPriceDesc sorts resale listings from highest to lowest
	// price.
	GiftForResaleOrderPriceDesc GiftForResaleOrder = "price_desc"
	// GiftForResaleOrderDateAsc sorts resale listings from oldest to newest
	// listing date.
	GiftForResaleOrderDateAsc GiftForResaleOrder = "date_asc"
	// GiftForResaleOrderDateDesc sorts resale listings from newest to oldest
	// listing date.
	GiftForResaleOrderDateDesc GiftForResaleOrder = "date_desc"
)

// String returns the string representation of the GiftForResaleOrder.
func (g GiftForResaleOrder) String() string { return string(g) }
