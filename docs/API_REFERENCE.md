# mtgo API Reference

> mtgo (MTProto Go) is a Go Telegram MTProto client library.
> This reference is generated from the source tree by `cmd/apicoverage`; do not edit by hand.

- [Architecture](#architecture)
- [Quick start](#quick-start)
- [Client operations by domain](#client-operations-by-domain)
- [Bound methods by type](#bound-methods-by-type)
- [Option structs](#option-structs)

## Architecture

`telegram.Client` is composed of twenty domain method groups. Every group method is
promoted onto the client itself, so `client.SendMessage` and `client.MessageMethods.SendMessage`
are the same call; a static collision test guarantees no name is ever defined twice.

Updates arrive through handlers (the `telegram.Handler` interface, registered with
`AddHandler`, plus middleware) and carry typed payloads (Message, CallbackQuery, WidgetQuery, …).
Payload types constructed by the client carry a minimal binder interface, so their convenience
methods (Reply, Edit, Answer, …) work without holding a client reference — and every
convenience method has a `NameContext` explicit-context form.

Low-level needs are served by `client.Raw()` (typed raw RPC), `InvokeJSON`, and the
`tg` generated types (TL layer 233). The full generated inventory — including the
deliberately unwrapped TL-function list — lives in [api-coverage.md](api-coverage.md).

## Quick start

```go
client, _ := telegram.NewClient(apiID, "api-hash", &telegram.Config{...})
_ = client.Connect(30 * time.Second)

// Phone-code sign-in.
sent, _ := client.SendPhoneNumberCode(ctx, "+1234567890")
user, err := client.SignIn(ctx, "+1234567890", sent.PhoneCodeHash, "12345")
if errors.Is(err, telegram.Err2FARequired) {
    user, err = client.CheckPassword(ctx, "cloud-password")
}

// Send media and text; optional parameters live in params structs.
msg, _ := client.SendMessage(ctx, chatID, "hello",
    &params.SendMessage{ReplyToMessageID: 42})
client.SendPhoto(ctx, chatID, &telegram.InputFile{Path: "cat.jpg"}, "caption")

// Bound convenience methods.
_ = msg.Reply("right back")
updated, _ := msg.EditWidgetData(`{"v":2}`, "")
```

## Client operations by domain

### Account

account-level settings — `*AccountMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AcceptTermsOfService` | `AcceptTermsOfService(ctx context.Context, terms *tg.HelpTermsOfService) (error)` | AcceptTermsOfService accepts a pending terms-of-service revision, as returned by GetTermsOfServiceUpdate. |
| `GetAccountTTL` | `GetAccountTTL(ctx context.Context) (int32, error)` | GetAccountTTL returns the number of days after which the account will self-destruct if the user does not log in. |
| `GetGlobalPrivacySettings` | `GetGlobalPrivacySettings(ctx context.Context) (*tg.GlobalPrivacySettings, error)` | GetGlobalPrivacySettings returns the current global privacy settings for the account. |
| `GetPrivacy` | `GetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass) ([]tg.PrivacyRuleClass, error)` | GetPrivacy returns the current privacy rules for the given privacy key. |
| `GetTermsOfServiceUpdate` | `GetTermsOfServiceUpdate(ctx context.Context) (*tg.HelpTermsOfService, error)` | GetTermsOfServiceUpdate returns the pending terms-of-service revision the logged-in user has not accepted yet, or nil when the current terms are already accepted. |
| `SetAccountTTL` | `SetAccountTTL(ctx context.Context, days int32) (error)` | SetAccountTTL sets the number of days after which the account will self-destruct if the user does not log in. |
| `SetGlobalPrivacySettings` | `SetGlobalPrivacySettings(ctx context.Context, settings *tg.GlobalPrivacySettings) (error)` | SetGlobalPrivacySettings applies the given global privacy settings to the account. |
| `SetPrivacy` | `SetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass, rules []tg.InputPrivacyRuleClass) (error)` | SetPrivacy sets the privacy rules for the given privacy key. |

### Advanced

raw RPC escape hatches — `*AdvancedMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `ConnectionSnapshot` | `ConnectionSnapshot() (ConnectionSnapshot)` | ConnectionSnapshot returns transport and connection lifecycle diagnostics. |
| `InvokeJSON` | `InvokeJSON(ctx context.Context, functionName string, payload []byte, useSnakeCase bool) ([]byte, error)` | InvokeJSON sends a raw Telegram API call encoded as JSON. |
| `LoadSnapshot` | `LoadSnapshot() (LoadSnapshot)` | LoadSnapshot returns an aggregated view of the client's internal load state. |
| `RPC` | `RPC() (*tg.RPCClient)` | RPC is an alias for [Client.Raw]. |
| `Raw` | `Raw() (*tg.RPCClient)` | Raw returns a low-level tg.RPCClient that sends TL objects directly to the Telegram server. |

### Auth

authorization — `*AuthMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `ChangePhoneNumber` | `ChangePhoneNumber(ctx context.Context, phoneNumber, phoneCodeHash, phoneCode string) (*types.User, error)` | ChangePhoneNumber completes a phone-number change with the code delivered to the new number and returns the updated user. |
| `CheckPassword` | `CheckPassword(ctx context.Context, password string) (*types.User, error)` | CheckPassword completes authentication for an account that has two-factor authentication (2FA) enabled. |
| `CheckQRCodeLoginToken` | `CheckQRCodeLoginToken(ctx context.Context, token []byte) (*types.User, error)` | CheckQRCodeLoginToken checks whether the given token has been scanned by an authenticated Telegram client. |
| `GetActiveSessions` | `GetActiveSessions(ctx context.Context) (*ActiveSessions, error)` | GetActiveSessions retrieves all active sessions for the currently authorized account. |
| `GetPasswordHint` | `GetPasswordHint(ctx context.Context) (string, error)` | GetPasswordHint retrieves the optional hint string configured for the account's two-factor authentication password. |
| `GetQRCodeLoginToken` | `GetQRCodeLoginToken(ctx context.Context) (*QRLoginToken, error)` | GetQRCodeLoginToken requests a new QR code login token from Telegram. |
| `RecoverPassword` | `RecoverPassword(ctx context.Context, code string) (*types.User, error)` | RecoverPassword authenticates the user using a recovery code sent to the account's recovery email address. |
| `ResendPhoneNumberCode` | `ResendPhoneNumberCode(ctx context.Context, phoneNumber, phoneCodeHash, reason string) (*SendCodeResult, error)` | ResendPhoneNumberCode re-requests the verification code for an ongoing sign-in, typically to deliver it through the next code type (for example from SMS to a phone call). |
| `ResetSession` | `ResetSession(ctx context.Context, hash int64) (error)` | ResetSession terminates a specific session identified by its hash. |
| `SendChangePhoneNumberCode` | `SendChangePhoneNumberCode(ctx context.Context, phoneNumber string) (*SendCodeResult, error)` | SendChangePhoneNumberCode starts a phone-number change by requesting a verification code for the new number. |
| `SendPhoneNumberCode` | `SendPhoneNumberCode(ctx context.Context, phoneNumber string) (*SendCodeResult, error)` | SendPhoneNumberCode requests Telegram to send a verification code to the given phone number. |
| `SignIn` | `SignIn(ctx context.Context, phoneNumber, phoneCodeHash, phoneCode string) (*types.User, error)` | SignIn authenticates a user with the phone number, verification code, and the phone code hash returned by SendPhoneNumberCode. |
| `SignOut` | `SignOut(ctx context.Context) (bool, error)` | SignOut logs the current session out of Telegram and disconnects the client from the server. |
| `SignUp` | `SignUp(ctx context.Context, phoneNumber, phoneCodeHash, firstName string, lastName ...string) (*types.User, error)` | SignUp registers a new Telegram account for a phone number that is not yet associated with any account. |

### Bots

bot surface — `*BotsMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AllowBotToMessage` | `AllowBotToMessage(ctx context.Context, botID int64) (error)` | AllowBotToMessage grants the given business bot permission to message the logged-in user. |
| `AnswerCallbackQuery` | `AnswerCallbackQuery(ctx context.Context, callbackQueryID int64, text string, showAlert bool, url string, cacheTime int) (error)` | AnswerCallbackQuery sends an answer to a callback query originated from an inline button press. |
| `AnswerGuestQuery` | `AnswerGuestQuery(ctx context.Context, guestQueryID string, result tg.InputBotInlineResultClass) (*types.SentGuestMessage, error)` | AnswerGuestQuery sends an inline result as a reply to a received guest message. |
| `AnswerInlineQuery` | `AnswerInlineQuery(ctx context.Context, queryID int64, results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) (error)` | AnswerInlineQuery sends results for an inline query originated from the @bot search box. |
| `AnswerWebAppQuery` | `AnswerWebAppQuery(ctx context.Context, queryID string, result tg.InputBotInlineResultClass) (*tg.WebViewMessageSent, error)` | AnswerWebAppQuery sends the result of an interaction with a Web App to the Telegram server. |
| `AnswerWidgetQuery` | `AnswerWidgetQuery(ctx context.Context, queryID int64, result string) (error)` | AnswerWidgetQuery answers a widget query issued by a widget message of the logged-in bot. |
| `CanMessageBot` | `CanMessageBot(ctx context.Context, botID int64) (bool, error)` | CanMessageBot reports whether the logged-in user may message the given business bot. |
| `DeleteBotCommands` | `DeleteBotCommands(ctx context.Context, scope tg.BotCommandScopeClass, langCode string) (error)` | DeleteBotCommands resets (deletes) all bot commands for the given scope and language code. |
| `GetBotCommands` | `GetBotCommands(ctx context.Context, langCode string) ([]*tg.BotCommand, error)` | GetBotCommands retrieves the current list of bot commands for the specified language code. |
| `GetBotInfoDescription` | `GetBotInfoDescription(ctx context.Context, langCode string) (string, error)` | GetBotInfoDescription retrieves the bot's long description text for the specified language code. |
| `GetBotInfoShortDescription` | `GetBotInfoShortDescription(ctx context.Context, langCode string) (string, error)` | GetBotInfoShortDescription retrieves the bot's short "about" text for the specified language code. |
| `GetBotName` | `GetBotName(ctx context.Context, langCode string) (string, error)` | GetBotName retrieves the bot's display name for the specified language code. |
| `GetBotRecommendations` | `GetBotRecommendations(ctx context.Context, botID int64) ([]*types.User, error)` | GetBotRecommendations returns bots recommended alongside the given bot. |
| `GetBotUserMenuButton` | `GetBotUserMenuButton(ctx context.Context, userID int64) (tg.BotMenuButtonClass, error)` | GetBotUserMenuButton returns the menu button a specific user sees for the logged-in bot. |
| `GetChatMenuButton` | `GetChatMenuButton(ctx context.Context, userID int64) (tg.BotMenuButtonClass, error)` | GetChatMenuButton returns the current menu button configured for the specified user. |
| `GetGameHighScores` | `GetGameHighScores(ctx context.Context, chatID int64, messageID int64, userID int64) ([]*tg.HighScore, error)` | GetGameHighScores returns the high score table for a game message. |
| `GetInlineBotResults` | `GetInlineBotResults(ctx context.Context, bot int64, chatID int64, query, offset string) (*tg.MessagesBotResults, error)` | GetInlineBotResults requests inline results from a bot for the given query. |
| `GetOwnedBots` | `GetOwnedBots(ctx context.Context) ([]*types.User, error)` | GetOwnedBots lists the bots created by the logged-in user. |
| `GetPopularAppBots` | `GetPopularAppBots(ctx context.Context, offset string, limit int) ([]*types.User, string, error)` | GetPopularAppBots searches popular mini-app bots by query. |
| `InvokeWebViewCustomMethod` | `InvokeWebViewCustomMethod(ctx context.Context, botID int64, method, params string) (string, error)` | InvokeWebViewCustomMethod invokes a custom mini-app method on behalf of the logged-in mini app with the given bot. |
| `ReorderBotUsernames` | `ReorderBotUsernames(ctx context.Context, botID int64, order []string) (error)` | ReorderBotUsernames changes the display order of an owned bot's active usernames. |
| `RequestCallbackAnswer` | `RequestCallbackAnswer(ctx context.Context, chatID int64, messageID int64, data []byte) (*tg.MessagesBotCallbackAnswer, error)` | RequestCallbackAnswer requests the bot's callback answer for a message in a chat. |
| `SendInlineBotResult` | `SendInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, opts ...*SendInlineBotResultOption) (*types.Message, error)` | SendInlineBotResult sends a specific inline bot result to a chat. |
| `SetBotCommands` | `SetBotCommands(ctx context.Context, scope tg.BotCommandScopeClass, langCode string, commands []*tg.BotCommand) (error)` | SetBotCommands sets the list of bot commands for the given scope and language code. |
| `SetBotDefaultAdminRights` | `SetBotDefaultAdminRights(ctx context.Context, forChannels bool, rights *tg.ChatAdminRights) (error)` | SetBotDefaultAdminRights sets the default administrator rights the logged-in bot requests when added to channels (forChannels true) or groups. |
| `SetBotInfoDescription` | `SetBotInfoDescription(ctx context.Context, langCode, description string) (error)` | SetBotInfoDescription sets the bot's long description text for the specified language code. |
| `SetBotInfoShortDescription` | `SetBotInfoShortDescription(ctx context.Context, langCode, description string) (error)` | SetBotInfoShortDescription sets the bot's short "about" text for the specified language code. |
| `SetBotName` | `SetBotName(ctx context.Context, langCode, name string) (error)` | SetBotName sets the bot's display name for the specified language code. |
| `SetBotUserMenuButton` | `SetBotUserMenuButton(ctx context.Context, userID int64, button tg.BotMenuButtonClass) (error)` | SetBotUserMenuButton sets the menu button a specific user sees for the logged-in bot. |
| `SetChatMenuButton` | `SetChatMenuButton(ctx context.Context, userID int64, button tg.BotMenuButtonClass) (error)` | SetChatMenuButton sets the menu button for the specified user in the bot's chat. |
| `SetGameScore` | `SetGameScore(ctx context.Context, chatID int64, messageID int64, userID int64, score int, force bool) (*types.Message, error)` | SetGameScore sets the game score for the specified user in a game message. |
| `ToggleBotUsername` | `ToggleBotUsername(ctx context.Context, botID int64, username string, active bool) (error)` | ToggleBotUsername activates or deactivates one of an owned bot's usernames. |

### Business

business profile and links — `*BusinessMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `CreateBusinessChatLink` | `CreateBusinessChatLink(ctx context.Context, link *tg.InputBusinessChatLink) (*tg.BusinessChatLink, error)` | CreateBusinessChatLink creates a new chat link for the logged-in business account from the given link definition. |
| `DeleteBusinessChatLink` | `DeleteBusinessChatLink(ctx context.Context, slug string) (error)` | DeleteBusinessChatLink deletes a business chat link by its slug. |
| `EditBusinessChatLink` | `EditBusinessChatLink(ctx context.Context, slug string, link *tg.InputBusinessChatLink) (*tg.BusinessChatLink, error)` | EditBusinessChatLink replaces the definition of an existing business chat link, identified by its slug. |
| `GetBusinessChatLinks` | `GetBusinessChatLinks(ctx context.Context) ([]*tg.BusinessChatLink, error)` | GetBusinessChatLinks lists the chat links configured for the logged-in business account. |
| `GetBusinessConnection` | `GetBusinessConnection(ctx context.Context, connectionID string) (*tg.BotBusinessConnection, error)` | GetBusinessConnection retrieves a business connection by its ID. |
| `ResolveBusinessChatLink` | `ResolveBusinessChatLink(ctx context.Context, slug string) (*tg.AccountResolvedBusinessChatLinks, error)` | ResolveBusinessChatLink resolves a business chat link slug and returns the destination peer and greeting message. |
| `SetBusinessAwayMessage` | `SetBusinessAwayMessage(ctx context.Context, message *tg.InputBusinessAwayMessage) (error)` | SetBusinessAwayMessage sets the automated away message sent while the business account is closed. |
| `SetBusinessGreetingMessage` | `SetBusinessGreetingMessage(ctx context.Context, message *tg.InputBusinessGreetingMessage) (error)` | SetBusinessGreetingMessage sets the automated greeting message sent to new business chats. |
| `SetBusinessIntro` | `SetBusinessIntro(ctx context.Context, intro *tg.InputBusinessIntro) (error)` | SetBusinessIntro sets the intro new customers see when opening the business chat. |
| `SetBusinessLocation` | `SetBusinessLocation(ctx context.Context, address string, lat, lng float64) (error)` | SetBusinessLocation sets the business location shown on the logged-in user's profile. |
| `SetBusinessWorkHours` | `SetBusinessWorkHours(ctx context.Context, hours *tg.BusinessWorkHours) (error)` | SetBusinessWorkHours sets the logged-in user's business opening hours shown on their profile. |

### Chat

chat and channel management — `*ChatMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AddChatMember` | `AddChatMember(ctx context.Context, chatID int64, userID int64) (error)` | AddChatMember adds a user to a channel or supergroup. |
| `ArchiveChat` | `ArchiveChat(ctx context.Context, chatID int64) (error)` | ArchiveChat moves the specified chat to the archived chats folder (folder ID 1). |
| `BanChatMember` | `BanChatMember(ctx context.Context, chatID int64, userID int64) (error)` | BanChatMember bans a user from a channel or supergroup, revoking all standard permissions (view messages, send media, embed links, etc.). |
| `CloseForumTopic` | `CloseForumTopic(ctx context.Context, chatID int64, topicID int32) (error)` |  |
| `CreateChannel` | `CreateChannel(ctx context.Context, title, about string, megagroup bool) (*types.Chat, error)` | CreateChannel creates a new channel or megagroup (supergroup). |
| `CreateForumTopic` | `CreateForumTopic(ctx context.Context, chatID int64, title string, iconColor *int32, iconEmojiID *int64) (*types.ForumTopic, error)` |  |
| `CreateGroup` | `CreateGroup(ctx context.Context, title string, userIDs []int64) (*types.Chat, error)` | CreateGroup creates a new basic group chat with the specified title and initial members. |
| `CreateSupergroup` | `CreateSupergroup(ctx context.Context, title, about string) (*types.Chat, error)` | CreateSupergroup creates a new supergroup (megagroup) with the given title and description. |
| `DeleteChat` | `DeleteChat(ctx context.Context, chatID int64) (error)` | DeleteChat deletes the specified chat or channel. |
| `DeleteChatPhoto` | `DeleteChatPhoto(ctx context.Context, chatID int64) (error)` | DeleteChatPhoto removes the photo from a chat or channel by setting it to empty. |
| `DeleteForumTopic` | `DeleteForumTopic(ctx context.Context, chatID int64, topicID int32) (error)` |  |
| `EditForumTopic` | `EditForumTopic(ctx context.Context, chatID int64, topicID int32, title *string, iconEmojiID *int64) (error)` |  |
| `GetChat` | `GetChat(ctx context.Context, chatID int64) (*types.Chat, error)` | GetChat retrieves full information about a chat or channel. |
| `GetChatEventLog` | `GetChatEventLog(ctx context.Context, chatID int64, query string, limit int) ([]*types.ChatEvent, error)` | GetChatEventLog retrieves administrative events from a channel or supergroup's audit log. |
| `GetChatMember` | `GetChatMember(ctx context.Context, chatID int64, userID int64) (*types.ChatMember, error)` | GetChatMember retrieves information about a single participant in a channel or supergroup. |
| `GetChatMembers` | `GetChatMembers(ctx context.Context, chatID int64, limit int, offset int) ([]*types.ChatMember, error)` | GetChatMembers retrieves a paginated list of participants in a channel or supergroup. |
| `GetChatMembersCount` | `GetChatMembersCount(ctx context.Context, chatID int64) (int, error)` | GetChatMembersCount returns the total number of participants in a channel or supergroup. |
| `GetChatNotificationSettings` | `GetChatNotificationSettings(ctx context.Context, chatID int64) (*tg.PeerNotifySettings, error)` | GetChatNotificationSettings returns the current notification settings of a chat. |
| `GetChatOnlineCount` | `GetChatOnlineCount(ctx context.Context, chatID int64) (int, error)` |  |
| `GetForumTopics` | `GetForumTopics(ctx context.Context, chatID int64, query string, limit int, offsetDate int32, offsetTopic int32) ([]*types.ForumTopic, error)` |  |
| `GetFullChannel` | `GetFullChannel(ctx context.Context, chatID int64) (*types.Chat, error)` |  |
| `GetSendAsChats` | `GetSendAsChats(ctx context.Context, chatID int64) ([]*types.Chat, error)` |  |
| `GetSimilarChannels` | `GetSimilarChannels(ctx context.Context, chatID int64) ([]*types.Chat, error)` | GetSimilarChannels returns channels similar to the given channel. |
| `GetSuitableDiscussionChats` | `GetSuitableDiscussionChats(ctx context.Context) ([]*types.Chat, error)` |  |
| `GetTopChats` | `GetTopChats(ctx context.Context, category TopChatCategory, limit int) ([]*types.Chat, error)` | GetTopChats returns the most frequently contacted chats in a category, ordered by interaction frequency. |
| `JoinChat` | `JoinChat(ctx context.Context, link string) (*types.Chat, error)` | JoinChat joins a chat or channel using a username, invite link, or invite hash. |
| `LeaveChat` | `LeaveChat(ctx context.Context, chatID int64) (error)` | LeaveChat leaves the specified chat or channel. |
| `MarkChatUnread` | `MarkChatUnread(ctx context.Context, chatID int64, unread bool) (error)` | MarkChatUnread toggles the unread status of a chat in the current user's chat list. |
| `MuteChat` | `MuteChat(ctx context.Context, chatID int64) (error)` | MuteChat disables all notifications for the specified chat indefinitely. |
| `PromoteChatMember` | `PromoteChatMember(ctx context.Context, chatID int64, userID int64, adminRights *tg.ChatAdminRights) (error)` | PromoteChatMember promotes a user to administrator in a channel with the specified admin rights. |
| `ReopenForumTopic` | `ReopenForumTopic(ctx context.Context, chatID int64, topicID int32) (error)` |  |
| `RestrictChatMember` | `RestrictChatMember(ctx context.Context, chatID int64, userID int64, bannedRights *tg.ChatBannedRights) (error)` | RestrictChatMember applies the given banned rights to a user in a channel or supergroup. |
| `SetAdministratorTitle` | `SetAdministratorTitle(ctx context.Context, chatID int64, userID int64, title string) (error)` | SetAdministratorTitle sets the custom administrator title (rank) displayed next to a user's name in a channel or supergroup. |
| `SetChatAccentColor` | `SetChatAccentColor(ctx context.Context, chatID int64, color int32, backgroundEmojiID int64, forProfile bool) (error)` | SetChatAccentColor sets the accent color palette index and background emoji of a channel. |
| `SetChatDescription` | `SetChatDescription(ctx context.Context, chatID int64, about string) (error)` | SetChatDescription sets the description (about text) of a chat or channel. |
| `SetChatDiscussionGroup` | `SetChatDiscussionGroup(ctx context.Context, broadcastChatID int64, groupChatID int64) (error)` |  |
| `SetChatEmojiStatus` | `SetChatEmojiStatus(ctx context.Context, chatID int64, customEmojiID int64) (error)` | SetChatEmojiStatus sets the custom emoji displayed next to a channel's title. |
| `SetChatNotificationSettings` | `SetChatNotificationSettings(ctx context.Context, chatID int64, settings *tg.InputPeerNotifySettings) (error)` | SetChatNotificationSettings replaces the notification settings of a chat. |
| `SetChatPermissions` | `SetChatPermissions(ctx context.Context, chatID int64, permissions *tg.ChatBannedRights) (error)` | SetChatPermissions sets the default banned rights for all members of a chat or channel. |
| `SetChatPhoto` | `SetChatPhoto(ctx context.Context, chatID int64, photo tg.InputChatPhotoClass) (error)` | SetChatPhoto sets or updates the photo for a chat or channel. |
| `SetChatTTL` | `SetChatTTL(ctx context.Context, chatID int64, ttl int) (error)` | SetChatTTL sets the Time-To-Live (auto-delete) period for messages in a chat. |
| `SetChatTitle` | `SetChatTitle(ctx context.Context, chatID int64, title string) (error)` | SetChatTitle sets the title of a chat or channel. |
| `SetChatUsername` | `SetChatUsername(ctx context.Context, chatID int64, username string) (error)` | SetChatUsername sets the public username of a channel. |
| `SetProtectedContent` | `SetProtectedContent(ctx context.Context, chatID int64, enabled bool) (error)` | SetProtectedContent toggles the "protected content" flag on a channel or supergroup. |
| `SetSendAsChat` | `SetSendAsChat(ctx context.Context, chatID int64, sendAs int64) (error)` |  |
| `SetSlowMode` | `SetSlowMode(ctx context.Context, chatID int64, seconds int) (error)` | SetSlowMode configures the slow mode delay for a channel or supergroup. |
| `TransferChatOwnership` | `TransferChatOwnership(ctx context.Context, chatID int64, userID int64, password string) (error)` |  |
| `UnarchiveChat` | `UnarchiveChat(ctx context.Context, chatID int64) (error)` | UnarchiveChat moves the specified chat from the archived folder back to the main chat list (folder ID 0). |
| `UnbanChatMember` | `UnbanChatMember(ctx context.Context, chatID int64, userID int64) (error)` | UnbanChatMember removes all previously applied ban restrictions from a user in a channel or supergroup, restoring their default permissions. |
| `UnmuteChat` | `UnmuteChat(ctx context.Context, chatID int64) (error)` | UnmuteChat re-enables notifications for the specified chat by setting the mute-until value to 0 (no mute). |

### Contacts

contact book — `*ContactsMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AddContact` | `AddContact(ctx context.Context, userID int64, firstName, lastName, phone string, share bool) (error)` | AddContact adds a user to the contact list using the provided first name, last name, and phone number. |
| `DeleteContacts` | `DeleteContacts(ctx context.Context, userIDs []int64) (error)` | DeleteContacts removes the specified users from the contact list. |
| `GetContacts` | `GetContacts(ctx context.Context, hash int64) (tg.ContactsClass, error)` | GetContacts retrieves the current user's contact list. |
| `SearchContacts` | `SearchContacts(ctx context.Context, query string, limit int) (*SearchResult, error)` | SearchContacts searches the user's Telegram contacts and global user database for the given query string. |
| `SetContactNote` | `SetContactNote(ctx context.Context, userID int64, note string) (error)` | SetContactNote attaches a short private note to a contact, visible only to the logged-in user. |

### Folder

chat folders — `*FolderMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `CheckFolderInviteLink` | `CheckFolderInviteLink(ctx context.Context, slug string) (tg.ChatlistInviteClass, error)` | CheckFolderInviteLink inspects a folder invite link without joining it. |
| `CreateFolder` | `CreateFolder(ctx context.Context, filter *tg.DialogFilter) (error)` | CreateFolder creates a new chat folder from the given filter definition. |
| `CreateFolderInviteLink` | `CreateFolderInviteLink(ctx context.Context, folderID int32, title string, chatIDs []int64) (*types.FolderInviteLink, error)` | CreateFolderInviteLink exports a new invite link for a shareable folder. |
| `DeleteFolder` | `DeleteFolder(ctx context.Context, folderID int32) (error)` | DeleteFolder removes a chat folder by ID. |
| `DeleteFolderInviteLink` | `DeleteFolderInviteLink(ctx context.Context, folderID int32, slug string) (error)` | DeleteFolderInviteLink revokes and deletes an exported folder invite link. |
| `EditFolder` | `EditFolder(ctx context.Context, filter *tg.DialogFilter) (error)` | EditFolder saves changes to an existing chat folder. |
| `EditFolderInviteLink` | `EditFolderInviteLink(ctx context.Context, folderID int32, slug, title string, chatIDs []int64) (*types.FolderInviteLink, error)` | EditFolderInviteLink changes the title or shared-chat set of an exported folder invite link, identified by its slug. |
| `ExcludeChat` | `ExcludeChat(ctx context.Context, folderID int32, chatID int64) (error)` | ExcludeChat adds a chat to a folder's exclusion list, so the folder never shows it even when the folder's rules would match. |
| `GetFolderInviteLinks` | `GetFolderInviteLinks(ctx context.Context, folderID int32) ([]*types.FolderInviteLink, error)` | GetFolderInviteLinks lists the invite links exported for a shareable folder (chatlist). |
| `GetFolderUpdates` | `GetFolderUpdates(ctx context.Context, folderID int32) (*tg.ChatlistsChatlistUpdates, error)` | GetFolderUpdates lists chats added to a shareable folder since the last join, so the user can accept or hide them. |
| `GetFolders` | `GetFolders(ctx context.Context) ([]*types.Folder, error)` | GetFolders returns the user's chat folders. |
| `GetLeaveFolderSuggestions` | `GetLeaveFolderSuggestions(ctx context.Context, folderID int32) ([]int64, error)` | GetLeaveFolderSuggestions lists chats the server suggests also leaving when leaving a shareable folder. |
| `HideFolderUpdates` | `HideFolderUpdates(ctx context.Context, folderID int32) (error)` | HideFolderUpdates dismisses newly added chats of a shareable folder without joining them. |
| `IncludeChat` | `IncludeChat(ctx context.Context, folderID int32, chatID int64) (error)` | IncludeChat adds a chat to a folder's explicit include list, so the folder always shows it regardless of the folder's rules. |
| `JoinFolderInviteLink` | `JoinFolderInviteLink(ctx context.Context, slug string, chatIDs []int64) (error)` | JoinFolderInviteLink joins a shareable folder via its invite link. |
| `JoinFolderUpdates` | `JoinFolderUpdates(ctx context.Context, folderID int32, chatIDs []int64) (error)` | JoinFolderUpdates accepts newly added chats of a shareable folder. |
| `LeaveFolder` | `LeaveFolder(ctx context.Context, folderID int32, chatIDs []int64) (error)` | LeaveFolder leaves a shareable folder. |
| `PinChatInFolder` | `PinChatInFolder(ctx context.Context, folderID int32, chatID int64) (error)` | PinChatInFolder pins a chat inside a folder, keeping it at the top of the folder's chat list. |
| `RemoveChatFromFolder` | `RemoveChatFromFolder(ctx context.Context, folderID int32, chatID int64) (error)` | RemoveChatFromFolder removes every reference to a chat from a folder: pin, include, and exclude entries. |
| `ReorderFolders` | `ReorderFolders(ctx context.Context, order []int32) (error)` | ReorderFolders changes the order of chat folders. |
| `UpdateFolderColor` | `UpdateFolderColor(ctx context.Context, folderID int32, color int32) (error)` | UpdateFolderColor sets the accent color of a folder. |

### InviteLink

invite links and join requests — `*InviteLinkMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `ApproveChatJoinRequest` | `ApproveChatJoinRequest(ctx context.Context, chatID int64, userID int64) (error)` | ApproveChatJoinRequest approves a pending join request from a user for a chat that requires admin approval. |
| `CreateChatInviteLink` | `CreateChatInviteLink(ctx context.Context, chatID int64, opts ...*InviteLinkOption) (*types.ChatInviteLink, error)` | CreateChatInviteLink generates a new invite link for the specified chat. |
| `DeclineChatJoinRequest` | `DeclineChatJoinRequest(ctx context.Context, chatID int64, userID int64) (error)` | DeclineChatJoinRequest declines (rejects) a pending join request from a user. |
| `DeleteChatInviteLink` | `DeleteChatInviteLink(ctx context.Context, chatID int64, link string) (error)` | DeleteChatInviteLink permanently deletes an invite link. |
| `EditChatInviteLink` | `EditChatInviteLink(ctx context.Context, chatID int64, link string, opts ...*InviteLinkOption) (*types.ChatInviteLink, error)` | EditChatInviteLink modifies an existing invite link's properties (expiration, usage limit, title). |
| `ExportChatInviteLink` | `ExportChatInviteLink(ctx context.Context, chatID int64) (string, error)` | ExportChatInviteLink creates a new primary invite link for the specified chat and returns the link string. |
| `GetChatAdminInviteLinks` | `GetChatAdminInviteLinks(ctx context.Context, chatID int64, adminID int64, limit int) ([]*types.ChatInviteLink, error)` | GetChatAdminInviteLinks retrieves invite links created by a specific admin in the specified chat. |
| `GetChatInviteLink` | `GetChatInviteLink(ctx context.Context, chatID int64, link string) (*types.ChatInviteLink, error)` | GetChatInviteLink retrieves an existing exported invite link for the specified chat. |
| `GetChatInviteLinkJoiners` | `GetChatInviteLinkJoiners(ctx context.Context, chatID int64, link string, limit int) ([]*types.ChatInviteLinkJoiner, error)` | GetChatInviteLinkJoiners retrieves the list of users who joined the chat via the specified invite link. |
| `RevokeChatInviteLink` | `RevokeChatInviteLink(ctx context.Context, chatID int64, link string) (*types.ChatInviteLink, error)` | RevokeChatInviteLink revokes an existing invite link, preventing new users from joining with it. |

### Message

message lifecycle — `*MessageMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AddChecklistTasks` | `AddChecklistTasks(ctx context.Context, chatID int64, messageID int32, tasks []*tg.TodoItem) (error)` | AddChecklistTasks appends tasks to a checklist (todo-list) message. |
| `CopyMediaGroup` | `CopyMediaGroup(ctx context.Context, chatID int64, fromChatID int64, groupedID int64) ([]*types.Message, error)` | CopyMediaGroup copies an entire album (grouped media messages) from one chat to another as a new album without forwarding attribution. |
| `CopyMessage` | `CopyMessage(ctx context.Context, chatID int64, fromChatID int64, messageID int32, opts ...*params.CopyMessage) (int64, error)` | CopyMessage copies a single message from one chat to another without indicating it was forwarded. |
| `DeleteChatHistory` | `DeleteChatHistory(ctx context.Context, chatID int64, maxID int32, revoke bool) (int, error)` | DeleteChatHistory deletes the entire message history of a chat for the current user. |
| `DeleteMessages` | `DeleteMessages(ctx context.Context, chatID int64, messageIDs []int32, opts ...*params.DeleteMessages) (int, error)` | DeleteMessages removes the specified messages from a chat. |
| `DownloadMedia` | `DownloadMedia(ctx context.Context, media types.Media, thumbSize string, opts *params.Download) ([]byte, error)` | DownloadMedia resolves the file location from the given Media interface and downloads its contents into memory. |
| `DownloadMediaToFile` | `DownloadMediaToFile(ctx context.Context, media types.Media, thumbSize string, filePath string, fileSize int64, opts *params.Download) (error)` | DownloadMediaToFile resolves the file location from the given Media interface and downloads it directly to disk. |
| `DownloadMediaToWriter` | `DownloadMediaToWriter(ctx context.Context, media types.Media, thumbSize string, w io.Writer, opts *params.Download) (int64, error)` | DownloadMediaToWriter resolves the file location from the given Media interface and streams the download into w as chunks arrive, returning the total number of bytes written. |
| `EditInlineCaption` | `EditInlineCaption(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, caption string, opts ...*EditInlineOpts) (bool, error)` | EditInlineCaption edits the caption of an inline media message sent via the bot. |
| `EditInlineMedia` | `EditInlineMedia(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, media tg.InputMediaClass, opts ...*EditInlineOpts) (bool, error)` | EditInlineMedia edits the media attachment of an inline message sent via the bot. |
| `EditInlineReplyMarkup` | `EditInlineReplyMarkup(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, replyMarkup tg.ReplyMarkupClass) (bool, error)` | EditInlineReplyMarkup replaces the inline keyboard of an inline message. |
| `EditInlineRichMessage` | `EditInlineRichMessage(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, rm tg.InputRichMessageClass, opts ...*EditInlineOpts) (bool, error)` | EditInlineRichMessage replaces the content of an inline message with a rich message (Bot API 10.1 InputRichMessageContent editing). |
| `EditInlineText` | `EditInlineText(ctx context.Context, inlineMessageID tg.InputBotInlineMessageIDClass, text string, opts ...*EditInlineOpts) (bool, error)` | EditInlineText edits the text content of an inline message sent via the bot. |
| `EditLiveLocation` | `EditLiveLocation(ctx context.Context, chatID int64, messageID int32, lat, lng float64) (*types.Message, error)` | EditLiveLocation updates the coordinates of a live-location message. |
| `EditMessageCaption` | `EditMessageCaption(ctx context.Context, chatID int64, messageID int32, caption string, opts ...*params.EditMessage) (*types.Message, error)` | EditMessageCaption changes the caption of a media message (photo, video, document, etc.). |
| `EditMessageMedia` | `EditMessageMedia(ctx context.Context, chatID int64, messageID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error)` | EditMessageMedia replaces the media content of an existing message (photo, video, document, etc.) with new media. |
| `EditMessageReplyMarkup` | `EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int32, replyMarkup tg.ReplyMarkupClass) (*types.Message, error)` | EditMessageReplyMarkup changes only the inline keyboard (reply markup) of an existing message without modifying its text or media. |
| `EditMessageText` | `EditMessageText(ctx context.Context, chatID int64, messageID int32, text string, opts ...*params.EditMessage) (*types.Message, error)` | EditMessageText modifies the text content of an existing message. |
| `EditMessageWidgetData` | `EditMessageWidgetData(ctx context.Context, chatID int64, messageID int32, publicData, chatData string) (*types.Message, error)` | EditMessageWidgetData updates the public or per-chat data of a widget message in place and returns the updated message. |
| `EditWidgetData` | `EditWidgetData(ctx context.Context, widgetID int64, publicData string, cacheTime int32) (error)` | EditWidgetData updates the public data of a shared widget by its widget ID. |
| `ForwardMediaGroup` | `ForwardMediaGroup(ctx context.Context, chatID int64, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error)` | ForwardMediaGroup forwards a group of media messages (an album) from one chat to another. |
| `ForwardMessages` | `ForwardMessages(ctx context.Context, chatID int64, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error)` | ForwardMessages forwards one or more messages from a source chat to a destination chat. |
| `GetAvailableReactions` | `GetAvailableReactions(ctx context.Context, hash int32) (*types.ReactionList, error)` |  |
| `GetChatHistory` | `GetChatHistory(ctx context.Context, chatID int64, limit int, offsetID int32) ([]*types.Message, error)` | GetChatHistory retrieves a slice of message history from the specified chat, paginated by offset ID. |
| `GetChatHistoryCount` | `GetChatHistoryCount(ctx context.Context, chatID int64) (int, error)` | GetChatHistoryCount returns the total number of messages in the specified chat. |
| `GetDiscussionMessage` | `GetDiscussionMessage(ctx context.Context, chatID int64, messageID int32) (*types.Message, error)` |  |
| `GetDiscussionReplies` | `GetDiscussionReplies(ctx context.Context, chatID int64, messageID int32, limit int, offsetID int32) ([]*types.Message, error)` |  |
| `GetDiscussionRepliesCount` | `GetDiscussionRepliesCount(ctx context.Context, chatID int64, messageID int32) (int, error)` |  |
| `GetMediaGroup` | `GetMediaGroup(ctx context.Context, chatID int64, messageID int32) ([]*types.Message, error)` | GetMediaGroup retrieves all messages belonging to the same album (grouped media) as the specified message. |
| `GetMessageReactionsList` | `GetMessageReactionsList(ctx context.Context, chatID int64, messageID int32, reaction *types.Reaction, offset string, limit int32) (*types.PeerReactionList, error)` |  |
| `GetMessages` | `GetMessages(ctx context.Context, chatID int64, messageIDs []int32) ([]*types.Message, error)` | GetMessages retrieves one or more messages by their IDs from a specific chat. |
| `GetRichMessage` | `GetRichMessage(ctx context.Context, chatID int64, messageID int32) (*tg.RichMessage, error)` | GetRichMessage fetches the rendered rich message content of an existing message via messages.getRichMessage. |
| `GetScheduledMessages` | `GetScheduledMessages(ctx context.Context, chatID int64, messageIDs []int32) ([]*types.Message, error)` | GetScheduledMessages retrieves scheduled messages of a chat. |
| `GetWidgetPublicData` | `GetWidgetPublicData(ctx context.Context, chatID int64, messageID int32, hash int64) (*WidgetPublicData, error)` | GetWidgetPublicData fetches the public data of a widget message. |
| `MarkChecklistTasksAsDone` | `MarkChecklistTasksAsDone(ctx context.Context, chatID int64, messageID int32, done, undone []int32) (error)` | MarkChecklistTasksAsDone toggles the completion state of checklist tasks: the tasks in done are marked completed, the tasks in undone are reverted. |
| `PinMessage` | `PinMessage(ctx context.Context, chatID int64, messageID int32, opts ...*params.PinMessage) (*types.Message, error)` | PinMessage pins a message to the top of the specified chat. |
| `ReadChannelHistory` | `ReadChannelHistory(ctx context.Context, chatID int64, maxID int32) (error)` | ReadChannelHistory marks channel/supergroup history as read up to and including maxID. |
| `ReadChatHistory` | `ReadChatHistory(ctx context.Context, chatID int64, maxID int32) (error)` | ReadChatHistory marks the chat history of the specified chat as read up to and including maxID. |
| `ReadMentions` | `ReadMentions(ctx context.Context, chatID int64) (error)` | ReadMentions marks all unread mentions (@-mentions and quote replies) in the specified chat as read. |
| `ReadReactions` | `ReadReactions(ctx context.Context, chatID int64) (error)` | ReadReactions marks all unread reaction notifications in the specified chat as read. |
| `RetractVote` | `RetractVote(ctx context.Context, chatID int64, messageID int32) (error)` |  |
| `SearchGlobal` | `SearchGlobal(ctx context.Context, query string, opts ...*SearchGlobalOption) ([]*types.Message, error)` | SearchGlobal performs a global search across all of the user's chats for messages matching the given query. |
| `SearchGlobalCount` | `SearchGlobalCount(ctx context.Context, query string) (int32, error)` | SearchGlobalCount returns the total number of messages matching the query across all of the user's chats. |
| `SearchMessages` | `SearchMessages(ctx context.Context, chatID int64, query string, opts ...*SearchMessagesOption) ([]*types.Message, error)` | SearchMessages searches for messages within a specific chat matching the given query string. |
| `SearchMessagesCount` | `SearchMessagesCount(ctx context.Context, chatID int64, query string) (int32, error)` | SearchMessagesCount returns the total number of messages matching the query within a specific chat. |
| `SearchPosts` | `SearchPosts(ctx context.Context, query, hashtag string, offsetRate int32, offsetID int32, offsetPeer int64, limit int) (*SearchPostsResult, error)` | SearchPosts searches public posts by hashtag or free-text query, ordered from oldest to newest. |
| `SendAnimation` | `SendAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*types.Message, error)` |  |
| `SendAudio` | `SendAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*types.Message, error)` |  |
| `SendCachedMedia` | `SendCachedMedia(ctx context.Context, chatID int64, fileID string, opts *params.SendMessage) (*types.Message, error)` | SendCachedMedia sends a previously uploaded file referenced by its file ID. |
| `SendChatAction` | `SendChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) (error)` | SendChatAction sends a chat action indicator to the specified chat, such as "typing" or "uploading photo". |
| `SendContact` | `SendContact(ctx context.Context, chatID int64, phoneNumber, firstName, lastName string, opts ...*params.SendContact) (*types.Message, error)` | SendContact sends a contact card (vCard) as a message to the specified chat. |
| `SendDice` | `SendDice(ctx context.Context, chatID int64, opts ...*params.SendDice) (*types.Message, error)` | SendDice sends an animated dice (or other animated emoji) message to the specified chat. |
| `SendDocument` | `SendDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error)` |  |
| `SendGame` | `SendGame(ctx context.Context, chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error)` | SendGame sends a game message to the specified chat. |
| `SendLivePhoto` | `SendLivePhoto(ctx context.Context, chatID int64, file *InputFile, ttlSeconds int32, caption string, opts ...*params.SendPhoto) (*types.Message, error)` | SendLivePhoto sends a photo marked as a self-destructing live photo with the given time-to-live in seconds. |
| `SendLocation` | `SendLocation(ctx context.Context, chatID int64, lat, lng float64, opts ...*params.SendLocation) (*types.Message, error)` | SendLocation sends a geographic point (latitude/longitude) as a message to the specified chat. |
| `SendMedia` | `SendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error)` | SendMedia sends a media-attached message (photo, document, video, etc.) to the specified chat. |
| `SendMediaGroup` | `SendMediaGroup(ctx context.Context, chatID int64, items []*tg.InputSingleMedia, opts ...*params.SendMessage) ([]*types.Message, error)` | SendMediaGroup sends an album of media items (multiple photos, videos, or documents) as a single grouped message to the specified chat. |
| `SendMessage` | `SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error)` | SendMessage sends a plain text message to the specified chat. |
| `SendMessageDraft` | `SendMessageDraft(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error)` |  |
| `SendPaidReaction` | `SendPaidReaction(ctx context.Context, chatID int64, messageID int32, amount int64, opts ...*params.SendPaidReactionOption) (error)` |  |
| `SendPhoto` | `SendPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*types.Message, error)` | SendPhoto uploads a photo and sends it to the specified chat. |
| `SendPoll` | `SendPoll(ctx context.Context, chatID int64, question string, options []string, opts ...*params.SendPoll) (*types.Message, error)` | SendPoll sends a poll question with the given answer options to the specified chat. |
| `SendReaction` | `SendReaction(ctx context.Context, chatID int64, messageID int32, reactions []types.Reaction, opts ...*params.SendReactionOption) (error)` |  |
| `SendRichMessage` | `SendRichMessage(ctx context.Context, chatID int64, rm tg.InputRichMessageClass, opts ...*params.SendMessage) (*types.Message, error)` | SendRichMessage sends a rich message (Bot API sendRichMessage). |
| `SendRichMessageHTML` | `SendRichMessageHTML(ctx context.Context, chatID int64, html string, opts ...*params.SendMessage) (*types.Message, error)` | SendRichMessageHTML sends a rich message built from Telegram HTML markup. |
| `SendRichMessageMarkdown` | `SendRichMessageMarkdown(ctx context.Context, chatID int64, markdown string, opts ...*params.SendMessage) (*types.Message, error)` | SendRichMessageMarkdown sends a rich message built from Telegram Markdown markup. |
| `SendScreenshotNotification` | `SendScreenshotNotification(ctx context.Context, chatID int64, replyToMessageID int32) (error)` | SendScreenshotNotification sends the secret-chat screenshot notification service message into a chat. |
| `SendSticker` | `SendSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*types.Message, error)` |  |
| `SendVenue` | `SendVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, opts ...*params.SendVenue) (*types.Message, error)` | SendVenue sends a venue (a named location with an address) as a message to the specified chat. |
| `SendVideo` | `SendVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*types.Message, error)` |  |
| `SendVideoNote` | `SendVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*types.Message, error)` |  |
| `SendVoice` | `SendVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*types.Message, error)` |  |
| `SendWidget` | `SendWidget(ctx context.Context, chatID int64, width, height int32, html string, opts ...*params.SendWidget) (*types.Message, error)` | SendWidget sends an interactive HTML widget message. |
| `StartBot` | `StartBot(ctx context.Context, botID int64, chatID int64, startParam string) (error)` | StartBot starts a conversation with a bot in the given chat, delivering the deep-link start parameter. |
| `StartMessageDraft` | `StartMessageDraft(ctx context.Context, chatID int64, opts ...*DraftOpts) (*MessageDraft, error)` | StartMessageDraft resolves the target chat and returns a streaming plain text draft bound to it. |
| `StartRichMessageDraft` | `StartRichMessageDraft(ctx context.Context, chatID int64, opts ...*DraftOpts) (*RichMessageDraft, error)` | StartRichMessageDraft resolves the target chat and returns a streaming rich message draft bound to it. |
| `StopLiveLocation` | `StopLiveLocation(ctx context.Context, chatID int64, messageID int32) (*types.Message, error)` | StopLiveLocation terminates a live-location message, keeping the last known coordinates. |
| `StopPoll` | `StopPoll(ctx context.Context, chatID int64, messageID int32) (error)` |  |
| `StreamMedia` | `StreamMedia(ctx context.Context, input any, opts *params.Download) (_, error)` | StreamMedia downloads media from a message or media object chunk by chunk, returning a channel that yields [StreamChunk] values. |
| `SummarizeMessage` | `SummarizeMessage(ctx context.Context, chatID int64, messageID int32) (*tg.TextWithEntities, error)` | SummarizeMessage asks the server to summarize a message and returns the summary as formatted text. |
| `TranslateMessageText` | `TranslateMessageText(ctx context.Context, chatID int64, messageIDs []int32, toLang string) ([]*tg.TextWithEntities, error)` | TranslateMessageText translates the given messages of a chat into the target language and returns the translated texts in input order. |
| `TranslateText` | `TranslateText(ctx context.Context, texts []*tg.TextWithEntities, toLang string) ([]*tg.TextWithEntities, error)` | TranslateText translates raw formatted texts into the target language and returns the translated texts in input order. |
| `UnpinAllMessages` | `UnpinAllMessages(ctx context.Context, chatID int64) (int, error)` | UnpinAllMessages removes all pinned messages from the specified chat in a single operation. |
| `UnpinMessage` | `UnpinMessage(ctx context.Context, chatID int64, messageID int32) (*types.Message, error)` | UnpinMessage removes a single message from the pinned messages list in the specified chat. |
| `ViewMessages` | `ViewMessages(ctx context.Context, chatID int64, messageIDs []int32, increment bool) ([]int32, error)` |  |
| `VotePoll` | `VotePoll(ctx context.Context, chatID int64, messageID int32, options [][]byte) (error)` |  |

### Password

cloud password — `*PasswordMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `ChangeCloudPassword` | `ChangeCloudPassword(ctx context.Context, currentPassword, newPassword, newHint string) (error)` | ChangeCloudPassword changes the existing cloud password to newPassword with the given newHint. |
| `EnableCloudPassword` | `EnableCloudPassword(ctx context.Context, password, hint string) (error)` | EnableCloudPassword enables two-factor authentication by setting a cloud password on the account. |
| `RemoveCloudPassword` | `RemoveCloudPassword(ctx context.Context, password string) (error)` | RemoveCloudPassword disables two-factor authentication by removing the cloud password from the account. |
| `ResetCloudPassword` | `ResetCloudPassword(ctx context.Context) (*ResetCloudPasswordResult, error)` | ResetCloudPassword starts resetting the cloud password when the current one is forgotten. |

### Payment

payments, stars, gifts — `*PaymentMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AnswerPreCheckoutQuery` | `AnswerPreCheckoutQuery(ctx context.Context, queryID int64, ok bool, errorMessage string) (error)` | AnswerPreCheckoutQuery responds to a pre-checkout query, approving or rejecting the payment. |
| `AnswerShippingQuery` | `AnswerShippingQuery(ctx context.Context, queryID int64, ok bool, options []*tg.ShippingOption) (error)` | AnswerShippingQuery responds to a shipping query with available shipping options. |
| `ApplyGiftCode` | `ApplyGiftCode(ctx context.Context, slug string) (error)` | ApplyGiftCode redeems a gift code (Telegram Premium or Stars) for the logged-in user. |
| `AssignAppStoreTransaction` | `AssignAppStoreTransaction(ctx context.Context, receipt []byte, purpose tg.InputStorePaymentPurposeClass) (error)` | AssignAppStoreTransaction applies an App Store in-app purchase receipt to its purpose. |
| `AssignPlayMarketTransaction` | `AssignPlayMarketTransaction(ctx context.Context, receiptJSON string, purpose tg.InputStorePaymentPurposeClass) (error)` | AssignPlayMarketTransaction applies a Google Play in-app purchase receipt (a JSON payload) to its purpose. |
| `BuyResaleGift` | `BuyResaleGift(ctx context.Context, slug string, recipientID int64, opts ...*params.BuyGift) (*types.Message, error)` |  |
| `CanPurchaseFromStore` | `CanPurchaseFromStore(ctx context.Context, purpose tg.InputStorePaymentPurposeClass) (bool, error)` | CanPurchaseFromStore checks whether a store purchase (Premium, Stars, gifts) is available for the given purpose before opening the store flow. |
| `CancelStarsSubscriptionByBot` | `CancelStarsSubscriptionByBot(ctx context.Context, userID int64, chargeID string, restore bool) (error)` | CancelStarsSubscriptionByBot cancels a user's subscription to the logged-in bot's paid channel; restore re-enables a previously canceled subscription. |
| `ChangeStarsSubscription` | `ChangeStarsSubscription(ctx context.Context, botID int64, subscriptionID string, canceled bool) (error)` | ChangeStarsSubscription cancels (or re-enables) one of the logged-in user's Stars subscriptions with a bot. |
| `CheckCanSendGift` | `CheckCanSendGift(ctx context.Context, giftID int64) (error)` | CheckCanSendGift verifies the logged-in user may send a specific gift (age, privacy, and regional restrictions included) before attempting it. |
| `CheckGiftCode` | `CheckGiftCode(ctx context.Context, slug string) (*tg.PaymentsCheckedGiftCode, error)` | CheckGiftCode inspects a gift-code slug without redeeming it. |
| `ClearSavedPaymentInfo` | `ClearSavedPaymentInfo(ctx context.Context, info, credentials bool) (error)` | ClearSavedPaymentInfo deletes the saved shipping info and/or payment credentials. |
| `ConvertGift` | `ConvertGift(ctx context.Context, gift tg.InputSavedStarGiftClass) (error)` |  |
| `CreateGiftCollection` | `CreateGiftCollection(ctx context.Context, chatID int64, title string, gifts []tg.InputSavedStarGiftClass) (*tg.StarGiftCollection, error)` | CreateGiftCollection groups owned gifts into a named showcase collection on the giver's profile. |
| `CreateInvoiceLink` | `CreateInvoiceLink(ctx context.Context, invoice *tg.InputMediaInvoice) (string, error)` |  |
| `DeleteGiftCollection` | `DeleteGiftCollection(ctx context.Context, chatID int64, collectionID int32) (error)` | DeleteGiftCollection deletes a gift collection. |
| `FulfillStarsSubscription` | `FulfillStarsSubscription(ctx context.Context, subscriberID int64, subscriptionID string) (error)` | FulfillStarsSubscription charges the subscriber for the next period of a bot's Stars subscription. |
| `GetActiveGiftAuctions` | `GetActiveGiftAuctions(ctx context.Context, hash int64) ([]*tg.StarGiftActiveAuctionState, error)` | GetActiveGiftAuctions lists currently running gift auctions. |
| `GetGiftAuctionAcquiredGifts` | `GetGiftAuctionAcquiredGifts(ctx context.Context, giftID int64) (*tg.PaymentsStarGiftAuctionAcquiredGifts, error)` | GetGiftAuctionAcquiredGifts lists the gifts acquired through auctions by a user's resale bot. |
| `GetGiftAuctionState` | `GetGiftAuctionState(ctx context.Context, auction tg.InputStarGiftAuctionClass, version int32) (tg.StarGiftAuctionStateClass, error)` | GetGiftAuctionState returns the live state of a gift auction; pass the last seen version to receive only changes. |
| `GetGiftCollections` | `GetGiftCollections(ctx context.Context, chatID int64, hash int64) ([]*tg.StarGiftCollection, error)` | GetGiftCollections lists a profile's gift collections. |
| `GetGiftValue` | `GetGiftValue(ctx context.Context, slug string) (*tg.PaymentsUniqueStarGiftValueInfo, error)` |  |
| `GetGiftWithdrawalURL` | `GetGiftWithdrawalURL(ctx context.Context, gift tg.InputSavedStarGiftClass, password string) (string, error)` |  |
| `GetGiveawayInfo` | `GetGiveawayInfo(ctx context.Context, chatID int64, messageID int32) (tg.GiveawayInfoClass, error)` | GetGiveawayInfo returns participation or result information about a giveaway message. |
| `GetPaymentForm` | `GetPaymentForm(ctx context.Context, chatID int64, messageID int32, opts ...*GetPaymentFormOption) (*types.PaymentForm, error)` | GetPaymentForm retrieves the payment form for an invoice message in the specified chat. |
| `GetPaymentReceipt` | `GetPaymentReceipt(ctx context.Context, chatID int64, messageID int32) (tg.PaymentReceiptClass, error)` | GetPaymentReceipt fetches the receipt for a paid invoice message. |
| `GetPremiumGiftCodeOptions` | `GetPremiumGiftCodeOptions(ctx context.Context, boostPeer int64) ([]*tg.PremiumGiftCodeOption, error)` | GetPremiumGiftCodeOptions returns purchasable Premium-gift-code packages (for gifting or boosting boostPeer). |
| `GetResaleGifts` | `GetResaleGifts(ctx context.Context, giftID int64, opts ...*params.GetGifts) (*tg.PaymentsResaleStarGifts, error)` |  |
| `GetSavedGiftByID` | `GetSavedGiftByID(ctx context.Context, gifts ...tg.InputSavedStarGiftClass) (*tg.PaymentsSavedStarGifts, error)` |  |
| `GetSavedGifts` | `GetSavedGifts(ctx context.Context, peerID int64, opts ...*params.GetGifts) (*tg.PaymentsSavedStarGifts, error)` |  |
| `GetSavedPaymentInfo` | `GetSavedPaymentInfo(ctx context.Context) (*tg.PaymentsSavedInfo, error)` | GetSavedPaymentInfo returns the saved shipping info and whether payment credentials are stored. |
| `GetStarGiftCatalog` | `GetStarGiftCatalog(ctx context.Context, hash int32) (tg.StarGiftsClass, error)` |  |
| `GetStarGiftOptions` | `GetStarGiftOptions(ctx context.Context) ([]*types.Gift, error)` |  |
| `GetStarGiftUpgradeOptions` | `GetStarGiftUpgradeOptions(ctx context.Context, giftID int64) (*tg.PaymentsStarGiftUpgradeAttributes, error)` |  |
| `GetStarsBalance` | `GetStarsBalance(ctx context.Context, chatID int64) (int64, error)` | GetStarsBalance returns the Telegram Stars balance for the specified peer. |
| `GetStarsGiveawayOptions` | `GetStarsGiveawayOptions(ctx context.Context) ([]*tg.StarsGiveawayOption, error)` | GetStarsGiveawayOptions returns the available Stars-giveaway boost packages for channels. |
| `GetStarsRevenueAdsAccountURL` | `GetStarsRevenueAdsAccountURL(ctx context.Context, chatID int64) (string, error)` | GetStarsRevenueAdsAccountURL returns the ad-platform account URL connected to a channel's Stars revenue. |
| `GetStarsRevenueStats` | `GetStarsRevenueStats(ctx context.Context, chatID int64, dark bool) (*tg.PaymentsStarsRevenueStats, error)` | GetStarsRevenueStats returns the Stars revenue statistics of a chat owned by the logged-in user. |
| `GetStarsSubscriptions` | `GetStarsSubscriptions(ctx context.Context, offset string, missingBalance bool) (*StarsSubscriptions, error)` | GetStarsSubscriptions lists the logged-in user's Stars subscriptions; missingBalance true lists only subscriptions the balance can no longer cover. |
| `GetStarsTopupOptions` | `GetStarsTopupOptions(ctx context.Context) ([]*tg.StarsTopupOption, error)` | GetStarsTopupOptions returns the available Telegram Stars top-up packages. |
| `GetStarsTransactions` | `GetStarsTransactions(ctx context.Context, chatID int64, inbound, outbound bool, offset string, limit int32, opts ...*params.GetStarsTransactionsOption) (*types.StarsStatus, error)` | GetStarsTransactions retrieves a paginated list of Telegram Stars transactions for the specified peer (user, bot, or channel). |
| `GetStarsTransactionsByID` | `GetStarsTransactionsByID(ctx context.Context, ids []*tg.InputStarsTransaction) ([]*tg.StarsTransaction, error)` | GetStarsTransactionsByID fetches specific Stars transactions of the logged-in user by identifier. |
| `GetUniqueGift` | `GetUniqueGift(ctx context.Context, slug string) (*tg.PaymentsUniqueStarGift, error)` |  |
| `HideGift` | `HideGift(ctx context.Context, gift tg.InputSavedStarGiftClass) (error)` |  |
| `LaunchPrepaidGiveaway` | `LaunchPrepaidGiveaway(ctx context.Context, chatID int64, giveawayID int64, purpose tg.InputStorePaymentPurposeClass) (error)` | LaunchPrepaidGiveaway launches a previously prepaid giveaway in a chat, using the giveaway identifier from the purchase receipt. |
| `PinGifts` | `PinGifts(ctx context.Context, gifts []tg.InputSavedStarGiftClass, peerID int64) (error)` |  |
| `PrepayGiftUpgrade` | `PrepayGiftUpgrade(ctx context.Context, peerID int64, hash string) (*types.Message, error)` |  |
| `RefundStarsCharge` | `RefundStarsCharge(ctx context.Context, userID int64, chargeID string) (error)` | RefundStarsCharge refunds a Telegram Stars charge. |
| `ReorderGiftCollections` | `ReorderGiftCollections(ctx context.Context, chatID int64, order []int32) (error)` | ReorderGiftCollections changes the display order of a profile's gift collections. |
| `ResolveGiftOffer` | `ResolveGiftOffer(ctx context.Context, messageID int32, accept bool) (*types.Message, error)` |  |
| `ResolveStarGiftOffer` | `ResolveStarGiftOffer(ctx context.Context, offerMessageID int32, decline bool) (*types.Message, error)` | ResolveStarGiftOffer accepts or declines a star-gift purchase offer message and returns the updated message. |
| `SendGift` | `SendGift(ctx context.Context, userID int64, giftID int64, message string, opts ...*params.GiftSend) (*types.Message, error)` |  |
| `SendGiftOffer` | `SendGiftOffer(ctx context.Context, peerID int64, slug string, price int64, duration int32) (*types.Message, error)` |  |
| `SendInvoice` | `SendInvoice(ctx context.Context, chatID int64, invoice *tg.InputMediaInvoice, caption string, opts ...*params.SendMessage) (*types.Message, error)` |  |
| `SendPaymentForm` | `SendPaymentForm(ctx context.Context, formID int64, chatID int64, messageID int32, creds tg.InputPaymentCredentialsClass, opts ...*SendPaymentFormOption) (tg.PaymentResultClass, error)` | SendPaymentForm submits payment credentials for a previously retrieved payment form. |
| `SetGiftChatNotifications` | `SetGiftChatNotifications(ctx context.Context, chatID int64, enabled bool) (error)` | SetGiftChatNotifications enables or disable notifications about new gifts in a chat. |
| `SetGiftResalePrice` | `SetGiftResalePrice(ctx context.Context, gift tg.InputSavedStarGiftClass, stars int64) (error)` |  |
| `ShowGift` | `ShowGift(ctx context.Context, gift tg.InputSavedStarGiftClass) (error)` |  |
| `TransferGift` | `TransferGift(ctx context.Context, gift tg.InputSavedStarGiftClass, recipientID int64) (*types.Message, error)` |  |
| `UpdateGiftCollection` | `UpdateGiftCollection(ctx context.Context, chatID int64, collectionID int32, title string, remove, add, order []tg.InputSavedStarGiftClass) (*tg.StarGiftCollection, error)` | UpdateGiftCollection edits a gift collection: renames it (title), removes or adds gifts, or reorders its gifts. |
| `UpgradeGift` | `UpgradeGift(ctx context.Context, gift tg.InputSavedStarGiftClass, keepOriginalDetails bool) (*types.Message, error)` |  |
| `ValidateRequestedInfo` | `ValidateRequestedInfo(ctx context.Context, invoice tg.InputInvoiceClass, info *tg.PaymentRequestedInfo, save bool) (*tg.PaymentsValidatedRequestedInfo, error)` | ValidateRequestedInfo validates shipping information for an invoice and returns the available shipping options; save stores the info for reuse. |
| `WithdrawStarsRevenue` | `WithdrawStarsRevenue(ctx context.Context, chatID int64, amount int64, password string) (string, error)` | WithdrawStarsRevenue opens a withdrawal URL for the Stars balance of a chat owned by the logged-in user. |

### Phone

calls and live streams — `*PhoneMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `CreateCall` | `CreateCall(ctx context.Context, peer int64, rtmp ...bool) (tg.PhoneCallClass, error)` | CreateCall creates a group call in the specified peer chat. |
| `GetActiveCall` | `GetActiveCall(ctx context.Context, chatID int64) (tg.InputGroupCallClass, error)` | GetActiveCall retrieves the currently active group call for the given chat or channel. |
| `NewBroadcastStream` | `NewBroadcastStream(chatID int64, cfg ...*BroadcastConfig) (*BroadcastStream, error)` | NewBroadcastStream creates a new BroadcastStream for the specified chat. |
| `NewCallReader` | `NewCallReader(ctx context.Context, chatID int64) (*CallReader, error)` | NewCallReader creates a CallReader for the active group call in the specified chat, fetching available stream channels automatically. |

### Premium

boosts and premium — `*PremiumMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `ApplyBoost` | `ApplyBoost(ctx context.Context, chatID int64, opts ...*ApplyBoostOption) ([]*tg.MyBoost, error)` | ApplyBoost applies one or more Telegram Premium boosts to the specified chat. |
| `GetBoostsStatus` | `GetBoostsStatus(ctx context.Context, chatID int64) (*tg.PremiumBoostsStatus, error)` | GetBoostsStatus retrieves the boost status for the specified chat, including the current boost level and how many more boosts are needed for the next level. |
| `GetChatBoosts` | `GetChatBoosts(ctx context.Context, chatID int64, offset string, limit int, giftsOnly bool) (*ChatBoosts, error)` | GetChatBoosts lists the boosts applied to a chat, most recent first. |
| `GetMyBoosts` | `GetMyBoosts(ctx context.Context, opts ...*GetBoostsOption) ([]*tg.MyBoost, error)` | GetMyBoosts retrieves all active boosts applied by the current user across all chats. |
| `GetUserBoosts` | `GetUserBoosts(ctx context.Context, chatID, userID int64) ([]*tg.Boost, error)` | GetUserBoosts lists the boosts a specific user has applied to a chat. |

### SecretChat

end-to-end secret chats (mtgo extension) — `*SecretChatMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AcceptSecretChat` | `AcceptSecretChat(ctx context.Context, chatID int32) (*SecretChat, error)` |  |
| `DecryptSecretFile` | `DecryptSecretFile(ctx context.Context, file *tg.EncryptedFile, fileKey, fileIV []byte) ([]byte, error)` |  |
| `DecryptSecretMessage` | `DecryptSecretMessage(chatID int32, ciphertext []byte) (*e2e.DecryptedMessageLayer, error)` |  |
| `DiscardSecretChat` | `DiscardSecretChat(ctx context.Context, chatID int32, deleteHistory bool) (error)` |  |
| `GetSecretChat` | `GetSecretChat(chatID int32) (*SecretChat, bool)` |  |
| `OnSecretChatRequest` | `OnSecretChatRequest(handler SecretChatRequestHandler)` |  |
| `OnSecretMessage` | `OnSecretMessage(handler SecretMessageHandler)` |  |
| `RequestSecretChat` | `RequestSecretChat(ctx context.Context, userID tg.InputUserClass) (*SecretChat, error)` |  |
| `SecretChatVisualization` | `SecretChatVisualization(chatID int32) ([]string)` |  |
| `SendLayerNotification` | `SendLayerNotification(ctx context.Context, chatID int32) (error)` |  |
| `SendSecretFile` | `SendSecretFile(ctx context.Context, chatID int32, reader io.Reader, fileName string, fileSize int64, opts *SecretFileOptions) (error)` |  |
| `SendSecretMessage` | `SendSecretMessage(ctx context.Context, chatID int32, text string) (int64, error)` |  |

### Sticker

stickers and custom emoji — `*StickerMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `AddStickerToSet` | `AddStickerToSet(ctx context.Context, set tg.InputStickerSetClass, sticker *tg.InputStickerSetItem) (*tg.MessagesStickerSet, error)` | AddStickerToSet appends a sticker to an existing set owned by the logged-in user and returns the updated set. |
| `ChangeSticker` | `ChangeSticker(ctx context.Context, sticker *tg.InputDocument, emoji, keywords string) (*tg.MessagesStickerSet, error)` | ChangeSticker updates the emoji list and search keywords of a sticker and returns the updated set. |
| `ChangeStickerPosition` | `ChangeStickerPosition(ctx context.Context, sticker *tg.InputDocument, position int32) (*tg.MessagesStickerSet, error)` | ChangeStickerPosition moves a sticker within its set and returns the updated set. |
| `CheckStickerSetShortName` | `CheckStickerSetShortName(ctx context.Context, shortName string) (bool, error)` | CheckStickerSetShortName reports whether the short name is available for a new sticker set. |
| `ClearRecentStickers` | `ClearRecentStickers(ctx context.Context, attached bool) (error)` | ClearRecentStickers clears the recent-stickers list; attached true clears the attached-stickers list. |
| `CreateStickerSet` | `CreateStickerSet(ctx context.Context, userID int64, title, shortName string, stickers []*tg.InputStickerSetItem, opts ...*CreateStickerSetOption) (*tg.MessagesStickerSet, error)` | CreateStickerSet creates a new sticker set owned by the given user (0 means the logged-in user) from already-uploaded documents. |
| `DeleteStickerSet` | `DeleteStickerSet(ctx context.Context, set tg.InputStickerSetClass) (error)` | DeleteStickerSet permanently deletes a sticker set owned by the logged-in user. |
| `FavoriteSticker` | `FavoriteSticker(ctx context.Context, document *tg.InputDocument) (error)` | FavoriteSticker adds a sticker to the logged-in user's favorites. |
| `GetAllStickerSets` | `GetAllStickerSets(ctx context.Context, hash int64) ([]*tg.StickerSet, error)` | GetAllStickerSets returns every installed sticker set. |
| `GetCustomEmojiStickers` | `GetCustomEmojiStickers(ctx context.Context, customEmojiIDs []string) ([]*types.Sticker, error)` | GetCustomEmojiStickers returns sticker metadata for the given custom emoji document identifiers. |
| `GetFeaturedStickerSets` | `GetFeaturedStickerSets(ctx context.Context, hash int64) ([]tg.StickerSetCoveredClass, error)` | GetFeaturedStickerSets returns the sticker sets featured by Telegram, with cover stickers for previews. |
| `GetRecentStickers` | `GetRecentStickers(ctx context.Context, attached bool, hash int64) ([]*types.Sticker, error)` | GetRecentStickers returns the stickers recently used by the logged-in user; attached true lists recently used attached stickers instead. |
| `GetSavedGIFs` | `GetSavedGIFs(ctx context.Context, hash int64) ([]*tg.Document, error)` | GetSavedGIFs returns the GIFs saved by the logged-in user. |
| `GetStickerSet` | `GetStickerSet(ctx context.Context, set tg.InputStickerSetClass, hash int32) (*tg.MessagesStickerSet, error)` | GetStickerSet fetches one sticker set with all of its stickers and emoji packs. |
| `InstallStickerSet` | `InstallStickerSet(ctx context.Context, set tg.InputStickerSetClass, archive bool) (error)` | InstallStickerSet installs a sticker set for the logged-in user. |
| `RemoveSavedGIF` | `RemoveSavedGIF(ctx context.Context, document *tg.InputDocument) (error)` | RemoveSavedGIF removes a GIF from the logged-in user's saved GIFs. |
| `RemoveStickerFromSet` | `RemoveStickerFromSet(ctx context.Context, sticker *tg.InputDocument) (*tg.MessagesStickerSet, error)` | RemoveStickerFromSet removes a sticker from the set that contains it and returns the updated set. |
| `RenameStickerSet` | `RenameStickerSet(ctx context.Context, set tg.InputStickerSetClass, title string) (*tg.MessagesStickerSet, error)` | RenameStickerSet changes a sticker set's title and returns the updated set. |
| `ReplaceSticker` | `ReplaceSticker(ctx context.Context, sticker *tg.InputDocument, newSticker *tg.InputStickerSetItem) (*tg.MessagesStickerSet, error)` | ReplaceSticker replaces a sticker's document in place and returns the updated set. |
| `SaveGIF` | `SaveGIF(ctx context.Context, document *tg.InputDocument) (error)` | SaveGIF saves a GIF for the logged-in user. |
| `SaveRecentSticker` | `SaveRecentSticker(ctx context.Context, document *tg.InputDocument, attached, unsave bool) (error)` | SaveRecentSticker adds a sticker to the recent list, or removes it when unsave is true; attached targets the attached-stickers list. |
| `SearchStickerSets` | `SearchStickerSets(ctx context.Context, query string, excludeFeatured bool, hash int64) ([]tg.StickerSetCoveredClass, error)` | SearchStickerSets searches installed and featured sticker sets by title or short name. |
| `SearchStickers` | `SearchStickers(ctx context.Context, emoticon string, hash int64) ([]*types.Sticker, error)` | SearchStickers returns stickers matching an emoji, such as "😀". |
| `SetStickerSetThumbnail` | `SetStickerSetThumbnail(ctx context.Context, set tg.InputStickerSetClass, thumb *tg.InputDocument) (*tg.MessagesStickerSet, error)` | SetStickerSetThumbnail sets the thumbnail document of a sticker set and returns the updated set. |
| `SuggestStickerSetShortName` | `SuggestStickerSetShortName(ctx context.Context, title string) (string, error)` | SuggestStickerSetShortName suggests an available short name derived from a sticker set title. |
| `UnfavoriteSticker` | `UnfavoriteSticker(ctx context.Context, document *tg.InputDocument) (error)` | UnfavoriteSticker removes a sticker from the logged-in user's favorites. |
| `UninstallStickerSet` | `UninstallStickerSet(ctx context.Context, set tg.InputStickerSetClass) (error)` | UninstallStickerSet removes a sticker set from the logged-in user's installed sets. |

### Story

stories — `*StoryMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `ActivateStealthMode` | `ActivateStealthMode(ctx context.Context, past, future bool) (error)` | ActivateStealthMode hides the logged-in user's story views from others: past hides views made during the last minutes, future hides upcoming views for a window. |
| `DeleteStories` | `DeleteStories(ctx context.Context, chatID int64, storyIDs []int32) (error)` | DeleteStories removes one or more stories from the specified chat. |
| `EditStoryCaption` | `EditStoryCaption(ctx context.Context, chatID int64, storyID int32, caption string) (*types.Story, error)` | EditStoryCaption updates the caption of an existing story identified by storyID owned by chatID. |
| `EditStoryMedia` | `EditStoryMedia(ctx context.Context, chatID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error)` | EditStoryMedia replaces the media content of an existing story identified by storyID owned by chatID. |
| `EditStoryPrivacy` | `EditStoryPrivacy(ctx context.Context, peerID int64, storyID int32, opts ...*params.EditPrivacy) (*types.Story, error)` | EditStoryPrivacy replaces the privacy rules of a story and returns the updated story. |
| `ExportStoryLink` | `ExportStoryLink(ctx context.Context, chatID int64, storyID int32) (string, error)` | ExportStoryLink returns the public t.me link of a story. |
| `ForwardStory` | `ForwardStory(ctx context.Context, targetChatID int64, sourceChatID int64, storyID int32) (*types.Message, error)` | ForwardStory forwards a story from sourceChatID to targetChatID. |
| `GetAllStories` | `GetAllStories(ctx context.Context, state string, next, hidden bool) (*StoriesFeed, error)` | GetAllStories returns the all-stories feed of the logged-in user's contacts. |
| `GetArchivedStories` | `GetArchivedStories(ctx context.Context, chatID int64, offsetID int32, limit int) ([]*types.Story, error)` | GetArchivedStories returns the archived stories of a peer. |
| `GetChatStories` | `GetChatStories(ctx context.Context, chatID int64) ([]*types.Story, error)` | GetChatStories retrieves all active stories posted by the specified chat or peer. |
| `GetChatsToSendStories` | `GetChatsToSendStories(ctx context.Context) ([]*types.Chat, error)` | GetChatsToSendStories returns the chats where the logged-in user may post stories (own profile plus admin channels). |
| `GetPinnedStories` | `GetPinnedStories(ctx context.Context, chatID int64, offsetID int32, limit int) ([]*types.Story, error)` | GetPinnedStories returns the stories a peer pinned to the top of its profile. |
| `GetRemainingStoryCount` | `GetRemainingStoryCount(ctx context.Context) (int32, error)` | GetRemainingStoryCount returns how many stories the logged-in user can still post today on their own profile. |
| `GetStories` | `GetStories(ctx context.Context, userID int64, storyIDs []int32) ([]*types.Story, error)` | GetStories retrieves one or more stories by ID from the specified user. |
| `GetStoryReactions` | `GetStoryReactions(ctx context.Context, chatID, storyID int64, reaction tg.ReactionClass, offset string, limit int, forwardsFirst bool) (*StoryReactionsPage, error)` | GetStoryReactions lists reactions to one story; pass a reaction to list only that reaction, and forwardsFirst to order forwards ahead. |
| `GetStoryViewers` | `GetStoryViewers(ctx context.Context, chatID, storyID int64, q string, justContacts, reactionsFirst, forwardsFirst bool, offset string, limit int) (*StoryViewersPage, error)` | GetStoryViewers lists the viewers of one story. |
| `GetStoryViews` | `GetStoryViews(ctx context.Context, chatID int64, storyIDs []int32) ([]*tg.StoryViews, error)` | GetStoryViews retrieves view statistics for the specified stories owned by chatID. |
| `PinChatStories` | `PinChatStories(ctx context.Context, chatID int64, storyIDs []int32) (error)` | PinChatStories pins the specified stories to the top of the chat's story list. |
| `ReadChatStories` | `ReadChatStories(ctx context.Context, chatID int64, storyIDs []int32) (error)` | ReadChatStories marks the specified stories as read (increments their view count) for the given chat. |
| `ReadStories` | `ReadStories(ctx context.Context, peerID int64, maxStoryID int32) (error)` | ReadStories marks the stories of a peer as read up to the given story ID. |
| `ReportStory` | `ReportStory(ctx context.Context, chatID int64, storyIDs []int32, option []byte, message string) (error)` | ReportStory reports stories that violate the terms of service. |
| `SendStory` | `SendStory(ctx context.Context, chatID int64, media tg.InputMediaClass, opts ...*SendStoryOption) (*types.Story, error)` | SendStory posts a new story to the specified peer's story feed. |
| `SendStoryReaction` | `SendStoryReaction(ctx context.Context, peerID int64, storyID int32, opts ...*params.React) (error)` | SendStoryReaction reacts to a story with an emoji. |
| `ToggleAllStoriesHidden` | `ToggleAllStoriesHidden(ctx context.Context, hidden bool) (error)` | ToggleAllStoriesHidden hides (or unhides) every contact's stories from the logged-in user's main feed. |
| `ToggleStoriesHidden` | `ToggleStoriesHidden(ctx context.Context, chatID int64, hidden bool) (error)` | ToggleStoriesHidden hides (or unhides) a peer's stories from the logged-in user's main feed. |

### User

user-profile operations — `*UserMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `BlockUser` | `BlockUser(ctx context.Context, userID int64) (error)` | BlockUser blocks the specified user from interacting with the current account. |
| `DeleteProfilePhoto` | `DeleteProfilePhoto(ctx context.Context, photoID int64) (error)` | DeleteProfilePhoto deletes a profile photo by its ID. |
| `GetBlocked` | `GetBlocked(ctx context.Context, limit, offset int) (tg.BlockedClass, error)` | GetBlocked retrieves a paginated list of blocked users. |
| `GetCommonChats` | `GetCommonChats(ctx context.Context, userID int64, limit int) ([]*types.Chat, error)` | GetCommonChats retrieves the list of chats (groups, channels, supergroups) that the current user shares with the specified user. |
| `GetMe` | `GetMe(ctx context.Context) (*types.User, error)` | GetMe retrieves the currently authenticated user's full information. |
| `GetProfilePhotos` | `GetProfilePhotos(ctx context.Context, userID int64, opts ...*GetProfilePhotosOption) ([]*types.ChatPhoto, error)` | GetProfilePhotos retrieves a paginated list of profile photos for the specified user. |
| `GetUser` | `GetUser(ctx context.Context, userID int64) (*types.User, error)` | GetUser retrieves full information about a single user by their ID. |
| `GetUsers` | `GetUsers(ctx context.Context, userIDs []int64) ([]*types.User, error)` | GetUsers retrieves full user information for each of the provided user IDs. |
| `ReorderActiveUsernames` | `ReorderActiveUsernames(ctx context.Context, order []string) (error)` | ReorderActiveUsernames changes the display order of the logged-in user's active usernames. |
| `SetBio` | `SetBio(ctx context.Context, bio string) (error)` | SetBio updates the current user's bio/about text. |
| `SetBirthday` | `SetBirthday(ctx context.Context, month, day, year int32) (error)` | SetBirthday sets the logged-in user's birthday shown to contacts. |
| `SetEmojiStatus` | `SetEmojiStatus(ctx context.Context, customEmojiID int64, until int32) (error)` | SetEmojiStatus sets the custom emoji displayed next to the logged-in user's name. |
| `SetProfilePhoto` | `SetProfilePhoto(ctx context.Context, photo tg.InputFileClass) (error)` | SetProfilePhoto uploads and sets a new profile photo for the current user. |
| `SetUsername` | `SetUsername(ctx context.Context, username string) (error)` | SetUsername updates the current user's username. |
| `ToggleUsername` | `ToggleUsername(ctx context.Context, username string, active bool) (error)` | ToggleUsername activates or deactivates one of the logged-in user's usernames. |
| `UnblockUser` | `UnblockUser(ctx context.Context, userID int64) (error)` | UnblockUser removes the specified user from the blocked list, allowing them to interact with the current account again. |
| `UpdateProfile` | `UpdateProfile(ctx context.Context, firstName, lastName, bio string) (*types.User, error)` | UpdateProfile updates the current user's first name, last name, and bio. |
| `UpdateStatus` | `UpdateStatus(ctx context.Context, offline bool) (error)` |  |

### Utility

dialogs, peers, files, config — `*UtilityMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `CachePeer` | `CachePeer(id int64, peer tg.InputPeerClass)` | CachePeer stores an InputPeer in the peer cache under the given numeric ID, canonicalizing the key and preserving known access hashes. |
| `Download` | `Download(ctx context.Context, input any, fileName string, progress params.ProgressFunc) (string, error)` | Download downloads media from a message or media object to a file and returns the absolute file path. |
| `DownloadBytes` | `DownloadBytes(ctx context.Context, input any, progress params.ProgressFunc) ([]byte, error)` | DownloadBytes downloads media from a message or media object into memory and returns the raw bytes. |
| `DownloadFile` | `DownloadFile(ctx context.Context, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) ([]byte, error)` | DownloadFile downloads a file from the given location into memory and returns its contents as a byte slice. |
| `DownloadToFile` | `DownloadToFile(ctx context.Context, location tg.InputFileLocationClass, filePath string, fileSize int64, opts *params.Download) (error)` | DownloadToFile downloads a file from the given location and writes it directly to disk at filePath. |
| `GetAppConfig` | `GetAppConfig(ctx context.Context, hash int32) (*tg.HelpAppConfig, error)` | GetAppConfig returns the server-side application configuration used for feature flags and UI limits. |
| `GetDialogs` | `GetDialogs(ctx context.Context, limit int, offsetDate int32) ([]*types.Chat, error)` | GetDialogs retrieves a slice of the user's dialogs (conversations). |
| `ResolvePeer` | `ResolvePeer(ctx context.Context, peerID any) (tg.InputPeerClass, error)` | ResolvePeer resolves a ChatRef (ID, username, or InputPeer) into an InputPeerClass suitable for use in API calls. |
| `ResolvePeerCache` | `ResolvePeerCache(id int64) (tg.InputPeerClass, error)` | ResolvePeerCache looks up a previously cached InputPeer by its numeric ID. |
| `ResolvePhone` | `ResolvePhone(ctx context.Context, phone string) (tg.InputPeerClass, error)` | ResolvePhone resolves a phone number to an [tg.InputPeerClass]. |
| `ResolveUsername` | `ResolveUsername(ctx context.Context, username string) (tg.InputPeerClass, error)` | ResolveUsername resolves a Telegram username (with or without the leading "@") to an [tg.InputPeerClass]. |
| `StreamFile` | `StreamFile(ctx context.Context, location tg.InputFileLocationClass, fileSize int64, opts *params.Download) (_, error)` | StreamFile downloads a file in chunks and delivers them over the returned read-only channel. |
| `UploadFile` | `UploadFile(ctx context.Context, reader io.Reader, fileName string, fileSize int64, opts *UploadOptions) (*UploadResult, error)` | UploadFile uploads a file to Telegram servers by splitting it into parts and sending them concurrently. |

### Wallet

TON wallet — `*WalletMethods`

| Operation | Signature | Purpose |
|---|---|---|
| `GetExistingWalletBalance` | `GetExistingWalletBalance(ctx context.Context) (*tg.WalletExistingBalance, error)` | GetExistingWalletBalance checks whether the logged-in user already owns a TON balance on an external wallet, returning the URL to import it. |
| `GetWalletNFTs` | `GetWalletNFTs(ctx context.Context, offset string, limit int) (*WalletNFTs, error)` | GetWalletNFTs lists the NFTs held by the wallet. |
| `GetWalletState` | `GetWalletState(ctx context.Context) (*tg.WalletState, error)` | GetWalletState returns the TON wallet state of the logged-in user: address, balance, and backup capability. |
| `GetWalletTransactions` | `GetWalletTransactions(ctx context.Context, inbound, outbound bool, offset string, limit int) (*WalletTransactions, error)` | GetWalletTransactions lists the wallet's transaction history. |
| `GetWalletTransactionsByIDs` | `GetWalletTransactionsByIDs(ctx context.Context, ids []string) ([]*tg.WalletTransaction, error)` | GetWalletTransactionsByIDs fetches specific wallet transactions by their identifiers. |
| `GetWalletTransactionsByMessageHash` | `GetWalletTransactionsByMessageHash(ctx context.Context, hashes []string) ([]*tg.WalletTransaction, error)` | GetWalletTransactionsByMessageHash fetches wallet transactions referenced by external message hashes. |
| `GetWalletUserAddresses` | `GetWalletUserAddresses(ctx context.Context, userIDs []int64, addresses []string, force bool) ([]*tg.WalletUserAddress, error)` | GetWalletUserAddresses resolves wallet addresses for users, or users for addresses. |
| `SendWalletTransfer` | `SendWalletTransfer(ctx context.Context, userID int64, dataNormal, dataGasless []byte) (error)` | SendWalletTransfer submits a signed TON transfer to the wallet. |

## Bound methods by type

Convenience forms are listed; each also has a `NameContext` form accepting a `context.Context`.

- **ActiveSession** (SessionBinder): Reset
- **CallbackQuery** (CallbackQueryBinder): Answer, AnswerAlert, AnswerURL, Delete, EditCaption, EditMedia, EditMessage, EditMessageCaption, EditMessageMedia, EditMessageReplyMarkup, EditMessageText, EditReplyMarkup, EditRich, Reply
- **Chat** (ChatBinder): AddMembers, Archive, BanMember, DC, DeletePhoto, ExportInviteLink, FullName, GetChat, GetEventLog, GetMember, GetMembers, Join, Leave, MarkUnread, MentionName, Mute, PromoteMember, RestrictMember, SetAdminTitle, SetDescription, SetPermissions, SetPhoto, SetProtectedContent, SetSlowMode, SetTTL, SetTitle, SetUsername, String, Unarchive, UnbanMember, Unmute, UnpinAll, UnpinAllMessages
- **ChatJoinRequest** (ChatJoinRequestBinder): Approve, Decline
- **Folder** (FolderBinder): Delete, Edit, ExcludeChat, IncludeChat, PinChat, RemoveChat, UpdateColor
- **Gift** (GiftBinder): Convert, Hide, Show, Transfer, Upgrade
- **InlineQuery** (InlineQueryBinder): Answer, AnswerArticle, AnswerArticles, AnswerDocument, AnswerDocuments, AnswerGame, AnswerPhoto, AnswerPhotos, AnswerResult, AnswerResults
- **Message** (MessageBinder): AcceptGiftPurchaseOffer, Answer, AnswerAnimation, AnswerAudio, AnswerCachedMedia, AnswerChecklist, AnswerContact, AnswerDice, AnswerDocument, AnswerGame, AnswerInlineBotResult, AnswerInvoice, AnswerLocation, AnswerMedia, AnswerMediaGroup, AnswerPaidMedia, AnswerPhoto, AnswerPoll, AnswerRich, AnswerSticker, AnswerVenue, AnswerVideo, AnswerVideoNote, AnswerVoice, Click, Copy, CopyMediaGroup, Delete, Download, DownloadTo, Edit, EditCaption, EditChecklist, EditLiveLocation, EditMedia, EditReplyMarkup, EditText, EditWidgetData, Forward, GetMediaGroup, Pay, Pin, React, Read, RejectGiftPurchaseOffer, Reply, ReplyAnimation, ReplyAudio, ReplyCachedMedia, ReplyChatAction, ReplyChecklist, ReplyContact, ReplyDice, ReplyDocument, ReplyGame, ReplyInlineBotResult, ReplyInvoice, ReplyLocation, ReplyMedia, ReplyMediaGroup, ReplyPaidMedia, ReplyPhoto, ReplyPoll, ReplyRich, ReplySticker, ReplyText, ReplyVenue, ReplyVideo, ReplyVideoNote, ReplyVoice, RetractVote, Send, SendMedia, SetTranslator, StopLiveLocation, String, Summarize, T, Unpin, View, Vote
- **PreCheckoutQuery** (PreCheckoutQueryBinder): Answer
- **ShippingQuery** (ShippingQueryBinder): Answer
- **Story** (StoryBinder): Copy, Delete, Download, EditCaption, EditMedia, EditPrivacy, Forward, React, Read, Reply, ReplyAnimation, ReplyAudio, ReplyCachedMedia, ReplyMedia, ReplyMediaGroup, ReplyPhoto, ReplySticker, ReplyText, ReplyVideo, ReplyVideoNote, ReplyVoice, View
- **User** (UserBinder): Archive, Block, DC, FullName, GetCommonChats, Mention, MentionName, String, Unarchive, Unblock, Verification
- **WidgetQuery** (WidgetQueryBinder): Answer

## Option structs

Optional parameters live in `telegram/params` structs passed as trailing variadic arguments
(for example `&params.SendMessage{ReplyToMessageID: 42}`); there are no `With*` helpers.

```
AnswerCallback, AnswerInlineQueryOption, AnswerPreCheckout, AnswerShipping, BuyGift, CopyMessage, DeleteMessages, Download, EditCaption, EditInlineOpts, EditMessage, EditPrivacy, Entity, ForwardMessages, GetGifts, GetPaymentFormOption, GetStarsTransactionsOption, GiftPurchaseOffer, GiftSend, InlineQuery, KeyboardButton, PinMessage, ProgressInfo, React, SendAnimation, SendAudio, SendChecklist, SendContact, SendDice, SendDiceOption, SendDocument, SendGame, SendInlineBotResult, SendInlineBotResultOption, SendLocation, SendMediaGroup, SendMessage, SendPaidReactionOption, SendPaymentFormOption, SendPhoto, SendPoll, SendReactionOption, SendSticker, SendVenue, SendVideo, SendVideoNote, SendVoice, SendWidget, StoryCopy, StoryForward
```
