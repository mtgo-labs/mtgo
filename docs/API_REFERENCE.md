# mtgo *(MTProto Go)* API Reference

> Complete API reference for mtgo — a Go Telegram MTProto client library.
>
> mtgo stands for **MTProto Go**. It is a Telegram client library and has no relation to Magic: The Gathering Online.

---

## Table of Contents

- [Packages](#packages)
- [Plugins](#plugins)
- [Middlewares](#middlewares)
- [telegram — High-Level Client](#telegram--high-level-client)
  - [Client](#client)
  - [Configuration](#configuration)
  - [Authentication](#authentication)
  - [Password Management](#password-management)
  - [QR Login](#qr-login)
  - [Messages](#messages)
    - [Send](#send-messages)
    - [Edit](#edit-messages)
    - [Copy](#copy-messages)
    - [Forward](#forward-messages)
    - [Delete](#delete-messages)
    - [Pin](#pin-messages)
    - [Read](#read-messages)
    - [Search](#search-messages)
    - [Reactions & Polls](#reactions--polls)
  - [Chats](#chats)
  - [Chat Actions](#chat-actions)
  - [Users](#users)
  - [Media & File Transfer](#media--file-transfer)
    - [Upload](#upload)
    - [Download](#download)
    - [Progress](#progress-tracking)
  - [Callback & Inline](#callback--inline)
  - [Stories](#stories)
  - [Payments](#payments)
  - [Business](#business)
  - [Premium & Boosts](#premium--boosts)
  - [Profile](#profile)
  - [Contacts](#contacts)
  - [Invite Links](#invite-links)
  - [Folders](#folders)
  - [Bot Commands](#bot-commands)
  - [Bot Info](#bot-info)
  - [Menu Button](#menu-button)
  - [Games](#games)
  - [Account & Privacy](#account--privacy)
  - [RPC Layer](#rpc-layer)
- [Update Handling](#update-handling)
  - [Update Struct](#update-struct)
  - [Dispatcher](#dispatcher)
  - [Handlers](#handlers)
  - [Filters](#filters)
- [Context — Handler Context](#context--handler-context)
  - [Context Struct](#context-struct)
  - [Context Methods](#context-methods)
    - [Message Methods](#message-methods)
    - [Chat Methods](#chat-methods)
    - [Callback Methods](#callback-methods)
    - [Inline Methods](#inline-methods)
    - [Stories Methods](#stories-methods)
    - [Payments Methods](#payments-methods)
    - [Account Methods](#account-methods)
    - [Premium Methods](#premium-methods)
- [Peer Resolution](#peer-resolution)
- [Utilities](#utilities)
- [tg — Generated TL Types](#tg--generated-tl-types)
  - [Core Interfaces](#core-interfaces)
  - [TL Primitives](#tl-primitives)
  - [Message & Container](#message--container)
  - [Gzip](#gzip)
  - [Generated Maps](#generated-maps)
- [tgerr — Error Handling](#tgerr--error-handling)
- [telegram/types — Domain Types](#telegrams--domain-types)
- [telegram/params — API Parameters](#telegramparams--api-parameters)
- [telegram/parser — Text Parsing](#telegramparser--text-parsing)
- [telegram/fileid — File ID](#telegramfileid--file-id)
- [compiler/tlgen — TL Code Generation](#compilertlgen--tl-code-generation)
- [internal — Internal Packages](#internal--internal-packages)
  - [crypto](#internalcrypto)
  - [session](#internalsession)
  - [storage](#internalstorage)
  - [transport](#internaltransport)

---

## Packages

| Package | Import Path | Description |
|---------|-------------|-------------|
| `telegram` | `github.com/mtgo-labs/mtgo/telegram` | High-level Telegram client |
| `tg` | `github.com/mtgo-labs/mtgo/tg` | Generated TL types and MTProto primitives |
| `tgerr` | `github.com/mtgo-labs/mtgo/tgerr` | RPC error types and handling |
| `telegram/types` | `github.com/mtgo-labs/mtgo/telegram/types` | Parsed domain types |
| `telegram/params` | `github.com/mtgo-labs/mtgo/telegram/params` | Option structs for API calls |
| `telegram/parser` | `github.com/mtgo-labs/mtgo/telegram/parser` | HTML/Markdown text parsing |
| `telegram/fileid` | `github.com/mtgo-labs/mtgo/telegram/fileid` | File ID encode/decode |
| `compiler/tlgen` | `github.com/mtgo-labs/mtgo/compiler/tlgen` | TL schema code generator |
| `internal/crypto` | `github.com/mtgo-labs/mtgo/internal/crypto` | Cryptographic primitives |
| `internal/session` | `github.com/mtgo-labs/mtgo/internal/session` | MTProto session management |
| `internal/storage` | `github.com/mtgo-labs/mtgo/internal/storage` | Session storage backends |
| `internal/transport` | `github.com/mtgo-labs/mtgo/internal/transport` | TCP transport implementations |

---

## telegram — High-Level Client

### Client

The `Client` struct is the main entry point. It wraps the MTProto session and provides high-level methods for all Telegram API operations.

#### Errors

```go
var ErrAlreadyConnected error
var ErrNotConnected      error
var ErrStillInitialized  error
var ErrPeerNotFound      error
```

#### Constructor

```go
func NewClient(apiID int, apiHash string, cfg *Config) (*Client, error)
```

Creates a new Client with a `*Config`. If `cfg` is `nil`, the default configuration is used. Only non-zero fields in `cfg` override the defaults.

**Parameters:**
- `apiID` — Telegram API application ID (from my.telegram.org)
- `apiHash` — Telegram API application hash
- `cfg` — Optional `*Config` (pass `nil` for defaults)

**Example:**
```go
client, err := telegram.NewClient(12345, "your_api_hash", &telegram.Config{
    SessionName: "my_session",
    BotToken:    "123456:ABC-DEF...",
})
if err != nil {
    log.Fatal(err)
}
```

#### Connection Lifecycle

```go
func (c *Client) Connect(timeout time.Duration) error
func (c *Client) Start() error
func (c *Client) Stop()
func (c *Client) Idle()
func (c *Client) Disconnect() error
func (c *Client) IsConnected() bool
func (c *Client) LogOut() error
func (c *Client) HandleUpdates(updates tg.UpdatesClass)
```

| Method | Description |
|--------|-------------|
| `Connect` | Creates storage, loads or generates auth key, starts encrypted session. If `timeout <= 0`, defaults to 30s |
| `Start` | Connects then blocks until `Stop()` is called. For bots: connect + idle in one call |
| `Stop` | Closes the stop channel and disconnects |
| `Idle` | Blocks until `Stop()` is called. Call after `Connect()` for long-running bots |
| `Disconnect` | Stops all sessions, closes storage, marks disconnected |
| `IsConnected` | Returns current connection state |
| `LogOut` | Disconnects without calling auth.logOut RPC |
| `HandleUpdates` | Processes raw MTProto updates, flattens and dispatches to registered handlers |

#### Accessors

```go
func (c *Client) Me() *types.User
func (c *Client) Session() *session.Session
func (c *Client) Storage() storage.Storage
func (c *Client) Config() Config
func (c *Client) SetMe(user *types.User)
func (c *Client) SetDispatcher(d Dispatcher)
func (c *Client) SetBotToken(token string)
func (c *Client) IsBot() bool
func (c *Client) ServerTime() int32
func (c *Client) APIID() int
func (c *Client) APIHash() string
func (c *Client) DC() int
func (c *Client) SessionName() string
func (c *Client) BotToken() string
func (c *Client) TestMode() bool
func (c *Client) IPv6() bool
func (c *Client) NoUpdates() bool
func (c *Client) Workers() int
func (c *Client) ParseMode() int
func (c *Client) SleepThreshold() time.Duration
func (c *Client) MaxConcurrentTransmissions() int
func (c *Client) MaxMessageCacheSize() int
```

#### Session Management

```go
func (c *Client) GetSession(ctx context.Context, dcID int, isMedia bool, isCDN bool) (*session.Session, error)
func (c *Client) ExportSessionString() (string, error)
```

| Method | Description |
|--------|-------------|
| `GetSession` | Get or create a session for a specific data center |
| `ExportSessionString` | Export the current session as a portable string for reuse |

#### Peer Cache

```go
func (c *Client) ResolvePeer(ctx context.Context, peerID interface{}) (tg.InputPeerClass, error)
func (c *Client) ResolvePeerCache(id int64) (tg.InputPeerClass, error)
func (c *Client) CachePeer(id int64, peer tg.InputPeerClass)
```

#### Handler Management

```go
func (c *Client) AddHandler(handler Handler, group ...int)
func (c *Client) RemoveHandler(handler Handler)
```

---

### Configuration

```go
type Proxy struct {
    Addr     string
    Username string
    Password string
}

type Config struct {
    APIID               int
    APIHash             string
    DC                  int
    SessionName         string
    BotToken            string
    SessionString       string
    PhoneNumber         string
    PhoneCode           string
    Password            string
    WorkDir             string
    InMemory            bool
    Proxy               *Proxy
    TestMode            bool
    IPv6                bool
    NoUpdates           bool
    SkipUpdates         bool
    Workers             int
    SleepThreshold      time.Duration
    HandlerTimeout      time.Duration
    MaxConcurrentTrans  int
    MaxMessageCacheSize int
    ParseMode           int
    HidePassword        bool
    FetchReplies        bool
    FetchTopics         bool
    FetchStories        bool
    FetchStickers       bool
    ClientPlatform      types.ClientPlatform
    AppVersion          string
    DeviceModel         string
    SystemVersion       string
    LangCode            string
    LangPack            string
    SystemLangCode      string
    TZOffset            int
    NetPoll             bool
    SavePeers           bool
}

var DefaultConfig = Config{
    Workers:             0,
    SleepThreshold:      10 * time.Second,
    MaxConcurrentTrans:  1,
    MaxMessageCacheSize: 1000,
    DeviceModel:         "MTGo",
    SystemVersion:       "1.0.0",
    AppVersion:          "1.0.0",
    LangPack:            "tdesktop",
    LangCode:            "en",
    SystemLangCode:      "en",
    SkipUpdates:         true,
    FetchReplies:        true,
    FetchTopics:         true,
    FetchStories:        true,
    FetchStickers:       true,
    ClientPlatform:      types.ClientPlatformAndroid,
}
```

When `Workers` is 0 (default), it is automatically set to `min(runtime.NumCPU()+4, 32)`.

---

### Authentication

```go
type SendCodeResult struct {
    PhoneCodeHash string
    Type           tg.SentCodeTypeClass
    NextType       tg.CodeTypeClass
    Timeout        int
}

func (c *Client) SendCode(ctx context.Context, phoneNumber string) (*SendCodeResult, error)
func (c *Client) SignIn(ctx context.Context, phoneNumber, phoneCodeHash, phoneCode string) (*types.User, error)
func (c *Client) SignUp(ctx context.Context, phoneNumber, phoneCodeHash, firstName string, lastName ...string) (*types.User, error)
func (c *Client) SignOut(ctx context.Context) (bool, error)
func (c *Client) GetPasswordHint(ctx context.Context) (string, error)
func (c *Client) CheckPassword(ctx context.Context, password string) (*types.User, error)
func (c *Client) RecoverPassword(ctx context.Context, code string) (*types.User, error)
```

| Method | Description |
|--------|-------------|
| `SendCode` | Send verification code to phone number |
| `SignIn` | Sign in with phone number, code hash, and verification code |
| `SignUp` | Register a new account (if sign-in requires registration) |
| `SignOut` | Sign out and disconnect |
| `GetPasswordHint` | Get 2FA password hint |
| `CheckPassword` | Verify 2FA cloud password |
| `RecoverPassword` | Recover account via email verification code |

**Example — User auth flow:**
```go
result, _ := client.SendCode(ctx, "+1234567890")
user, _ := client.SignIn(ctx, "+1234567890", result.PhoneCodeHash, "12345")
```

---

### Password Management

```go
var ErrPasswordAlreadyEnabled error
var ErrPasswordNotEnabled     error

func (c *Client) EnableCloudPassword(ctx context.Context, password, hint string) error
func (c *Client) ChangeCloudPassword(ctx context.Context, currentPassword, newPassword, newHint string) error
func (c *Client) RemoveCloudPassword(ctx context.Context, password string) error
```

---

### QR Login

```go
type QRLoginToken struct {
    Token   []byte
    Expires int32
}

func (c *Client) GetQRCodeLoginToken(ctx context.Context) (*QRLoginToken, error)
func (c *Client) CheckQRCodeLoginToken(ctx context.Context, token []byte) (*types.User, error)
```

| Method | Description |
|--------|-------------|
| `GetQRCodeLoginToken` | Generate a QR code login token for scanning |
| `CheckQRCodeLoginToken` | Poll to check if the QR code was scanned and accepted |

---

### Messages

#### Send Messages

Options are passed as variadic pointers to structs from the `params` package:

```go
type SendMessage struct {
    DisableWebPagePreview bool
    DisableNotification   bool
    Silent                bool
    Background            bool
    ClearDraft            bool
    NoForwards            bool
    InvertMedia           bool
    ReplyToMessageID      int32
    ReplyTo               tg.InputReplyToClass
    ReplyMarkup           tg.ReplyMarkupClass
    Entities              []tg.MessageEntityClass
    ParseMode             params.ParseMode
    ScheduleDate          *int32
    EffectID              *int64
    SendAs                tg.InputPeerClass
}

func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Client) SendMedia(ctx context.Context, chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Client) SendContact(ctx context.Context, chatID int64, phoneNumber, firstName, lastName string, opts ...*params.SendContact) (*types.Message, error)
func (c *Client) SendLocation(ctx context.Context, chatID int64, lat, lng float64, opts ...*params.SendLocation) (*types.Message, error)
func (c *Client) SendVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, opts ...*params.SendVenue) (*types.Message, error)
func (c *Client) SendPoll(ctx context.Context, chatID int64, question string, options []string, opts ...*params.SendPoll) (*types.Message, error)
```

```go
type SendDice struct {
    Emoticon string
}

func (c *Client) SendDice(ctx context.Context, chatID int64, opts ...*params.SendDice) (*types.Message, error)
```

**Example:**
```go
msg, err := client.SendMessage(ctx, chatID, "Hello, world!", &params.SendMessage{
    ReplyToMessageID:    42,
    DisableNotification: true,
})
```

#### Edit Messages

```go
type EditMessage struct {
    DisableWebPagePreview bool
    InvertMedia           bool
    ReplyMarkup           tg.ReplyMarkupClass
    ParseMode             params.ParseMode
    Entities              []tg.MessageEntityClass
    ScheduleDate          *int32
}

func (c *Client) EditMessageText(ctx context.Context, chatID int64, messageID int32, text string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Client) EditMessageCaption(ctx context.Context, chatID int64, messageID int32, caption string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Client) EditMessageMedia(ctx context.Context, chatID int64, messageID int32, media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error)
func (c *Client) EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int32, replyMarkup tg.ReplyMarkupClass) (*types.Message, error)
```

#### Copy Messages

```go
type CopyMessage struct {
    Caption             string
    DisableNotification bool
    ReplyToMessageID    int32
    ReplyMarkup         tg.ReplyMarkupClass
    ScheduleDate        *int32
    DropAuthor          bool
}

func (c *Client) CopyMessage(ctx context.Context, chatID, fromChatID int64, messageID int32, opts ...*params.CopyMessage) (int64, error)
func (c *Client) CopyMediaGroup(ctx context.Context, chatID, fromChatID int64, groupedID int64) ([]*types.Message, error)
```

#### Forward Messages

```go
type ForwardMessages struct {
    DisableNotification bool
    NoForwards          bool
    DropAuthor          bool
    DropMediaCaptions   bool
    ScheduleDate        *int32
}

func (c *Client) ForwardMessages(ctx context.Context, chatID, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error)
func (c *Client) ForwardMediaGroup(ctx context.Context, chatID, fromChatID int64, messageIDs []int32, opts ...*params.ForwardMessages) ([]*types.Message, error)
```

#### Delete Messages

```go
type DeleteMessages struct {
    Revoke bool
}

func (c *Client) DeleteMessages(ctx context.Context, chatID int64, messageIDs []int32, opts ...*params.DeleteMessages) (int, error)
func (c *Client) DeleteChatHistory(ctx context.Context, chatID int64, maxID int32, revoke bool) (int, error)
```

Both return the pts count on success.

#### Pin Messages

```go
type PinMessage struct {
    Silent bool
    Unpin  bool
}

func (c *Client) PinMessage(ctx context.Context, chatID int64, messageID int32, opts ...*params.PinMessage) (*types.Message, error)
func (c *Client) UnpinMessage(ctx context.Context, chatID int64, messageID int32) (*types.Message, error)
func (c *Client) UnpinAllMessages(ctx context.Context, chatID int64) (int, error)
```

#### Read Messages

```go
func (c *Client) ReadHistory(ctx context.Context, chatID int64, maxID int32) error
func (c *Client) ReadMentions(ctx context.Context, chatID int64) error
func (c *Client) ReadReactions(ctx context.Context, chatID int64) error
```

#### Search Messages

```go
type SearchMessages struct {
    Limit    int
    OffsetID int32
    MinDate  int32
    MaxDate  int32
    FromID   tg.InputPeerClass
    Filter   tg.MessagesFilterClass
    TopMsgID *int32
}

type SearchGlobal struct {
    Limit          int
    OffsetRate     int32
    OffsetID       int32
    OffsetPeer     tg.InputPeerClass
    MinDate        int32
    MaxDate        int32
    BroadcastsOnly bool
    GroupsOnly     bool
    FolderID       *int32
    Filter         tg.MessagesFilterClass
}

func (c *Client) SearchMessages(ctx context.Context, chatID int64, query string, opts ...*params.SearchMessages) ([]*types.Message, error)
func (c *Client) SearchGlobal(ctx context.Context, query string, opts ...*params.SearchGlobal) ([]*types.Message, error)
func (c *Client) SearchMessagesCount(ctx context.Context, chatID int64, query string) (int32, error)
func (c *Client) SearchGlobalCount(ctx context.Context, query string) (int32, error)
```

#### Reactions & Polls

```go
func (c *Client) SendReaction(ctx context.Context, chatID int64, messageID int32, reaction ...tg.ReactionClass) error
func (c *Client) SendPaidReaction(ctx context.Context, chatID int64, messageID int32, amount int64) error
func (c *Client) VotePoll(ctx context.Context, chatID int64, messageID int32, options [][]byte) error
func (c *Client) StopPoll(ctx context.Context, chatID int64, messageID int32) error
func (c *Client) RetractVote(ctx context.Context, chatID int64, messageID int32) error
```

---

### Chats

```go
func (c *Client) GetChat(ctx context.Context, chatID int64) (*types.Chat, error)
func (c *Client) JoinChat(ctx context.Context, inviteHash string) (*types.Chat, error)
func (c *Client) LeaveChat(ctx context.Context, chatID int64) error
func (c *Client) CreateChannel(ctx context.Context, title, about string, megagroup bool) (*types.Chat, error)
func (c *Client) CreateGroup(ctx context.Context, title string, userIDs []int64) (*types.Chat, error)
func (c *Client) CreateSupergroup(ctx context.Context, title, about string) (*types.Chat, error)
func (c *Client) DeleteChat(ctx context.Context, chatID int64) error
func (c *Client) SetChatTitle(ctx context.Context, chatID int64, title string) error
func (c *Client) SetChatDescription(ctx context.Context, chatID int64, about string) error
func (c *Client) SetChatUsername(ctx context.Context, chatID int64, username string) error
func (c *Client) BanChatMember(ctx context.Context, chatID int64, userID int64) error
func (c *Client) UnbanChatMember(ctx context.Context, chatID int64, userID int64) error
func (c *Client) PromoteChatMember(ctx context.Context, chatID int64, userID int64, adminRights *tg.ChatAdminRights) error
func (c *Client) RestrictChatMember(ctx context.Context, chatID int64, userID int64, rights *tg.ChatBannedRights) error
func (c *Client) SetAdministratorTitle(ctx context.Context, chatID int64, userID int64, title string) error
func (c *Client) GetChatMember(ctx context.Context, chatID int64, userID int64) (*types.ChatMember, error)
func (c *Client) GetChatMembers(ctx context.Context, chatID int64, limit, offset int) ([]*types.ChatMember, error)
func (c *Client) GetChatMembersCount(ctx context.Context, chatID int64) (int, error)
func (c *Client) AddChatMember(ctx context.Context, chatID int64, userID int64) error
func (c *Client) SetChatPhoto(ctx context.Context, chatID int64, photo tg.InputChatPhotoClass) error
func (c *Client) DeleteChatPhoto(ctx context.Context, chatID int64) error
func (c *Client) SetChatTTL(ctx context.Context, chatID int64, ttl int) error
func (c *Client) SetChatPermissions(ctx context.Context, chatID int64, permissions *tg.ChatBannedRights) error
func (c *Client) MarkChatUnread(ctx context.Context, chatID int64, unread bool) error
func (c *Client) SetProtectedContent(ctx context.Context, chatID int64, enabled bool) error
func (c *Client) SetSlowMode(ctx context.Context, chatID int64, seconds int) error
func (c *Client) GetChatEventLog(ctx context.Context, chatID int64, query string, limit int) ([]*types.ChatEvent, error)
func (c *Client) MuteChat(ctx context.Context, chatID int64) error
func (c *Client) UnmuteChat(ctx context.Context, chatID int64) error
```

---

### Chat Actions

```go
func (c *Client) SendChatAction(ctx context.Context, chatID int64, action tg.SendMessageActionClass) error
```

---

### Users

```go
func (c *Client) GetUsers(ctx context.Context, userIDs []int64) ([]*types.User, error)
func (c *Client) GetMe(ctx context.Context) (*types.User, error)
func (c *Client) GetCommonChats(ctx context.Context, userID int64) ([]*types.Chat, error)
func (c *Client) UpdateProfile(ctx context.Context, firstName, lastName, about string) error
```

---

### Media & File Transfer

#### InputFile

`InputFile` is the strongly-typed file parameter used by all media-sending methods. It lives in the `telegram/types` package and is re-exported from `telegram` via type alias.

```go
type InputFile struct { /* unexported fields */ }

// Constructors — use one based on the source of the file:
func FileID(s string) *InputFile                          // Telegram file_id string
func FromIDs(ID, accessHash int64, fileRef []byte) *InputFile // Raw Telegram file IDs
func URL(u string) *InputFile                              // Download from URL
func Path(p string) *InputFile                             // Local filesystem path
func Reader(r io.ReadSeeker, fileName string, size int64) *InputFile  // In-memory reader
func FromBytes(data []byte, fileName string) *InputFile    // Raw byte slice
```

Each `InputFile` is resolved automatically by the client: file IDs and raw IDs are sent as `InputMediaPhoto`/`InputMediaDocument`; URLs are passed as `InputMediaDocument` with `url`; paths and readers are uploaded first, then sent.

#### Upload

```go
type UploadResult struct {
    File   tg.InputFileClass
    Size   int64
    Name   string
    IsBig  bool
}

type UploadOptions struct {
    Workers  int
    Progress ProgressFunc
    FileName string
}

func (c *Client) UploadFile(ctx context.Context, reader io.Reader, fileName string, fileSize int64, opts *UploadOptions) (*UploadResult, error)
```

#### Send Media

Each media type has its own `params.SendXxx` struct with common fields (notification, reply, schedule, etc.) plus media-specific fields (duration, dimensions, thumbnail, etc.).

```go
func (c *Client) SendPhoto(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendPhoto) (*types.Message, error)
func (c *Client) SendDocument(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendDocument) (*types.Message, error)
func (c *Client) SendVideo(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVideo) (*types.Message, error)
func (c *Client) SendAudio(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAudio) (*types.Message, error)
func (c *Client) SendAnimation(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendAnimation) (*types.Message, error)
func (c *Client) SendVoice(ctx context.Context, chatID int64, file *InputFile, caption string, opts ...*params.SendVoice) (*types.Message, error)
func (c *Client) SendVideoNote(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendVideoNote) (*types.Message, error)
func (c *Client) SendSticker(ctx context.Context, chatID int64, file *InputFile, opts ...*params.SendSticker) (*types.Message, error)
```

All `params.SendXxx` structs share a common base of fields and add media-specific options:

```go
// Common fields present in every SendXxx struct:
DisableNotification bool
Silent              bool
Background          bool
ClearDraft          bool
NoForwards          bool
ReplyToMessageID    int32
ReplyTo             tg.InputReplyToClass
ReplyMarkup         tg.ReplyMarkupClass
ScheduleDate        *int32
EffectID            *int64
SendAs              tg.InputPeerClass

// Media-specific fields (varies by type):
// SendPhoto:      FileName
// SendDocument:   FileName, Thumb, MimeType
// SendVideo:      Duration float64, Width, Height, SupportsStreaming, FileName, Thumb
// SendAudio:      Duration int32, Performer, Title, FileName, Thumb
// SendAnimation:  FileName, Thumb
// SendVoice:      Duration int32, FileName
// SendVideoNote:  Duration float64, FileName, Thumb
// SendSticker:    FileName
```

**Example:**
```go
msg, err := client.SendPhoto(ctx, chatID, telegram.FileID("AABBCC..."), "Caption text", &params.SendPhoto{
    DisableNotification: true,
    ReplyToMessageID:    42,
})

msg, err := client.SendVideo(ctx, chatID, telegram.Path("/tmp/video.mp4"), "Video caption", &params.SendVideo{
    Duration:          30.5,
    Width:             1920,
    Height:            1080,
    SupportsStreaming: true,
})
```

#### Download

```go
type FileChunk struct {
    Data  []byte
    Err   error
    Bytes int64
    Total int64
}

type DownloadOptions struct {
    ChunkSize int32
    Progress  ProgressFunc
    DCID      int32
}

func (c *Client) DownloadFile(ctx context.Context, location tg.InputFileLocationClass, fileSize int64, opts *DownloadOptions) ([]byte, error)
func (c *Client) DownloadToFile(ctx context.Context, location tg.InputFileLocationClass, filePath string, fileSize int64, opts *DownloadOptions) error
func (c *Client) DownloadMedia(ctx context.Context, media types.Media, thumbSize string, opts *DownloadOptions) ([]byte, error)
func (c *Client) DownloadMediaToFile(ctx context.Context, media types.Media, thumbSize string, filePath string, fileSize int64, opts *DownloadOptions) error
func (c *Client) StreamFile(ctx context.Context, location tg.InputFileLocationClass, fileSize int64, opts *DownloadOptions) (<-chan FileChunk, error)
```

`GetFileLocation` resolves the download location from a parsed media object:
```go
func GetFileLocation(media types.Media, thumbSize string) (tg.InputFileLocationClass, int32, error)
```

#### Progress Tracking

```go
type ProgressInfo struct {
    FileName        string
    TotalBytes      int64
    UploadedBytes   int64
    DownloadedBytes int64
    IsUpload        bool
}

type ProgressFunc func(info ProgressInfo)

func (p ProgressInfo) Progress() float64 // Returns 0.0–100.0
```

---

### Callback & Inline

#### Callback Queries

```go
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID int64, text string, showAlert bool, url string, cacheTime int) error
func (c *Client) AnswerWebAppQuery(ctx context.Context, queryID string, result tg.InputBotInlineResultClass) (*tg.WebViewMessageSent, error)
func (c *Client) RequestCallbackAnswer(ctx context.Context, chatID int64, messageID int64, data []byte) (*tg.MessagesBotCallbackAnswer, error)
```

#### Inline Mode

```go
type AnswerInlineQuery struct { /* fields */ }
type SendInlineBotResult struct { /* fields */ }

func (c *Client) AnswerInlineQuery(ctx context.Context, queryID int64, results []tg.InputBotInlineResultClass, opts ...*params.AnswerInlineQuery) error
func (c *Client) GetInlineBotResults(ctx context.Context, bot int64, chatID int64, query, offset string) (*tg.MessagesBotResults, error)
func (c *Client) SendInlineBotResult(ctx context.Context, chatID int64, queryID int64, resultID string, opts ...*params.SendInlineBotResult) (*types.Message, error)
```

---

### Stories

```go
type SendStoryOption struct {
    Pinned       bool
    NoForwards   bool
    Period       *int32
    PrivacyRules []tg.InputPrivacyRuleClass
}

func (c *Client) SendStory(ctx context.Context, chatID int64, media tg.InputMediaClass, opts ...*SendStoryOption) (*types.Story, error)
func (c *Client) EditStoryCaption(ctx context.Context, chatID int64, storyID int32, caption string) (*types.Story, error)
func (c *Client) EditStoryMedia(ctx context.Context, chatID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error)
func (c *Client) DeleteStories(ctx context.Context, chatID int64, storyIDs []int32) error
func (c *Client) GetStories(ctx context.Context, userID int64, storyIDs []int32) ([]*types.Story, error)
func (c *Client) GetChatStories(ctx context.Context, chatID int64) ([]*types.Story, error)
func (c *Client) GetStoryViews(ctx context.Context, chatID int64, storyIDs []int32) ([]*tg.StoryViews, error)
func (c *Client) ForwardStory(ctx context.Context, targetChatID int64, sourceChatID int64, storyID int32) (*types.Message, error)
func (c *Client) PinChatStories(ctx context.Context, chatID int64, storyIDs []int32) error
func (c *Client) ReadChatStories(ctx context.Context, chatID int64, storyIDs []int32) error
```

---

### Payments

```go
type GetPaymentFormOption struct {
    ThemeParams *string
}

type SendPaymentFormOption struct {
    RequestedInfoID  *string
    ShippingOptionID *string
    TipAmount        *int64
}

func (c *Client) GetPaymentForm(ctx context.Context, chatID int64, messageID int32, opts ...*GetPaymentFormOption) (tg.PaymentFormClass, error)
func (c *Client) SendPaymentForm(ctx context.Context, formID int64, chatID int64, messageID int32, credentials tg.InputPaymentCredentialsClass, opts ...*SendPaymentFormOption) (tg.PaymentResultClass, error)
func (c *Client) GetStarsBalance(ctx context.Context, chatID int64) (int64, error)
func (c *Client) SendGift(ctx context.Context, userID int64, giftID int64, message string) error
func (c *Client) AnswerPreCheckoutQuery(ctx context.Context, queryID int64, ok bool, errorMessage string) error
func (c *Client) AnswerShippingQuery(ctx context.Context, queryID int64, ok bool, shippingOptions []*tg.ShippingOption) error
```

---

### Business

```go
func (c *Client) GetBusinessConnection(ctx context.Context, connectionID string) (*tg.BotBusinessConnection, error)
```

---

### Premium & Boosts

```go
type ApplyBoostOption struct {
    Slots []int32
}

type GetBoostsOption struct {
    Offset string
    Limit  int32
}

func (c *Client) ApplyBoost(ctx context.Context, chatID int64, opts ...*ApplyBoostOption) ([]*tg.MyBoost, error)
func (c *Client) GetBoostsStatus(ctx context.Context, chatID int64) (*tg.PremiumBoostsStatus, error)
func (c *Client) GetBoosts(ctx context.Context, opts ...*GetBoostsOption) ([]*tg.MyBoost, error)
```

---

### Profile

```go
type GetProfilePhotosOption struct {
    Offset int32
    Limit  int32
    MaxID  int64
}

func (c *Client) SetProfilePhoto(ctx context.Context, photo tg.InputFileClass) error
func (c *Client) SetUsername(ctx context.Context, username string) error
func (c *Client) SetBio(ctx context.Context, bio string) error
func (c *Client) DeleteProfilePhoto(ctx context.Context, photoID int64) error
func (c *Client) GetProfilePhotos(ctx context.Context, userID int64, opts ...*GetProfilePhotosOption) ([]*types.ChatPhoto, error)
```

---

### Contacts

```go
func (c *Client) AddContact(ctx context.Context, userID int64, firstName, lastName, phone string, share bool) error
func (c *Client) DeleteContacts(ctx context.Context, userIDs []int64) error
func (c *Client) GetContacts(ctx context.Context, hash int64) (tg.ContactsClass, error)
func (c *Client) BlockUser(ctx context.Context, userID int64) error
func (c *Client) UnblockUser(ctx context.Context, userID int64) error
func (c *Client) GetBlocked(ctx context.Context, limit, offset int) (tg.BlockedClass, error)
```

---

### Invite Links

```go
type InviteLinkOption struct {
    ExpireDate *int32
    UsageLimit *int32
    Title      *string
}

func (c *Client) GetChatInviteLink(ctx context.Context, chatID int64, link string) (*types.ChatInviteLink, error)
func (c *Client) CreateChatInviteLink(ctx context.Context, chatID int64, opts ...*InviteLinkOption) (*types.ChatInviteLink, error)
func (c *Client) EditChatInviteLink(ctx context.Context, chatID int64, link string, opts ...*InviteLinkOption) (*types.ChatInviteLink, error)
func (c *Client) RevokeChatInviteLink(ctx context.Context, chatID int64, link string) (*types.ChatInviteLink, error)
func (c *Client) ExportChatInviteLink(ctx context.Context, chatID int64) (string, error)
func (c *Client) GetChatInviteLinkJoiners(ctx context.Context, chatID int64, link string, limit int) ([]*types.ChatInviteLinkJoiner, error)
func (c *Client) GetChatAdminInviteLinks(ctx context.Context, chatID int64, adminID int64, limit int) ([]*types.ChatInviteLink, error)
func (c *Client) DeleteChatInviteLink(ctx context.Context, chatID int64, link string) error
func (c *Client) ApproveChatJoinRequest(ctx context.Context, chatID int64, userID int64) error
func (c *Client) DeclineChatJoinRequest(ctx context.Context, chatID int64, userID int64) error
```

---

### Folders

```go
func (c *Client) ArchiveChat(ctx context.Context, chatID int64) error
func (c *Client) UnarchiveChat(ctx context.Context, chatID int64) error
```

---

### Bot Commands

```go
func (c *Client) SetBotCommands(ctx context.Context, scope tg.BotCommandScopeClass, langCode string, commands []*tg.BotCommand) error
func (c *Client) GetBotCommands(ctx context.Context, langCode string) ([]*tg.BotCommand, error)
func (c *Client) DeleteBotCommands(ctx context.Context, scope tg.BotCommandScopeClass, langCode string) error
```

---

### Bot Info

```go
func (c *Client) SetBotInfoDescription(ctx context.Context, langCode, description string) error
func (c *Client) GetBotInfoDescription(ctx context.Context, langCode string) (string, error)
func (c *Client) SetBotInfoShortDescription(ctx context.Context, langCode, description string) error
func (c *Client) GetBotInfoShortDescription(ctx context.Context, langCode string) (string, error)
func (c *Client) SetBotName(ctx context.Context, langCode, name string) error
func (c *Client) GetBotName(ctx context.Context, langCode string) (string, error)
```

---

### Menu Button

```go
func (c *Client) SetChatMenuButton(ctx context.Context, userID int64, button tg.BotMenuButtonClass) error
func (c *Client) GetChatMenuButton(ctx context.Context, userID int64) (tg.BotMenuButtonClass, error)
```

---

### Games

```go
func (c *Client) SendGame(ctx context.Context, chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Client) SetGameScore(ctx context.Context, chatID int64, messageID int64, userID int64, score int, force, noForward bool) (*types.Message, error)
func (c *Client) GetGameHighScores(ctx context.Context, chatID int64, messageID int64, userID int64) ([]*tg.HighScore, error)
```

---

### Account & Privacy

```go
func (c *Client) SetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass, rules []tg.InputPrivacyRuleClass) error
func (c *Client) GetPrivacy(ctx context.Context, key tg.InputPrivacyKeyClass) ([]tg.PrivacyRuleClass, error)
func (c *Client) SetGlobalPrivacySettings(ctx context.Context, settings *tg.GlobalPrivacySettings) error
func (c *Client) GetGlobalPrivacySettings(ctx context.Context) (*tg.GlobalPrivacySettings, error)
func (c *Client) SetAccountTTL(ctx context.Context, days int32) error
func (c *Client) GetAccountTTL(ctx context.Context) (int32, error)
```

---

### RPC Layer

```go
func (c *Client) Invoke(query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error)
func (c *Client) InvokeRaw(query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error)
func (c *Client) InvokeWithRawResult(ctx context.Context, query tg.TLObject) ([]byte, error)
func (c *Client) InvokeJSON(ctx context.Context, functionName string, payload []byte, useSnakeCase bool) ([]byte, error)
func (c *Client) Raw() *tg.RPCClient
func (c *Client) RPC() *tg.RPCClient
```

| Method | Description |
|--------|-------------|
| `Invoke` | High-level TL object invocation with wrapped errors |
| `InvokeRaw` | Low-level TL invocation returning raw errors |
| `InvokeWithRawResult` | Returns raw MTProto `rpc_result.result:Object` payload bytes; not a decoded Go struct and not necessarily gzip-unpacked |
| `InvokeJSON` | JSON-based RPC proxy (name-based invocation) |
| `Raw` / `RPC` | Returns the typed `RPCClient` for direct TL function calls |

---

## Update Handling

### Update Struct

The `Update` struct is populated by `HandleUpdates` and dispatched to handlers. It contains all possible update types:

```go
type Update struct {
    Users                   map[int64]*types.User
    Chats                   map[int64]*types.Chat
    Message                 *types.Message
    EditedMessage           *types.Message
    BusinessMessage         *types.Message
    EditedBusinessMessage   *types.Message
    DeletedMessages         *types.DeletedMessages
    DeletedBusinessMessages *types.DeletedMessages
    CallbackQuery           *types.CallbackQuery
    InlineQuery             *types.InlineQuery
    ChosenInlineResult      *types.ChosenInlineResult
    UserStatus              *types.UserStatusUpdated
    ChatMember              *types.ChatMemberUpdated
    MessageReaction         *types.MessageReactions
    MessageReactionCount    *types.MessageReactions
    Poll                    *types.PollUpdate
    BusinessConnection      *types.BusinessConnection
    Story                   *types.Story
    ChatBoost               *types.ChatBoostUpdated
    ChatJoinRequest         *types.ChatJoinRequest
    PreCheckoutQuery        *types.PreCheckoutQuery
    ShippingQuery           *types.ShippingQuery
    PurchasedPaidMedia      *types.PurchasedPaidMedia
    ManagedBot              *types.ManagedBotUpdated
    Error                   error
    Connected               bool
    Disconnected            bool
    Started                 bool
    Stopped                 bool
    Raw                     tg.TLObject
}
```

### Dispatcher

```go
type Dispatcher interface {
    Start(workers int) error
    Stop() error
    AddHandler(handler Handler, group int)
    RemoveHandler(handler Handler, group int)
    Enqueue(packet UpdatePacket) error
}

type UpdatePacket struct {
    Update tg.UpdatesClass
    Users  map[int64]*types.User
    Chats  map[int64]*types.Chat
}

type HandlerDispatcher struct { /* ... */ }

func NewHandlerDispatcher() *HandlerDispatcher
func (d *HandlerDispatcher) AddHandler(h Handler, group ...int)
func (d *HandlerDispatcher) RemoveHandler(h Handler)
func (d *HandlerDispatcher) Dispatch(client *Client, update *Update)
```

`HandlerDispatcher` is the built-in dispatcher. Handlers are sorted by group number; within a group, they run in insertion order. If a handler calls `StopPropagation()`, remaining handlers are skipped.

### Handlers

```go
type Handler interface {
    Check(update *Update) bool
    Handle(ctx *Context)
}

type Filter func(*Context) bool
type RemoveFunc func()
```

All handler constructors take `callback interface{}` which accepts multiple callback signatures (e.g. `func(*Context)`, `func(*Client, *types.Message)`, `func(*Context, *types.Message)` for message handlers). Lifecycle and error handlers take `func(*Context)`.

#### Handler Types (26 total)

| Type | Constructor | Triggers On |
|------|-------------|-------------|
| `MessageHandler` | `NewMessageHandler(cb, filters...)` | New text/media messages |
| `EditedMessageHandler` | `NewEditedMessageHandler(cb, filters...)` | Edited messages |
| `BusinessMessageHandler` | `NewBusinessMessageHandler(cb, filters...)` | New business messages |
| `EditedBusinessMessageHandler` | `NewEditedBusinessMessageHandler(cb, filters...)` | Edited business messages |
| `DeletedMessagesHandler` | `NewDeletedMessagesHandler(cb, filters...)` | Deleted messages |
| `DeletedBusinessMessagesHandler` | `NewDeletedBusinessMessagesHandler(cb, filters...)` | Deleted business messages |
| `CallbackQueryHandler` | `NewCallbackQueryHandler(cb, filters...)` | Callback queries |
| `InlineQueryHandler` | `NewInlineQueryHandler(cb, filters...)` | Inline queries |
| `ChosenInlineResultHandler` | `NewChosenInlineResultHandler(cb, filters...)` | Chosen inline results |
| `UserStatusHandler` | `NewUserStatusHandler(cb, filters...)` | User status updates |
| `ChatMemberHandler` | `NewChatMemberHandler(cb, filters...)` | Chat member updates |
| `MessageReactionHandler` | `NewMessageReactionHandler(cb, filters...)` | Message reactions |
| `MessageReactionCountHandler` | `NewMessageReactionCountHandler(cb, filters...)` | Reaction count updates |
| `PollHandler` | `NewPollHandler(cb, filters...)` | Poll updates |
| `BusinessConnectionHandler` | `NewBusinessConnectionHandler(cb, filters...)` | Business connection updates |
| `StoryHandler` | `NewStoryHandler(cb, filters...)` | Story updates |
| `ChatBoostHandler` | `NewChatBoostHandler(cb, filters...)` | Chat boost updates |
| `ChatJoinRequestHandler` | `NewChatJoinRequestHandler(cb, filters...)` | Chat join requests |
| `PreCheckoutQueryHandler` | `NewPreCheckoutQueryHandler(cb, filters...)` | Pre-checkout queries |
| `ShippingQueryHandler` | `NewShippingQueryHandler(cb, filters...)` | Shipping queries |
| `PurchasedPaidMediaHandler` | `NewPurchasedPaidMediaHandler(cb, filters...)` | Purchased paid media |
| `ManagedBotHandler` | `NewManagedBotHandler(cb, filters...)` | Managed bot updates |
| `RawUpdateHandler` | `NewRawUpdateHandler(cb, filters...)` | Any raw update |
| `LifecycleHandler` | `NewStartHandler(cb)` / `NewStopHandler(cb)` / `NewConnectHandler(cb)` / `NewDisconnectHandler(cb)` | Lifecycle events |
| `ErrorHandler` | `NewErrorHandler(cb, exceptions...)` | Error events |

#### Client Handler Registration (On* Methods)

All `On*` methods (except lifecycle/error) take `callback interface{}` and optional filters:

```go
func (c *Client) OnMessage(callback interface{}, filters ...Filter) Handler
func (c *Client) OnEditedMessage(callback interface{}, filters ...Filter) Handler
func (c *Client) OnBusinessMessage(callback interface{}, filters ...Filter) Handler
func (c *Client) OnEditedBusinessMessage(callback interface{}, filters ...Filter) Handler
func (c *Client) OnDeletedMessages(callback interface{}, filters ...Filter) Handler
func (c *Client) OnDeletedBusinessMessages(callback interface{}, filters ...Filter) Handler
func (c *Client) OnCallbackQuery(callback interface{}, filters ...Filter) Handler
func (c *Client) OnInlineQuery(callback interface{}, filters ...Filter) Handler
func (c *Client) OnChosenInlineResult(callback interface{}, filters ...Filter) Handler
func (c *Client) OnUserStatus(callback interface{}, filters ...Filter) Handler
func (c *Client) OnChatMember(callback interface{}, filters ...Filter) Handler
func (c *Client) OnMessageReaction(callback interface{}, filters ...Filter) Handler
func (c *Client) OnMessageReactionCount(callback interface{}, filters ...Filter) Handler
func (c *Client) OnPoll(callback interface{}, filters ...Filter) Handler
func (c *Client) OnBusinessConnection(callback interface{}, filters ...Filter) Handler
func (c *Client) OnStory(callback interface{}, filters ...Filter) Handler
func (c *Client) OnChatBoost(callback interface{}, filters ...Filter) Handler
func (c *Client) OnChatJoinRequest(callback interface{}, filters ...Filter) Handler
func (c *Client) OnPreCheckoutQuery(callback interface{}, filters ...Filter) Handler
func (c *Client) OnShippingQuery(callback interface{}, filters ...Filter) Handler
func (c *Client) OnPurchasedPaidMedia(callback interface{}, filters ...Filter) Handler
func (c *Client) OnManagedBot(callback interface{}, filters ...Filter) Handler
func (c *Client) OnRawUpdate(callback interface{}, filters ...Filter) Handler
```

Lifecycle and error handlers take `func(*Context)`:

```go
func (c *Client) OnStart(callback func(*Context)) Handler
func (c *Client) OnStop(callback func(*Context)) Handler
func (c *Client) OnConnect(callback func(*Context)) Handler
func (c *Client) OnDisconnect(callback func(*Context)) Handler
func (c *Client) OnError(callback func(*Context), exceptions ...error) Handler
```

**Example:**
```go
client.OnMessage(func(ctx *telegram.Context) {
    ctx.Reply("Echo: " + ctx.Message.Text)
}, telegram.Command("start"))
```

### Filters

#### Message Content Filters

```go
func Text(s string) Filter
func Command(commands ...string) Filter
func Regex(pattern string) Filter
func All() Filter
func Caption() Filter
```

#### Media Filters

```go
func Audio() Filter
func Video() Filter
func Animation() Filter
func Voice() Filter
func VideoNote() Filter
func Sticker() Filter
func Photo() Filter
func Document() Filter
func Contact() Filter
func Location() Filter
func LiveLocation() Filter
func Venue() Filter
func WebPage() Filter
func Poll() Filter
func Dice() Filter
func Game() Filter
func Giveaway() Filter
func GiveawayWinners() Filter
func Story() Filter
func PaidMedia() Filter
func Invoice() Filter
func MediaGroup() Filter
func MediaSpoiler() Filter
func MediaFilter() Filter
func HasMedia() Filter
```

#### Direction & Origin Filters

```go
func Me() Filter
func Bot() Filter
func Incoming() Filter
func Outgoing() Filter
func Forwarded() Filter
func Reply() Filter
func Mentioned() Filter
func ViaBot() Filter
func Direct() Filter
func SenderChat(chatIDs ...int64) Filter
```

#### Chat Type Filters

```go
func Private() Filter
func Group() Filter
func Channel() Filter
func Forum() Filter
func Business() Filter
func Topic(topicIDs ...int32) Filter
```

#### Service Message Filters

```go
func Service() Filter
func NewChatMembers() Filter
func LeftChatMember() Filter
func NewChatTitle() Filter
func NewChatPhoto() Filter
func DeleteChatPhoto() Filter
func GroupChatCreated() Filter
func SupergroupChatCreated() Filter
func ChannelChatCreated() Filter
func MigrateToChatID() Filter
func MigrateFromChatID() Filter
func PinnedMessage() Filter
func GameHighScore() Filter
func VideoChatStarted() Filter
func VideoChatEnded() Filter
func VideoChatMembersInvited() Filter
func SuccessfulPayment() Filter
```

#### Message Property Filters

```go
func User(userIDs ...int64) Filter
func Chat(chatIDs ...int64) Filter
func GuestMessage() Filter
func Scheduled() Filter
func FromScheduled() Filter
func PaidMessage() Filter
func LinkedChannel() Filter
func ReplyKeyboard() Filter
func InlineKeyboard() Filter
func SelfDestruction() Filter
```

#### Callback & Inline Filters

```go
func CallbackData(data string) Filter
func CallbackRegex(pattern string) Filter
func InlineQueryText(s string) Filter
func ChatActionFilter(chatID int64) Filter
```

#### Special Filters

```go
func Quote() Filter
func Admin() Filter
func Create(fn func(*Client, *Context) bool) Filter
func NewCommand(commands []string, prefixes []string, caseSensitive bool) Filter
```

#### Filter Combinators

```go
func (f Filter) And(other Filter) Filter
func (f Filter) Or(other Filter) Filter
func (f Filter) Not() Filter
```

**Example:**
```go
telegram.Private().And(telegram.Command("start"))
telegram.Photo().Or(telegram.Document())
```

---

## Context — Handler Context

### Context Struct

`Context` wraps a `Client`, an `Update`, and a `context.Context` for use inside handlers.

```go
type Context struct {
    Ctx     context.Context
    Client  *Client
    Update  *Update
    Stopped bool

    Message                 *types.Message
    EditedMessage           *types.Message
    BusinessMessage         *types.Message
    EditedBusinessMessage   *types.Message
    DeletedMessages         *types.DeletedMessages
    DeletedBusinessMessages *types.DeletedMessages
    CallbackQuery           *types.CallbackQuery
    InlineQuery             *types.InlineQuery
    ChosenInlineResult      *types.ChosenInlineResult
    UserStatus              *types.UserStatusUpdated
    ChatMember              *types.ChatMemberUpdated
    MessageReaction         *types.MessageReactions
    MessageReactionCount    *types.MessageReactions
    Poll                    *types.PollUpdate
    BusinessConnection      *types.BusinessConnection
    Story                   *types.Story
    ChatBoost               *types.ChatBoostUpdated
    ChatJoinRequest         *types.ChatJoinRequest
    PreCheckoutQuery        *types.PreCheckoutQuery
    ShippingQuery           *types.ShippingQuery
    PurchasedPaidMedia      *types.PurchasedPaidMedia
    ManagedBot              *types.ManagedBotUpdated
    Error                   error
    Connected               bool
    Disconnected            bool
    Started                 bool
}
```

### Constructor

```go
func (c *Client) NewContext(ctx context.Context) *Context
```

### Core Methods

```go
func (c *Context) StopPropagation()
func (c *Context) ResolvePeer(id int64) interface{}
```

### Context Methods

#### Message Methods

```go
func (c *Context) Reply(text string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Context) Edit(text string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) EditText(text string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) EditCaption(caption string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) EditMedia(media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) EditReplyMarkup(replyMarkup tg.ReplyMarkupClass) (*types.Message, error)
func (c *Context) Delete(opts ...*params.DeleteMessages) (int, error)
func (c *Context) Forward(chatID int64, opts ...*params.ForwardMessages) (*types.Message, error)
func (c *Context) Copy(chatID int64, opts ...*params.CopyMessage) (int64, error)
func (c *Context) Send(chatID int64, text string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Context) SendMedia(chatID int64, media tg.InputMediaClass, caption string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Context) React(reaction ...tg.ReactionClass) error
func (c *Context) SendPaidReaction(amount int64) error
func (c *Context) Read() error
func (c *Context) View() error
func (c *Context) DownloadMedia() ([]byte, error)
func (c *Context) DownloadMediaToFile(filePath string, fileSize int64) error
func (c *Context) Pin(opts ...*params.PinMessage) error
func (c *Context) Unpin() error
func (c *Context) SendChatAction(action tg.SendMessageActionClass) error
func (c *Context) GetMediaGroup() ([]*types.Message, error)
func (c *Context) Vote(options [][]byte) error
func (c *Context) StopPoll() error
func (c *Context) RetractVote() error
func (c *Context) DeleteChatHistory(revoke bool) (int, error)
func (c *Context) GetChatHistoryCount() (int, error)
func (c *Context) ForwardMediaGroup(chatID int64, opts ...*params.ForwardMessages) ([]*types.Message, error)
func (c *Context) SendGame(chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error)
func (c *Context) ReadMentions() error
func (c *Context) ReadReactions() error
```

#### Chat Methods

```go
func (c *Context) GetChat() (*types.Chat, error)
func (c *Context) LeaveChat() error
func (c *Context) Ban(userID int64) error
func (c *Context) Unban(userID int64) error
func (c *Context) Restrict(userID int64, rights *tg.ChatBannedRights) error
func (c *Context) Promote(userID int64, rights *tg.ChatAdminRights) error
func (c *Context) SetAdministratorTitle(userID int64, title string) error
func (c *Context) GetMember(userID int64) (*types.ChatMember, error)
func (c *Context) GetMembers(limit, offset int) ([]*types.ChatMember, error)
func (c *Context) SetTitle(title string) error
func (c *Context) SetDescription(about string) error
func (c *Context) SetPhoto(photo tg.InputChatPhotoClass) error
func (c *Context) DeleteChatPhoto() error
func (c *Context) SetTTL(ttl int) error
func (c *Context) SetPermissions(permissions *tg.ChatBannedRights) error
func (c *Context) ExportInviteLink() (string, error)
func (c *Context) Archive() error
func (c *Context) Unarchive() error
func (c *Context) MarkUnread(unread bool) error
func (c *Context) SetProtectedContent(enabled bool) error
func (c *Context) UnpinAllMessages() (int, error)
func (c *Context) Mute() error
func (c *Context) Unmute() error
func (c *Context) AddMembers(userIDs []int64) error
func (c *Context) SetSlowMode(seconds int) error
func (c *Context) GetChatEventLog(query string, limit int) ([]*types.ChatEvent, error)
func (c *Context) SearchMessages(query string, opts ...*params.SearchMessages) ([]*types.Message, error)
func (c *Context) GetHistory(limit int, offsetID int32) ([]*types.Message, error)
func (c *Context) GetCommonChats(userID int64, limit int) ([]*types.Chat, error)
```

#### Callback Methods

```go
func (c *Context) Answer(text string, showAlert bool) error
func (c *Context) AnswerCallbackQuery(text string, showAlert bool) error
func (c *Context) AnswerCallback(text string, showAlert bool) error
func (c *Context) CallbackEditText(text string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) CallbackEditCaption(caption string, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) CallbackEditMedia(media tg.InputMediaClass, opts ...*params.EditMessage) (*types.Message, error)
func (c *Context) CallbackEditReplyMarkup(replyMarkup tg.ReplyMarkupClass) (*types.Message, error)
```

#### Inline Methods

```go
func (c *Context) AnswerInlineQuery(results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error
func (c *Context) AnswerInline(results []tg.InputBotInlineResultClass, opts ...*AnswerInlineQueryOption) error
func (c *Context) AnswerShipping(queryID int64, ok bool, options []*tg.ShippingOption) error
func (c *Context) AnswerPreCheckout(queryID int64, ok bool, errorMessage string) error
```

#### Stories Methods

```go
func (c *Context) SendStory(chatID int64, media tg.InputMediaClass, opts ...*SendStoryOption) (*types.Story, error)
func (c *Context) EditStoryCaption(chatID int64, storyID int32, caption string) (*types.Story, error)
func (c *Context) EditStoryMedia(chatID int64, storyID int32, media tg.InputMediaClass) (*types.Story, error)
func (c *Context) DeleteStories(chatID int64, storyIDs []int32) error
func (c *Context) GetStories(userID int64, storyIDs []int32) ([]*types.Story, error)
func (c *Context) GetChatStories(chatID int64) ([]*types.Story, error)
func (c *Context) GetStoryViews(chatID int64, storyIDs []int32) ([]*tg.StoryViews, error)
func (c *Context) ForwardStory(target, source int64, storyID int32) (*types.Message, error)
func (c *Context) ReadChatStories(chatID int64, storyIDs []int32) error
```

#### Payments Methods

```go
func (c *Context) GetPaymentForm(chatID int64, messageID int32, opts ...*GetPaymentFormOption) (tg.PaymentFormClass, error)
func (c *Context) SendPaymentForm(formID int64, chatID int64, messageID int32, creds tg.InputPaymentCredentialsClass, opts ...*SendPaymentFormOption) (tg.PaymentResultClass, error)
func (c *Context) GetStarsBalance(chatID int64) (int64, error)
func (c *Context) SendGift(userID int64, giftID int64, message string) error
```

#### Account Methods

```go
func (c *Context) GetBusinessConnection(connectionID string) (*tg.BotBusinessConnection, error)
```

#### Premium Methods

```go
func (c *Context) ApplyBoost(chatID int64, opts ...*ApplyBoostOption) ([]*tg.MyBoost, error)
func (c *Context) GetBoostsStatus(chatID int64) (*tg.PremiumBoostsStatus, error)
func (c *Context) GetBoosts(opts ...*GetBoostsOption) ([]*tg.MyBoost, error)
```

---

## Peer Resolution

### ChatRef / UserRef

Opaque peer references used throughout the API:

```go
type ChatRef struct { /* unexported */ }
type UserRef struct { /* unexported */ }

func ChatID(id int64) ChatRef
func ChatUsername(username string) ChatRef
func ChatPeer(peer tg.InputPeerClass) ChatRef
func UserID(id int64) UserRef
func UserUsername(username string) UserRef
func UserInput(user tg.InputUserClass) UserRef
```

### Resolver Helpers

```go
type PeerResolver interface {
    ResolvePeerCache(id int64) (tg.InputPeerClass, error)
    ResolveUsername(ctx context.Context, username string) (tg.InputPeerClass, error)
}

func PeerToInputPeer(peer tg.PeerClass, users []tg.UserClass, chats []tg.ChatClass) (tg.InputPeerClass, error)
```

---

## Utilities

```go
var TestDCs  map[int]string
var ProdDCs  map[int]string
var DefaultTestPort  int = 80
var DefaultProdPort  int = 443

func ServerTime(offset int) int32
func ResolveDCAddress(dcID int, testMode bool) string
func DefaultDCPort(testMode bool) int
func GuessMIMEType(filename string) string
func GuessExtension(mime string) string

func (c *Client) GetDialogs(ctx context.Context, limit int, offsetDate int32) ([]*types.Chat, error)
```

---

## tg — Generated TL Types

### Core Interfaces

```go
type TLObject interface {
    Encode(b *bytes.Buffer) error
    ConstructorID() uint32
}

func EncodeTLObject(b *bytes.Buffer, obj TLObject) error
func ReadTLObject(r io.Reader) (TLObject, error)

var Registry map[uint32]func(io.Reader) (TLObject, error)
```

### Invoker

```go
type Invoker interface {
    RPCInvoke(ctx context.Context, input TLObject, decode func(io.Reader) (TLObject, error)) (TLObject, error)
}

type Client struct { rpc Invoker }
type RPCClient struct { rpc Invoker }

func NewClient(invoker Invoker) *Client
func (c *RPCClient) RPC() Invoker
func (c *RPCClient) Invoke(ctx context.Context, input TLObject, decode func(io.Reader) (TLObject, error)) (TLObject, error)
func (c *RPCClient) InvokeJSON(ctx context.Context, functionName string, payload []byte, useSnakeCase bool) ([]byte, error)
```

### TL Primitives

```go
func ReadInt(r io.Reader) uint32
func WriteInt(b *bytes.Buffer, v uint32)
func ReadLong(r io.Reader) int64
func WriteLong(b *bytes.Buffer, v int64)
func ReadInt128(r io.Reader) [16]byte
func WriteInt128(b *bytes.Buffer, v [16]byte)
func ReadInt256(r io.Reader) [32]byte
func WriteInt256(b *bytes.Buffer, v [32]byte)
func ReadDouble(r io.Reader) float64
func WriteDouble(b *bytes.Buffer, v float64)
func ReadBool(r io.Reader) bool
func WriteBool(b *bytes.Buffer, v bool)
func ReadBytes(r io.Reader) []byte
func WriteBytes(b *bytes.Buffer, v []byte)
func ReadString(r io.Reader) string
func WriteString(b *bytes.Buffer, v string)
func ReadVectorInt(r io.Reader) []int32
func WriteVectorInt(b *bytes.Buffer, v []int32)
func ReadVectorLong(r io.Reader) []int64
func WriteVectorLong(b *bytes.Buffer, v []int64)
func ReadVectorString(r io.Reader) []string
func WriteVectorString(b *bytes.Buffer, v []string)
func ReadVectorBytes(r io.Reader) [][]byte
func WriteVectorBytes(b *bytes.Buffer, v [][]byte)

const BoolTrueID  uint32 = 0x997275B5
const BoolFalseID uint32 = 0xBC799737
const VectorID    uint32 = 0x1CB5C415

type TLBool bool // implements TLObject
```

### Layer

```go
const Layer int = 224
```

### Message & Container

```go
const MessageID uint32 = 0x5BB8E511

type Message struct {
    MsgID int64
    SeqNo uint32
    Body  TLObject
}

func (m *Message) ConstructorID() uint32
func (m *Message) Encode(b *bytes.Buffer) error
func DecodeMessage(r io.Reader) (*Message, error)

const MsgContainerID uint32 = 0x73F1F8DC

type MsgContainer struct {
    Messages []*Message
}

func DecodeMsgContainer(r io.Reader) (*MsgContainer, error)
```

### Gzip

```go
const GzipPackedID uint32 = 0x3072CFA1

type GzipPacked struct {
    Data TLObject
}

func DecodeGzipPacked(r io.Reader) (*GzipPacked, error)
```

### Generated Maps

```go
var NamesMap        map[string]uint32                    // qualified name → constructor ID
var FunctionsMap    map[uint32]func() TLObject           // function ID → factory
var ConstructorMap  map[uint32]func() TLObject           // constructor ID → factory
```

### Generated Types

The `tg` package contains **365+ generated types** from the Telegram TL schema:

- **~90 concrete struct types** — e.g. `UserTL`, `MessageTL`, `Channel`, `PeerUser`, `InputPeerUser`, `InputMediaPhoto`, `ChatAdminRights`, `BotCommand`, etc.
- **~275 interface types** (suffixed `Class`) — e.g. `UserClass`, `MessageClass`, `ChatClass`, `InputPeerClass`, `UpdateClass`, `MessageMediaClass`, `ReplyMarkupClass`, etc.
- **~500+ RPC function request types** — each with a corresponding method on `RPCClient`

Each generated type implements the `TLObject` interface with `Encode` and `ConstructorID` methods, plus a `Decode<Name>` function for deserialization.

---

## tgerr — Error Handling

### Error Type

```go
type Error struct {
    Code     int
    Message  string
    Type     string
    Argument int
}

func New(code int, msg string) *Error
func (e *Error) Error() string
func (e *Error) IsType(t string) bool
func (e *Error) IsCode(code int) bool
func (e *Error) IsOneOf(tt ...string) bool
func (e *Error) IsCodeOneOf(codes ...int) bool

func AsType(err error, t string) (rpcErr *Error, ok bool)
func As(err error) (rpcErr *Error, ok bool)
func Is(err error, tt ...string) bool
func IsCode(err error, code ...int) bool
```

### Flood Wait

```go
var FloodWaitErrors []string

func AsFloodWait(err error) (d time.Duration, ok bool)
func FloodWait(ctx context.Context, err error) (bool, error)
```

`AsFloodWait` extracts the flood wait duration from an error. `FloodWait` sleeps for that duration, respecting context cancellation.

### Security Checks

```go
type SecurityCheckMismatch struct {
    Name string
}

func (e *SecurityCheckMismatch) Error() string
func Check(ok bool, name string)
```

`Check` panics with `SecurityCheckMismatch` if `ok` is false. Used for MTProto integrity verification.

### Error Constants

~280+ error type constants. Key examples:

```go
const ErrFloodWait            = "FLOOD_WAIT"
const ErrFloodPremiumWait     = "FLOOD_PREMIUM_WAIT"
const ErrPhoneNumberInvalid   = "PHONE_NUMBER_INVALID"
const ErrSessionPasswordNeeded = "SESSION_PASSWORD_NEEDED"
const ErrChatWriteForbidden   = "CHAT_WRITE_FORBIDDEN"
const ErrPeerFlood            = "PEER_FLOOD"
const ErrUserDeactivated      = "USER_DEACTIVATED"
const ErrApiIdInvalid         = "API_ID_INVALID"
const ErrAuthKeyInvalid       = "AUTH_KEY_INVALID"
```

Each constant has a matching `Is<Name>(err error) bool` function, e.g. `IsFloodWait(err)`, `IsPhoneNumberInvalid(err)`.

---

## telegram/types — Domain Types

Parsed, human-friendly domain types converted from raw TL objects.

### Enum Types

| Type | Key Constants |
|------|---------------|
| `ChatType` | `ChatTypePrivate`, `ChatTypeBot`, `ChatTypeGroup`, `ChatTypeSupergroup`, `ChatTypeChannel`, `ChatTypeForum` |
| `ChatAction` | `ChatActionTyping`, `ChatActionUploadPhoto`, `ChatActionRecordVideo`, etc. |
| `ParseMode` | `ParseModeDefault`, `ParseModeMarkdown`, `ParseModeHTML`, `ParseModeDisabled` |
| `MessageEntityType` | `MessageEntityTypeMention`, `Bold`, `Italic`, `Code`, `Pre`, `Blockquote`, `CustomEmoji`, etc. |
| `MessageMediaType` | `MessageMediaTypePhoto`, `Audio`, `Document`, `Video`, `Sticker`, `Animation`, `Voice`, etc. |
| `UserStatus` | `UserStatusOnline`, `Offline`, `Recently`, `LastWeek`, `LastMonth`, `LongAgo` |
| `ChatMemberStatus` | `ChatMemberStatusOwner`, `Administrator`, `Member`, `Restricted`, `Left`, `Kicked` |
| `StickerType` | `StickerTypeRegular`, `StickerTypeMask`, `StickerTypeCustomEmoji` |
| `PrivacyKey` | `PrivacyKeyStatusTimestamp`, `PrivacyKeyChatInvite`, etc. |
| `PollType` | `PollTypeRegular`, `PollTypeQuiz` |
| `ClientPlatform` | `ClientPlatformAndroid`, `ClientPlatformiOS`, `ClientPlatformDesktop`, etc. |

### Core Structs

```go
type Message struct { /* ID, Text, Date, From, Chat, ForwardHeader, ReplyTo, Media, Entities, Reactions, etc. */ }
type User struct { /* ID, FirstName, LastName, Username, Bot, Status, Photo, etc. */ }
type Chat struct { /* ID, Title, Type, Username, MembersCount, Permissions, Photo, etc. */ }
type ChatPreview struct { /* Title, Type, MembersCount, Photo, etc. */ }
type Story struct { /* ID, Date, FromID, Media, Caption, Views, etc. */ }
type CallbackQuery struct { /* ID, UserID, ChatInstance, MessageID, Data, GameShortName */ }
type InlineQuery struct { /* ID, UserID, Query, Offset */ }
type ChosenInlineResult struct { /* ID, UserID, Query, ResultID, Location */ }
type ChatMember struct { /* User, Status, CustomTitle, JoinedDate, AdminRights, BannedRights */ }
type ChatMemberUpdated struct { /* Chat, From, Date, OldStatus, NewStatus */ }
type ChatInviteLink struct { /* Link, Title, AdminID, Date, ExpireDate, UsageLimit, etc. */ }
type DeletedMessages struct { /* ChatID, MessageIDs */ }
type MessageEntity struct { /* Type, Offset, Length, URL, User, Language, CustomEmojiID */ }
type MessageReactions struct { /* ChatID, MessageID, Reactions */ }
type PollUpdate struct { /* ChatID, MessageID, Poll */ }
type UserStatusUpdated struct { /* UserID, Status */ }
type ChatPhoto struct { /* SmallFileID, BigFileID, HasAnimation */ }
type ChatPermissions struct { /* CanSendMessages, CanSendMedia, CanSendPolls, CanChangeInfo, etc. */ }
type ChatAdminRights struct { /* CanChangeInfo, CanPostMessages, CanEditMessages, CanDeleteMessages, etc. */ }
type ChatBannedRights struct { /* UntilDate, ViewMessages, SendMessages, SendMedia, etc. */ }
type BusinessConnection struct { /* ID, UserID, DCID, Date */ }
type ChatBoostUpdated struct { /* ChatID, Boost */ }
type ChatJoinRequest struct { /* ChatID, UserID, Date, Bio */ }
type PreCheckoutQuery struct { /* ID, UserID, Currency, TotalAmount, Payload */ }
type ShippingQuery struct { /* ID, UserID, Payload, ShippingAddress */ }
type PurchasedPaidMedia struct { /* UserID, Payload */ }
type ManagedBotUpdated struct { /* UserID, BotID */ }
```

### InputFile

Strongly-typed file parameter for all media-sending methods. Re-exported from the `telegram` package via type alias.

```go
type InputFile struct { /* unexported fields */ }

func FileID(s string) *InputFile
func FromIDs(ID, accessHash int64, fileRef []byte) *InputFile
func URL(u string) *InputFile
func Path(p string) *InputFile
func Reader(r io.ReadSeeker, fileName string, size int64) *InputFile
func FromBytes(data []byte, fileName string) *InputFile

type MediaKind int
// MediaKindAuto, MediaKindPhoto, MediaKindDocument, MediaKindAudio,
// MediaKindVideo, MediaKindAnimation, MediaKindVoice, MediaKindVideoNote,
// MediaKindSticker
```

### Media Interface

```go
type Media interface {
    MediaType() MessageMediaType
}
```

Implementations: `PhotoMedia`, `DocumentMedia`, `ContactMedia`, `LocationMedia`, `VenueMedia`, `WebPageMedia`, `PollMedia`, `DiceMedia`, `GameMedia`, `InvoiceMedia`, `StoryMedia`, `GiveawayMedia`, `GiveawayResultsMedia`, `PaidMedia`.

### Reply Markup

```go
type ReplyMarkup struct {
    Type    ReplyMarkupType
    Buttons [][]Button
    /* InlineKeyboard, ReplyKeyboard, etc. */
}

type Button struct {
    Text string
    Type ButtonType
    URL  string
    Data string
    /* Callback, URL, SwitchInline, etc. */
}
```

### Parsing Functions

```go
func ParseMessage(raw tl.MessageClass, pm *PeerMap) *Message
func ParseUser(raw tl.UserClass) *User
func ParseChatFromUser(raw tl.UserClass) *Chat
func ParseChatFromChat(raw tl.ChatClass) *Chat
func ParseChatFromPeer(peer tl.PeerClass, pm *PeerMap) *Chat
func ParseMedia(raw tl.MessageMediaClass) Media
func ParseReplyMarkup(raw tl.ReplyMarkupClass) *ReplyMarkup
func ParseMessageEntity(raw tl.MessageEntityClass) *MessageEntity
func ParseMessageEntities(raw []tl.MessageEntityClass) []*MessageEntity
func ParseStory(raw tl.StoryItemClass, pm *PeerMap) *Story
func ParseCallbackQuery(raw tl.UpdateClass) *CallbackQuery
func ParseInlineQuery(raw tl.UpdateClass) *InlineQuery
func ParseChatPermissions(raw *tl.ChatBannedRights) *ChatPermissions
func ParseChatAdminRights(raw *tl.ChatAdminRights) *ChatAdminRights
func ParseChatBannedRights(raw *tl.ChatBannedRights) *ChatBannedRights
func ParseChatParticipant(raw tl.ChatParticipantClass, users map[int64]tl.UserClass) *ChatMember
func ParseChannelParticipant(raw tl.ChannelParticipantClass, users map[int64]tl.UserClass) *ChatMember
func ParseChatInviteLink(raw *tl.ChatInviteExported, users map[int64]tl.UserClass) *ChatInviteLink
func NewPeerMap(users []*tl.UserTL, chats []*tl.ChatTL, channels []*tl.Channel) *PeerMap
func NewPeerMapFromClasses(users []tl.UserClass, chats []tl.ChatClass) *PeerMap
func GetPeerID(peer tl.PeerClass) int64
```

---

## telegram/params — API Parameters

Option and parameter structs used to configure Telegram API calls. Each struct corresponds to a specific API method group.

### ParseMode

```go
type ParseMode string

const (
    ParseModeDefault   ParseMode = "default"
    ParseModeMarkdown  ParseMode = "markdown"
    ParseModeHTML      ParseMode = "html"
    ParseModeDisabled  ParseMode = "disabled"
    MarkdownV2         ParseMode = "MarkdownV2"
)
```

### Message Option Structs

```go
type SendMessage struct { /* DisableNotification, Silent, Background, ClearDraft, NoForwards,
    InvertMedia, ReplyToMessageID, ReplyTo, ReplyMarkup, Entities, ParseMode,
    ScheduleDate, EffectID, SendAs */ }

type EditMessage struct { /* DisableWebPagePreview, InvertMedia, ReplyMarkup, ParseMode,
    Entities, ScheduleDate */ }

type ForwardMessages struct { /* DisableNotification, NoForwards, DropAuthor, DropMediaCaptions,
    ScheduleDate */ }

type CopyMessage struct { /* Caption, DisableNotification, ReplyToMessageID, ReplyMarkup,
    ScheduleDate, DropAuthor */ }

type DeleteMessages struct { /* Revoke */ }
type PinMessage struct { /* Silent, Unpin */ }

type SendMediaGroup struct { /* DisableNotification, Silent, Background, ClearDraft, NoForwards,
    ReplyToMessageID, ReplyTo, ScheduleDate, EffectID, SendAs */ }

type SearchMessages struct { /* Limit, OffsetID, MinDate, MaxDate, FromID, Filter, TopMsgID */ }
type SearchGlobal struct { /* Limit, OffsetRate, OffsetID, OffsetPeer, MinDate, MaxDate,
    BroadcastsOnly, GroupsOnly, FolderID, Filter */ }
```

### Media Option Structs

Each media type has its own struct with common fields plus media-specific options:

```go
type SendPhoto struct { /* common... + FileName */ }
type SendDocument struct { /* common... + FileName, Thumb, MimeType */ }
type SendVideo struct { /* common... + Duration float64, Width, Height, SupportsStreaming, FileName, Thumb */ }
type SendAudio struct { /* common... + Duration int32, Performer, Title, FileName, Thumb */ }
type SendAnimation struct { /* common... + FileName, Thumb */ }
type SendVoice struct { /* common... + Duration int32, FileName */ }
type SendVideoNote struct { /* common... + Duration float64, FileName, Thumb */ }
type SendSticker struct { /* common... + FileName */ }
```

### Special Option Structs

```go
type SendContact struct { /* common... */ }
type SendLocation struct { /* common... */ }
type SendVenue struct { /* common... */ }
type SendPoll struct { /* common... */ }
type SendDice struct { /* Emoticon */ }
type SendGame struct { /* common... */ }
type SendChecklist struct { /* common... */ }
type SendInlineBotResult struct { /* common... + QueryID, ResultID, HideVia */ }
```

### Utility

```go
func GetOptDef[T comparable](def T, opts ...T) T
```

Returns the first valid option from `opts`, or `def` when empty. Panics if more than one option is passed.

---

## telegram/parser — Text Parsing

```go
type ParseMode int

const (
    ParseModeDefault  ParseMode = iota
    ParseModeHTML
    ParseModeMarkdown
    ParseModeDisabled
)

func Parse(mode ParseMode, text string) (string, []tl.MessageEntityClass, error)
```

### HTML Parser

```go
type HTMLParser struct {}

func NewHTMLParser() *HTMLParser
func (p *HTMLParser) Parse(html string) (string, []tl.MessageEntityClass, error)
```

Supported tags: `<b>`, `<i>`, `<u>`, `<s>`, `<a>`, `<code>`, `<pre>`, `<blockquote>`, `<spoiler>`, `<tg-spoiler>`.

### Markdown Parser

```go
type MarkdownParser struct {}

func NewMarkdownParser() *MarkdownParser
func (p *MarkdownParser) Parse(md string) (string, []tl.MessageEntityClass, error)
```

Supported: `**bold**`, `__italic__`, `` `code` ``, ` ```pre``` `, `--underline--`, `~~strikethrough~~`, `[text](url)`.

### Utilities

```go
func AddSurrogates(text string) string
func RemoveSurrogates(text string) (string, error)
func ReplaceOnce(source, old, newStr string, start int) string
```

---

## telegram/fileid — File ID

### Types

```go
type FileType byte

const (
    FileTypeThumbnail       FileType = 0
    FileTypePhoto            FileType = 1
    FileTypeVoice            FileType = 2
    FileTypeVideo            FileType = 3
    FileTypeDocument         FileType = 4
    FileTypeEncrypted        FileType = 5
    FileTypeTemp             FileType = 6
    FileTypeSticker          FileType = 7
    FileTypeAudio            FileType = 8
    FileTypeAnimation        FileType = 9
    FileTypeVideoNote        FileType = 10
    FileTypeSecureRaw        FileType = 11
    FileTypeSecureDocument   FileType = 12
    FileTypeBackground       FileType = 13
    FileTypeDocumentPhoto    FileType = 14
)

func (ft FileType) IsPhoto() bool

type ThumbnailSource byte
const (
    ThumbnailSourceLegacy           ThumbnailSource = 0
    ThumbnailSourceThumbnail        ThumbnailSource = 1
    ThumbnailSourceDialogPhotoSmall ThumbnailSource = 2
    ThumbnailSourceDialogPhotoBig   ThumbnailSource = 3
    ThumbnailSourceStickerSetThumb  ThumbnailSource = 4
)

type FileUniqueType byte
const (
    FileUniqueTypeWeb       FileUniqueType = 0
    FileUniqueTypePhoto     FileUniqueType = 1
    FileUniqueTypeDocument  FileUniqueType = 2
    FileUniqueTypeSecure    FileUniqueType = 3
    FileUniqueTypeEncrypted FileUniqueType = 4
    FileUniqueTypeTemp      FileUniqueType = 5
)
```

### File ID Encode/Decode

```go
type PhotoSizeSource struct {
    Type               ThumbnailSource
    Secret             int64
    VolumeID           int64
    LocalID            int32
    PhotoID            int64
    ChatID             int64
    ChatAccessHash     int64
    StickerSetID       int64
    StickerSetAccessHash int64
    ThumbnailFileType  FileType
    ThumbnailSize      int32
}

type FileID struct {
    Type       FileType
    DCID       int32
    ID         int64
    AccessHash int64
    VolumeID   int64
    LocalID    int32
    Source     PhotoSizeSource
}

func Encode(f FileID) (string, error)
func Decode(s string) (FileID, error)
```

---

## compiler/tlgen — TL Code Generation

### Types

```go
type Section int
const (
    SectionTypes     Section = iota
    SectionFunctions
)

type Arg struct {
    Name     string
    Type     string
    FlagBit  int
    FlagName string
    Generic  bool
}

type Combinator struct {
    Section    Section
    QualName   string
    Namespace  string
    Name       string
    ID         uint32
    HasFlags   bool
    Args       []Arg
    QualType   string
    TypeSpace  string
    Type       string
    Category   string
    IsBare     bool
}

func (c *Combinator) FlagArgs() []Arg
func (c *Combinator) NonFlagArgs() []Arg
```

### Parser

```go
func Parse(r io.Reader) ([]Combinator, error)
```

Parses a TL schema definition file into `Combinator` slices.

### Generator

```go
func SnakeToPascal(s string) string
func CamelToSnake(s string) string
func GenerateTypes(outDir string, combos []Combinator, layer int) error
func GenerateGroupedTypes(outDir string, combos []Combinator, layer int) error
func GenerateBases(outDir string, combos []Combinator) error
func GenerateFunctions(outDir string, combos []Combinator, layer int) error
func GenerateGroupedFunctions(outDir string, combos []Combinator, layer int) error
func GeneratePackageFiles(outDir, pkgName string, layer int) error
func GenerateNamesMap(outDir string, combos []Combinator) error
func GenerateFunctionsMap(outDir string, combos []Combinator) error
func GenerateConstructorsMap(outDir string, combos []Combinator) error
```

| Function | Description |
|----------|-------------|
| `GenerateTypes` | Generate all TL types into a single file |
| `GenerateGroupedTypes` | Generate TL types, one file per base type |
| `GenerateBases` | Generate base interface types |
| `GenerateFunctions` | Generate all TL RPC functions into one file |
| `GenerateGroupedFunctions` | Generate TL functions, one file per function |
| `GeneratePackageFiles` | Generate package doc file |
| `GenerateNamesMap` | Generate the `NamesMap` variable |
| `GenerateFunctionsMap` | Generate the `FunctionsMap` variable |
| `GenerateConstructorsMap` | Generate the `ConstructorMap` variable |

---

## internal — Internal Packages

> These packages are internal and not importable outside the module. Documented for contributors.

### internal/crypto

```go
type ServerKey struct {
    N *big.Int
    E *big.Int
}

var ServerPublicKeys map[int64]*ServerKey

func GetServerKey(fingerprint int64) (*ServerKey, bool)
func RSAEncrypt(data []byte, fingerprint int64) ([]byte, error)
func KDF(authKey, msgKey []byte, outgoing bool) (aesKey, aesIV []byte)
func Pack(message *tg.Message, salt int64, sessionID, authKey, authKeyID []byte) []byte
func Unpack(data, sessionID, authKey, authKeyID []byte) (*tg.Message, []byte)
func IGEEncrypt(data, key, iv []byte) []byte
func IGEDecrypt(data, key, iv []byte) []byte
func CTREncrypt(data, key, iv []byte) []byte
func CTRDecrypt(data, key, iv []byte) []byte

type CTRCipher struct { /* ... */ }
func NewCTRCipher(key, iv []byte) *CTRCipher
func (c *CTRCipher) Process(data []byte) []byte

var CurrentDHPrime *big.Int
func Decompose(pq int64) int64
func ComputePasswordHash(password string, salt1, salt2 []byte) []byte

type SRPResult struct {
    SrpID int64
    A     []byte
    M1    []byte
}

func ComputeSRP(salt1, salt2 []byte, g, p *big.Int, srpB []byte, srpID int64, password string) (*SRPResult, error)
```

### internal/session

```go
type Transport interface {
    Send(data []byte) error
    Recv() ([]byte, error)
    Close() error
    IsConnected() bool
}

type AuthFunc func(transport Transport) (*AuthResult, error)

type AuthResult struct {
    AuthKey   []byte
    ServerSalt int64
    ServerTime int32
}

type Session struct { /* ... */ }

func NewSession(dc DataCenter, st storage.Storage, deviceModel, appVersion, systemLang, langCode string) (*Session, error)

func (s *Session) DC() DataCenter
func (s *Session) SessionID() int64
func (s *Session) AuthKey() []byte
func (s *Session) IsConnected() bool
func (s *Session) SetAuthKey(key []byte)
func (s *Session) SetServerSalt(salt int64)
func (s *Session) SetServerTime(t time.Time)
func (s *Session) SetTransport(t Transport)
func (s *Session) SetUpdateHandler(fn func(tg.TLObject))
func (s *Session) Send(msgID int64, seqNo uint32, body tg.TLObject, timeout time.Duration) ([]byte, error)
func (s *Session) Invoke(query tg.TLObject, retries int, timeout time.Duration) (tg.TLObject, error)
func (s *Session) Start(timeout time.Duration) error
func (s *Session) Stop()
func (s *Session) Connect(transport Transport, timeout time.Duration) error
func (s *Session) ConnectWithAuth(transport Transport, authFunc AuthFunc, timeout time.Duration) error

type DataCenter struct {
    ID       int
    TestMode bool
    IPv6     bool
}

func (dc *DataCenter) Address() string
func (dc *DataCenter) Port() int
func (dc *DataCenter) String() string
```

### internal/storage

```go
type Storage interface {
    DCID() int
    SetDCID(id int)
    APIID() int
    SetAPIID(id int)
    TestMode() bool
    SetTestMode(v bool)
    AuthKey() []byte
    SetAuthKey(key []byte)
    UserID() int64
    SetUserID(id int64)
    IsBot() bool
    SetIsBot(v bool)
    Date() int32
    SetDate(d int32)
    ServerAddress() string
    SetServerAddress(addr string)
    Port() int
    SetPort(port int)
    State() int
    SetState(state int)
    ExportSessionString() (string, error)
    Close() error
}

type MemoryStorage struct { /* ... */ }
func NewMemoryStorage() *MemoryStorage

type SQLiteStorage struct { /* ... */ }
func NewSQLiteStorage(path string) (*SQLiteStorage, error)
```

### internal/transport

```go
type Transport interface {
    Send(buf *bytes.Buffer) error
    Recv() ([]byte, error)
}
```

| Type | Constructor | Description |
|------|-------------|-------------|
| `TCPFull` | `NewTCPFull(conn net.Conn)` | Full transport with length prefix, seq_no, CRC32 |
| `TCPAbridged` | `NewTCPAbridged(conn net.Conn)` | Abridged transport (0xEF marker, compact length) |
| `TCPIntermediate` | `NewTCPIntermediate(conn net.Conn)` | Intermediate transport (0xEEEEEEEE marker, 4-byte length) |
| `TCPObfuscated` | `NewTCPObfuscated(inner Transport, marker byte)` | AES-CTR obfuscated wrapper |
| `TCPAbridgedO` | `NewTCPAbridgedO(conn net.Conn)` | Obfuscated + Abridged |
| `TCPIntermediateO` | `NewTCPIntermediateO(conn net.Conn)` | Obfuscated + Intermediate |

All transport types implement `Connect() error`, `Send(buf *bytes.Buffer) error`, and `Recv() ([]byte, error)`.

---

## Quick Start Example

```go
package main

import (
    "context"
    "log"

    "github.com/mtgo-labs/mtgo/telegram"
    "github.com/mtgo-labs/mtgo/telegram/types"
)

func main() {
    client, err := telegram.NewClient(12345, "api_hash", &telegram.Config{
        SessionName: "my_bot",
        BotToken:    "123456:ABCDEF",
    })
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Connect(0); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect()

    // Register message handler
    client.OnMessage(func(ctx *telegram.Context) {
        if ctx.Message == nil {
            return
        }
        ctx.Reply("Echo: " + ctx.Message.Text)
    }, telegram.Command("echo"))

    // Send a message
    msg, err := client.SendMessage(context.Background(), telegram.ChatID(-100123456), "Hello!")
    if err != nil {
        log.Fatal(err)
    }
    _ = msg

    client.Idle() // block until Stop()
}
```
