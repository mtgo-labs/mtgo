package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// PaymentForm represents a payment invoice form with details about the seller,
// payment provider, invoice items, and checkout options.
//
// Example:
//
//	form := types.ParsePaymentForm(rawForm)
//	fmt.Printf("Payment: %s (%s)\n", form.Title, form.Description)
type PaymentForm struct {
	ID                       int64
	Type                     PaymentFormType
	Title                    string
	Description              string
	Photo                    *Photo
	SellerBotUserID          int64
	SellerBot                *User
	PaymentProviderUserID    int64
	PaymentProvider          *User
	AdditionalPaymentOptions []*PaymentOption
	SavedCredentials         []*SavedCredentials
	Invoice                  *Invoice
	URL                      string
	IsCanSaveCredentials     bool
	IsNeedPassword           bool
	NativeProvider           string
}

// ParsePaymentForm converts a TL PaymentFormClass into a PaymentForm, handling
// regular, stars, and star gift payment forms. Returns nil if raw is nil.
//
// Example:
//
//	form := types.ParsePaymentForm(rawForm)
//	if form != nil {
//	    fmt.Printf("Form ID: %d, type: %s\n", form.ID, form.Type)
//	}
func ParsePaymentForm(raw tg.PaymentFormClass) *PaymentForm {
	if raw == nil {
		return nil
	}
	switch form := raw.(type) {
	case *tg.PaymentsPaymentForm:
		users := make(map[int64]tg.UserClass, len(form.Users))
		for _, u := range form.Users {
			if v, ok := u.(*tg.User); ok && v != nil {
				users[v.ID] = u
			}
		}
		pf := &PaymentForm{
			ID:                    form.FormID,
			Type:                  PaymentFormTypeRegular,
			Title:                 form.Title,
			Description:           form.Description,
			SellerBotUserID:       form.BotID,
			PaymentProviderUserID: form.ProviderID,
			Invoice:               ParseInvoice(form.Invoice),
			URL:                   form.URL,
			IsCanSaveCredentials:  form.CanSaveCredentials,
			IsNeedPassword:        form.PasswordMissing,
			NativeProvider:        form.NativeProvider,
		}
		if form.Photo != nil {
			pf.Photo = parseWebDocumentPhoto(form.Photo)
		}
		pf.SellerBot = getUser(users, form.BotID)
		pf.PaymentProvider = getUser(users, form.ProviderID)
		for _, m := range form.AdditionalMethods {
			if m != nil {
				pf.AdditionalPaymentOptions = append(pf.AdditionalPaymentOptions, &PaymentOption{
					URL:   m.URL,
					Title: m.Title,
				})
			}
		}
		for _, c := range form.SavedCredentials {
			if c != nil {
				pf.SavedCredentials = append(pf.SavedCredentials, &SavedCredentials{
					ID:    c.ID,
					Title: c.Title,
				})
			}
		}
		return pf
	case *tg.PaymentsPaymentFormStars:
		users := make(map[int64]tg.UserClass, len(form.Users))
		for _, u := range form.Users {
			if v, ok := u.(*tg.User); ok && v != nil {
				users[v.ID] = u
			}
		}
		pf := &PaymentForm{
			ID:              form.FormID,
			Type:            PaymentFormTypeStars,
			Title:           form.Title,
			Description:     form.Description,
			SellerBotUserID: form.BotID,
			Invoice:         ParseInvoice(form.Invoice),
		}
		if form.Photo != nil {
			pf.Photo = parseWebDocumentPhoto(form.Photo)
		}
		pf.SellerBot = getUser(users, form.BotID)
		return pf
	case *tg.PaymentsPaymentFormStarGift:
		return &PaymentForm{
			ID:      form.FormID,
			Type:    PaymentFormTypeStarSubscription,
			Invoice: ParseInvoice(form.Invoice),
		}
	default:
		return nil
	}
}

func parseWebDocumentPhoto(raw tg.WebDocumentClass) *Photo {
	if raw == nil {
		return nil
	}
	return nil
}
