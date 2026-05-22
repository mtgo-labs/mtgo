package types

import "github.com/mtgo-labs/mtgo/tg"

// PeerMap stores indexed lookups for users, chats, and channels by their IDs.
// It is used to resolve peer references in updates and messages without
// additional API calls.
//
// Example:
//
//	pm := types.NewPeerMap(users, chats, channels)
//	if user, ok := pm.Users[userID]; ok {
//	    fmt.Println("Found user:", user.FirstName)
//	}
type PeerMap struct {
	// Users maps user IDs to their TL user objects.
	Users map[int64]*tg.User
	// Chats maps basic group IDs to their TL chat objects.
	Chats map[int64]*tg.Chat
	// Channels maps channel/supergroup IDs to their TL channel objects.
	Channels map[int64]*tg.Channel
}

// NewPeerMap creates a PeerMap from typed slices of users, chats, and channels.
// Each slice is indexed by ID for O(1) lookups. Nil entries are skipped.
//
// Example:
//
//	pm := types.NewPeerMap(update.Users, update.Chats, update.Channels)
//	msg := types.ParseMessage(update.Message, pm)
func NewPeerMap(users []*tg.User, chats []*tg.Chat, channels []*tg.Channel) *PeerMap {
	pm := &PeerMap{
		Users:    make(map[int64]*tg.User, len(users)),
		Chats:    make(map[int64]*tg.Chat, len(chats)),
		Channels: make(map[int64]*tg.Channel, len(channels)),
	}
	for _, u := range users {
		if u != nil {
			pm.Users[u.ID] = u
		}
	}
	for _, c := range chats {
		if c != nil {
			pm.Chats[c.ID] = c
		}
	}
	for _, ch := range channels {
		if ch != nil {
			pm.Channels[ch.ID] = ch
		}
	}
	return pm
}

// NewPeerMapFromClasses creates a PeerMap from interface slices of UserClass and ChatClass,
// type-asserting each element to its concrete type.
// Use this when the caller has heterogeneous slices (e.g. from TL updates) rather
// than pre-separated typed slices.
func NewPeerMapFromClasses(users []tg.UserClass, chats []tg.ChatClass) *PeerMap {
	pm := &PeerMap{
		Users:    make(map[int64]*tg.User, len(users)),
		Chats:    make(map[int64]*tg.Chat),
		Channels: make(map[int64]*tg.Channel),
	}
	for _, u := range users {
		if v, ok := u.(*tg.User); ok && v != nil {
			pm.Users[v.ID] = v
		}
	}
	for _, c := range chats {
		switch v := c.(type) {
		case *tg.Chat:
			if v != nil {
				pm.Chats[v.ID] = v
			}
		case *tg.Channel:
			if v != nil {
				pm.Channels[v.ID] = v
			}
		}
	}
	return pm
}

// GetPeerID returns the raw peer ID from a PeerClass value.
// For users it returns the positive user ID; for chats and channels it returns
// a negative ID to distinguish them from user IDs.
// Returns 0 if peer is nil or unrecognized.
//
// Example:
//
//	id := types.GetPeerID(peer)
//	if id > 0 {
//	    fmt.Println("User peer:", id)
//	} else {
//	    fmt.Println("Group/channel peer:", id)
//	}
func GetPeerID(peer tg.PeerClass) int64 {
	if peer == nil {
		return 0
	}
	switch p := peer.(type) {
	case *tg.PeerUser:
		return p.UserID
	case *tg.PeerChat:
		return -p.ChatID
	case *tg.PeerChannel:
		return channelChatID(p.ChannelID)
	}
	return 0
}

func getUserFromPM(pm *PeerMap, id int64) *User {
	if pm == nil || pm.Users == nil {
		return &User{ID: id}
	}
	if u, ok := pm.Users[id]; ok && u != nil {
		return ParseUser(u)
	}
	return &User{ID: id}
}
