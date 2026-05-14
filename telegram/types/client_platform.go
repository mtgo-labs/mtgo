package types

// ClientPlatform identifies the platform of a Telegram client.
type ClientPlatform string

// Client platform constants representing the different Telegram client platforms.
const (
	ClientPlatformAndroid ClientPlatform = "android"
	ClientPlatformIOS     ClientPlatform = "ios"
	ClientPlatformWP      ClientPlatform = "wp"
	ClientPlatformBB      ClientPlatform = "bb"
	ClientPlatformDesktop ClientPlatform = "desktop"
	ClientPlatformWeb     ClientPlatform = "web"
	ClientPlatformUBP     ClientPlatform = "ubp"
	ClientPlatformOther   ClientPlatform = "other"
)

// String returns the string representation of the ClientPlatform.
func (c ClientPlatform) String() string { return string(c) }
