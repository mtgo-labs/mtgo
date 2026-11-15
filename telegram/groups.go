package telegram

// Domain method groups.
//
// Every client-facing operation lives on exactly one group, and each group
// holds a reference back to the Client it serves. Client embeds all groups, so
// group methods are promoted onto *Client itself: client.SendMessage resolves
// to MessageMethods.SendMessage. Adding a method to a group makes it available
// on the client automatically; the promoted-method collision test in
// collisions_test.go guarantees no name is ever defined twice across groups or
// shadowed by a direct Client method.

// MessageMethods groups message lifecycle operations: sending (text, media,
// polls, games, rich content), editing (normal and inline), forwarding,
// copying, deleting, searching, reading, reacting, and media download for
// message-bound content.
type MessageMethods struct{ *Client }

// ChatMethods groups chat and channel management: metadata changes, membership
// and administration, permissions, forum topics, and chat archive state.
type ChatMethods struct{ *Client }

// UserMethods groups user-profile operations: profile inspection and updates,
// blocking, and user-oriented media listings.
type UserMethods struct{ *Client }

// ContactsMethods groups the contact-book operations: adding, deleting,
// searching, and listing contacts.
type ContactsMethods struct{ *Client }

// AccountMethods groups account-level settings: privacy rules, global privacy,
// and account lifetime.
type AccountMethods struct{ *Client }

// AuthMethods groups authorization: phone-code sign-in flows, sign-up, QR
// login, password recovery, session listings, and logout.
type AuthMethods struct{ *Client }

// PasswordMethods groups two-step-verification (cloud password) management.
type PasswordMethods struct{ *Client }

// BotsMethods groups bot-facing operations: commands, bot info, menu buttons,
// inline queries, callback queries, and game scoreboards.
type BotsMethods struct{ *Client }

// BusinessMethods groups bot business-connection operations.
type BusinessMethods struct{ *Client }

// PremiumMethods group boost operations for channels and the premium state of
// the logged-in user.
type PremiumMethods struct{ *Client }

// StoryMethods groups story lifecycle operations: posting, editing, deleting,
// fetching, viewing, and story reactions.
type StoryMethods struct{ *Client }

// PaymentMethods groups payments: invoices, checkout forms, Stars balances and
// transactions, and the gift catalog and gift lifecycle.
type PaymentMethods struct{ *Client }

// StickerMethods groups sticker and custom-emoji operations.
type StickerMethods struct{ *Client }

// FolderMethods groups chat-folder (dialog filter) operations: folder CRUD,
// folder membership, and shareable folder invite links.
type FolderMethods struct{ *Client }

// InviteLinkMethods groups chat invite-link management and join-request
// handling.
type InviteLinkMethods struct{ *Client }

// UtilityMethods groups cross-domain helpers: dialog listing, peer resolution,
// raw file transfer by location, and app-level configuration.
type UtilityMethods struct{ *Client }

// AdvancedMethods groups low-level escape hatches: typed raw RPC clients, the
// JSON RPC bridge, and connection introspection.
type AdvancedMethods struct{ *Client }

// PhoneMethods groups voice/video call and live-broadcast streaming
// operations.
type PhoneMethods struct{ *Client }

// WalletMethods groups the on-platform wallet operations for third-party
// value transfer.
type WalletMethods struct{ *Client }

// SecretChatMethods groups end-to-end encrypted chat operations. This group is
// an mtgo extension with no counterpart in the shared client domains.
type SecretChatMethods struct{ *Client }

// initMethodGroups wires every embedded group back to this client. It is
// idempotent and must run before any promoted group method is called.
func (c *Client) initMethodGroups() {
	c.MessageMethods = &MessageMethods{Client: c}
	c.ChatMethods = &ChatMethods{Client: c}
	c.UserMethods = &UserMethods{Client: c}
	c.ContactsMethods = &ContactsMethods{Client: c}
	c.AccountMethods = &AccountMethods{Client: c}
	c.AuthMethods = &AuthMethods{Client: c}
	c.PasswordMethods = &PasswordMethods{Client: c}
	c.BotsMethods = &BotsMethods{Client: c}
	c.BusinessMethods = &BusinessMethods{Client: c}
	c.PremiumMethods = &PremiumMethods{Client: c}
	c.StoryMethods = &StoryMethods{Client: c}
	c.PaymentMethods = &PaymentMethods{Client: c}
	c.StickerMethods = &StickerMethods{Client: c}
	c.FolderMethods = &FolderMethods{Client: c}
	c.InviteLinkMethods = &InviteLinkMethods{Client: c}
	c.UtilityMethods = &UtilityMethods{Client: c}
	c.AdvancedMethods = &AdvancedMethods{Client: c}
	c.PhoneMethods = &PhoneMethods{Client: c}
	c.WalletMethods = &WalletMethods{Client: c}
	c.SecretChatMethods = &SecretChatMethods{Client: c}
}

// initClient is a helper for constructing bare clients in tests: it builds the
// struct literal and wires the method groups in one expression.
func initClient(c *Client) *Client {
	c.initMethodGroups()
	return c
}
