package types

import "github.com/mtgo-labs/mtgo/tg"

// SuggestedPostPrice represents the price of a suggested post, which can be
// denominated in Telegram Stars or TON.
type SuggestedPostPrice struct {
	// Amount is the price amount.
	Amount int64
	// Currency is the currency code ("XTR" for Stars, "TON" for TON).
	Currency string
}

// SuggestedPostParameters represents parameters for a suggested post.
type SuggestedPostParameters struct {
	// Author is the peer that authored the suggested post.
	Author string
	// Price is the optional price for the suggested post.
	Price *SuggestedPostPrice
}

// ParseSuggestedPostParameters converts a TL SuggestedPost into SuggestedPostParameters.
func ParseSuggestedPostParameters(raw *tg.SuggestedPost) *SuggestedPostParameters {
	if raw == nil {
		return nil
	}
	out := &SuggestedPostParameters{}
	if raw.Price != nil {
		switch p := raw.Price.(type) {
		case *tg.StarsAmount:
			out.Price = &SuggestedPostPrice{
				Amount:   p.Amount,
				Currency: "XTR",
			}
		case *tg.StarsTonAmount:
			out.Price = &SuggestedPostPrice{
				Amount:   p.Amount,
				Currency: "TON",
			}
		}
	}
	return out
}
