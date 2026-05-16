package types

import (
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

// SessionBinder provides the ability to terminate an active session through
// a bound client. Injected into ActiveSession instances so they can self-reset.
type SessionBinder interface {
	BoundResetSession(hash int64) error
}

// ActiveSession represents an authenticated session on a specific device,
// including device model, platform, app version, IP, and location info.
// Call Reset to terminate the session.
//
// Example:
//
//	sessions, _ := client.GetActiveSessions(ctx)
//	for _, s := range sessions {
//	    fmt.Printf("%s on %s (last active: %s)\n", s.ApplicationName, s.DeviceModel, s.DateActive)
//	}
type ActiveSession struct {
	Hash               int64
	DeviceModel        string
	Platform           string
	SystemVersion      string
	ApplicationName    string
	ApplicationVersion string
	DateCreated        time.Time
	DateActive         time.Time
	IP                 string
	Country            string
	Region             string
	binder             SessionBinder
}

func (s *ActiveSession) SetBinder(b SessionBinder) {
	s.binder = b
}

func (s *ActiveSession) Reset() error {
	if s.binder == nil {
		return ErrNoBinder
	}
	return s.binder.BoundResetSession(s.Hash)
}

// ParseActiveSession converts an MTProto AuthorizationTL into an ActiveSession.
// Returns nil if raw is nil.
func ParseActiveSession(raw *tg.Authorization) *ActiveSession {
	if raw == nil {
		return nil
	}
	return &ActiveSession{
		Hash:               raw.Hash,
		DeviceModel:        raw.DeviceModel,
		Platform:           raw.Platform,
		SystemVersion:      raw.SystemVersion,
		ApplicationName:    raw.AppName,
		ApplicationVersion: raw.AppVersion,
		DateCreated:        time.Unix(int64(raw.DateCreated), 0),
		DateActive:         time.Unix(int64(raw.DateActive), 0),
		IP:                 raw.Ip,
		Country:            raw.Country,
		Region:             raw.Region,
	}
}

// ActiveSessions holds the list of active sessions and web sessions for the
// current account. Use this to present a complete session management view.
type ActiveSessions struct {
	// Sessions is the list of native (mobile, desktop) active sessions.
	Sessions []*ActiveSession
	// WebSessions is the list of active web (browser) sessions.
	WebSessions []*ActiveSession
	// InactiveSessionTtlDays is the number of days after which inactive
	// sessions are automatically terminated.
	InactiveSessionTTLDays int32
}

// SentCodeInfo describes the result of a send-code request during
// authentication. It tells the client how the verification code was delivered
// and what fallback methods are available.
//
// Example:
//
//	info, err := client.SendCode(ctx, "+1234567890")
//	fmt.Printf("Code type: %s, hash: %s\n", info.Type, info.PhoneCodeHash)
type SentCodeInfo struct {
	// Type indicates how the code was delivered (e.g., SMS, call, Telegram app).
	Type SentCodeType
	// PhoneCodeHash is the hash that must be passed back when calling
	// auth.signIn or auth.signUp to verify the code.
	PhoneCodeHash string
	// NextType indicates the fallback delivery method the user can request
	// if the primary method fails. Empty when no fallback is available.
	NextType SentCodeType
	// Timeout is the number of seconds the user must wait before requesting
	// the next delivery method. Zero when no waiting period applies.
	Timeout int32
}

// ParseSentCodeInfo converts an MTProto AuthSentCode into a SentCodeInfo.
// Returns nil if raw is nil.
func ParseSentCodeInfo(raw *tg.AuthSentCode) *SentCodeInfo {
	if raw == nil {
		return nil
	}
	info := &SentCodeInfo{
		PhoneCodeHash: raw.PhoneCodeHash,
	}
	info.Type = parseSentCodeTypeClass(raw.Type)
	if raw.NextType != nil {
		info.NextType = parseCodeTypeClass(raw.NextType)
	}
	if raw.Timeout != 0 {
		info.Timeout = raw.Timeout
	}
	return info
}

func parseSentCodeTypeClass(raw tg.SentCodeTypeClass) SentCodeType {
	if raw == nil {
		return ""
	}
	switch raw.(type) {
	case *tg.AuthSentCodeTypeApp:
		return SentCodeTypeApp
	case *tg.AuthSentCodeTypeCall:
		return SentCodeTypeCall
	case *tg.AuthSentCodeTypeFlashCall:
		return SentCodeTypeFlashCall
	case *tg.AuthSentCodeTypeMissedCall:
		return SentCodeTypeMissedCall
	case *tg.AuthSentCodeTypeSms:
		return SentCodeTypeSMS
	case *tg.AuthSentCodeTypeFragmentSms:
		return SentCodeTypeFragmentSMS
	case *tg.AuthSentCodeTypeEmailCode:
		return SentCodeTypeEmailCode
	case *tg.AuthSentCodeTypeFirebaseSms:
		return SentCodeTypeFirebaseSMS
	case *tg.AuthSentCodeTypeSetUpEmailRequired:
		return SentCodeTypeSetupEmailRequired
	case *tg.AuthSentCodeTypeSmsPhrase:
		return SentCodeTypeSmsPhrase
	case *tg.AuthSentCodeTypeSmsWord:
		return SentCodeTypeSmsWord
	default:
		return ""
	}
}

func parseCodeTypeClass(raw tg.CodeTypeClass) SentCodeType {
	if raw == nil {
		return ""
	}
	switch raw.(type) {
	case *tg.AuthCodeTypeSms:
		return SentCodeTypeSMS
	case *tg.AuthCodeTypeCall:
		return SentCodeTypeCall
	case *tg.AuthCodeTypeFlashCall:
		return SentCodeTypeFlashCall
	case *tg.AuthCodeTypeMissedCall:
		return SentCodeTypeMissedCall
	case *tg.AuthCodeTypeFragmentSms:
		return SentCodeTypeFragmentSMS
	default:
		return ""
	}
}

// TermsOfService represents the Telegram Terms of Service presented to the
// user during authorization. The user must accept these terms before proceeding.
type TermsOfService struct {
	// ID is the unique identifier of the terms of service document, used to
	// confirm acceptance via account.acceptTermsOfService.
	ID string
	// Text is the full text content of the terms of service.
	Text string
	// Entities contains the formatting entities (bold, links, etc.) applied
	// to Text for rich rendering.
	Entities []*MessageEntity
	// MinAgeConfirm is the minimum age the user must confirm to accept the
	// terms. Zero when no age confirmation is required.
	MinAgeConfirm int32
}

// ParseTermsOfService converts an MTProto HelpTermsOfService into a
// TermsOfService. Returns nil if raw is nil.
func ParseTermsOfService(raw *tg.HelpTermsOfService) *TermsOfService {
	if raw == nil {
		return nil
	}
	tos := &TermsOfService{
		Text:     raw.Text,
		Entities: ParseMessageEntities(raw.Entities),
	}
	if raw.ID != nil {
		tos.ID = raw.ID.Data
	}
	if raw.MinAgeConfirm != 0 {
		tos.MinAgeConfirm = raw.MinAgeConfirm
	}
	return tos
}

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
