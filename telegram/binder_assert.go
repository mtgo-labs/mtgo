package telegram

import "github.com/mtgo-labs/mtgo/telegram/types"

// Compile-time proof that *Client satisfies every per-type binder interface
// through promoted group methods. If a group method signature drifts, the
// corresponding assertion here fails the build.
var (
	_ types.MessageBinder          = (*Client)(nil)
	_ types.ChatBinder             = (*Client)(nil)
	_ types.UserBinder             = (*Client)(nil)
	_ types.StoryBinder            = (*Client)(nil)
	_ types.GiftBinder             = (*Client)(nil)
	_ types.FolderBinder           = (*Client)(nil)
	_ types.CallbackQueryBinder    = (*Client)(nil)
	_ types.InlineQueryBinder      = (*Client)(nil)
	_ types.ChatJoinRequestBinder  = (*Client)(nil)
	_ types.PreCheckoutQueryBinder = (*Client)(nil)
	_ types.ShippingQueryBinder    = (*Client)(nil)
	_ types.SessionBinder          = (*Client)(nil)
)
