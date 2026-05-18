package telegram

import (
	"errors"
	"fmt"
)

// Client connection and lifecycle errors.
//
// These errors indicate the state of the client's connection to Telegram.
// Use errors.Is to check for specific conditions.
//
// Example:
//
//	err := client.Connect(ctx)
//	if errors.Is(err, telegram.ErrNotConnected) {
//		log.Println("client is not connected")
//	}
var (
	// ErrNotConnected is returned when an operation requires an active connection
	// but the client is currently disconnected.
	ErrNotConnected = errors.New("client: not connected")
	// ErrAlreadyConnected is returned when Connect is called on a client that is
	// already connected to Telegram.
	ErrAlreadyConnected = errors.New("client: already connected")
	// ErrPeerNotFound is returned when a peer (user, chat, or channel) cannot be
	// resolved from the given identifier.
	ErrPeerNotFound = errors.New("client: peer not found")
	// ErrClientClosed is returned when an operation is attempted on a client that
	// has been closed via Disconnect or Close.
	ErrClientClosed = errors.New("client: closed")
	// ErrReconnectFailed is returned when the client was unable to reconnect to
	// Telegram after the configured number of retry attempts.
	ErrReconnectFailed = errors.New("client: reconnect failed")
	// ErrHealthTimeout is returned when a health check ping to the server does not
	// receive a response within the expected timeout.
	ErrHealthTimeout = errors.New("client: health check timeout")
	// ErrMigrationFailed is returned when a DC migration attempt fails entirely.
	ErrMigrationFailed = errors.New("client: dc migration failed")
	// ErrMigrationUnsafe is returned when a non-idempotent request (e.g.
	// ForwardMessages) is interrupted by a DC migration and cannot be safely
	// retried because it may have already been applied.
	ErrMigrationUnsafe = errors.New("client: dc migration unsafe for non-idempotent request")
	// ErrMigrationUnknown is returned when the server requests migration to an
	// unknown data center that the client has no configuration for.
	ErrMigrationUnknown = errors.New("client: dc migration to unknown dc")
)

// Client initialization and authentication errors.
//
// Example:
//
//	client, err := telegram.NewClient(0, "", nil)
//	if errors.Is(err, telegram.ErrAPIIDRequired) {
//		log.Fatal("apiID is required")
//	}
var (
	// ErrAPIIDRequired is returned by NewClient when the apiID parameter is zero.
	ErrAPIIDRequired = errors.New("telegram: apiID is required")
	// ErrAPIHashRequired is returned by NewClient when the apiHash parameter is empty.
	ErrAPIHashRequired = errors.New("telegram: apiHash is required")
	// ErrNoStorage is returned during connection when no storage backend is
	// configured and InMemory is not enabled. Set Storage, SessionName, or
	// enable InMemory in the Config.
	ErrNoStorage = errors.New("telegram: no storage configured; set Storage, SessionName, or enable InMemory")
	// ErrAlreadyAuthed is returned when attempting to authenticate a client that
	// is already authorized (e.g. calling SignIn when already logged in).
	ErrAlreadyAuthed = errors.New("telegram: already authenticated")
	// Err2FARequired is returned when SignCheckPassword is needed instead of
	// SignIn because the account has two-factor authentication enabled.
	Err2FARequired = errors.New("2FA is enabled: use CheckPassword instead")
	// ErrSignUpRequired is returned when the phone number is registered but
	// requires completing the sign-up flow via the SignUp method.
	ErrSignUpRequired = errors.New("telegram: sign up required: use SignUp method")
)

// Context errors returned by context-based helper methods on Message, Chat,
// CallbackQuery, etc.
//
// These indicate missing state (e.g. calling a method on a context that was
// not created by a client). Use errors.Is to match specific conditions.
//
// Example:
//
//	_, err := msg.Reply("hello")
//	if errors.Is(err, telegram.ErrContextNoClient) {
//		log.Println("message was not created by a client")
//	}
var (
	// ErrContextNoClient is returned when a bound method is called on a type
	// that was not created by a client (e.g. manually constructed).
	ErrContextNoClient = errors.New("context: no client")
	// ErrContextNoChat is returned when a context method requires a chat but
	// none is available.
	ErrContextNoChat = errors.New("context: no chat available")
	// ErrContextNoMessage is returned when a context method requires a message
	// but none is available.
	ErrContextNoMessage = errors.New("context: no message")
	// ErrContextNoMedia is returned when a download or media operation is
	// attempted on a message that has no attached media.
	ErrContextNoMedia = errors.New("context: message has no media")
	// ErrContextNoCallback is returned when a callback operation is attempted
	// but no callback query is present in the context.
	ErrContextNoCallback = errors.New("context: no callback query")
	// ErrContextEditInline is returned when trying to edit an inline message
	// by its numeric message ID, which is not supported. Use the inline
	// message ID instead.
	ErrContextEditInline = errors.New("context: cannot edit inline message by id")
)

// Media processing errors.
//
// These are returned when parsing or validating photo and document objects
// from Telegram API responses.
//
// Example:
//
//	photo, _, err := telegram.GetPhotoFileLocation(media)
//	if errors.Is(err, telegram.ErrPhotoNoSizes) {
//		log.Println("photo has no downloadable sizes")
//	}
var (
	// ErrMediaNil is returned when a nil media object is passed to a media
	// processing function.
	ErrMediaNil = errors.New("media: nil media")
	// ErrPhotoNoData is returned when a Photo object has no inner PhotoData
	// field populated.
	ErrPhotoNoData = errors.New("media: photo has no photo data")
	// ErrPhotoNoSizes is returned when a Photo has no downloadable size
	// variants available.
	ErrPhotoNoSizes = errors.New("media: no photo sizes available")
	// ErrPhotoNil is returned when a photo-size reference within a larger
	// media object is unexpectedly nil.
	ErrPhotoNil = errors.New("media: photo is nil")
	// ErrDocumentNil is returned when a document reference within a larger
	// media object is unexpectedly nil.
	ErrDocumentNil = errors.New("media: document is nil")
	// ErrNoDownloadableMedia is returned when attempting to download media
	// from a message that contains no downloadable content.
	ErrNoDownloadableMedia = errors.New("message has no downloadable media")
)

// Message retrieval errors.
//
// Returned when a send or edit operation completes server-side but the
// resulting message cannot be extracted from the Telegram Updates response.
var (
	// ErrNoMessage is returned when a method cannot find the expected message.
	ErrNoMessage = errors.New("message not found")
	// ErrNoMessageUpdates is returned when the API returns an UpdatesTL
	// response but no message can be extracted from it.
	ErrNoMessageUpdates = errors.New("no message found in UpdatesTL")
	// ErrNoMessageShort is returned when the API returns an UpdateShort
	// response but no message can be extracted from it.
	ErrNoMessageShort = errors.New("no message found in UpdateShort")
)

// Story-related validation errors.
var (
	// ErrMediaRequired is returned when creating or editing a story without
	// providing media input.
	ErrMediaRequired = errors.New("media is required")
	// ErrCaptionRequired is returned when a story operation requires a caption
	// but none was provided.
	ErrCaptionRequired = errors.New("caption is required")
	// ErrStoryIDsRequired is returned when a story method requires one or more
	// story IDs but the input slice is empty.
	ErrStoryIDsRequired = errors.New("story ids are required")
	// ErrStoryNotInUpdates is returned when the expected story object is not
	// found in the Updates container returned by the API.
	ErrStoryNotInUpdates = errors.New("no story found in Updates")
	// ErrStoryNotInShort is returned when the expected story object is not
	// found in the UpdateShort container returned by the API.
	ErrStoryNotInShort = errors.New("no story found in UpdateShort")
	// ErrStorySchema is returned when a story feature is not supported by the
	// current TL schema version.
	ErrStorySchema = errors.New("not available in current tl schema")
)

// Upload, download, and input validation errors.
var (
	// ErrNoUserInResponse is returned when an API call that should return a
	// user object receives a response without a user.
	ErrNoUserInResponse = errors.New("no user in response")
	// ErrInputFileEmpty is returned when attempting to upload a file with no
	// content (empty bytes or nil reader).
	ErrInputFileEmpty = errors.New("input_file: empty input")
	// ErrUploadNoData is returned when a streaming upload completes but
	// produced no data chunks.
	ErrUploadNoData = errors.New("upload: streamed upload produced no data")
)

// Proxy and connection transport errors.
var (
	// ErrMTProxySecretRequired is returned when configuring an MTProxy
	// connection without providing the required secret parameter.
	ErrMTProxySecretRequired = errors.New("telegram: mtproxy: secret is required")
	// ErrProxyParamsRequired is returned when parsing a tg://proxy URI that
	// is missing the server, port, or secret fields.
	ErrProxyParamsRequired = errors.New("telegram: tg://proxy requires server, port, and secret")
	// ErrSocks4Domain is returned when a SOCKS4 proxy receives a domain name
	// instead of an IP address, which SOCKS4 does not support.
	ErrSocks4Domain = errors.New("socks4: domain not supported, need ip")
	// ErrSocks4IPv6 is returned when a SOCKS4 proxy receives an IPv6 address,
	// which SOCKS4 does not support. Use SOCKS5 for IPv6.
	ErrSocks4IPv6 = errors.New("socks4: ipv6 not supported")
	// ErrProxyResponseTooLarge is returned when the proxy handshake response
	// exceeds the expected maximum size.
	ErrProxyResponseTooLarge = errors.New("response too large")
)

// Business, secret chat, forum, and group call errors.
var (
	// ErrNoSecretChats is returned when trying to accept or read a secret chat
	// but no pending secret chats are available.
	ErrNoSecretChats = errors.New("telegram: no pending secret chats")
	// ErrBusinessConnIDRequired is returned when a business connection operation
	// is attempted without providing the required connection ID.
	ErrBusinessConnIDRequired = errors.New("business: connection id is required")
	// ErrNoBusinessConnection is returned when the expected business connection
	// cannot be found in the update received from Telegram.
	ErrNoBusinessConnection = errors.New("business: no business connection found in update")
	// ErrForumTopicNotFound is returned when creating or editing a forum topic
	// but the topic cannot be extracted from the API response.
	ErrForumTopicNotFound = errors.New("forum topic not found in updates")
	// ErrCallNoChannel is returned when the group call reader starts without a
	// selected channel.
	ErrCallNoChannel = errors.New("call reader: no channel selected")
	// ErrCallChannelNil is returned when the group call reader receives a nil
	// channel reference.
	ErrCallChannelNil = errors.New("call reader: channel is nil")
	// ErrCallCDNNotSupported is returned when the group call stream redirects
	// to a CDN endpoint, which is not yet supported.
	ErrCallCDNNotSupported = errors.New("call reader: cdn redirect not supported")
	// ErrCallNoStreams is returned when a group call has no available stream
	// channels to read from.
	ErrCallNoStreams = errors.New("call reader: no stream channels available")
	// ErrCallNotFound is returned when creating a phone call but the call
	// object cannot be extracted from the API response.
	ErrCallNotFound = errors.New("create call: phone call not found in updates")
	// ErrPaymentsCredentialsRequired is returned when sending a payment form
	// without providing the required payment credentials.
	ErrPaymentsCredentialsRequired = errors.New("send payment form: credentials required")
	// ErrPrivacySettingsRequired is returned when setting global privacy
	// settings but the settings parameter is nil.
	ErrPrivacySettingsRequired = errors.New("set global privacy settings: settings is required")
)

// Chat management errors.
//
// These are returned by methods in chats.go that manage channels, groups,
// members, and invite links.
var (
	// ErrGetChatNotChat is returned when GetChat is called with a peer that
	// resolves to a user instead of a chat or channel.
	ErrGetChatNotChat = errors.New("GetChat: peer is a user, not a chat")
	// ErrJoinNoInfo is returned when joining a chat succeeds but the chat
	// information cannot be extracted from the response.
	ErrJoinNoInfo = errors.New("joined chat but could not extract chat info")
	// ErrChannelNoInfo is returned when creating a channel succeeds but the
	// channel information cannot be extracted from the response.
	ErrChannelNoInfo = errors.New("created channel but could not extract info")
	// ErrBanSupergroupOnly is returned when BanChatMember is called on a
	// peer that is not a channel or supergroup.
	ErrBanSupergroupOnly = errors.New("BanChatMember: only channels/supergroups are supported")
	// ErrUnbanSupergroupOnly is returned when UnbanChatMember is called on a
	// peer that is not a channel or supergroup.
	ErrUnbanSupergroupOnly = errors.New("UnbanChatMember: only channels/supergroups are supported")
	// ErrRestrictSupergroupOnly is returned when RestrictChatMember is called
	// on a peer that is not a channel or supergroup.
	ErrRestrictSupergroupOnly = errors.New("RestrictChatMember: only channels/supergroups supported")
	// ErrGroupNoInfo is returned when creating a group succeeds but the group
	// information cannot be extracted from the response.
	ErrGroupNoInfo = errors.New("created group but could not extract info")
	// ErrJoinRequiresInvite is returned when attempting to join a chat without
	// providing either a username or an invite hash.
	ErrJoinRequiresInvite = errors.New("join requires a username or invite hash")
)

// Context message operation errors.
//
// These are returned by the convenience methods on Message (Reply, Edit,
// Forward, Copy, React, etc.) when the required preconditions are not met.
var (
	// ErrContextNoReply is returned when calling Reply on a nil message context.
	ErrContextNoReply = errors.New("context: no message to reply to")
	// ErrContextNoEdit is returned when calling Edit on a nil message context.
	ErrContextNoEdit = errors.New("context: no message to edit")
	// ErrContextNoForward is returned when calling Forward on a nil message context.
	ErrContextNoForward = errors.New("context: no message to forward")
	// ErrContextNoForwardResult is returned when Forward completes but the
	// forwarded message cannot be found in the API response.
	ErrContextNoForwardResult = errors.New("context: no forwarded message returned")
	// ErrContextNoCopy is returned when calling Copy on a nil message context.
	ErrContextNoCopy = errors.New("context: no message to copy")
	// ErrContextNoReact is returned when calling React on a nil message context.
	ErrContextNoReact = errors.New("context: no message to react to")
	// ErrContextNotMediaGroup is returned when attempting to retrieve or copy a
	// media group on a message that is not part of an album.
	ErrContextNotMediaGroup = errors.New("context: message is not part of a media group")
)

// ReconnectError indicates that reconnection attempts were exhausted.
//
// Example:
//
//	err := client.Connect(ctx)
//	if err != nil {
//		var reconnErr *telegram.ReconnectError
//		if errors.As(err, &reconnErr) {
//			fmt.Printf("gave up after %d attempts: %v\n", reconnErr.Attempts, reconnErr.Err)
//		}
//	}
type ReconnectError struct {
	Attempts int
	Err      error
}

func (e *ReconnectError) Error() string {
	return fmt.Sprintf("client: reconnect failed after %d attempts: %v", e.Attempts, e.Err)
}

func (e *ReconnectError) Unwrap() error { return e.Err }

// MigrationError indicates a failure to migrate the connection to a different DC.
//
// Example:
//
//	_, err := client.SendMessage(ctx, peer, "hello")
//	if err != nil {
//		var migErr *telegram.MigrationError
//		if errors.As(err, &migErr) {
//			fmt.Printf("migration to DC %d failed: %v\n", migErr.TargetDC, migErr.Err)
//		}
//	}
type MigrationError struct {
	TargetDC int
	Err      error
}

func (e *MigrationError) Error() string {
	return fmt.Sprintf("client: dc migration to dc %d failed: %v", e.TargetDC, e.Err)
}

func (e *MigrationError) Unwrap() error { return e.Err }

// UnsafeMigrationError indicates a non-idempotent request was interrupted by a DC migration.
// The request is not automatically retried because it may have already been applied.
//
// Example:
//
//	_, err := client.ForwardMessages(ctx, peer, msgIDs)
//	if err != nil {
//		var unsafeErr *telegram.UnsafeMigrationError
//		if errors.As(err, &unsafeErr) {
//			fmt.Printf("non-idempotent %q interrupted by migration to DC %d\n",
//				unsafeErr.Method, unsafeErr.TargetDC)
//		}
//	}
type UnsafeMigrationError struct {
	TargetDC int
	Method   string
}

func (e *UnsafeMigrationError) Error() string {
	return fmt.Sprintf("client: refusing to retry non-idempotent %q after dc migration to dc %d", e.Method, e.TargetDC)
}
