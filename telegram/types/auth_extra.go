package types

// FirebaseAuthenticationSettings represents Firebase authentication settings.
type FirebaseAuthenticationSettings struct {
	Platform string
}

// FirebaseAuthenticationSettingsAndroid represents Firebase settings for Android.
type FirebaseAuthenticationSettingsAndroid struct{}

// FirebaseAuthenticationSettingsIos represents Firebase settings for iOS.
type FirebaseAuthenticationSettingsIos struct{}

// PhoneNumberAuthenticationSettings controls how phone number verification is performed.
type PhoneNumberAuthenticationSettings struct {
	AllowFlashcall  bool
	CurrentNumber   bool
	AllowAppHash    bool
	AllowMissedCall bool
	AllowFirebase   bool
}
