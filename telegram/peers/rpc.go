package peers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mtgo-labs/mtgo/internal/storage"

	"github.com/mtgo-labs/mtgo/internal/peerid"
	"github.com/mtgo-labs/mtgo/tg"
)

// InputPeerByUsername resolves a Telegram username (with or without the
// leading "@") to an input peer. The local username cache is checked first;
// on a miss, contacts.resolveUsername is invoked under a single-flight
// coalescer and every entity in the response is cached.
func (m *Manager) InputPeerByUsername(ctx context.Context, username string) (tg.InputPeerClass, error) {
	username = strings.TrimPrefix(username, "@")
	m.debugf("ResolveUsername @%s", username)

	if p, ok := m.cachedByUsername(username); ok {
		m.debugf("ResolveUsername cache hit @%s", username)
		return p, nil
	}

	result, err := m.ResolveUsernameFull(ctx, username)
	if err != nil {
		return nil, err
	}
	inputPeer, err := PeerToInputPeer(result.Peer, result.Users, result.Chats)
	if err != nil {
		return nil, fmt.Errorf("%w: @%s: %w", ErrNotFound, username, err)
	}
	return inputPeer, nil
}

// ResolveUsernameFull resolves a username through the coalesced cascade and
// returns the complete contacts.resolveUsername response (including the
// users and chats slices) after caching every entity. Use it where callers
// need rich entity data, not just the input peer.
func (m *Manager) ResolveUsernameFull(ctx context.Context, username string) (*tg.ContactsResolvedPeer, error) {
	username = strings.TrimPrefix(username, "@")
	if p, ok := m.cachedByUsername(username); ok {
		return resolvedFromCached(p), nil
	}
	result, err := coalesce(m, ctx, "username:"+username, func() (*tg.ContactsResolvedPeer, error) {
		// Double-check cache inside the coalescer — another goroutine
		// may have resolved it while we were waiting for the lock.
		if p, ok := m.cachedByUsername(username); ok {
			return resolvedFromCached(p), nil
		}
		result, err := m.invoker().ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: username,
		})
		if err != nil {
			if isNotFoundRPC(err) {
				return nil, &NotFoundError{Ref: "@" + username, Cause: err}
			}
			// Transient failure (flood, network, auth): surface it unmasked.
			return nil, fmt.Errorf("resolve @%s: %w", username, err)
		}
		m.ingestResolved(result)
		return result, nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// resolvedFromCached wraps a cached input peer as a minimal resolve result.
// Rich entity data is unavailable on cache hits; callers that need it should
// use the store.
func resolvedFromCached(p tg.InputPeerClass) *tg.ContactsResolvedPeer {
	res := &tg.ContactsResolvedPeer{Peer: peerClassFromInput(p)}
	switch v := p.(type) {
	case *tg.InputPeerUser:
		res.Users = []tg.UserClass{&tg.User{ID: v.UserID, AccessHash: v.AccessHash}}
	case *tg.InputPeerChat:
		res.Chats = []tg.ChatClass{&tg.Chat{ID: v.ChatID}}
	case *tg.InputPeerChannel:
		res.Chats = []tg.ChatClass{&tg.Channel{ID: v.ChannelID, AccessHash: v.AccessHash}}
	}
	return res
}

func peerClassFromInput(p tg.InputPeerClass) tg.PeerClass {
	switch v := p.(type) {
	case *tg.InputPeerUser:
		return &tg.PeerUser{UserID: v.UserID}
	case *tg.InputPeerChat:
		return &tg.PeerChat{ChatID: v.ChatID}
	case *tg.InputPeerChannel:
		return &tg.PeerChannel{ChannelID: v.ChannelID}
	default:
		return nil
	}
}

// InputPeerByPhone resolves a phone number to an input peer. The number is
// normalized automatically (leading "+" and "00" prefixes are stripped) and
// contacts.resolvePhone is invoked under a single-flight coalescer.
func (m *Manager) InputPeerByPhone(ctx context.Context, phone string) (tg.InputPeerClass, error) {
	phone = NormalizePhone(phone)
	m.debugf("ResolvePhone")
	if p, ok := m.cachedByPhone(phone); ok {
		return p, nil
	}
	if ps := phoneStoreFrom(m.store()); ps != nil {
		if entry, err := ps.GetPeerByPhone(phone); err == nil && entry != nil && entry.Type == storage.PeerTypeUser {
			peer := &tg.InputPeerUser{UserID: entry.ID, AccessHash: entry.AccessHash}
			m.Cache(entry.ID, peer)
			m.CachePhone(phone, entry.ID)
			return peer, nil
		}
	}
	return coalesce(m, ctx, "phone:"+phone, func() (tg.InputPeerClass, error) {
		result, err := m.invoker().ContactsResolvePhone(ctx, &tg.ContactsResolvePhoneRequest{
			Phone: phone,
		})
		if err != nil {
			if isNotFoundRPC(err) {
				return nil, &NotFoundError{Ref: "phone " + phone, Cause: err}
			}
			return nil, fmt.Errorf("resolve phone %s: %w", phone, err)
		}
		m.ingestResolved(result)
		inputPeer, err := PeerToInputPeer(result.Peer, result.Users, result.Chats)
		if err != nil {
			return nil, fmt.Errorf("%w: phone %s: %w", ErrNotFound, phone, err)
		}
		return inputPeer, nil
	})
}

// numericForBot resolves a numeric ID when authorized as a bot: bare
// negated IDs become input chats directly, channels are fetched with a zero
// hash via channels.getChannels, and users via users.getUsers.
func (m *Manager) numericForBot(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	if peer, ok := inputPeerFromBareChatID(id); ok {
		return peer, nil
	}
	if raw, ok := peerid.UnmarkChannel(id); ok {
		return m.botChannelAccessHash(ctx, raw)
	}
	if id > 0 {
		peer, err := m.EnsureUsable(ctx, &tg.InputPeerUser{UserID: id})
		if err != nil {
			return nil, fmt.Errorf("could not resolve chat: %w", err)
		}
		return peer, nil
	}
	return nil, fmt.Errorf("could not resolve chat: %w", ErrNotFound)
}

func (m *Manager) botUserAccessHash(ctx context.Context, userID int64) (tg.InputPeerClass, error) {
	result, err := m.invoker().UsersGetUsers(ctx, &tg.UsersGetUsersRequest{
		ID: []tg.InputUserClass{
			&tg.InputUser{UserID: userID, AccessHash: 0},
		},
	})
	if err != nil {
		if isNotFoundRPC(err) {
			return nil, fmt.Errorf("%w: get user %d: %w", ErrNotFound, userID, err)
		}
		return nil, fmt.Errorf("get user %d: %w", userID, err)
	}
	users := usersFromUsersGetUsers(result)
	m.Ingest(users, nil)
	for _, u := range users {
		user, ok := u.(*tg.User)
		if ok && user.ID == userID && user.AccessHash != 0 {
			peer := &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}
			m.Cache(user.ID, peer)
			return peer, nil
		}
	}
	return nil, ErrNotFound
}

func (m *Manager) botChannelAccessHash(ctx context.Context, channelID int64) (tg.InputPeerClass, error) {
	result, err := m.invoker().ChannelsGetChannels(ctx, &tg.ChannelsGetChannelsRequest{
		ID: []tg.InputChannelClass{
			&tg.InputChannel{ChannelID: channelID, AccessHash: 0},
		},
	})
	if err != nil {
		if isNotFoundRPC(err) {
			return nil, fmt.Errorf("%w: get channel %d: %w", ErrNotFound, channelID, err)
		}
		return nil, fmt.Errorf("get channel %d: %w", channelID, err)
	}
	chats := chatsFromChatsClass(result)
	m.Ingest(nil, chats)
	for _, ch := range chats {
		channel, ok := ch.(*tg.Channel)
		if ok && channel.ID == channelID && channel.AccessHash != 0 {
			peer := &tg.InputPeerChannel{ChannelID: channel.ID, AccessHash: channel.AccessHash}
			m.Cache(channel.ID, peer)
			return peer, nil
		}
	}
	return nil, ErrNotFound
}

// numericForAccount resolves a numeric ID when authorized as a user account:
// bare negated IDs become input chats, otherwise up to 20 pages of dialogs
// are preloaded to find the peer's access hash, with a username-based
// resolve as the last resort for min entities.
func (m *Manager) numericForAccount(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	if peer, ok := inputPeerFromBareChatID(id); ok {
		return peer, nil
	}
	if preloadErr := m.preloadDialogPeer(ctx, id); preloadErr != nil && preloadErr != ErrNotFound {
		return nil, preloadErr
	}
	peer, err := m.Cached(id)
	if err != nil {
		return nil, fmt.Errorf("could not resolve chat: %w", ErrNotFound)
	}
	// If dialog preload couldn't find a full hash, try cached-credential
	// re-resolution (username, then phone) before returning a zero-hash
	// peer.
	if !hasAccessHash(peer) {
		if resolved, err := m.peerByUsername(ctx, id); err == nil {
			return resolved, nil
		}
		if phone := m.LookupPhone(id); phone != "" {
			if resolved, err := m.InputPeerByPhone(ctx, phone); err == nil {
				return resolved, nil
			}
		}
		if resolved, ok := m.anchorInputPeer(id); ok {
			return resolved, nil
		}
	}
	return peer, nil
}

// peerByUsername attempts to resolve a peer via its cached username when the
// access hash is unknown (min entity). This is the fallback path after
// dialog preload fails to find the peer.
func (m *Manager) peerByUsername(ctx context.Context, id int64) (tg.InputPeerClass, error) {
	username := m.LookupUsername(id)
	if username == "" {
		return nil, ErrNotFound
	}
	return m.InputPeerByUsername(ctx, username)
}

func (m *Manager) preloadDialogPeer(ctx context.Context, id int64) error {
	const (
		dialogPageLimit = 100
		maxDialogPages  = 20
	)

	offsetPeer := tg.InputPeerClass(&tg.InputPeerEmpty{})
	var offsetDate int32
	var offsetID int32

	for page := 0; page < maxDialogPages; page++ {
		result, err := m.invoker().MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
			OffsetDate: offsetDate,
			OffsetID:   offsetID,
			OffsetPeer: offsetPeer,
			Limit:      dialogPageLimit,
		})
		if err != nil {
			return err
		}

		dialogs, messages, users, chats, ok := unpackDialogs(result)
		if !ok {
			return ErrNotFound
		}

		m.Ingest(users, chats)
		if _, err := m.Cached(id); err == nil {
			return nil
		}
		if len(dialogs) == 0 {
			break
		}

		nextPeer, nextID, nextDate, ok := m.nextDialogOffset(dialogs, messages)
		if !ok {
			break
		}
		offsetPeer = nextPeer
		offsetID = nextID
		offsetDate = nextDate
	}

	return ErrNotFound
}

func (m *Manager) nextDialogOffset(dialogs []tg.DialogClass, messages []tg.MessageClass) (tg.InputPeerClass, int32, int32, bool) {
	last, ok := dialogs[len(dialogs)-1].(*tg.Dialog)
	if !ok || last.Peer == nil {
		return nil, 0, 0, false
	}
	peerID, ok := peerid.RawFromPeer(last.Peer)
	if !ok {
		return nil, 0, 0, false
	}
	peer, err := m.Cached(peerID)
	if err != nil {
		return nil, 0, 0, false
	}
	return peer, last.TopMessage, messageDate(messages, last.TopMessage), true
}

func messageDate(messages []tg.MessageClass, id int32) int32 {
	for _, msg := range messages {
		switch m := msg.(type) {
		case *tg.Message:
			if m.ID == id {
				return m.Date
			}
		case *tg.MessageService:
			if m.ID == id {
				return m.Date
			}
		}
	}
	return 0
}

func unpackDialogs(result tg.DialogsClass) ([]tg.DialogClass, []tg.MessageClass, []tg.UserClass, []tg.ChatClass, bool) {
	switch v := result.(type) {
	case *tg.MessagesDialogs:
		return v.Dialogs, v.Messages, v.Users, v.Chats, true
	case *tg.MessagesDialogsSlice:
		return v.Dialogs, v.Messages, v.Users, v.Chats, true
	case *tg.MessagesDialogsNotModified:
		return nil, nil, nil, nil, false
	default:
		return nil, nil, nil, nil, false
	}
}

func chatsFromChatsClass(result tg.TLObject) []tg.ChatClass {
	switch v := result.(type) {
	case *tg.MessagesChats:
		return v.Chats
	case *tg.MessagesChatsSlice:
		return v.Chats
	default:
		return nil
	}
}

func usersFromUsersGetUsers(result tg.TLObject) []tg.UserClass {
	vector, ok := result.(*tg.GenericVector)
	if !ok {
		return nil
	}
	users := make([]tg.UserClass, 0, len(vector.Items))
	for _, item := range vector.Items {
		if user, ok := item.(tg.UserClass); ok {
			users = append(users, user)
		}
	}
	return users
}

// NormalizePhone strips whitespace and the leading "+"/"00" prefixes
// from a phone number.
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimPrefix(phone, "00")
	return phone
}

// PeerToInputPeer converts a high-level tg.PeerClass (as returned by
// Telegram updates or API responses) into a tg.InputPeerClass, using the
// users and chats slices for access hashes and metadata.
func PeerToInputPeer(peer tg.PeerClass, users []tg.UserClass, chats []tg.ChatClass) (tg.InputPeerClass, error) {
	userMap := makeUserMap(users)
	chatMap := makeChatMap(chats)
	switch p := peer.(type) {
	case *tg.PeerUser:
		user, ok := userMap[p.UserID]
		if !ok {
			if p.UserID == 0 {
				return &tg.InputPeerSelf{}, nil
			}
			return nil, fmt.Errorf("%w: user %d not found in resolved peers", ErrNotFound, p.UserID)
		}
		if user.AccessHash != 0 {
			return &tg.InputPeerUser{UserID: user.ID, AccessHash: user.AccessHash}, nil
		}
		return nil, fmt.Errorf("%w: user %d carries no usable access hash", ErrNotFound, user.ID)
	case *tg.PeerChat:
		if chat, ok := chatMap[p.ChatID]; ok {
			return &tg.InputPeerChat{ChatID: chat.id}, nil
		}
		return nil, fmt.Errorf("%w: chat %d not found in resolved peers", ErrNotFound, p.ChatID)
	case *tg.PeerChannel:
		ch, ok := chatMap[p.ChannelID]
		if !ok {
			return nil, fmt.Errorf("%w: channel %d not found in resolved peers", ErrNotFound, p.ChannelID)
		}
		return &tg.InputPeerChannel{ChannelID: ch.id, AccessHash: ch.accessHash}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported peer type %T", ErrNotFound, peer)
	}
}

// InputPeerToUser converts an input peer into an input user.
func InputPeerToUser(peer tg.InputPeerClass) (tg.InputUserClass, error) {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return &tg.InputUser{UserID: p.UserID, AccessHash: p.AccessHash}, nil
	case *tg.InputPeerSelf:
		return &tg.InputUserSelf{}, nil
	default:
		return nil, NotUserError{Peer: peer}
	}
}

// InputPeerToChannel converts an input peer into an input channel.
func InputPeerToChannel(peer tg.InputPeerClass) (tg.InputChannelClass, error) {
	switch p := peer.(type) {
	case *tg.InputPeerChannel:
		return &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}, nil
	case *tg.InputPeerSelf:
		return &tg.InputChannelEmpty{}, nil
	default:
		return nil, NotChannelError{Peer: peer}
	}
}

func makeUserMap(users []tg.UserClass) map[int64]*tg.User {
	m := make(map[int64]*tg.User, len(users))
	for _, u := range users {
		user, ok := u.(*tg.User)
		if ok && user.ID != 0 {
			m[user.ID] = user
		}
	}
	return m
}

type chatInfo struct {
	id         int64
	accessHash int64
}

func makeChatMap(chats []tg.ChatClass) map[int64]chatInfo {
	m := make(map[int64]chatInfo, len(chats))
	for _, c := range chats {
		switch v := c.(type) {
		case *tg.Chat:
			m[v.ID] = chatInfo{id: v.ID}
		case *tg.Channel:
			m[v.ID] = chatInfo{id: v.ID, accessHash: v.AccessHash}
		}
	}
	return m
}
