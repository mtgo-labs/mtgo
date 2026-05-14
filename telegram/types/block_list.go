package types

// BlockList represents the Telegram block list category a user belongs to.
type BlockList string

// Block list categories.
const (
	// BlockListMain blocks the user from the main block list.
	BlockListMain BlockList = "main"
	// BlockListStories blocks the user from viewing stories.
	BlockListStories BlockList = "stories"
)

// String returns the string representation of the BlockList.
func (b BlockList) String() string { return string(b) }
