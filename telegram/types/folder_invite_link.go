package types

import "github.com/mtgo-labs/mtgo/tg"

// FolderInviteLink represents an invite link for a chat folder, including the
// link URL, display name, and associated chat IDs.
//
// Example:
//
//	link := types.ParseFolderInviteLink(rawInvite)
//	fmt.Printf("Folder invite: %s (%d chats)\n", link.Name, len(link.ChatIDs))
type FolderInviteLink struct {
	InviteLink string
	Name       string
	ChatIDs    []int64
}

// ParseFolderInviteLink converts a TL ExportedChatlistInvite into a FolderInviteLink.
// Returns nil if raw is nil.
//
// Example:
//
//	link := types.ParseFolderInviteLink(rawInvite)
//	if link != nil {
//	    fmt.Println("Invite:", link.InviteLink)
//	}
func ParseFolderInviteLink(raw *tg.ExportedChatlistInvite) *FolderInviteLink {
	if raw == nil {
		return nil
	}
	link := &FolderInviteLink{
		InviteLink: raw.URL,
		Name:       raw.Title,
	}
	for _, peer := range raw.Peers {
		link.ChatIDs = append(link.ChatIDs, getPeerID(peer))
	}
	return link
}
