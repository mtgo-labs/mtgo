package types_test

import (
	"testing"

	"github.com/mtgo-labs/mtgo/telegram/types"
)

func TestChatType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.ChatType
		expected string
	}{
		{"private", types.ChatTypePrivate, "private"},
		{"bot", types.ChatTypeBot, "bot"},
		{"group", types.ChatTypeGroup, "group"},
		{"supergroup", types.ChatTypeSupergroup, "supergroup"},
		{"channel", types.ChatTypeChannel, "channel"},
		{"forum", types.ChatTypeForum, "forum"},
		{"direct", types.ChatTypeDirect, "direct"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.ChatType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		name     string
		value    types.ParseMode
		expected string
	}{
		{"default", types.ParseModeDefault, "default"},
		{"markdown", types.ParseModeMarkdown, "markdown"},
		{"html", types.ParseModeHTML, "html"},
		{"disabled", types.ParseModeDisabled, "disabled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.ParseMode.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestUserStatus(t *testing.T) {
	tests := []struct {
		name     string
		value    types.UserStatus
		expected string
	}{
		{"online", types.UserStatusOnline, "online"},
		{"offline", types.UserStatusOffline, "offline"},
		{"recently", types.UserStatusRecently, "recently"},
		{"last_week", types.UserStatusLastWeek, "last_week"},
		{"last_month", types.UserStatusLastMonth, "last_month"},
		{"long_ago", types.UserStatusLongAgo, "long_ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.UserStatus.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestChatMemberStatus(t *testing.T) {
	tests := []struct {
		name     string
		value    types.ChatMemberStatus
		expected string
	}{
		{"owner", types.ChatMemberStatusOwner, "owner"},
		{"administrator", types.ChatMemberStatusAdministrator, "administrator"},
		{"member", types.ChatMemberStatusMember, "member"},
		{"restricted", types.ChatMemberStatusRestricted, "restricted"},
		{"left", types.ChatMemberStatusLeft, "left"},
		{"banned", types.ChatMemberStatusBanned, "banned"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.ChatMemberStatus.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestPollType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.PollType
		expected string
	}{
		{"quiz", types.PollTypeQuiz, "quiz"},
		{"regular", types.PollTypeRegular, "regular"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.PollType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestStickerType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.StickerType
		expected string
	}{
		{"regular", types.StickerTypeRegular, "regular"},
		{"mask", types.StickerTypeMask, "mask"},
		{"custom_emoji", types.StickerTypeCustomEmoji, "custom_emoji"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.StickerType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestClientPlatform(t *testing.T) {
	tests := []struct {
		name     string
		value    types.ClientPlatform
		expected string
	}{
		{"android", types.ClientPlatformAndroid, "android"},
		{"ios", types.ClientPlatformIOS, "ios"},
		{"wp", types.ClientPlatformWP, "wp"},
		{"bb", types.ClientPlatformBB, "bb"},
		{"desktop", types.ClientPlatformDesktop, "desktop"},
		{"web", types.ClientPlatformWeb, "web"},
		{"ubp", types.ClientPlatformUBP, "ubp"},
		{"other", types.ClientPlatformOther, "other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.ClientPlatform.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestButtonStyle(t *testing.T) {
	tests := []struct {
		name     string
		value    types.ButtonStyle
		expected string
	}{
		{"default", types.ButtonStyleDefault, "default"},
		{"primary", types.ButtonStylePrimary, "primary"},
		{"danger", types.ButtonStyleDanger, "danger"},
		{"success", types.ButtonStyleSuccess, "success"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.ButtonStyle.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestBlockList(t *testing.T) {
	tests := []struct {
		name     string
		value    types.BlockList
		expected string
	}{
		{"main", types.BlockListMain, "main"},
		{"stories", types.BlockListStories, "stories"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.BlockList.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestMessageEntityType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.MessageEntityType
		expected string
	}{
		{"mention", types.MessageEntityTypeMention, "mention"},
		{"hashtag", types.MessageEntityTypeHashtag, "hashtag"},
		{"cashtag", types.MessageEntityTypeCashtag, "cashtag"},
		{"bot_command", types.MessageEntityTypeBotCommand, "bot_command"},
		{"url", types.MessageEntityTypeURL, "url"},
		{"email", types.MessageEntityTypeEmail, "email"},
		{"phone_number", types.MessageEntityTypePhoneNumber, "phone_number"},
		{"bold", types.MessageEntityTypeBold, "bold"},
		{"italic", types.MessageEntityTypeItalic, "italic"},
		{"underline", types.MessageEntityTypeUnderline, "underline"},
		{"strikethrough", types.MessageEntityTypeStrikethrough, "strikethrough"},
		{"spoiler", types.MessageEntityTypeSpoiler, "spoiler"},
		{"code", types.MessageEntityTypeCode, "code"},
		{"pre", types.MessageEntityTypePre, "pre"},
		{"blockquote", types.MessageEntityTypeBlockquote, "blockquote"},
		{"text_link", types.MessageEntityTypeTextLink, "text_link"},
		{"text_mention", types.MessageEntityTypeTextMention, "text_mention"},
		{"bank_card", types.MessageEntityTypeBankCard, "bank_card"},
		{"custom_emoji", types.MessageEntityTypeCustomEmoji, "custom_emoji"},
		{"date_time", types.MessageEntityTypeDateTime, "date_time"},
		{"unknown", types.MessageEntityTypeUnknown, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.MessageEntityType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestMessageMediaType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.MessageMediaType
		expected string
	}{
		{"unsupported", types.MessageMediaTypeUnsupported, "unsupported"},
		{"audio", types.MessageMediaTypeAudio, "audio"},
		{"document", types.MessageMediaTypeDocument, "document"},
		{"photo", types.MessageMediaTypePhoto, "photo"},
		{"sticker", types.MessageMediaTypeSticker, "sticker"},
		{"video", types.MessageMediaTypeVideo, "video"},
		{"animation", types.MessageMediaTypeAnimation, "animation"},
		{"voice", types.MessageMediaTypeVoice, "voice"},
		{"video_note", types.MessageMediaTypeVideoNote, "video_note"},
		{"contact", types.MessageMediaTypeContact, "contact"},
		{"location", types.MessageMediaTypeLocation, "location"},
		{"venue", types.MessageMediaTypeVenue, "venue"},
		{"poll", types.MessageMediaTypePoll, "poll"},
		{"web_page", types.MessageMediaTypeWebPage, "web_page"},
		{"dice", types.MessageMediaTypeDice, "dice"},
		{"game", types.MessageMediaTypeGame, "game"},
		{"giveaway", types.MessageMediaTypeGiveaway, "giveaway"},
		{"giveaway_winners", types.MessageMediaTypeGiveawayWinners, "giveaway_winners"},
		{"story", types.MessageMediaTypeStory, "story"},
		{"invoice", types.MessageMediaTypeInvoice, "invoice"},
		{"paid_media", types.MessageMediaTypePaidMedia, "paid_media"},
		{"checklist", types.MessageMediaTypeChecklist, "checklist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.MessageMediaType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestMessageServiceType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.MessageServiceType
		expected string
	}{
		{"unsupported", types.MessageServiceTypeUnsupported, "unsupported"},
		{"custom_action", types.MessageServiceTypeCustomAction, "custom_action"},
		{"new_chat_members", types.MessageServiceTypeNewChatMembers, "new_chat_members"},
		{"left_chat_member", types.MessageServiceTypeLeftChatMember, "left_chat_member"},
		{"chat_owner_left", types.MessageServiceTypeChatOwnerLeft, "chat_owner_left"},
		{"chat_owner_changed", types.MessageServiceTypeChatOwnerChanged, "chat_owner_changed"},
		{"new_chat_title", types.MessageServiceTypeNewChatTitle, "new_chat_title"},
		{"new_chat_photo", types.MessageServiceTypeNewChatPhoto, "new_chat_photo"},
		{"delete_chat_photo", types.MessageServiceTypeDeleteChatPhoto, "delete_chat_photo"},
		{"forum_topic_created", types.MessageServiceTypeForumTopicCreated, "forum_topic_created"},
		{"forum_topic_closed", types.MessageServiceTypeForumTopicClosed, "forum_topic_closed"},
		{"forum_topic_reopened", types.MessageServiceTypeForumTopicReopened, "forum_topic_reopened"},
		{"forum_topic_edited", types.MessageServiceTypeForumTopicEdited, "forum_topic_edited"},
		{"general_forum_topic_hidden", types.MessageServiceTypeGeneralForumTopicHidden, "general_forum_topic_hidden"},
		{"general_forum_topic_unhidden", types.MessageServiceTypeGeneralForumTopicUnhidden, "general_forum_topic_unhidden"},
		{"group_chat_created", types.MessageServiceTypeGroupChatCreated, "group_chat_created"},
		{"channel_chat_created", types.MessageServiceTypeChannelChatCreated, "channel_chat_created"},
		{"supergroup_chat_created", types.MessageServiceTypeSupergroupChatCreated, "supergroup_chat_created"},
		{"migrate_to_chat_id", types.MessageServiceTypeMigrateToChatID, "migrate_to_chat_id"},
		{"migrate_from_chat_id", types.MessageServiceTypeMigrateFromChatID, "migrate_from_chat_id"},
		{"pinned_message", types.MessageServiceTypePinnedMessage, "pinned_message"},
		{"game_high_score", types.MessageServiceTypeGameHighScore, "game_high_score"},
		{"giveaway_created", types.MessageServiceTypeGiveawayCreated, "giveaway_created"},
		{"giveaway_completed", types.MessageServiceTypeGiveawayCompleted, "giveaway_completed"},
		{"managed_bot_created", types.MessageServiceTypeManagedBotCreated, "managed_bot_created"},
		{"poll_option_added", types.MessageServiceTypePollOptionAdded, "poll_option_added"},
		{"poll_option_deleted", types.MessageServiceTypePollOptionDeleted, "poll_option_deleted"},
		{"premium_gift_code", types.MessageServiceTypePremiumGiftCode, "premium_gift_code"},
		{"gifted_premium", types.MessageServiceTypeGiftedPremium, "gifted_premium"},
		{"gifted_stars", types.MessageServiceTypeGiftedStars, "gifted_stars"},
		{"gifted_ton", types.MessageServiceTypeGiftedTon, "gifted_ton"},
		{"video_chat_started", types.MessageServiceTypeVideoChatStarted, "video_chat_started"},
		{"video_chat_ended", types.MessageServiceTypeVideoChatEnded, "video_chat_ended"},
		{"video_chat_scheduled", types.MessageServiceTypeVideoChatScheduled, "video_chat_scheduled"},
		{"video_chat_members_invited", types.MessageServiceTypeVideoChatMembersInvited, "video_chat_members_invited"},
		{"phone_call_started", types.MessageServiceTypePhoneCallStarted, "phone_call_started"},
		{"phone_call_ended", types.MessageServiceTypePhoneCallEnded, "phone_call_ended"},
		{"web_app_data", types.MessageServiceTypeWebAppData, "web_app_data"},
		{"users_shared", types.MessageServiceTypeUsersShared, "users_shared"},
		{"chat_shared", types.MessageServiceTypeChatShared, "chat_shared"},
		{"successful_payment", types.MessageServiceTypeSuccessfulPayment, "successful_payment"},
		{"refunded_payment", types.MessageServiceTypeRefundedPayment, "refunded_payment"},
		{"screenshot_taken", types.MessageServiceTypeScreenshotTaken, "screenshot_taken"},
		{"gift", types.MessageServiceTypeGift, "gift"},
		{"set_message_auto_delete_time", types.MessageServiceTypeSetMessageAutoDeleteTime, "set_message_auto_delete_time"},
		{"chat_boost", types.MessageServiceTypeChatBoost, "chat_boost"},
		{"connected_website", types.MessageServiceTypeConnectedWebsite, "connected_website"},
		{"write_access_allowed", types.MessageServiceTypeWriteAccessAllowed, "write_access_allowed"},
		{"contact_registered", types.MessageServiceTypeContactRegistered, "contact_registered"},
		{"proximity_alert_triggered", types.MessageServiceTypeProximityAlertTriggered, "proximity_alert_triggered"},
		{"history_cleared", types.MessageServiceTypeHistoryCleared, "history_cleared"},
		{"suggest_profile_photo", types.MessageServiceTypeSuggestProfilePhoto, "suggest_profile_photo"},
		{"suggest_birthday", types.MessageServiceTypeSuggestBirthday, "suggest_birthday"},
		{"chat_set_background", types.MessageServiceTypeChatSetBackground, "chat_set_background"},
		{"chat_set_theme", types.MessageServiceTypeChatSetTheme, "chat_set_theme"},
		{"giveaway_prize_stars", types.MessageServiceTypeGiveawayPrizeStars, "giveaway_prize_stars"},
		{"paid_messages_refunded", types.MessageServiceTypePaidMessagesRefunded, "paid_messages_refunded"},
		{"paid_messages_price_changed", types.MessageServiceTypePaidMessagesPriceChanged, "paid_messages_price_changed"},
		{"direct_message_price_changed", types.MessageServiceTypeDirectMessagePriceChanged, "direct_message_price_changed"},
		{"checklist_tasks_done", types.MessageServiceTypeChecklistTasksDone, "checklist_tasks_done"},
		{"checklist_tasks_added", types.MessageServiceTypeChecklistTasksAdded, "checklist_tasks_added"},
		{"upgraded_gift_purchase_offer", types.MessageServiceTypeUpgradedGiftPurchaseOffer, "upgraded_gift_purchase_offer"},
		{"upgraded_gift_purchase_offer_rejected", types.MessageServiceTypeUpgradedGiftPurchaseOfferRejected, "upgraded_gift_purchase_offer_rejected"},
		{"chat_has_protected_content_toggled", types.MessageServiceTypeChatHasProtectedContentToggled, "chat_has_protected_content_toggled"},
		{"chat_has_protected_content_disable_requested", types.MessageServiceTypeChatHasProtectedContentDisableRequested, "chat_has_protected_content_disable_requested"},
		{"suggested_post_approval_failed", types.MessageServiceTypeSuggestedPostApprovalFailed, "suggested_post_approval_failed"},
		{"suggested_post_approved", types.MessageServiceTypeSuggestedPostApproved, "suggested_post_approved"},
		{"suggested_post_declined", types.MessageServiceTypeSuggestedPostDeclined, "suggested_post_declined"},
		{"suggested_post_paid", types.MessageServiceTypeSuggestedPostPaid, "suggested_post_paid"},
		{"suggested_post_refunded", types.MessageServiceTypeSuggestedPostRefunded, "suggested_post_refunded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.MessageServiceType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

func TestMessageOriginType(t *testing.T) {
	tests := []struct {
		name     string
		value    types.MessageOriginType
		expected string
	}{
		{"channel", types.MessageOriginTypeChannel, "channel"},
		{"chat", types.MessageOriginTypeChat, "chat"},
		{"hidden_user", types.MessageOriginTypeHiddenUser, "hidden_user"},
		{"import", types.MessageOriginTypeImport, "import"},
		{"user", types.MessageOriginTypeUser, "user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.String(); got != tt.expected {
				t.Errorf("types.MessageOriginType.%s = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}
