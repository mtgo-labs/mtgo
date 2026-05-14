package types

import "github.com/mtgo-labs/mtgo/tg"

// MenuButtonType enumerates the kinds of bot menu buttons shown next to the
// text input field in a chat with the bot.
type MenuButtonType string

const (
	// MenuButtonCommands shows the bot's command list when tapped.
	MenuButtonCommands MenuButtonType = "commands"
	// MenuButtonWebApp opens a Web App when tapped.
	MenuButtonWebApp MenuButtonType = "web_app"
	// MenuButtonDefault uses the default menu button configured in BotFather.
	MenuButtonDefault MenuButtonType = "default"
)

// MenuButton describes a bot's menu button shown next to the text input field.
// Depending on Type, it either shows commands, opens a Web App, or uses the default.
type MenuButton struct {
	// Type identifies the kind of menu button (commands, web_app, or default).
	Type MenuButtonType
	// Text is the label shown on the menu button for Web App type buttons.
	Text string
	// URL is the Web App URL opened when the button is tapped, for Web App type buttons.
	URL string
}

// ParseMenuButton converts a TL BotMenuButtonClass into a MenuButton.
// Returns nil if raw is nil.
func ParseMenuButton(raw tg.BotMenuButtonClass) *MenuButton {
	if raw == nil {
		return nil
	}
	switch raw := raw.(type) {
	case *tg.BotMenuButtonCommands:
		return &MenuButton{Type: MenuButtonCommands}
	case *tg.BotMenuButtonDefault:
		return &MenuButton{Type: MenuButtonDefault}
	case *tg.BotMenuButton:
		btn := raw
		return &MenuButton{
			Type: MenuButtonWebApp,
			Text: btn.Text,
			URL:  btn.URL,
		}
	default:
		return &MenuButton{Type: MenuButtonDefault}
	}
}
