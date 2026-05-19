---
name: mtgo-mtproto-go
description: Build Telegram bots and userbots with mtgo (MTProto Go) — a fast, idiomatic Go client for the Telegram MTProto API. Use this skill whenever the user mentions Telegram bots, Telegram bots in Go, userbots, MTProto, mtgo, Telegram client library, or wants to build any Telegram-related application in Go — even if they don't explicitly say "mtgo". Also use when the user asks about Telegram message handlers, inline keyboards, callback queries, media upload/download, session management, Telegram authentication (bot token, phone number, QR login, session strings), MTProxy, middleware, plugins, conversations, i18n, storage backends, multi-client setups, creating groups, managing Telegram groups, creating bots via BotFather, testing Telegram bots, or high-performance Telegram operations via mtgo-cli.
---

# mtgo — Telegram MTProto Client for Go

mtgo is a Go library for building Telegram bots and userbots using the MTProto 2.0 protocol. It provides a high-level client API with handlers, filters, middleware, plugins, and storage backends.

## Quick Reference

**Module:** `github.com/mtgo-labs/mtgo`  
**API Reference:** <https://pkg.go.dev/github.com/mtgo-labs/mtgo>

Use `go doc` to look up types and methods:
```
go doc github.com/mtgo-labs/mtgo/telegram Client
go doc github.com/mtgo-labs/mtgo/telegram Filter
go doc github.com/mtgo-labs/mtgo/telegram/params SendMessage
```

**Key packages:**
- `telegram` — high-level client, handlers, filters, middleware, keyboards
- `telegram/types` — Message, User, Chat, CallbackQuery, media types
- `telegram/params` — SendMessage, SendPhoto, Download, ProgressInfo, entities
- `tg` — generated TL types and RPC methods (low-level)
- `tgerr` — generated error types and error constants
- `session` — session string import/export (Telethon, Pyrogram, GramJS, mtcute)

**Ecosystem packages:**
- `github.com/mtgo-labs/storage/sqlite` — SQLite storage
- `github.com/mtgo-labs/storage/postgres` — PostgreSQL storage
- `github.com/mtgo-labs/storage/mongodb` — MongoDB storage
- `github.com/mtgo-labs/storage` — storage.NewAdapter() wrapper
- `github.com/mtgo-labs/plugins/conversations` — conversation/state machine plugin
- `github.com/mtgo-labs/plugins/i18n` — internationalization plugin
- `github.com/mtgo-labs/middlewares/floodwait` — flood wait auto-retry middleware
- `github.com/mtgo-labs/middlewares/ratelimit` — rate limiting middleware

## Client Creation

```go
import "github.com/mtgo-labs/mtgo/telegram"

// Bot with token
client, err := telegram.NewClient(apiID, apiHash, &telegram.Config{
    BotToken:    os.Getenv("BOT_TOKEN"),
    SessionName: "my_bot",
    SavePeers:   true,
})

// Bot with in-memory session
client, err := telegram.NewClient(apiID, apiHash, &telegram.Config{
    BotToken:    botToken,
    SessionName: "my_bot",
    InMemory:    true,
    SavePeers:   true,
    ParseMode:   telegram.HTML,
})

// Userbot (phone number) — terminal prompts for code/password
client, err := telegram.NewClient(apiID, apiHash, &telegram.Config{
    PhoneNumber: "+1234567890",
    SessionName: "my_userbot",
})

// Session string import (auto-detects format)
client, err := telegram.NewClient(apiID, apiHash, &telegram.Config{
    SessionString: sessionStr,
    InMemory:      true,
    SavePeers:     true,
})

// With storage backend
client, err := telegram.NewClient(apiID, apiHash, &telegram.Config{
    BotToken:    botToken,
    SessionName: "my_bot",
    Storage:     sqlite.New(),
})
```

The `apiID` is `int32` and `apiHash` is `string`, obtained from https://my.telegram.org.

### Config fields

| Field | Type | Purpose |
|---|---|---|
| `BotToken` | `string` | Bot authentication |
| `PhoneNumber` | `string` | Userbot authentication |
| `SessionString` | `string` | Import existing session |
| `SessionName` | `string` | Session identifier |
| `InMemory` | `bool` | No session file on disk |
| `SavePeers` | `bool` | Cache peer info |
| `ParseMode` | `string` | Default parse mode |
| `Storage` | `storage.Storage` | Storage backend |
| `MTProxy` | `*telegram.MTProxyConfig` | MTProxy config |
| `NoUpdates` | `bool` | Skip receiving updates |
| `AutoConnect` | `bool` | Lazy connect on first use |
| `WebSocket` | `bool` | MTProto over WebSocket |
| `HandlerTimeout` | `time.Duration` | Max handler runtime |
| `ReqTimeout` | `time.Duration` | Default RPC timeout (60s) |
| `Retries` | `int` | RPC retry count |
| `ReconnectEnabled` | `bool` | Auto-reconnect (default true) |

## Lifecycle

The client is started with **`Connect(0)`** (zero means auto-detect the nearest DC), then blocked with **`Idle()`** until stopped. Always defer `Stop()` so the session is properly persisted on exit. There is no `Start()` method — `Connect` plus `Idle` is the only pattern.

```go
// Every bot/userbot follows this exact skeleton:
client.Connect(0)
defer client.Stop()
client.Idle() // blocks until Stop() is called
```

If the program exits without calling `Stop()`, the session won't be saved to disk/storage, so the next start will require a fresh login. **Always defer `client.Stop()` right after `Connect(0)`.**

For multi-client setups use `telegram.Compose` or `telegram.Idle`:
```go
telegram.Compose(bot1, bot2)  // block until any stops
telegram.Idle()               // block until ALL registered clients stop
```

Graceful shutdown with OS signals — use this when you need to stop from outside Idle():
```go
shutdownCtx, stopNotify := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stopNotify()
go func() {
    <-shutdownCtx.Done()
    client.Stop() // this will cause Idle() to return
}()
client.Idle()
```

## Handlers

### Registration methods

```go
client.OnMessage(callback, filters...)        // new messages
client.OnEditedMessage(callback, filters...)  // edited messages
client.OnCallbackQuery(callback, filters...)  // inline button presses
client.OnInlineQuery(callback, filters...)    // inline mode queries
client.OnDeletedMessages(callback, filters...) // deleted messages
client.OnUserStatus(callback, filters...)     // user online/offline
client.OnChatMember(callback, filters...)     // member status changes
client.OnMessageReaction(callback, filters...) // reactions
client.OnPoll(callback, filters...)           // poll updates
client.OnStory(callback, filters...)          // stories
client.OnChatJoinRequest(callback, filters...) // join requests
client.OnPreCheckoutQuery(callback, filters...) // payment checkout
client.OnShippingQuery(callback, filters...)  // shipping options
client.OnRawUpdate(callback, filters...)      // raw MTProto updates
```

### Handler callback signatures

The framework accepts multiple callback signatures via reflection. Use whichever feels most natural:

```go
// Style 1: Context only (access message via ctx.Message)
client.OnMessage(func(ctx *telegram.Context) {
    ctx.Reply("Hello!")
})

// Style 2: Context + Message
client.OnMessage(func(ctx *telegram.Context, msg *types.Message) {
    ctx.Reply("Got: " + msg.Text)
})

// Style 3: Client + Message (direct API access)
client.OnMessage(func(client *telegram.Client, msg *types.Message) {
    msg.Reply("Echo: " + msg.Text)
})
```

Callback queries work the same way — access data via `ctx.CallbackQuery`:
```go
client.OnCallbackQuery(func(ctx *telegram.Context) {
    data := string(ctx.CallbackQuery.Data)
    ctx.CallbackEditText("You pressed: " + data, nil)
    ctx.Answer("", false)
})
```

### Handler groups (ordered dispatch)

Lower group numbers execute first. When a handler calls `ctx.StopPropagation()`, no further groups process the update. Use this to build a pipeline — for example: logging first, then moderation, then commands.

```go
// Group -10: Logging (runs first, never stops propagation)
loggingHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
    if ctx.Message != nil {
        log.Printf("[%d] %s", ctx.Message.ChatID, ctx.Message.Text)
    }
})
client.AddHandler(loggingHandler, -10)

// Group 0: Moderation (runs second, stops propagation on match)
urlHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
    ctx.Delete()
    ctx.Reply("Links are not allowed")
    ctx.StopPropagation()
}, telegram.Regex(`https?://[\S]+`))
client.AddHandler(urlHandler, 0)

// Group 10: Commands (runs last, only if moderation didn't match)
spamHandler := telegram.NewMessageHandler(func(ctx *telegram.Context) {
    ctx.Reply("Goodbye!")
    client.BanChatMember(ctx.Ctx, ctx.Message.ChatID, ctx.Message.FromID)
}, telegram.Command("spam"))
client.AddHandler(spamHandler, 10)
```

The priority numbers are arbitrary — lower numbers fire first. Common conventions: `-10` for logging/middleware, `0` for business logic, `10` for catch-all commands.

`client.AddHandler` is the primary way to register handlers with a priority. You can also use `client.OnMessage(callback, filters...)` for simple cases (registered at priority 0), but `AddHandler` gives you explicit control over dispatch order.

## Filters

Filters restrict which updates trigger a handler. Pass them as variadic arguments after the callback:

```go
client.OnMessage(handler, telegram.Private)        // one filter
client.OnMessage(handler, telegram.Command("start")) // command filter
client.OnMessage(handler, telegram.Private.And(telegram.HasText)) // combined
```

### Built-in filters

**Chat type:** `Private`, `Group`, `Channel`, `Direct`, `Forum`, `Business`

**Message content:** `HasText`, `Media`, `Photo`, `Video`, `Audio`, `Voice`, `VideoNote`, `Sticker`, `Animation`, `Document`, `Contact`, `Location`, `Venue`, `Poll`, `Game`, `Dice`, `Invoice`, `PaidMedia`, `WebPage`, `Caption`, `MediaGroup`, `MediaSpoiler`, `Story`, `Service`

**Message properties:** `Incoming`, `Outgoing`, `Me`, `Bot`, `Forwarded`, `Reply`, `Mentioned`, `ViaBot`, `Pinned`, `LinkedChannel`, `SelfDestruction`

**Service messages:** `NewChatMembers`, `LeftChatMember`, `NewChatTitle`, `NewChatPhoto`, `DeleteChatPhoto`, `GroupChatCreated`, `SupergroupChatCreated`, `ChannelChatCreated`, `PinnedMessage`, `VideoChatStarted`, `VideoChatEnded`, `GameHighScore`

**Parameterized filters:**
```go
telegram.Command("start", "help")           // match /start or /help
telegram.Text("exact match")                // exact text match
telegram.Regex(`\d+`)                       // regex match
telegram.User(123456, 789012)              // specific user IDs
telegram.Chat(-1001234567890)              // specific chat IDs
telegram.Topic(42)                          // forum topic ID
telegram.SenderChat(-1001234567890)        // sender chat ID
telegram.CallbackData("approve")           // exact callback data
telegram.CallbackRegex(`^page_\d+$`)       // callback regex
telegram.InlineQueryText("search")         // exact inline query
telegram.NewCommand([]string{"start"}, []string{"/", "!"}, false) // custom prefix
telegram.Create(func(c *telegram.Client, ctx *telegram.Context) bool {
    return isAdmin(c, ctx.Message.FromID)
})
```

**Composing filters:**
```go
privateText := telegram.Private.And(telegram.HasText)
notBot := telegram.Bot.Not()
mediaOrCommand := telegram.Media.Or(telegram.Command("upload"))
```

## Middleware

Two middleware levels for different concerns:

### Handler middleware (update dispatch)

Wraps the handler chain. Lower priority number = runs first:

```go
// Logging middleware
client.UseMiddleware(func(next telegram.Handler) telegram.Handler {
    return &telegram.FuncHandler{Fn: func(ctx *telegram.Context) {
        if ctx.Message != nil {
            log.Printf("[%d] %s", ctx.Message.ChatID, ctx.Message.Text)
        }
        next.Handle(ctx)
    }}
}, -10) // priority -10 = outermost

// Auth guard
client.UseMiddleware(func(next telegram.Handler) telegram.Handler {
    return &telegram.FuncHandler{Fn: func(ctx *telegram.Context) {
        if ctx.Message != nil && ctx.Message.FromID != adminID {
            ctx.Reply("Unauthorized")
            ctx.Stopped = true
            return
        }
        next.Handle(ctx)
    }}
})
```

### Invoker middleware (RPC calls)

Intercepts all outgoing Telegram API calls:

```go
// Flood wait auto-retry
waiter := floodwait.New()
waiter.OnWait(func(d time.Duration) {
    log.Printf("flood wait: sleeping %v", d)
})
client.UseInvokerMiddleware(waiter.Middleware())

// Rate limiting
limiter := ratelimit.New(20, 5)
client.UseInvokerMiddleware(limiter.Middleware())

// Custom invoker middleware (e.g., force silent messages)
client.UseInvokerMiddleware(func(next tg.Invoker) tg.Invoker {
    return tg.InvokerFunc(func(ctx context.Context, input tg.TLObject, decode func(io.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
        if req, ok := input.(*tg.MessagesSendMessageRequest); ok {
            req.Silent = true
            req.SetFlags()
        }
        return next.RPCInvoke(ctx, input, decode)
    })
})
```

## Plugins

Plugins implement the `Plugin` interface with lifecycle hooks:

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my_plugin" }
func (p *MyPlugin) Start(ctx context.Context, client *telegram.Client) error { return nil }
func (p *MyPlugin) Stop(ctx context.Context) error { return nil }

client.Use(&MyPlugin{})
```

Plugins start/stop automatically with the client.

## Keyboards

Fluent builder: `telegram.Keyboard()` → chain buttons → `.Build()` (inline) or `.BuildReply(opts)` (reply).

```go
// Inline keyboard
markup := telegram.Keyboard().
    Callback("Yes", "yes").
    Callback("No", "no").
    Next().
    URL("Docs", "https://example.com").
    Build()

ctx.Reply("Choose:", &params.SendMessage{ReplyMarkup: markup})

// Reply keyboard
markup := telegram.Keyboard().
    Text("Option A").
    Text("Option B").
    BuildReply(telegram.ReplyOpts{Resize: true, OneTime: true})

ctx.Reply("Pick one:", &params.SendMessage{ReplyMarkup: markup})

// Remove keyboard
ctx.Reply("Done", &params.SendMessage{ReplyMarkup: telegram.RemoveKeyboard()})

// Force reply
ctx.Reply("Type something:", &params.SendMessage{ReplyMarkup: telegram.ForceReplyMarkup()})
```

**Inline buttons:** `Callback(text, data)`, `URL(text, url)`, `Switch(text, samePeer, query)`, `Copy(text, copyText)`, `Game(text)`, `Buy(text)`, `WebApp(text, url)`

**Reply buttons:** `Text(text)`, `RequestUser(text, id, max, opts)`, `RequestChannel(text, id)`, `RequestGroup(text, id)`

## Sending Messages

```go
// Simple reply
ctx.Reply("Hello!")
msg.Reply("Hello!")

// With options
ctx.Reply("<b>Bold</b>", &params.SendMessage{
    ParseMode: params.ParseModeHTML,
    ReplyMarkup: markup,
})

// To specific chat
client.SendMessage(ctx, chatID, "Hello", &params.SendMessage{})

// Entities (formatted text without parse mode)
ctx.Reply("Bold Italic Code", &params.SendMessage{
    Entities: params.Entities(
        params.Bold(0, 4),
        params.Italic(5, 6),
        params.Code(12, 4),
    ),
})
```

## Media

### Sending media

File sources: `telegram.Path("file.jpg")`, `telegram.URL("https://...")`, `telegram.FileID("...")`, `telegram.FromBytes([]byte{...}, "name.png")`

```go
client.SendPhoto(ctx, chatID, telegram.Path("photo.jpg"), "caption", &params.SendPhoto{})
client.SendVideo(ctx, chatID, telegram.Path("clip.mp4"), "caption", &params.SendVideo{
    Duration: 12.5, Width: 1280, Height: 720,
})
client.SendAudio(ctx, chatID, telegram.Path("song.mp3"), "caption", &params.SendAudio{
    Duration: 245, Performer: "Artist", Title: "Track",
})
client.SendDocument(ctx, chatID, telegram.Path("file.pdf"), "caption", nil)
client.SendAnimation(ctx, chatID, telegram.Path("meme.gif"), "caption", nil)
client.SendVoice(ctx, chatID, telegram.Path("voice.ogg"), "caption", nil)
client.SendSticker(ctx, chatID, telegram.Path("sticker.webp"))
client.SendVideoNote(ctx, chatID, telegram.Path("round.mp4"), nil)
```

### Downloading media

```go
// To memory
data, err := client.DownloadMedia(ctx, media, "", &params.Download{
    Progress: func(info params.ProgressInfo) {
        fmt.Printf("progress: %d/%d\n", info.DownloadedBytes, info.TotalBytes)
    },
})

// To file
err := client.DownloadMediaToFile(ctx, media, "", destPath, fileSize, &params.Download{})

// Media type switch
switch m := msg.Media.(type) {
case *types.PhotoMedia:
case *types.DocumentMedia:
    fmt.Println(m.FileName, m.FileSize, m.MimeType)
}
```

### Upload with progress

```go
result, err := client.UploadFile(ctx, file, filename, fileSize, &telegram.UploadOptions{
    Workers: 4,
    Progress: func(info params.ProgressInfo) {
        fmt.Printf("upload: %d/%d\n", info.UploadedBytes, info.TotalBytes)
    },
})
```

## Raw RPC / Invoke

When high-level methods aren't enough, use the generated TL RPC methods. For a full list of available Telegram API methods, refer to:

> **https://corefork.telegram.org/methods** — official Telegram API method reference
> **https://corefork.telegram.org/schema** — raw TL schema for constructors, interfaces, and method signatures

```go
rpc := client.Raw() // or client.RPC()

// Typed TL method
result, err := rpc.AuthSendCode(ctx, &tg.AuthSendCodeRequest{
    PhoneNumber: "+1234567890",
    APIID:       apiID,
    APIHash:     apiHash,
    Settings:    &tg.CodeSettings{},
})

// Resolve a username
peer, err := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
    Username: "durov",
})

// JSON RPC (dynamic calls)
resp, err := client.InvokeJSON(ctx, "messages.SendMessage", []byte(`{
    "peer": "inputPeerSelf",
    "message": "hello"
}`), false)
```

Context deadlines are extracted automatically for RPC timeouts. When no deadline is set, `Config.ReqTimeout` (default 60s) is used:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
result, err := rpc.MessagesGetMessages(ctx, req)
```

### InvokeRaw — skip error wrapping

`client.InvokeRaw(ctx, query, retries, timeout)` sends a TL object and returns the decoded TL response without wrapping or transforming RPC errors. Use this when you need to inspect the original RPC error directly instead of the processed version. For raw MTProto result payload bytes, use `InvokeWithRawResult`.

```go
// Get the decoded TL response with original RPC error handling
raw, err := client.InvokeRaw(ctx, &tg.MessagesGetHistoryRequest{
    Peer:  &tg.InputPeerChannel{ChannelID: channelID, AccessHash: hash},
    Limit: 100,
}, 3, 10*time.Second)
```

### InvokeWithRawResult — raw MTProto result payload

`client.InvokeWithRawResult(ctx, query)` sends a TL object and returns the raw MTProto `rpc_result.result:Object` payload bytes. The return value is not a decoded Go struct, and it is not necessarily gzip-unpacked; if Telegram returns `gzip_packed`, the returned bytes start with the `gzip_packed` constructor.

```go
// Fast ping: get the raw rpc_result payload bytes without TL decoding.
rawBytes, err := client.InvokeWithRawResult(ctx, &tg.PingRequest{PingID: rand.Int63()})

// Batch operation: check multiple peer access hashes without decode overhead
for _, peer := range peers {
    rawBytes, err := client.InvokeWithRawResult(ctx, &tg.MessagesGetHistoryRequest{
        Peer:  peer.InputPeer(),
        Limit: 1,
    })
    // rawBytes is the undecoded rpc_result payload; parse only what you need.
}
```

Use `InvokeWithRawResult` when:
- You're doing bulk operations and don't need the full typed response
- You only need part of the response and want to skip full TL deserialization
- You're implementing a custom parser that's faster than the generated decoder

## Userbot Authentication

### Phone number flow

The typical userbot auth flow is: `SendCode` → `SignIn` → handle `Err2FARequired` or `ErrSignUpRequired`.

```go
client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
    PhoneNumber: "+1234567890",
    SessionName: "my_userbot",
})
client.Connect(0)

ctx := context.Background()

// 1. Send verification code
result, err := client.SendCode(ctx, "+1234567890")
if err != nil {
    log.Fatal(err)
}

// 2. Sign in with the code the user received
user, err := client.SignIn(ctx, "+1234567890", result.PhoneCodeHash, "12345")
if err != nil {
    if errors.Is(err, telegram.Err2FARequired) {
        // Account has 2FA — provide password
        user, err = client.CheckPassword(ctx, "my_password")
    } else if errors.Is(err, telegram.ErrSignUpRequired) {
        // Phone not registered — create account
        user, err = client.SignUp(ctx, "+1234567890", result.PhoneCodeHash, "First", "Last")
    }
}
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Logged in as %s (ID: %d)\n", user.FirstName, user.ID)
```

### Auth methods

| Method | Returns | Description |
|--------|---------|-------------|
| `client.SendCode(ctx, phone)` | `*SendCodeResult` | Request OTP code. Result has `PhoneCodeHash`, `Type`, `NextType`, `Timeout` |
| `client.SignIn(ctx, phone, hash, code)` | `*types.User` | Verify code and sign in |
| `client.SignUp(ctx, phone, hash, first, last...)` | `*types.User` | Register new account |
| `client.CheckPassword(ctx, password)` | `*types.User` | Complete 2FA (SRP-based, plaintext never sent) |
| `client.RecoverPassword(ctx, code)` | `*types.User` | Fallback via email recovery code |
| `client.GetPasswordHint(ctx)` | `string` | Get the user's 2FA password hint |
| `client.SignOut(ctx)` | `(bool, error)` | Log out and invalidate session |
| `client.GetActiveSessions(ctx)` | `*ActiveSessions` | List all logged-in devices |
| `client.ResetSession(ctx, hash)` | `error` | Terminate a specific session |

Sentinel errors to check after `SignIn`:
- `telegram.Err2FARequired` — account has 2FA, call `CheckPassword`
- `telegram.ErrSignUpRequired` — phone not registered, call `SignUp`
- `telegram.ErrAlreadyAuthed` — session already authorized

### QR login

```go
token, err := client.GetQRCodeLoginToken(ctx)
if err != nil {
    log.Fatal(err)
}

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
// Export current session
sessionStr, err := client.ExportSessionString()

// Import — auto-detects format (Telethon, Pyrogram, GramJS, mtcute)
import "github.com/mtgo-labs/mtgo/session"

client, _ := telegram.NewClient(apiID, apiHash, &telegram.Config{
    SessionString: session.MustTelethon("1BVusO..."),
    InMemory:      true,
    SavePeers:     true,
})
```

## Context and Cancellation

`telegram.Context` carries a `context.Context` via `ctx.Ctx`:

```go
// In a handler, use ctx.Ctx for downstream calls
func myHandler(ctx *telegram.Context) {
    // Create a timeout for a specific RPC call
    rpcCtx, cancel := context.WithTimeout(ctx.Ctx, 5*time.Second)
    defer cancel()
    peer, err := ctx.Client.ResolvePeer(rpcCtx, "@username")
}

// Outside handlers, create a Context manually
bgCtx := context.Background()
tc := client.NewContext(bgCtx)

// Cancellation propagates through RPC calls
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
// if cancel() is called, pending RPC calls are aborted
```

## Context Helper Methods

```go
ctx.Reply(text, opts)                    // reply to current message
ctx.Sender()                             // get sender User
ctx.StopPropagation()                    // stop handler chain
ctx.T("key", args...)                    // i18n translate
ctx.ResolvePeer(id)                      // lookup peer by ID
ctx.CallbackEditText(text, opts)         // edit callback message
ctx.Answer(text, alert)                  // answer callback query
ctx.Client                               // access the Client directly
ctx.Message                              // current message (may be nil)
ctx.CallbackQuery                        // current callback query (may be nil)
ctx.InlineQuery                          // current inline query (may be nil)
```

## Error Handling

```go
import "github.com/mtgo-labs/mtgo/tgerr"

result, err := rpc.SomeMethod(ctx, req)
if err != nil {
    var rpcErr *tgerr.Error
    if errors.As(err, &rpcErr) {
        switch {
        case tgerr.Is(err, tgerr.ErrFloodWait):
            // handle rate limit
        case tgerr.Is(err, tgerr.ErrSessionPasswordNeeded):
            // 2FA required
        }
    }
    return err
}
```

## WebApp Validation

```go
secretKey := telegram.CreateWebAppSecretKey(botToken)
data, err := telegram.ParseWebAppData(secretKey, initData, 5*time.Minute)
// or simply:
data, err := telegram.ValidateWebAppData(botToken, initData, 5*time.Minute)
data.User.ID
data.User.FirstName
```

## Testing and Operations with mtgo-cli

For fast one-off operations (testing, debugging, group management), use `mtgo-cli` — see the `mtgo-cli` skill for full CLI reference. Install: `go install github.com/mtgo-labs/mtgo-cli/cmd/mtgo-cli@latest`

**Quick patterns:**
```bash
export MTGO_CLI_API_ID=12345 MTGO_CLI_API_HASH=... MTGO_CLI_BOT_TOKEN=...
mtgo-cli get-me --format json                           # identity check
mtgo-cli invoke users.getFullUser '{"id":{"_":"inputUserSelf"}}'  # raw TL call
mtgo-cli listen &                                       # persistent listener (reuse connection)
```

## Creating a Bot via BotFather

Use the raw RPC to interact with BotFather programmatically. Requires a **userbot session** (phone login, not bot token) — bots can't create other bots.

### Step 1: Resolve BotFather

```go
rpc := client.Raw()
result, err := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
    Username: "botfather",
})
// BotFather's UserID is typically 93372553
// Note the UserID and AccessHash from the response for subsequent calls
```

Or with mtgo-cli:
```bash
mtgo-cli invoke contacts.resolveUsername '{"Username":"botfather"}'
```

### Step 2: Send /newbot

```go
_, err = rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer:     &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Message:  "/newbot",
    RandomID: rand.Int63(),
})
```

### Step 3: Send bot display name and username

```go
// Display name
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer:     &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Message:  "My Test Bot",
    RandomID: rand.Int63(),
})
// Username (must end in "bot")
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer:     &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Message:  "my_test_bot",
    RandomID: rand.Int63(),
})
```

### Step 4: Extract the bot token

```go
// Read BotFather's response to get the token
history, err := rpc.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
    Peer:  &tg.InputPeerUser{UserID: 93372553, AccessHash: botfatherHash},
    Limit: 5,
})
// Parse history for the message containing "token:"
```

Or faster with mtgo-cli:
```bash
mtgo-cli invoke messages.getHistory '{"peer":{"_":"inputPeerUser","user_id":93372553,"access_hash":'$HASH'},"limit":5}' --format json | jq '.messages[].message' | grep token
```

## Group Management

Groups require a **userbot session** (not bot token) for creation, adding members, and admin promotion.

### Create a group

```go
// Create basic group
result, err := rpc.MessagesCreateChat(ctx, &tg.MessagesCreateChatRequest{
    Users: []tg.InputUserClass{&tg.InputUserEmpty{}},
    Title: "Bot Test Suite",
})
// result is *tg.Updates — extract chat_id from the update

// Upgrade to supergroup
for _, u := range result.Updates {
    if msg, ok := u.(*tg.UpdateNewChat); ok {
        migrateResult, err := rpc.MessagesMigrateChat(ctx, &tg.MessagesMigrateChatRequest{
            ChatID: msg.ChatID,
        })
        // migrateResult gives channel_id and access_hash
        _ = migrateResult
    }
}
```

> **Note:** `MessagesMigrateChat` takes a plain `int64` chat_id, NOT an `InputPeer`. Passing a wrapped peer causes `CHAT_ID_INVALID`.

### Add a bot to the group

**For supergroups/channels** (after `MigrateChat`):
```go
rpc.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
    Channel: &tg.InputChannel{ChannelID: channelID, AccessHash: channelHash},
    Users:   []tg.InputUserClass{
        &tg.InputUser{UserID: botUserID, AccessHash: botHash},
    },
})
```

**For basic groups** (not migrated):
```go
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
        ChangeInfo:     true,
        PostMessages:   true,
        EditMessages:   true,
        DeleteMessages: true,
        BanUsers:       true,
        InviteUsers:    true,
        PinMessages:    true,
        ManageTopics:   true,
    },
    Rank: "admin",
})
```

### Fast group setup with mtgo-cli

For quick one-off operations, mtgo-cli is faster than writing code:
```bash
# Create group
mtgo-cli invoke messages.createChat '{"Users":[{"_":"inputUserEmpty"}],"Title":"Test Group"}'
# Migrate (use chat_id from response)
mtgo-cli invoke messages.migrateChat '{"chat_id": CHAT_ID}'
# Add bot (use channel_id and access_hash from migrate response)
mtgo-cli invoke channels.inviteToChannel '{"channel":{"_":"inputChannel","channel_id":CID,"access_hash":HASH},"users":[{"_":"inputUser","user_id":BOT_ID,"access_hash":BOT_HASH}]}'
```

## Testing Bots with Userbots

When you need to test a bot end-to-end, use a userbot session to simulate real user interactions. The bot must already exist and have its `UserID` and `AccessHash` resolved.

### Resolve the bot

```go
result, err := rpc.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
    Username: "my_test_bot",
})
// Extract UserID and AccessHash for all subsequent calls
```

Or with mtgo-cli:
```bash
mtgo-cli get-user my_test_bot --format json
```

### Send /start in DM

```go
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer:     &tg.InputPeerUser{UserID: botUserID, AccessHash: botHash},
    Message:  "/start",
    RandomID: rand.Int63(),
})
```

### Send /start in group

```go
rpc.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
    Peer:     &tg.InputPeerChannel{ChannelID: channelID, AccessHash: channelHash},
    Message:  "/start@my_test_bot",
    RandomID: rand.Int63(),
})
```

### Click inline buttons

```go
rpc.MessagesGetBotCallbackAnswer(ctx, &tg.MessagesGetBotCallbackAnswerRequest{
    Peer:   &tg.InputPeerUser{UserID: botUserID, AccessHash: botHash},
    MsgID:  msgID,
    Data:   []byte(callbackData),
})
```

### Verify bot responses

Use `mtgo-cli trace` to watch all updates in real-time with correlation IDs:
```bash
mtgo-cli trace &
# In another terminal, send test messages — watch the trace output
mtgo-cli send-message my_test_bot "/start"
# Trace shows: [1] UPDATE updateNewMessage from bot
```

The trace output links each RPC call to its response with timing:
```
[1] >> messages.sendMessage
[1] << messages.sendMessage [45ms]
[2] UPDATE updateNewMessage
[2]    {Message:{ID:42, Text:"Welcome!"}}
```

### Testing checklist

After each interaction, verify:
- Bot responds in DM (check for `updateNewMessage` from bot ID)
- Bot responds in group context (same check, with `inputPeerChannel`)
- Inline buttons produce callback queries
- Error messages are appropriate for invalid input
- Deep links (e.g., `/start ref_code`) resolve correctly

For more CLI patterns (bulk operations, fast path, trace mode), see the `mtgo-cli` skill.
```
