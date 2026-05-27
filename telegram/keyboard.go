package telegram

import (
	"github.com/mtgo-labs/mtgo/tg"
)

// KeyboardBuilder provides a fluent interface for constructing Telegram
// keyboards. Add buttons to the current row, call Next() to start a new row,
// then Build() for inline or BuildReply() for a reply keyboard.
//
// Inline:
//
//	markup := telegram.Keyboard().
//	    Callback("Yes", "yes").Primary().
//	    Callback("No", "no").Danger().
//	    Next().
//	    URL("Docs", "https://example.com").
//	    Build()
//
// Reply:
//
//	markup := telegram.Keyboard().
//	    Text("Option A").
//	    Text("Option B").
//	    BuildReply(telegram.ReplyOpts{Resize: true, OneTime: true})
type KeyboardBuilder struct {
	rows [][]tg.KeyboardButtonClass
	row  []tg.KeyboardButtonClass
}

// Keyboard returns a new keyboard builder.
func Keyboard() *KeyboardBuilder {
	return &KeyboardBuilder{
		rows: make([][]tg.KeyboardButtonClass, 0, 4),
		row:  make([]tg.KeyboardButtonClass, 0, 6),
	}
}

func (b *KeyboardBuilder) add(btn tg.KeyboardButtonClass) *KeyboardBuilder {
	b.row = append(b.row, btn)
	return b
}

// Next finalizes the current row and starts a new one. No-op if the row is empty.
func (b *KeyboardBuilder) Next() *KeyboardBuilder {
	if len(b.row) > 0 {
		b.rows = append(b.rows, b.row)
		b.row = make([]tg.KeyboardButtonClass, 0, cap(b.row))
	}
	return b
}

// Row appends a pre-built row of buttons.
func (b *KeyboardBuilder) Row(buttons ...tg.KeyboardButtonClass) *KeyboardBuilder {
	if len(buttons) > 0 {
		b.rows = append(b.rows, buttons)
	}
	return b
}

// ---------------------------------------------------------------------------
// Inline buttons
// ---------------------------------------------------------------------------

// Callback adds a callback button. data is truncated to 64 bytes (Telegram limit).
func (b *KeyboardBuilder) Callback(text, data string) *KeyboardBuilder {
	d := []byte(data)
	if len(d) > 64 {
		d = d[:64]
	}
	return b.add(&tg.KeyboardButtonCallback{Text: text, Data: d})
}

// URL adds a button that opens url when tapped.
func (b *KeyboardBuilder) URL(text, url string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonURL{Text: text, URL: url})
}

// Switch adds a button that switches the user to inline mode.
// samePeer=true sends the query in the current chat; false lets the user pick.
func (b *KeyboardBuilder) Switch(text string, samePeer bool, query string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonSwitchInline{Text: text, Query: query, SamePeer: samePeer})
}

// Copy adds a button that copies copyText to the user's clipboard.
func (b *KeyboardBuilder) Copy(text, copyText string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonCopy{Text: text, CopyText: copyText})
}

// Game adds an HTML5 game button.
func (b *KeyboardBuilder) Game(text string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonGame{Text: text})
}

// Buy adds a payment button.
func (b *KeyboardBuilder) Buy(text string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonBuy{Text: text})
}

// WebApp adds a button that opens a Telegram Mini App.
func (b *KeyboardBuilder) WebApp(text, url string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonWebView{Text: text, URL: url})
}

// ---------------------------------------------------------------------------
// Reply buttons
// ---------------------------------------------------------------------------

// Text adds a text button (sends its text as a message when tapped).
func (b *KeyboardBuilder) Text(text string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButton{Text: text})
}

// RequestPhone adds a button that requests the user's phone number.
func (b *KeyboardBuilder) RequestPhone(text string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonRequestPhone{Text: text})
}

// RequestPeer adds a button that lets the user share a chat, channel, or user.
// buttonID identifies which button was pressed in the response.
// peerType is one of &tg.RequestPeerTypeUser{}, &tg.RequestPeerTypeChat{},
// or &tg.RequestPeerTypeBroadcast{}.
// maxQuantity controls how many peers the user can share (for users only).
// PeerUserOpts controls optional filters when requesting a user peer.
type PeerUserOpts struct {
	Bot     bool
	Premium bool
}

// PeerGroupOpts controls optional filters when requesting a group peer.
type PeerGroupOpts struct {
	Creator        bool
	BotParticipant bool
	HasUsername    bool
	Forum          bool
}

// PeerChannelOpts controls optional filters when requesting a channel peer.
type PeerChannelOpts struct {
	Creator     bool
	HasUsername bool
}

func (b *KeyboardBuilder) RequestPeer(text string, buttonID int32, peerType tg.RequestPeerTypeClass, maxQuantity int32) *KeyboardBuilder {
	return b.add(&tg.InputKeyboardButtonRequestPeer{
		Text:        text,
		ButtonID:    buttonID,
		PeerType:    peerType,
		MaxQuantity: maxQuantity,
	})
}

func (b *KeyboardBuilder) RequestUser(text string, buttonID int32, maxQuantity int32, opts ...PeerUserOpts) *KeyboardBuilder {
	o := getOptDef(PeerUserOpts{}, opts...)
	return b.RequestPeer(text, buttonID, &tg.RequestPeerTypeUser{
		Bot:     o.Bot,
		Premium: o.Premium,
	}, maxQuantity)
}

func (b *KeyboardBuilder) RequestGroup(text string, buttonID int32, opts ...PeerGroupOpts) *KeyboardBuilder {
	o := getOptDef(PeerGroupOpts{}, opts...)
	return b.RequestPeer(text, buttonID, &tg.RequestPeerTypeChat{
		Creator:        o.Creator,
		BotParticipant: o.BotParticipant,
		HasUsername:    o.HasUsername,
		Forum:          o.Forum,
	}, 1)
}

func (b *KeyboardBuilder) RequestChannel(text string, buttonID int32, opts ...PeerChannelOpts) *KeyboardBuilder {
	o := getOptDef(PeerChannelOpts{}, opts...)
	return b.RequestPeer(text, buttonID, &tg.RequestPeerTypeBroadcast{
		Creator:     o.Creator,
		HasUsername: o.HasUsername,
	}, 1)
}

// RequestGeo adds a button that requests the user's location.
func (b *KeyboardBuilder) RequestGeo(text string) *KeyboardBuilder {
	return b.add(&tg.KeyboardButtonRequestGeoLocation{Text: text})
}

// RequestPoll adds a button that prompts the user to create a poll or quiz.
func (b *KeyboardBuilder) RequestPoll(text string, quiz bool) *KeyboardBuilder {
	btn := &tg.KeyboardButtonRequestPoll{Text: text, Quiz: quiz}
	btn.Flags.Set(0)
	return b.add(btn)
}

// ---------------------------------------------------------------------------
// Style modifiers (applied to the last button in the current row)
// ---------------------------------------------------------------------------

// Primary highlights the last button with a primary background color.
func (b *KeyboardBuilder) Primary() *KeyboardBuilder { return b.applyStyle(primary) }

// Danger marks the last button as destructive (red background).
func (b *KeyboardBuilder) Danger() *KeyboardBuilder { return b.applyStyle(danger) }

// Success marks the last button as positive (green background).
func (b *KeyboardBuilder) Success() *KeyboardBuilder { return b.applyStyle(success) }

// Icon sets a custom emoji icon on the last button by custom emoji document ID.
func (b *KeyboardBuilder) Icon(docID int64) *KeyboardBuilder {
	return b.applyStyle(func(s *tg.KeyboardButtonStyle) { s.Icon = docID })
}

func primary(s *tg.KeyboardButtonStyle) { s.BgPrimary = true }
func danger(s *tg.KeyboardButtonStyle)  { s.BgDanger = true }
func success(s *tg.KeyboardButtonStyle) { s.BgSuccess = true }

// applyStyle modifies the Style field of the last button in the current row.
// No-op if the row is empty or the button doesn't support styling.
func (b *KeyboardBuilder) applyStyle(fn func(*tg.KeyboardButtonStyle)) *KeyboardBuilder {
	if len(b.row) == 0 {
		return b
	}
	btn := b.row[len(b.row)-1]
	if s := styleOf(btn); s != nil {
		fn(s)
	}
	return b
}

// styleOf returns a pointer to the button's Style field, or nil if unsupported.
func styleOf(btn tg.KeyboardButtonClass) *tg.KeyboardButtonStyle {
	switch b := btn.(type) {
	case *tg.KeyboardButton:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonURL:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonCallback:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonRequestPhone:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonRequestGeoLocation:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonSwitchInline:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonGame:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonBuy:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonURLAuth:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonRequestPoll:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonUserProfile:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonWebView:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonSimpleWebView:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	case *tg.KeyboardButtonCopy:
		if b.Style == nil {
			b.Style = &tg.KeyboardButtonStyle{}
		}
		return b.Style
	}
	return nil
}

// ---------------------------------------------------------------------------
// Build
// ---------------------------------------------------------------------------

// ReplyOpts controls the appearance and behaviour of a reply keyboard.
type ReplyOpts struct {
	Resize      bool   // Shrink keyboard to fit fewer buttons.
	OneTime     bool   // Hide after first press.
	Selective   bool   // Show only to @mentioned or replied-to users.
	Persistent  bool   // Don't auto-hide when another keyboard is sent.
	Placeholder string // Hint text in the input field. Empty = default.
}

// buildRows finalizes the current row and returns all accumulated rows, or nil.
func (b *KeyboardBuilder) buildRows() []*tg.KeyboardButtonRow {
	b.Next()
	if len(b.rows) == 0 {
		return nil
	}
	out := make([]*tg.KeyboardButtonRow, len(b.rows))
	for i, row := range b.rows {
		out[i] = &tg.KeyboardButtonRow{Buttons: row}
	}
	return out
}

// Build produces an inline keyboard (tg.ReplyInlineMarkup).
// Returns nil if no buttons were added.
func (b *KeyboardBuilder) Build() tg.ReplyMarkupClass {
	rows := b.buildRows()
	if rows == nil {
		return nil
	}
	return &tg.ReplyInlineMarkup{Rows: rows}
}

// BuildReply produces a reply keyboard (tg.ReplyKeyboardMarkup).
// Returns nil if no buttons were added.
func (b *KeyboardBuilder) BuildReply(opts ...ReplyOpts) tg.ReplyMarkupClass {
	rows := b.buildRows()
	if rows == nil {
		return nil
	}

	var o ReplyOpts
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &tg.ReplyKeyboardMarkup{
		Rows:       rows,
		Resize:     o.Resize,
		SingleUse:  o.OneTime,
		Selective:  o.Selective,
		Persistent: o.Persistent,
	}
	if o.Resize {
		m.Flags |= 1 << 0
	}
	if o.OneTime {
		m.Flags |= 1 << 1
	}
	if o.Selective {
		m.Flags |= 1 << 2
	}
	if o.Persistent {
		m.Flags |= 1 << 4
	}
	if o.Placeholder != "" {
		m.Placeholder = o.Placeholder
		m.Flags |= 1 << 3
	}
	return m
}

// ---------------------------------------------------------------------------
// Standalone markups
// ---------------------------------------------------------------------------

// ForceReplyMarkup returns a markup that forces the user to reply.
func ForceReplyMarkup(opts ...ReplyOpts) *tg.ReplyKeyboardForceReply {
	var o ReplyOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	m := &tg.ReplyKeyboardForceReply{
		SingleUse: o.OneTime,
		Selective: o.Selective,
	}
	if o.OneTime {
		m.Flags |= 1 << 1
	}
	if o.Selective {
		m.Flags |= 1 << 2
	}
	if o.Placeholder != "" {
		m.Placeholder = o.Placeholder
		m.Flags |= 1 << 3
	}
	return m
}

// RemoveKeyboard returns a markup that removes the bot's reply keyboard.
func RemoveKeyboard(opts ...ReplyOpts) *tg.ReplyKeyboardHide {
	var o ReplyOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	m := &tg.ReplyKeyboardHide{
		Selective: o.Selective,
	}
	if o.Selective {
		m.Flags |= 1 << 2
	}
	return m
}
