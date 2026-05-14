package types

// SuggestedPostRefundReason indicates why a suggested post payment was
// refunded. Use this when processing payment reversal events to explain the
// refund cause to the user.
type SuggestedPostRefundReason string

const (
	// SuggestedPostRefundReasonSuggestedPostUnavailable indicates the refund
	// was issued because the suggested post was deleted or became unavailable
	// before publication.
	SuggestedPostRefundReasonSuggestedPostUnavailable SuggestedPostRefundReason = "suggested_post_unavailable"
	// SuggestedPostRefundReasonFailedToDeliver indicates the refund was issued
	// because the platform failed to deliver the payment to the intended
	// recipient.
	SuggestedPostRefundReasonFailedToDeliver SuggestedPostRefundReason = "failed_to_deliver"
	// SuggestedPostRefundReasonOther indicates the refund was issued for a
	// reason not covered by the specific categories.
	SuggestedPostRefundReasonOther SuggestedPostRefundReason = "other"
)

// String returns the string representation of the SuggestedPostRefundReason.
func (s SuggestedPostRefundReason) String() string { return string(s) }
