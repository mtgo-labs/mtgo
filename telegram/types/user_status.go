package types

// UserStatus represents the online presence status of a Telegram user.
type UserStatus string

// UserStatus constants enumerate the possible user presence states.
const (
	UserStatusOnline    UserStatus = "online"
	UserStatusOffline   UserStatus = "offline"
	UserStatusRecently  UserStatus = "recently"
	UserStatusLastWeek  UserStatus = "last_week"
	UserStatusLastMonth UserStatus = "last_month"
	UserStatusLongAgo   UserStatus = "long_ago"
)

// String returns the string representation of the UserStatus.
func (u UserStatus) String() string { return string(u) }
