package types

// TopChatCategory selects the category of peers used when requesting the most
// frequently interacted-with chats from the server. Pass the desired category
// to getTopPeers to retrieve the corresponding top list.
type TopChatCategory int

const (
	// TopChatCategoryUsers selects top peers for private message conversations.
	TopChatCategoryUsers TopChatCategory = iota
	// TopChatCategoryGroups selects top peers for group chats.
	TopChatCategoryGroups
	// TopChatCategoryChannels selects top peers for channels.
	TopChatCategoryChannels
	// TopChatCategoryBots selects top peers for bot chats.
	TopChatCategoryBots
	// TopChatCategoryBotsInline selects top peers for inline bot usage.
	TopChatCategoryBotsInline
	// TopChatCategoryPhoneCalls selects top peers for phone calls.
	TopChatCategoryPhoneCalls
	// TopChatCategoryForwardUsers selects top peers that the user has forwarded
	// messages to in private chats.
	TopChatCategoryForwardUsers
	// TopChatCategoryForwardChats selects top peers that the user has forwarded
	// messages to in groups or channels.
	TopChatCategoryForwardChats
	// TopChatCategoryBotsApp selects top mini-app bots opened by the user.
	TopChatCategoryBotsApp
	// TopChatCategoryBotsGuestChat selects top bots used in guest chat mode.
	TopChatCategoryBotsGuestChat
	// TopChatCategoryPeers selects top peers across all conversation types.
	TopChatCategoryPeers
)
