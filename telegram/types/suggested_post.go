package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// SuggestedPostPrice represents the price of a suggested post, which can be
// denominated in Telegram Stars or TON.
type SuggestedPostPrice struct {
	Amount    int64
	Currency  string
	StarCount int64
}

// SuggestedPostParameters represents parameters for a suggested post.
type SuggestedPostParameters struct {
	Author   string
	Price    *SuggestedPostPrice
	SendDate time.Time
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
