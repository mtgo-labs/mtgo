<div align="center">

# mtgo

A fast, idiomatic Go client for the [Telegram MTProto API](https://core.telegram.org/mtproto).

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/mtgo-labs/mtgo.svg)](https://pkg.go.dev/github.com/mtgo-labs/mtgo)

</div>

## Features

- **Full MTProto 2.0** — encryption, key generation, CDNs, file transfers
- **High-level client API** — messages, chats, media, inline, stories, payments, business connections
- **Handler-based updates** — register handlers with filters, priorities, and handler groups
- **Middleware** — invoker-level (RPC calls) and handler-level (update dispatch) middleware chains
- **Plugin system** — lifecycle plugins with `Start`/`Stop` hooks
- **Multi-client** — run multiple bots/users in one process with `Compose` or `Idle`
- **Session import/export** — Telethon, Pyrogram, GramJS, mtcute, tdesktop session formats
- **Storage backends** — [SQLite](https://github.com/mtgo-labs/storage), [PostgreSQL](https://github.com/mtgo-labs/storage), [MongoDB](https://github.com/mtgo-labs/storage), or bring your own adapter
- **MTProxy support** — dd/ee/simple secrets with obfuscated2 and fake TLS
- **WebSocket transport** — MTProto over WebSocket for restrictive networks
- **Auto-reconnect** — exponential backoff with jitter and configurable max attempts
- **Health checks** — periodic ping/pong keepalive with configurable timeout
- **Generated TL layer** — auto-generated from Telegram schemas; update with one command
- **Pure Go** — no CGO required (SQLite via modernc.org/sqlite)

## Quick Start

```bash
go get github.com/mtgo-labs/mtgo
```

```go
package main

import (
    "fmt"
    "log"
    "os"

    tg "github.com/mtgo-labs/mtgo/telegram"
    "github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
    client, err := tg.NewClient(apiID, apiHash, &tg.Config{
        BotToken:    os.Getenv("BOT_TOKEN"),
        SessionName: "my_bot",
        SavePeers:   true,
    })
    if err != nil {
        log.Fatal(err)
    }

    client.OnMessage(func(client *tg.Client, msg *types.Message) {
        msg.Reply(msg.Text)
    }, tg.Private)

    if err := client.Connect(0); err != nil {
        log.Fatal(err)
    }
    defer client.Stop()

    fmt.Println("bot is running")
    client.Idle()
}
```

See [`examples/`](examples/) for more: middleware, conversations, SQLite, MongoDB, keyboards, media, MTProxy, webapp, and more.

## Authentication

### Bot

```go
client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
    BotToken: "123456:ABC-DEF",
})
```

### User (Phone Number)

```go
client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
    PhoneNumber: "+1234567890",
})
client.Connect(0)
// Terminal prompts for code/password automatically
```

### QR Login

```go
client, _ := tg.NewClient(apiID, apiHash, &tg.Config{})
client.QRLogin(context.Background())
// Displays QR code link for Telegram mobile scanning
```

### Session Strings

Import existing sessions from other frameworks:

```go
import "github.com/mtgo-labs/mtgo/session"

// From Telethon
client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
    SessionString: session.MustTelethon("1BVusO..."),
})

// From Pyrogram
client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
    SessionString: session.MustPyrogram("BAAJbwI..."),
})
```

## Handlers and Filters

```go
// Command handler
client.OnMessage(func(client *tg.Client, msg *types.Message) {
    msg.Reply("Welcome!")
}, tg.Command("start"))

// Regex filter
client.OnMessage(func(client *tg.Client, msg *types.Message) {
    msg.Reply("Got a number!")
}, telegram.Regex(`\d+`))

// Combined filters
client.OnMessage(handlePrivate, tg.Private.And(tg.HasText))

// Callback queries
client.OnCallbackQuery(func(client *tg.Client, cb *types.CallbackQuery) {
    cb.Answer("Pressed!", false)
})
```

Built-in filters: `Private`, `Group`, `HasText`, `Command`, `Regex`, `Media`, `Photo`, `Video`, `Document`, `Audio`, `Voice`, `VideoNote`, `Sticker`, `Animation`, `Contact`, `Location`, `Venue`, `Poll`, `Game`, `Forwarded`, `Reply`, `Outgoing`, and composable with `.And()`, `.Or()`, `.Not()`.

## Middleware

Two middleware levels for different concerns:

| Level | Method | Intercepts | Use case |
|-------|--------|------------|----------|
| **Invoker** | `UseInvokerMiddleware` | Outgoing RPC calls | Rate limiting, flood wait, logging, metrics |
| **Handler** | `UseMiddleware` | Incoming update dispatch | Auth, i18n, conversation state |

```go
// Invoker middleware (wraps RPC calls)
mw := floodwait.New()
client.UseInvokerMiddleware(mw.Middleware())

// Handler middleware (wraps update dispatch)
client.UseMiddleware(authMiddleware, -10) // lower priority = outermost
client.UseMiddleware(loggingMiddleware, 0)
```

Ready-made middlewares: [`floodwait`](https://github.com/mtgo-labs/middlewares/floodwait), [`ratelimit`](https://github.com/mtgo-labs/middlewares/ratelimit).

## Plugins

```go
// Register plugins before connecting
client.Use(i18nPlugin)
client.Use(conversationsPlugin)

// Plugins start/stop automatically with the client
client.Connect(0)
```

Available plugins: [`conversations`](https://github.com/mtgo-labs/plugins/conversations), [`i18n`](https://github.com/mtgo-labs/plugins/i18n).

## Storage

```go
import (
    "github.com/mtgo-labs/storage"
    "github.com/mtgo-labs/storage/sqlite"
)

ext, _ := sqlite.Open("session.db")
defer ext.Close()

client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
    BotToken:    botToken,
    SessionName: "my_bot",
    Storage:     storage.NewAdapter(ext),
})
```

Backends: [SQLite](https://github.com/mtgo-labs/storage/sqlite), [PostgreSQL](https://github.com/mtgo-labs/storage/postgres), [MongoDB](https://github.com/mtgo-labs/storage/mongodb). See the [storage repo](https://github.com/mtgo-labs/storage) for custom adapter docs.

## Multi-Client

```go
bot1, _ := tg.NewClient(apiID, apiHash, &tg.Config{BotToken: token1})
bot2, _ := tg.NewClient(apiID, apiHash, &tg.Config{BotToken: token2})

// Block until both stop
telegram.Compose(bot1, bot2)
```

## Project Structure

```
cmd/            Code generators (TL schema, error types)
compiler/       TL compiler and templates
docs/           API reference
examples/       Working examples
internal/       MTProto internals (crypto, session, transport)
session/        Session string import/export
tg/             Generated TL types (do not edit directly)
tgerr/          Generated error types
telegram/       High-level client API, handlers, filters, middleware
mtproxy/        MTProxy obfuscated2/fake-TLS transport
```

## Code Generation

```bash
# Regenerate TL types from schema
go run cmd/tlgen/main.go

# Regenerate error types
go run cmd/errgen/main.go
```

Do not edit generated files directly. Modify the schema in `compiler/` or `cmd/` instead.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, and PR process.

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## License

Licensed under the [Apache License 2.0](LICENSE).

## Ecosystem

| Repository | Description |
|------------|-------------|
| [storage](https://github.com/mtgo-labs/storage) | Persistent storage adapters (SQLite, PostgreSQL, MongoDB) |
| [plugins](https://github.com/mtgo-labs/plugins) | Official plugins (conversations, i18n) |
| [middlewares](https://github.com/mtgo-labs/middlewares) | Invoker middleware (flood wait, rate limiting) |
| [plugins-template](https://github.com/mtgo-labs/plugins-template) | Template for new plugins |
| [middlewares-template](https://github.com/mtgo-labs/middlewares-template) | Template for new middlewares |
