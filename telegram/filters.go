package telegram

import (
	"regexp"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram/types"
)

// HasText matches messages that contain any non-empty text body.
// This is the argument-free equivalent of Pyrogram's "text" filter.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("You sent text!")
//	}, telegram.HasText)
var HasText Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Text != ""
}

// Text returns a Filter that matches messages whose text content exactly equals s.
// For substring or pattern matching use Regex instead.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("pong")
//	}, telegram.Text("ping"))
func Text(s string) Filter {
	return func(ctx *Context) bool {
		return ctx.Message != nil && ctx.Message.Text == s
	}
}

// Command returns a Filter that matches messages starting with one of the given
// bot commands (e.g. "/start", "/help"). The leading "/" is prepended automatically.
// Commands are matched case-insensitively and support @bot_username suffixes.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Welcome!")
//	}, telegram.Command("start"))
//
// Multiple commands:
//
//	client.OnMessage(handleHelp, telegram.Command("help", "info"))
func Command(commands ...string) Filter {
	normalized := make(map[string]bool, len(commands))
	for _, c := range commands {
		normalized["/"+c] = true
	}
	return func(ctx *Context) bool {
		if ctx.Message == nil || ctx.Message.Text == "" {
			return false
		}
		text := ctx.Message.Text
		if text[0] != '/' {
			return false
		}
		if idx := strings.IndexByte(text, '@'); idx > 0 {
			text = text[:idx]
		}
		if idx := strings.IndexByte(text, ' '); idx > 0 {
			text = text[:idx]
		}
		return normalized[text]
	}
}

// Regex returns a Filter that matches messages, callback queries, inline queries,
// or chosen inline results whose text content matches the given regular expression.
// Panics if the pattern is invalid.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Got a number!")
//	}, telegram.Regex(`\d+`))
func Regex(pattern string) Filter {
	re := regexp.MustCompile(pattern)
	return func(ctx *Context) bool {
		text := ""
		switch {
		case ctx.Message != nil:
			text = ctx.Message.Text
		case ctx.CallbackQuery != nil:
			text = string(ctx.CallbackQuery.Data)
		case ctx.InlineQuery != nil:
			text = ctx.InlineQuery.Query
		case ctx.ChosenInlineResult != nil:
			text = ctx.ChosenInlineResult.Query
		}
		return text != "" && re.MatchString(text)
	}
}

// All unconditionally matches every update. Use it when a handler should fire
// regardless of message content or type.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("received update in chat %d", ctx.Message.ChatID)
//	}, telegram.All)
var All Filter = func(ctx *Context) bool { return true }

// Me matches only outgoing messages sent by the current user.
// Use it to monitor or process the user's own messages across all chats.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("you sent: %s", ctx.Message.Text)
//	}, telegram.Me)
var Me Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Out
}

// Bot matches messages sent by bot users. It checks the sender map in the
// update to determine whether the author is a bot account.
// Useful for ignoring or specifically handling automated messages.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Hello, bot!")
//	}, telegram.Bot)
var Bot Filter = func(ctx *Context) bool {
	if ctx.Update == nil || ctx.Update.Users == nil || ctx.Message == nil {
		return false
	}
	if u, ok := ctx.Update.Users[ctx.Message.FromID]; ok {
		return u.IsBot
	}
	return false
}

// Incoming matches only incoming (non-outgoing) messages.
// Useful for bots that should ignore their own echoed messages.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("You said: " + ctx.Message.Text)
//	}, telegram.Incoming)
var Incoming Filter = func(ctx *Context) bool {
	return ctx.Message != nil && !ctx.Message.Out
}

// Outgoing matches only outgoing messages sent by the current user.
// Use for logging or processing messages the user sends from other clients.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("you sent a message to chat %d", ctx.Message.ChatID)
//	}, telegram.Outgoing)
var Outgoing Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Out
}

// Caption matches messages that have a media attachment but no text body.
// This is useful when you want to handle media that carries its description
// in the caption field instead of the main text.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("received media with caption only")
//	}, telegram.Caption)
var Caption Filter = func(ctx *Context) bool {
	return ctx.Message != nil && hasAnyMedia(ctx) && ctx.Message.Text == ""
}

// Audio matches messages containing audio attachments (e.g. MP3, FLAC).
// Use to build music bots or audio processing pipelines.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Nice track!")
//	}, telegram.Audio)
var Audio Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeAudio)
}

// Video matches messages containing video attachments (e.g. MP4, MOV).
// Use to handle video uploads or transcode triggers.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Received your video!")
//	}, telegram.Video)
var Video Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeVideo)
}

// Animation matches messages containing GIF/animations (MPEG4 or GIF format).
// Use to react to inline GIF results or auto-tag animated content.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("received a GIF")
//	}, telegram.Animation)
var Animation Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeAnimation)
}

// Voice matches messages containing voice notes (recorded audio messages).
// Useful for voice-command bots or transcription pipelines.
//
// Example:
//
//	client.OnMessage(handleVoiceNote, telegram.Voice)
var Voice Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeVoice)
}

// VideoNote matches messages containing round video notes (bubble videos).
// These are short circular video recordings unique to Telegram.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Got your video note!")
//	}, telegram.VideoNote)
var VideoNote Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeVideoNote)
}

// Sticker matches messages containing stickers. Use to build sticker
// management bots or react to specific sticker packs.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Cute sticker!")
//	}, telegram.Sticker)
var Sticker Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeSticker)
}

// Photo matches messages containing photos.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Nice photo!")
//	}, telegram.Photo)
var Photo Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypePhoto)
}

// Document matches messages containing generic file documents.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Received your file!")
//	}, telegram.Document)
var Document Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeDocument)
}

// Contact matches messages containing shared contact cards.
// Useful for CRM bots or contact-import workflows.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("received a contact card")
//	}, telegram.Contact)
var Contact Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeContact)
}

// Location matches messages containing static location pins.
// For continuously updating locations use LiveLocation instead.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("received a location pin")
//	}, telegram.Location)
var Location Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeLocation)
}

// LiveLocation matches messages containing live (updating) locations.
// Live locations update periodically until the sender stops sharing.
//
// Example:
//
//	client.OnMessage(trackLiveLocation, telegram.LiveLocation)
var LiveLocation Filter = func(ctx *Context) bool {
	if ctx.Message == nil || ctx.Message.Media == nil {
		return false
	}
	if loc, ok := ctx.Message.Media.(*types.LocationMedia); ok {
		return loc.Live
	}
	return false
}

// Venue matches messages containing venue information (name, address, location).
// Useful for location-based services and check-in bots.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("received a venue")
//	}, telegram.Venue)
var Venue Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeVenue)
}

// WebPage matches messages containing link previews (web page attachments).
// Fires when Telegram auto-generates a preview for a shared URL.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("message has a link preview")
//	}, telegram.WebPage)
var WebPage Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeWebPage)
}

// Poll matches messages containing polls. Use to auto-vote, log, or
// respond to poll creations in groups.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("a new poll was created")
//	}, telegram.Poll)
var Poll Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypePoll)
}

// Dice matches messages containing dice (animated emoji with random values).
// Telegram supports standard dice, darts, basketball, and other animated emoji.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("You rolled the dice!")
//	}, telegram.Dice)
var Dice Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeDice)
}

// Game matches messages containing game attachments. Telegram games are
// HTML5 mini-games that can be launched inline.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Let's play!")
//	}, telegram.Game)
var Game Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeGame)
}

// Giveaway matches messages containing giveaway announcements.
// Use to detect and track Telegram premium/channel giveaways.
//
// Example:
//
//	client.OnMessage(logGiveaway, telegram.Giveaway)
var Giveaway Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeGiveaway)
}

// GiveawayWinners matches messages containing giveaway winner announcements.
// These are service messages that list the winners of a completed giveaway.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("giveaway winners announced!")
//	}, telegram.GiveawayWinners)
var GiveawayWinners Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeGiveawayWinners)
}

// Story matches messages containing forwarded stories.
// Use to detect when users share stories into chats.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("received a forwarded story")
//	}, telegram.Story)
var Story Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeStory)
}

// PaidMedia matches messages containing paid media that requires a
// Telegram Stars payment to unlock.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("paid media received")
//	}, telegram.PaidMedia)
var PaidMedia Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypePaidMedia)
}

// Invoice matches messages containing payment invoices. These are
// Telegram-native payment requests for goods and services.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("payment invoice received")
//	}, telegram.Invoice)
var Invoice Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeInvoice)
}

// MediaGroup matches messages belonging to a media group (album).
// Albums are sets of photos or videos sent together with a shared GroupedID.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("album message, group=%d", ctx.Message.GroupedID)
//	}, telegram.MediaGroup)
var MediaGroup Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.GroupedID != 0
}

// MediaSpoiler matches messages whose media is hidden behind a spoiler overlay.
// The user must tap to reveal the content.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("I see you sent a spoiler media!")
//	}, telegram.MediaSpoiler)
var MediaSpoiler Filter = func(ctx *Context) bool {
	if ctx.Message == nil || ctx.Message.Media == nil {
		return false
	}
	switch m := ctx.Message.Media.(type) {
	case *types.PhotoMedia:
		return m.IsSpoiler
	case *types.DocumentMedia:
		return m.IsSpoiler
	}
	return false
}

// Media matches messages containing any type of media attachment (photo, video,
// audio, document, sticker, etc.). This is a broad catch-all; prefer specific
// filters like Photo or Document when possible.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Got some media!")
//	}, telegram.Media)
var Media Filter = func(ctx *Context) bool {
	_, ok := hasMedia(ctx)
	return ok
}

// HasMedia is an alias for Media. See Media for documentation.
//
// Example:
//
//	client.OnMessage(handleAnyMedia, telegram.HasMedia)
var HasMedia = Media

// Service matches service messages (system messages) such as group creation,
// member joins, photo changes, and pin events. Use with specific service
// action filters for finer granularity.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("service message received")
//	}, telegram.Service)
var Service Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil
}

// Private matches messages from private (1-on-1) chats.
// These are identified by a positive ChatID.
//
// Example:
//
//	client.OnMessage(handlePrivate, telegram.Private)
var Private Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ChatID > 0
}

// Group matches messages from group chats (basic groups and supergroups).
// These are identified by a negative ChatID.
//
// Example:
//
//	client.OnMessage(handleGroup, telegram.Group)
var Group Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ChatID < 0
}

// Channel matches messages from broadcast channels (not supergroups).
// Supergroups and forums also have negative ChatIDs — this filter
// distinguishes channels by checking the chat type in the update's peer map.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("channel post in %d", ctx.Message.ChatID)
//	}, telegram.Channel)
var Channel Filter = func(ctx *Context) bool {
	if ctx.Message == nil || ctx.Update == nil || ctx.Update.Chats == nil {
		return false
	}
	ch, ok := ctx.Update.Chats[ctx.Message.ChatID]
	return ok && ch.Type == types.ChatTypeChannel
}

// Forum matches messages from forum-enabled supergroups. Forums organize
// conversations into topics; use the Topic filter for topic-level matching.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("message from a forum supergroup")
//	}, telegram.Forum)
var Forum Filter = func(ctx *Context) bool {
	if ctx.Update == nil || ctx.Update.Chats == nil || ctx.Message == nil {
		return false
	}
	if ch, ok := ctx.Update.Chats[ctx.Message.ChatID]; ok {
		return ch.IsForum
	}
	return false
}

// Business matches updates related to Telegram Business connections.
// These are messages exchanged through a Telegram Business account.
//
// Example:
//
//	client.OnMessage(handleBusinessMsg, telegram.Business)
var Business Filter = func(ctx *Context) bool {
	return ctx.BusinessMessage != nil
}

// GuestMessage matches messages sent by guest users pending approval in
// a Telegram Business chat.
//
// Example:
//
//	client.OnMessage(handleGuest, telegram.GuestMessage)
var GuestMessage Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.IsFromPending
}

// Scheduled is a placeholder for scheduled messages. Always returns false
// in the current implementation. Reserved for future use.
var Scheduled Filter = func(ctx *Context) bool { return false }

// FromScheduled is a placeholder for messages originally sent as scheduled
// messages. Always returns false in the current implementation.
var FromScheduled Filter = func(ctx *Context) bool { return false }

// PaidMessage matches messages that require payment (Telegram Stars) to view.
// Similar to PaidMedia but checked at the message level.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("paid message received")
//	}, telegram.PaidMessage)
var PaidMessage Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypePaidMedia)
}

// LinkedChannel matches messages forwarded from a linked channel into a
// discussion group. Detected by the presence of a PostAuthor field.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("channel forward by %s", ctx.Message.PostAuthor)
//	}, telegram.LinkedChannel)
var LinkedChannel Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.PostAuthor != ""
}

// Forwarded matches messages forwarded from another chat. The original
// source information is available in Message.FwdFrom.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("this message was forwarded")
//	}, telegram.Forwarded)
var Forwarded Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.FwdFrom != nil
}

// Reply matches messages that are replies to another message. The parent
// message ID is available in Message.ReplyToID.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("reply to message %d", ctx.Message.ReplyToID)
//	}, telegram.Reply)
var Reply Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ReplyToID != 0
}

// Mentioned matches messages where the current user was @mentioned.
// Useful for building notification or highlight responders.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("You mentioned me!")
//	}, telegram.Mentioned)
var Mentioned Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Mentioned
}

// ViaBot matches messages sent via an inline bot (messages where the user
// chose a result from an inline query).
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("sent via bot %d", ctx.Message.ViaBotID)
//	}, telegram.ViaBot)
var ViaBot Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ViaBotID != 0
}

// Pinned matches messages that are currently pinned in their chat.
// Useful for logging or auto-unpinning old pinned messages.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("pinned message detected")
//	}, telegram.Pinned)
var Pinned Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Pinned
}

// NewChatMembers matches service messages for new members joining a group.
// Access the new member list through the service message fields.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Welcome to the group!")
//	}, telegram.NewChatMembers)
var NewChatMembers Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupAddMembers
}

// LeftChatMember matches service messages for a member leaving or being
// removed from a group.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("a member left the group")
//	}, telegram.LeftChatMember)
var LeftChatMember Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupRemoveMember
}

// NewChatTitle matches service messages for group title changes.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Printf("group renamed")
//	}, telegram.NewChatTitle)
var NewChatTitle Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupEditTitle
}

// NewChatPhoto matches service messages for group photo changes.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("group photo changed")
//	}, telegram.NewChatPhoto)
var NewChatPhoto Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupEditPhoto
}

// DeleteChatPhoto matches service messages for group photo deletion.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("group photo removed")
//	}, telegram.DeleteChatPhoto)
var DeleteChatPhoto Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupDeletePhoto
}

// GroupChatCreated matches service messages for basic group creation events.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("New group created!")
//	}, telegram.GroupChatCreated)
var GroupChatCreated Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupCreate
}

// SupergroupChatCreated matches service messages for supergroup creation
// (when a basic group is upgraded to a supergroup).
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("group upgraded to supergroup")
//	}, telegram.SupergroupChatCreated)
var SupergroupChatCreated Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionChannelMigrateFrom
}

// ChannelChatCreated matches service messages for channel creation events.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("new channel created")
//	}, telegram.ChannelChatCreated)
var ChannelChatCreated Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionChannelCreate
}

// MigrateToChatID matches service messages for group-to-supergroup migration.
// The new supergroup ChatID is embedded in the service message.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("chat migrating to supergroup")
//	}, telegram.MigrateToChatID)
var MigrateToChatID Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGroupMigrateTo
}

// MigrateFromChatID matches service messages for channel-from-group migration.
// Fires when a channel is converted from a basic group.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("migration from group completed")
//	}, telegram.MigrateFromChatID)
var MigrateFromChatID Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionChannelMigrateFrom
}

// PinnedMessage matches service messages for message pin events. Note:
// this targets the pin notification itself, not the pinned message content.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("a message was pinned")
//	}, telegram.PinnedMessage)
var PinnedMessage Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionPinMessage
}

// GameHighScore matches service messages for game score updates. Fired when
// a player's high score changes in an inline game.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("game high score updated")
//	}, telegram.GameHighScore)
var GameHighScore Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.Service != nil &&
		ctx.Message.Service.Type == types.ServiceActionGameScore
}

// VideoChatStarted matches service messages for video chat (group call) start.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("video chat started")
//	}, telegram.VideoChatStarted)
var VideoChatStarted Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.VideoChatStarted != nil
}

// VideoChatEnded matches service messages for video chat (group call) end.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("video chat ended")
//	}, telegram.VideoChatEnded)
var VideoChatEnded Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.VideoChatEnded != nil
}

// VideoChatMembersInvited matches service messages for video chat invitations.
// Fired when participants are invited to an ongoing group call.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("members invited to video chat")
//	}, telegram.VideoChatMembersInvited)
var VideoChatMembersInvited Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.VideoChatMembersInvited != nil
}

// SuccessfulPayment matches messages confirming a completed payment.
// Contains the transaction details in the invoice media.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("payment confirmed")
//	}, telegram.SuccessfulPayment)
var SuccessfulPayment Filter = func(ctx *Context) bool {
	return mediaTypeIs(ctx, types.MessageMediaTypeInvoice)
}

// ReplyKeyboard matches messages that carry a reply keyboard markup.
// Use to detect when the bot's own reply keyboard is shown.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("message has a reply keyboard")
//	}, telegram.ReplyKeyboard)
var ReplyKeyboard Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ReplyMarkup != nil &&
		ctx.Message.ReplyMarkup.Type == types.ReplyMarkupKeyboard
}

// InlineKeyboard matches messages that carry an inline keyboard markup.
// Use to detect messages that have interactive inline buttons attached.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("message has inline buttons")
//	}, telegram.InlineKeyboard)
var InlineKeyboard Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ReplyMarkup != nil &&
		ctx.Message.ReplyMarkup.Type == types.ReplyMarkupInline
}

// SelfDestruction matches messages with self-destructing media (secret chat
// media with a TTL). After the timer expires the media becomes inaccessible.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    log.Print("self-destructing media received")
//	}, telegram.SelfDestruction)
var SelfDestruction Filter = func(ctx *Context) bool {
	if ctx.Message == nil || ctx.Message.Media == nil {
		return false
	}
	switch m := ctx.Message.Media.(type) {
	case *types.PhotoMedia:
		return m.TTLSeconds > 0
	case *types.DocumentMedia:
		return m.TTLSeconds > 0
	}
	return false
}

// Direct matches incoming private (1-on-1) chat messages only.
// Combines Private and Incoming: positive ChatID and Out is false.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Hello in private!")
//	}, telegram.Direct)
var Direct Filter = func(ctx *Context) bool {
	return ctx.Message != nil && ctx.Message.ChatID > 0 && !ctx.Message.Out
}

// Quote is a placeholder for quoted message matching. Always returns false
// in the current implementation. Reserved for future use.
var Quote Filter = func(ctx *Context) bool { return false }

// Admin is a placeholder for admin user matching. Always returns false
// in the current implementation. Reserved for future use.
var Admin Filter = func(ctx *Context) bool { return false }

// User returns a filter that matches updates from one of the specified user IDs.
// It checks Message.FromID, CallbackQuery.UserID, and InlineQuery.UserID.
//
// Example:
//
//	// Only respond to messages from specific admins
//	client.OnMessage(handleAdmin, telegram.User(12345678, 87654321))
func User(userIDs ...int64) Filter {
	allowed := make(map[int64]bool, len(userIDs))
	for _, id := range userIDs {
		allowed[id] = true
	}
	return func(ctx *Context) bool {
		if ctx.Message != nil {
			return allowed[ctx.Message.FromID]
		}
		if ctx.CallbackQuery != nil {
			return allowed[ctx.CallbackQuery.UserID]
		}
		if ctx.InlineQuery != nil {
			return allowed[ctx.InlineQuery.UserID]
		}
		return false
	}
}

// Chat returns a filter that matches updates from one of the specified chat IDs.
// It checks Message.ChatID, EditedMessage.ChatID, and CallbackQuery.ChatID.
//
// Example:
//
//	client.OnMessage(handleSpecificGroup, telegram.Chat(-1001234567890))
func Chat(chatIDs ...int64) Filter {
	allowed := make(map[int64]bool, len(chatIDs))
	for _, id := range chatIDs {
		allowed[id] = true
	}
	return func(ctx *Context) bool {
		if ctx.Message != nil {
			return allowed[ctx.Message.ChatID]
		}
		if ctx.EditedMessage != nil {
			return allowed[ctx.EditedMessage.ChatID]
		}
		if ctx.CallbackQuery != nil {
			return allowed[ctx.CallbackQuery.ChatID]
		}
		return false
	}
}

// Topic returns a filter that matches messages from specific forum topics.
// Topic IDs are set by the supergroup when topics (threads) are enabled.
//
// Parameters:
//   - topicIDs: one or more forum topic IDs to match.
//
// Example:
//
//	client.OnMessage(handleAnnouncements, telegram.Topic(42))
func Topic(topicIDs ...int32) Filter {
	allowed := make(map[int32]bool, len(topicIDs))
	for _, id := range topicIDs {
		allowed[id] = true
	}
	return func(ctx *Context) bool {
		return ctx.Message != nil && allowed[ctx.Message.TopicID]
	}
}

// SenderChat returns a filter that matches messages sent on behalf of a
// specific sender chat (e.g. channel posts forwarded into a linked group).
//
// Parameters:
//   - chatIDs: one or more sender chat IDs to match.
//
// Example:
//
//	client.OnMessage(handleChannelPost, telegram.SenderChat(-1001234567890))
func SenderChat(chatIDs ...int64) Filter {
	allowed := make(map[int64]bool, len(chatIDs))
	for _, id := range chatIDs {
		allowed[id] = true
	}
	return func(ctx *Context) bool {
		return ctx.Message != nil && ctx.Message.SenderChatID != 0 && allowed[ctx.Message.SenderChatID]
	}
}

// CallbackData returns a filter that matches callback queries whose data
// exactly equals the given string. Use for inline button handlers that
// dispatch on a fixed callback payload.
//
// Parameters:
//   - data: exact callback data string to match.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    // handle button with callback data "approve"
//	}, telegram.CallbackData("approve"))
func CallbackData(data string) Filter {
	return func(ctx *Context) bool {
		return ctx.CallbackQuery != nil && string(ctx.CallbackQuery.Data) == data
	}
}

// CallbackRegex returns a filter that matches callback queries whose data
// matches the given regular expression. Panics if the pattern is invalid.
//
// Parameters:
//   - pattern: regular expression to match against callback data bytes.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    // match callbacks like "page_1", "page_2", etc.
//	}, telegram.CallbackRegex(`^page_\d+$`))
func CallbackRegex(pattern string) Filter {
	re := regexp.MustCompile(pattern)
	return func(ctx *Context) bool {
		return ctx.CallbackQuery != nil && re.Match(ctx.CallbackQuery.Data)
	}
}

// InlineQueryText returns a filter that matches inline queries whose text
// exactly equals the given string. Useful for providing fixed inline results.
//
// Parameters:
//   - s: exact query text to match.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    // respond when user types exactly "search"
//	}, telegram.InlineQueryText("search"))
func InlineQueryText(s string) Filter {
	return func(ctx *Context) bool {
		return ctx.InlineQuery != nil && ctx.InlineQuery.Query == s
	}
}

// ChatActionFilter returns a placeholder for chat action updates.
// Always returns false in the current implementation. Reserved for future use.
//
// Parameters:
//   - chatID: chat ID to filter actions for (unused).
func ChatActionFilter(chatID int64) Filter {
	return func(ctx *Context) bool { return false }
}

// Create builds a custom filter from the provided function. The function
// receives both the Client and Context, enabling filters that need access
// to client methods (e.g. API calls, state lookups).
//
// Parameters:
//   - fn: predicate function receiving the Client and Context.
//
// Returns:
//   - A Filter wrapping the provided function.
//
// Example:
//
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("You are an admin!")
//	}, telegram.Create(func(c *telegram.Client, ctx *telegram.Context) bool {
//	    return isAdmin(c, ctx.Message.FromID)
//	}))
func Create(fn func(*Client, *Context) bool) Filter {
	return func(ctx *Context) bool {
		return fn(ctx.Client, ctx)
	}
}

// CommandFilter is a configurable command filter with custom prefixes and
// case sensitivity. Unlike the Command function which always uses "/" as the
// prefix and is case-insensitive, CommandFilter allows arbitrary prefixes
// (e.g. "!", ".") and optional case-sensitive matching.
//
// Example:
//
//	cf := &telegram.CommandFilter{
//	    prefixes:      []string{"/", "!"},
//	    caseSensitive: false,
//	    commands:      []string{"start", "help"},
//	}
//	client.OnMessage(func(ctx *telegram.Context) {
//	    ctx.Message.Reply("Command matched!")
//	}, cf.Check)
type CommandFilter struct {
	prefixes      []string
	caseSensitive bool
	commands      []string
}

// Check evaluates whether the context's message matches any of the configured
// commands. It strips the prefix, removes @bot suffixes, and compares the
// resulting command string against the allowed list.
//
// Parameters:
//   - ctx: the update context to evaluate.
//
// Returns:
//   - true if the message matches a configured command, false otherwise.
func (cf *CommandFilter) Check(ctx *Context) bool {
	if ctx.Message == nil || ctx.Message.Text == "" {
		return false
	}
	text := ctx.Message.Text
	if len(text) == 0 {
		return false
	}
	prefixOK := false
	for _, p := range cf.prefixes {
		if strings.HasPrefix(text, p) {
			prefixOK = true
			text = text[len(p):]
			break
		}
	}
	if !prefixOK {
		return false
	}
	if !cf.caseSensitive {
		text = strings.ToLower(text)
	}
	if idx := strings.IndexByte(text, '@'); idx > 0 {
		text = text[:idx]
	}
	if idx := strings.IndexByte(text, ' '); idx > 0 {
		text = text[:idx]
	}
	for _, cmd := range cf.commands {
		if text == cmd {
			return true
		}
	}
	return false
}

// NewCommand creates a configurable command filter and returns it as a Filter.
// Defaults to the "/" prefix when none is provided.
//
// Parameters:
//   - commands: list of command names to match (without the prefix).
//   - prefixes: list of valid prefix characters. Defaults to ["/"] if empty.
//   - caseSensitive: whether command matching should be case-sensitive.
//
// Returns:
//   - A Filter that uses CommandFilter.Check internally.
//
// Example:
//
//	// Match "!start" and "!START" with case-insensitive matching
//	client.OnMessage(handleStart, telegram.NewCommand(
//	    []string{"start"},
//	    []string{"!"},
//	    false,
//	))
func NewCommand(commands []string, prefixes []string, caseSensitive bool) Filter {
	cmds := commands
	if !caseSensitive {
		cmds = make([]string, len(commands))
		for i, c := range commands {
			cmds[i] = strings.ToLower(c)
		}
	}
	cf := &CommandFilter{
		prefixes:      prefixes,
		caseSensitive: caseSensitive,
		commands:      cmds,
	}
	if len(cf.prefixes) == 0 {
		cf.prefixes = []string{"/"}
	}
	return cf.Check
}

func hasMedia(ctx *Context) (types.Media, bool) {
	if ctx.Message == nil || ctx.Message.Media == nil {
		return nil, false
	}
	return ctx.Message.Media, true
}

func hasAnyMedia(ctx *Context) bool {
	_, ok := hasMedia(ctx)
	return ok
}

func mediaTypeIs(ctx *Context, want types.MessageMediaType) bool {
	media, ok := hasMedia(ctx)
	if !ok {
		return false
	}
	return media.MediaType() == want
}
