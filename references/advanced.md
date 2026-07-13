# Advanced Topics — mtgo Reference

## Full Config Reference

| Field | Type | Default | Purpose |
|---|---|---|---|
| `BotToken` | `string` | | Bot authentication |
| `PhoneNumber` | `string` | | Userbot phone number |
| `PhoneCode` | `string` | | OTP code (automated flows) |
| `Password` | `string` | | 2FA password (automated flows) |
| `CodeFunc` | `CodeFunc` | terminal prompt | Function to provide OTP code |
| `PasswordFunc` | `PasswordFunc` | terminal prompt | Function to provide 2FA password |
| `SessionString` | `string` | | Import existing session |
| `SessionName` | `string` | | Session identifier |
| `InMemory` | `bool` | false | No session file on disk |
| `SavePeers` | `bool` | false | Cache peer info |
| `Storage` | `storage.Storage` | | Storage backend |
| `ParseMode` | `params.ParseMode` | | Default parse mode |
| `WorkDir` | `string` | cwd | Session file directory |
| `DC` | `int` | 0 (auto) | Datacenter number |
| `ServerAddr` | `string` | | Override DC address |
| `LocalAddr` | `string` | | Local bind address |
| `AutoConnect` | `bool` | false | Lazy connect on first use |
| `NoUpdates` | `bool` | false | Skip receiving updates |
| `SkipUpdates` | `bool` | false | Discard stale updates on reconnect |
| `Takeout` | `bool` | false | Enable takeout session for data export |
| `MTProxy` | `*MTProxyConfig` | | MTProxy config |
| `Proxy` | `*Proxy` | | SOCKS5/HTTP proxy |
| `TestMode` | `bool` | false | Connect to test DCs |
| `IPv6` | `bool` | false | Force IPv6 |
| `WebSocket` | `bool` | false | MTProto over WebSocket |
| `WebSocketTLS` | `bool` | false | TLS on WebSocket |
| `TransportMode` | `string` | Abridged | TCP framing mode |
| `Device` | `DeviceConfig` | | Device identity |
| `ClientPlatform` | `types.ClientPlatform` | | Platform identifier |
| `LinkPreviewOptions` | `*types.LinkPreviewOptions` | | Global link preview settings |
| `HidePassword` | `bool` | false | Mask 2FA password in logs |
| `Timeout` | `time.Duration` | 60s | TCP connection timeout |
| `ReqTimeout` | `time.Duration` | 60s | Default RPC timeout |
| `Retries` | `int` | 1 | RPC retry count |
| `HandlerTimeout` | `time.Duration` | 0 (none) | Max handler runtime |
| `SleepThreshold` | `time.Duration` | 0 | Flood-wait sleep duration |
| `MaxConcurrentTrans` | `int` | 0 (unlimited) | Max parallel file transfers |
| `DispatchWorkers` | `int` | GOMAXPROCS | TL-decode worker count |
| `DispatchQueueSize` | `int` | 256 | Incoming message queue size |
| `MaxMessageCacheSize` | `int` | 0 (unlimited) | Message cache limit |
| `MaxTopicCacheSize` | `int` | 1000 | Forum topic cache limit |
| `PeerCacheSize` | `int` | 0 (unlimited) | Peer cache limit (recommended: 5000) |
| `FetchReplies` | `bool` | false | Resolve reply-to references |
| `FetchTopics` | `bool` | false | Load forum topic metadata |
| `FetchStories` | `bool` | false | Retrieve stories |
| `FetchStickers` | `bool` | false | Download sticker metadata |
| `ReconnectEnabled` | `bool` | true | Auto-reconnect |
| `ReconnectBaseDelay` | `time.Duration` | 1s | Initial reconnect delay |
| `ReconnectMaxDelay` | `time.Duration` | 60s | Max reconnect delay |
| `ReconnectMaxAttempts` | `int` | 0 (unlimited) | Max reconnect tries |
| `HealthEnabled` | `bool` | true | Periodic health pings |
| `HealthPingInterval` | `time.Duration` | 60s | Health check interval |
| `HealthPongTimeout` | `time.Duration` | 30s | Pong timeout |
| `UpdateQueueSize` | `int` | 1024 | Update channel capacity |
| `DurableUpdateQueue` | `bool` | true | Persist updates across reconnects |
| `MaxUpdateHandlerRetry` | `int` | 3 | Handler error retries |
| `Log` | `LogConfig` | | Logging configuration |

### DeviceConfig

```go
telegram.DeviceConfig{
    DeviceModel:    "Samsung Galaxy S24",
    SystemVersion:  "Android 14",
    AppVersion:     "1.0.0",
    LangCode:       "en",
    LangPack:       "tdesktop",
    SystemLangCode: "en",
    TZOffset:       0,
    ClientPlatform: types.ClientPlatformAndroid,
}
```

### LogConfig

```go
telegram.LogConfig{
    Level:  telegram.LogLevelDebug,  // Debug, Info, Warn, Error
    File:   "/var/log/mtgo.log",
    MaxSize: 10 * 1024 * 1024,      // 10MB
    Logger: customLogger,            // *telegram.Logger override
}
```

### MTProxyConfig

```go
telegram.MTProxyConfig{
    Addr:   "proxy.example.com:443",
    Secret: "dd05fb7acb549be047a7c585116581418",
}
```

## Userbot Authentication

### Phone number flow

```go
client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
    PhoneNumber: "+1234567890",
    SessionName: "my_userbot",
})
client.Connect(0)

ctx := context.Background()
result, err := client.SendCode(ctx, "+1234567890")
if err != nil {
    log.Fatal(err)
}

user, err := client.SignIn(ctx, "+1234567890", result.PhoneCodeHash, "12345")
if err != nil {
    if errors.Is(err, telegram.Err2FARequired) {
        user, err = client.CheckPassword(ctx, "my_password")
    } else if errors.Is(err, telegram.ErrSignUpRequired) {
        user, err = client.SignUp(ctx, "+1234567890", result.PhoneCodeHash, "First", "Last")
    }
}
```

### Auth methods

| Method | Returns | Description |
|--------|---------|-------------|
| `client.SendCode(ctx, phone)` | `*SendCodeResult` | Request OTP |
| `client.SignIn(ctx, phone, hash, code)` | `*types.User` | Verify code |
| `client.SignUp(ctx, phone, hash, first, last...)` | `*types.User` | Register account |
| `client.CheckPassword(ctx, password)` | `*types.User` | Complete 2FA |
| `client.RecoverPassword(ctx, code)` | `*types.User` | Email recovery |
| `client.GetPasswordHint(ctx)` | `string` | 2FA hint |
| `client.SignOut(ctx)` | `(bool, error)` | Logout |
| `client.GetActiveSessions(ctx)` | `*ActiveSessions` | List devices |
| `client.ResetSession(ctx, hash)` | `error` | Terminate session |

Sentinel errors: `Err2FARequired`, `ErrSignUpRequired`, `ErrAlreadyAuthed`

### Custom code/password functions

```go
client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
    PhoneNumber: "+1234567890",
    SessionName: "my_userbot",
    CodeFunc: func(ctx context.Context, phone string) (string, error) {
        return readCodeFromWebhook(ctx)
    },
    PasswordFunc: func(ctx context.Context, hint string) (string, error) {
        return readPasswordFromVault()
    },
})
```

### QR login

```go
token, err := client.GetQRCodeLoginToken(ctx)
for {
    user, err := client.CheckQRCodeLoginToken(ctx, token.Token)
    if err != nil {
        time.Sleep(2 * time.Second)
        continue
    }
    fmt.Printf("Logged in as %s\n", user.FirstName)
    break
}
```

### Session export/import

```go
sessionStr, err := client.ExportSessionString()

import "github.com/mtgo-labs/mtgo/session"
str, err := session.String("AQFvs2s...")
client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
    SessionString: str,
    InMemory:      true,
    SavePeers:     true,
})
```

Auto-detects format (Telethon, Pyrogram, GramJS, mtcute, mtgo).

## Advanced RPC

### InvokeRaw — skip error wrapping

`client.InvokeRaw(ctx, query, retries, timeout)` returns the decoded TL response with original RPC error handling. Use when you need to inspect original RPC errors directly.

```go
raw, err := client.InvokeRaw(ctx, &tg.MessagesGetHistoryRequest{
    Peer:  &tg.InputPeerChannel{ChannelID: channelID, AccessHash: hash},
    Limit: 100,
}, 3, 10*time.Second)
```

### InvokeWithRawResult — raw MTProto result payload

Returns the raw `rpc_result.result:Object` bytes without gzip unpacking or TL decoding. RPC errors are returned as `*tgerr.Error`, not included in the raw bytes. DC migration is handled automatically.

```go
rawBytes, err := client.InvokeWithRawResult(ctx, &tg.PingRequest{PingID: rand.Int63()})
```

Use when doing bulk operations and don't need the full typed response, or implementing a custom parser.

### TDLib JSON compatibility

mtgo provides a TDLib-compatible JSON API for tools that expect tdlib-style responses:

```go
// JSON RPC via TDLib-compatible interface
resp, err := client.InvokeJSON(ctx, "messages.SendMessage", jsonBody, false)

// Get JSON-formatted updates
client.OnJSONUpdate(func(update json.RawMessage) { ... })
```

## Group Management

Groups require a **userbot session** (not bot token) for creation, adding members, and admin promotion.

### Create a group

```go
result, err := rpc.MessagesCreateChat(ctx, &tg.MessagesCreateChatRequest{
    Users: []tg.InputUserClass{&tg.InputUserEmpty{}},
    Title: "Bot Test Suite",
})

// Upgrade to supergroup
for _, u := range result.Updates {
    if msg, ok := u.(*tg.UpdateNewChat); ok {
        migrateResult, err := rpc.MessagesMigrateChat(ctx, &tg.MessagesMigrateChatRequest{
            ChatID: msg.ChatID,
        })
        _ = migrateResult
    }
}
```

> **Note:** `MessagesMigrateChat` takes a plain `int64` chat_id, NOT an `InputPeer`.

### Add bot to group

```go
// Supergroups/channels:
rpc.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
    Channel: &tg.InputChannel{ChannelID: channelID, AccessHash: channelHash},
    Users:   []tg.InputUserClass{&tg.InputUser{UserID: botUserID, AccessHash: botHash}},
})

// Basic groups:
rpc.MessagesAddChatUser(ctx, &tg.MessagesAddChatUserRequest{
    ChatID: chatID,
    UserID: &tg.InputUser{UserID: botUserID, AccessHash: botHash},
    FwdLimit: 100,
})
```

### Promote bot to admin (supergroup only)

```go
rpc.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
    Channel: &tg.InputChannel{ChannelID: channelID, AccessHash: channelHash},
    UserID:  &tg.InputUser{UserID: botUserID, AccessHash: botHash},
    AdminRights: &tg.ChatAdminRights{
        ChangeInfo: true, PostMessages: true, EditMessages: true,
        DeleteMessages: true, BanUsers: true, InviteUsers: true,
        PinMessages: true, ManageTopics: true,
    },
    Rank: "admin",
})
```

### Fast group setup with mtgo-cli

```bash
mtgo-cli invoke messages.createChat '{"Users":[{"_":"inputUserEmpty"}],"Title":"Test Group"}'
mtgo-cli invoke messages.migrateChat '{"chat_id": CHAT_ID}'
mtgo-cli invoke channels.inviteToChannel '{"channel":{"_":"inputChannel","channel_id":CID,"access_hash":HASH},"users":[{"_":"inputUser","user_id":BOT_ID,"access_hash":BOT_HASH}]}'
```

## Creating a Bot via BotFather

Requires a **userbot session** (bots can't create other bots).

```go
rpc := client.Raw()
// 1. Resolve BotFather
result, _ := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: "botfather"})
// BotFather's UserID is typically 93372553

// 2. Send /newbot
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer: &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Message: "/newbot", RandomID: rand.Int63(),
})

// 3. Send display name and username
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer: &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Message: "My Test Bot", RandomID: rand.Int63(),
})
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer: &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Message: "my_test_bot", RandomID: rand.Int63(),
})

// 4. Read response to extract token
history, _ := rpc.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
    Peer: &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Limit: 5,
})
```

## Testing Bots with Userbots

Use a userbot session to simulate real user interactions:

```go
// Resolve the bot
result, _ := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: "my_test_bot"})

// Send /start in DM
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer: &tg.InputPeerUser{UserID: botUserID, AccessHash: botHash},
    Message: "/start", RandomID: rand.Int63(),
})

// Send /start in group
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer: &tg.InputPeerChannel{ChannelID: channelID, AccessHash: channelHash},
    Message: "/start@my_test_bot", RandomID: rand.Int63(),
})

// Click inline buttons
rpc.MessagesGetBotCallbackAnswer(ctx, &tg.MessagesGetBotCallbackAnswerRequest{
    Peer: &tg.InputPeerUser{UserID: botUserID, AccessHash: botHash},
    MsgID: msgID, Data: []byte(callbackData),
})
```

### Testing checklist

- Bot responds in DM (check `updateNewMessage` from bot ID)
- Bot responds in group context
- Inline buttons produce callback queries
- Error messages are appropriate for invalid input
- Deep links resolve correctly

### mtgo-cli trace mode

```bash
mtgo-cli trace &
mtgo-cli send-message my_test_bot "/start"
# Trace shows: [1] >> messages.sendMessage / [1] << messages.sendMessage [45ms]
# [2] UPDATE updateNewMessage from bot
```

## File Transfer Performance

File uploads and downloads are **parallelized by default** — large files are split into chunks and transferred concurrently for maximum throughput.

- **Uploads** run on a **dedicated session** to avoid blocking update delivery and RPC calls on the main session. No configuration needed — this is automatic.
- **Downloads** fan out across CDN and DC workers with cascade-kill safety for same-DC sessions.

Control parallelism with the `MaxConcurrentTrans` Config field:

```go
client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
    MaxConcurrentTrans: 4, // limit to 4 concurrent chunk transfers (0 = unlimited)
})
```

The `Progress` callback on `DownloadMedia` receives `ProgressInfo` with `DownloadedBytes` and `TotalBytes` for progress bars.
