# New Features — mtgo Reference

## Business Connections

Telegram Business lets businesses connect bots to their accounts. The bot can read and send messages on behalf of the business.

```go
// Handle business messages
client.OnBusinessMessage(func(ctx *telegram.Context, msg *types.Message) {
    ctx.Reply("Thank you for contacting us!")
})

// Handle edited business messages
client.OnEditedBusinessMessage(func(ctx *telegram.Context, msg *types.Message) {
    log.Printf("Business message edited: %d", msg.ID)
})

// Handle deleted business messages
client.OnDeletedBusinessMessages(func(ctx *telegram.Context, deleted *types.DeletedMessages) {
    log.Printf("Business messages deleted: %v", deleted.IDs)
})

// Business connection lifecycle
client.OnBusinessConnection(func(ctx *telegram.Context) {
    bc := ctx.BusinessConnection
    log.Printf("Business connection: %s (enabled=%v)", bc.ID, bc.Enabled)
})

// Guest messages (users pending approval in business chats)
client.OnGuestMessage(func(ctx *telegram.Context, msg *types.Message) {
    ctx.Reply("Welcome! A team member will be with you shortly.")
})
```

## Managed Bots

Track when a managed bot's connection state or settings change:

```go
client.OnManagedBot(func(ctx *telegram.Context) {
    mb := ctx.ManagedBot
    log.Printf("Managed bot updated: %+v", mb)
})
```

## Secret Chats

End-to-end encrypted chats using the `SecretChatManager`:

```go
// Register secret chat request handler
client.OnSecretChatRequest(func(chatID int32, fromID int64) bool {
    log.Printf("Secret chat request from %d", fromID)
    return true // accept
})

// Register secret message handler
client.OnSecretMessage(func(ctx *telegram.Context) {
    msg := ctx.SecretMessage
    log.Printf("Decrypted secret message: %v", msg)
})
```

Secret chat messages are automatically decrypted. The `SecretChat` field on Context provides the secret chat state.

## Cloud Password Management

Manage 2FA passwords programmatically:

```go
// Enable cloud password
err := client.EnableCloudPassword(ctx, "new_password", "optional_hint")

// Change cloud password
err := client.ChangeCloudPassword(ctx, "old_password", "new_password", "optional_hint")

// Remove cloud password
err := client.RemoveCloudPassword(ctx, "current_password")
```

## Gifts & Star Gifts

```go
// Get available star gift options
gifts, err := client.GetStarGiftOptions(ctx)
for _, g := range gifts {
    fmt.Printf("Gift: %s (stars: %d)\n", g.Title, g.Stars)
}

// Resolve a gift offer (accept or decline)
msg, err := client.ResolveGiftOffer(ctx, messageID, true) // accept=true
```

Gift types in `telegram/types/gift.go` include `Gift`, `GiftUpgraded`, `GiftResale`, `GiftAttribute`, and `GiftForResaleOrder`.

## Paid Media

Handle purchases of paid media:

```go
client.OnPurchasedPaidMedia(func(ctx *telegram.Context) {
    pm := ctx.PurchasedPaidMedia
    log.Printf("Paid media purchased by %d", pm.UserID)
})
```

Use the `PaidMedia` filter to restrict handlers to paid media messages.

## Live Broadcasting

Stream live video to group calls via RTMP/FFmpeg:

```go
stream, err := client.NewBroadcastStream(chatID)
if err != nil {
    log.Fatal(err)
}

// Fetch RTMP URL from Telegram
err = stream.FetchRTMPURL(ctx)

// Configure FFmpeg options
stream.SetVideoCodec("libx264")
stream.SetAudioCodec("aac")
stream.SetResolution(1280, 720)
stream.SetFramerate(30)
stream.SetBitrate(2500)

// Start from file
err = stream.StartFile("input.mp4")
// Start from URL
err = stream.StartURL("rtsp://source.example.com/live")
// Start from pipe (stdin)
err = stream.StartPipe()

// Control playback
stream.Stop()
stream.Pause()
stream.Resume()

// Set loop count (0 = infinite)
stream.SetLoopCount(0)

// Callbacks
stream.OnEnd(func(chatID int64) { log.Println("stream ended") })
stream.OnError(func(chatID int64, err error) { log.Printf("stream error: %v", err) })

// Refresh RTMP credentials (revokes old ones)
err = stream.RefreshRTMPURL(ctx)
```

## TDLib JSON Compatibility

For tools and libraries expecting tdlib-style JSON responses:

```go
// JSON RPC calls
resp, err := client.InvokeJSON(ctx, "messages.SendMessage", jsonBody, false)

// Receive JSON-formatted updates
client.OnJSONUpdate(func(update json.RawMessage) {
    fmt.Println(string(update))
})

// TDLib-compatible client wrapper
tdlibClient := telegram.NewTDLibJSONClient(client)
```

This provides compatibility with tools designed for the TDLib JSON interface.

## Account Privacy Settings

```go
// Set privacy for a specific key
client.SetPrivacy(ctx, types.PrivacyKeyStatusTimestamp, []types.InputPrivacyRuleClass{
    &types.PrivacyValueAllowAll{},
})

// Get current privacy rules
rules, err := client.GetPrivacy(ctx, types.PrivacyKeyStatusTimestamp)

// Global privacy settings
client.SetGlobalPrivacySettings(ctx, &types.GlobalPrivacySettings{
    ArchiveAndMuteNewNoncontactPeers: true,
})
settings, err := client.GetGlobalPrivacySettings(ctx)

// Account TTL
client.SetAccountTTL(ctx, 180) // days
ttl, err := client.GetAccountTTL(ctx)
```

Privacy keys: `PrivacyKeyStatusTimestamp`, `PrivacyKeyChatInvite`, `PrivacyKeyPhoneCall`, `PrivacyKeyPhoneP2P`, `PrivacyKeyForwards`, `PrivacyKeyProfilePhoto`, `PrivacyKeyPhoneNumber`, `PrivacyKeyAddedByPhone`, `PrivacyKeyVoiceMessages`, `PrivacyKeyAbout`, `PrivacyKeyBirthday`

## Profile Management

```go
// Update profile
err := client.SetProfileName(ctx, "First", "Last")
err := client.SetProfileBio(ctx, "My bio text")
err := client.SetProfileUsername(ctx, "new_username")

// Profile photo
client.SetProfilePhoto(ctx, telegram.Path("avatar.jpg"), nil)
client.DeleteProfilePhoto(ctx, photoID)
```

## Bot Info & Commands

```go
// Bot info
client.SetBotInfoDescription(ctx, botID, langCode, "Bot description text")
client.SetBotInfoShortDescription(ctx, botID, langCode, "Short description")
desc, _ := client.GetBotInfoDescription(ctx, botID, langCode)
client.SetBotName(ctx, botID, langCode, "Bot Display Name")
name, _ := client.GetBotName(ctx, botID, langCode)

// Bot commands
client.SetBotCommands(ctx, []types.BotCommand{
    {Command: "start", Description: "Start the bot"},
    {Command: "help", Description: "Show help"},
})

// Menu button
client.SetMenuButton(ctx, userID, &types.MenuButtonWebApp{
    Text: "Open App",
    URL:  "https://example.com/app",
})
```

## Forum Topics

```go
// Create topic
result, err := client.CreateForumTopic(ctx, chatID, "Announcements")

// Edit topic
client.EditForumTopic(ctx, chatID, topicID, "New Name")

// Close/open topic
client.CloseForumTopic(ctx, chatID, topicID)
client.OpenForumTopic(ctx, chatID, topicID)

// Delete topic
client.DeleteForumTopic(ctx, chatID, topicID)
```

Use `telegram.Topic(42)` filter to scope handlers to specific topics.

## Invite Links

```go
// Create invite link
link, err := client.CreateChatInviteLink(ctx, chatID, &params.CreateChatInviteLink{
    ExpireDate:  time.Now().Add(24 * time.Hour),
    MemberLimit: 10,
})

// Edit invite link
client.EditChatInviteLink(ctx, chatID, link.Link, &params.EditChatInviteLink{
    MemberLimit: 20,
})

// Revoke invite link
client.RevokeChatInviteLink(ctx, chatID, link.Link)

// Get invite link importers
importers, _ := client.GetChatInviteLinkImporters(ctx, chatID, link.Link)
```

## Chat Folders

```go
// Create chat folder
client.CreateChatFolder(ctx, &types.ChatFolder{
    Title:   "Work",
    IncludedChats: []int64{chatID1, chatID2},
})
```

## Stories

```go
client.OnStory(func(ctx *telegram.Context) {
    story := ctx.Story
    fmt.Printf("Story from %d: %v\n", story.FromID, story.Media)
})
```

## Premium Features

```go
// Check premium status
user, _ := client.GetMe(ctx)
if user.Premium {
    // premium-only features
}
```

## Inline Mode

```go
client.OnInlineQuery(func(ctx *telegram.Context) {
    query := ctx.InlineQuery
    results := []tg.InputBotInlineResultClass{
        &tg.InputBotInlineResultDocument{
            ID:       "1",
            Type:     "article",
            Title:    "Result 1",
            Document: &tg.InputDocument{ID: docID, AccessHash: hash},
        },
    }
    ctx.AnswerInlineQuery(results)
})

client.OnChosenInlineResult(func(ctx *telegram.Context, result *types.ChosenInlineResult) {
    log.Printf("User chose result: %s", result.ResultID)
})
```

## Error Handler

Register a handler for update processing errors:

```go
client.AddHandler(telegram.NewErrorHandler(func(ctx *telegram.Context) {
    log.Printf("Update error: %v", ctx.Error)
}))
```

## Lifecycle Handlers

React to client lifecycle events for initialization and cleanup:

```go
// Runs after successful connection
client.AddHandler(telegram.NewConnectHandler(func(ctx *telegram.Context) {
    log.Println("Connected!")
    client.SendMessage(ctx.Ctx, adminChatID, "Bot is online", nil)
}))

// Runs on disconnection
client.AddHandler(telegram.NewDisconnectHandler(func(ctx *telegram.Context) {
    log.Println("Disconnected")
}))

// Runs at startup, before connection
client.AddHandler(telegram.NewStartHandler(func(ctx *telegram.Context) {
    log.Println("Starting up...")
}))

// Runs at shutdown
client.AddHandler(telegram.NewStopHandler(func(ctx *telegram.Context) {
    log.Println("Shutting down...")
}))
```

The Context fields `Connected`, `Disconnected`, `Started`, and `Stopped` are set for lifecycle events.
