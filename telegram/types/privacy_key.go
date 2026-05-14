package types

// PrivacyKey represents a Telegram user privacy setting category. Each key
// controls who can access a specific piece of user information or feature.
// Pass a PrivacyKey to account.getPrivacy or account.setPrivacy to read or
// modify the corresponding privacy rules.
type PrivacyKey string

const (
	// PrivacyKeyAbout controls who can view the user's bio text.
	PrivacyKeyAbout PrivacyKey = "about"
	// PrivacyKeyAddedByPhone controls who can find the user by their phone
	// number in global search.
	PrivacyKeyAddedByPhone PrivacyKey = "added_by_phone"
	// PrivacyKeyBirthday controls who can view the user's birthday.
	PrivacyKeyBirthday PrivacyKey = "birthday"
	// PrivacyKeyChatInvite controls who can invite the user to chats and
	// channels.
	PrivacyKeyChatInvite PrivacyKey = "chat_invite"
	// PrivacyKeyForwards controls who can view the original author of
	// forwarded messages from this user.
	PrivacyKeyForwards PrivacyKey = "forwards"
	// PrivacyKeyPhoneCall controls who can initiate voice calls with the user.
	PrivacyKeyPhoneCall PrivacyKey = "phone_call"
	// PrivacyKeyPhoneNumber controls who can view the user's phone number.
	PrivacyKeyPhoneNumber PrivacyKey = "phone_number"
	// PrivacyKeyPhoneP2P controls who can use peer-to-peer connections during
	// calls with the user.
	PrivacyKeyPhoneP2P PrivacyKey = "phone_p2p"
	// PrivacyKeyProfilePhoto controls who can view the user's profile photo.
	PrivacyKeyProfilePhoto PrivacyKey = "profile_photo"
	// PrivacyKeyStatus controls who can view the user's last seen timestamp.
	PrivacyKeyStatus PrivacyKey = "status"
	// PrivacyKeyVoiceMessages controls who can send voice messages to the user.
	PrivacyKeyVoiceMessages PrivacyKey = "voice_messages"
	// PrivacyKeyGifts controls who can send gifts to the user.
	PrivacyKeyGifts PrivacyKey = "gifts"
	// PrivacyKeyNoPaidMessages controls whether the user accepts paid messages
	// from non-contacts.
	PrivacyKeyNoPaidMessages PrivacyKey = "no_paid_messages"
	// PrivacyKeySavedMusic controls who can view the user's saved music.
	PrivacyKeySavedMusic PrivacyKey = "saved_music"
)

// String returns the string representation of the PrivacyKey.
func (p PrivacyKey) String() string { return string(p) }
