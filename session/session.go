package session

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// Format identifies a string session encoding format.
type Format string

const (
	FormatTelethon     Format = "telethon"
	FormatPyrogram     Format = "pyrogram"
	FormatGramJS       Format = "gramjs"
	FormatMtcute       Format = "mtcute"
	FormatGotgExtended Format = "gotg_extended"
	FormatUnknown      Format = ""
)

// -------------------------------------------------------------------
// String-returning functions — use with Config.SessionString
//
//	sessionStr, _ := session.Pyrogram("pyrogram_string")
//	client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
//	    SessionString: sessionStr,
//	})
// -------------------------------------------------------------------

// Telethon validates a Telethon session string and returns it.
func Telethon(s string) (string, error) {
	if _, err := DecodeTelethon(s); err != nil {
		return "", fmt.Errorf("telethon: %w", err)
	}
	return s, nil
}

// Pyrogram decodes a Pyrogram session string and returns a Telethon-format string.
func Pyrogram(s string) (string, error) {
	data, err := DecodePyrogram(s)
	if err != nil {
		return "", fmt.Errorf("pyrogram: %w", err)
	}
	return EncodeTelethon(data)
}

// Gramjs decodes a GramJS session string and returns a Telethon-format string.
func Gramjs(s string) (string, error) {
	data, err := DecodeGramjs(s)
	if err != nil {
		return "", fmt.Errorf("gramjs: %w", err)
	}
	return EncodeTelethon(data)
}

// Mtcute decodes an mtcute session string and returns a Telethon-format string.
func Mtcute(s string) (string, error) {
	data, err := DecodeMtcute(s)
	if err != nil {
		return "", fmt.Errorf("mtcute: %w", err)
	}
	return EncodeTelethon(data)
}

// String auto-detects the session format and returns a Telethon-format string.
func String(s string) (string, error) {
	data, _, err := Decode(s)
	if err != nil {
		return "", err
	}
	return EncodeTelethon(data)
}

// -------------------------------------------------------------------
// Session struct — implements storage.Storage for Config.Storage
//
//	s, _ := session.TelethonSession("1abc...")
//	client, _ := tg.NewClient(apiID, apiHash, &tg.Config{
//	    Storage: s,
//	})
// -------------------------------------------------------------------

// Session holds decoded session data and satisfies the storage.Storage
// interface via structural typing.
type Session struct {
	dcID          int
	apiID         int32
	apiHash       string
	testMode      bool
	authKey       []byte
	userID        int64
	isBot         bool
	firstName     string
	lastName      string
	username      string
	date          int
	serverAddress string
	port          int
	state         []byte

	// Format is the detected source format (set by constructors).
	Format Format
}

// String returns the session as a Telethon-format string.
func (s *Session) String() string {
	str, _ := s.ExportSessionString()
	return str
}

// ---------- Session constructors (return *Session for Config.Storage) ----------

// TelethonSession decodes a Telethon-format session string into a *Session.
func TelethonSession(s string) (*Session, error) {
	data, err := DecodeTelethon(s)
	if err != nil {
		return nil, fmt.Errorf("telethon: %w", err)
	}
	return dataToSession(data, FormatTelethon), nil
}

// PyrogramSession decodes a Pyrogram-format session string into a *Session.
func PyrogramSession(s string) (*Session, error) {
	data, err := DecodePyrogram(s)
	if err != nil {
		return nil, fmt.Errorf("pyrogram: %w", err)
	}
	return dataToSession(data, FormatPyrogram), nil
}

// GramjsSession decodes a GramJS-format session string into a *Session.
func GramjsSession(s string) (*Session, error) {
	data, err := DecodeGramjs(s)
	if err != nil {
		return nil, fmt.Errorf("gramjs: %w", err)
	}
	return dataToSession(data, FormatGramJS), nil
}

// MtcuteSession decodes an mtcute-format session string into a *Session.
func MtcuteSession(s string) (*Session, error) {
	data, err := DecodeMtcute(s)
	if err != nil {
		return nil, fmt.Errorf("mtcute: %w", err)
	}
	return dataToSession(data, FormatMtcute), nil
}

// StringSession auto-detects the format and decodes into a *Session.
func StringSession(s string) (*Session, error) {
	data, format, err := Decode(s)
	if err != nil {
		return nil, err
	}
	return dataToSession(data, format), nil
}

func dataToSession(d *SessionData, f Format) *Session {
	return &Session{
		dcID:          d.DCID,
		serverAddress: d.ServerAddress,
		port:          d.Port,
		authKey:       d.AuthKey,
		apiID:         d.AppID,
		testMode:      d.TestMode,
		userID:        d.UserID,
		isBot:         d.IsBot,
		Format:        f,
	}
}

// ---------- storage.Storage interface (structural typing) ----------

func (s *Session) DCID() (int, error)              { return s.dcID, nil }
func (s *Session) SetDCID(v int) error             { s.dcID = v; return nil }
func (s *Session) APIID() (int32, error)           { return s.apiID, nil }
func (s *Session) SetAPIID(v int32) error          { s.apiID = v; return nil }
func (s *Session) APIHash() (string, error)        { return s.apiHash, nil }
func (s *Session) SetAPIHash(v string) error       { s.apiHash = v; return nil }
func (s *Session) TestMode() (bool, error)         { return s.testMode, nil }
func (s *Session) SetTestMode(v bool) error        { s.testMode = v; return nil }
func (s *Session) AuthKey() ([]byte, error)        { return append([]byte(nil), s.authKey...), nil }
func (s *Session) SetAuthKey(v []byte) error       { s.authKey = v; return nil }
func (s *Session) UserID() (int64, error)          { return s.userID, nil }
func (s *Session) SetUserID(v int64) error         { s.userID = v; return nil }
func (s *Session) IsBot() (bool, error)            { return s.isBot, nil }
func (s *Session) SetIsBot(v bool) error           { s.isBot = v; return nil }
func (s *Session) FirstName() (string, error)      { return s.firstName, nil }
func (s *Session) SetFirstName(v string) error     { s.firstName = v; return nil }
func (s *Session) LastName() (string, error)       { return s.lastName, nil }
func (s *Session) SetLastName(v string) error      { s.lastName = v; return nil }
func (s *Session) Username() (string, error)       { return s.username, nil }
func (s *Session) SetUsername(v string) error      { s.username = v; return nil }
func (s *Session) Date() (int, error)              { return s.date, nil }
func (s *Session) SetDate(v int) error             { s.date = v; return nil }
func (s *Session) ServerAddress() (string, error)  { return s.serverAddress, nil }
func (s *Session) SetServerAddress(v string) error { s.serverAddress = v; return nil }
func (s *Session) Port() (int, error)              { return s.port, nil }
func (s *Session) SetPort(v int) error             { s.port = v; return nil }
func (s *Session) State() ([]byte, error)          { return append([]byte(nil), s.state...), nil }
func (s *Session) SetState(v []byte) error         { s.state = v; return nil }
func (s *Session) Close() error                    { return nil }

// ExportSessionString returns the Telethon-format session string.
func (s *Session) ExportSessionString() (string, error) {
	return EncodePyrogram(&SessionData{
		DCID:     s.dcID,
		AppID:    s.apiID,
		TestMode: s.testMode,
		AuthKey:  s.authKey,
		UserID:   s.userID,
		IsBot:    s.isBot,
	})
}

// ---------- ImportSessionString — used by connectTransport ----------

// ImportSessionString decodes a Telethon-format string and populates the
// storage with DCID, ServerAddress, Port, and AuthKey.
func ImportSessionString(st interface {
	DCID() (int, error)
	SetDCID(int) error
	ServerAddress() (string, error)
	SetServerAddress(string) error
	Port() (int, error)
	SetPort(int) error
	AuthKey() ([]byte, error)
	SetAuthKey([]byte) error
}, s string,
) error {
	data, err := DecodeTelethon(s)
	if err != nil {
		return fmt.Errorf("session: invalid session string: %w", err)
	}
	_ = st.SetDCID(data.DCID)
	_ = st.SetServerAddress(data.ServerAddress)
	_ = st.SetPort(data.Port)
	_ = st.SetAuthKey(data.AuthKey)
	return nil
}

// ---------- detection ----------

// DetectFormat attempts to identify which encoding format a string session
// uses without fully decoding it.
func DetectFormat(s string) Format {
	if s == "" {
		return FormatUnknown
	}

	if s[0] == '1' && len(s) > 1 {
		rest := s[1:]

		if payload, err := base64.URLEncoding.DecodeString(padBase64(rest)); err == nil {
			switch len(payload) {
			case 263, 275:
				return FormatTelethon
			}
		}

		if payload, err := base64.StdEncoding.DecodeString(rest); err == nil && len(payload) >= 5 {
			addrLen := uint16(payload[1])<<8 | uint16(payload[2])
			if addrLen >= 4 && addrLen <= 253 {
				expected := 1 + 2 + int(addrLen) + 2 + 256
				if len(payload) == expected {
					return FormatGramJS
				}
			}
		}
	}

	if s[0] == '2' && len(s) > 1 {
		rest := s[1:]
		if payload, err := base64.URLEncoding.DecodeString(padBase64(rest)); err == nil {
			switch len(payload) {
			case 267, 279: // v2: IPv4 or IPv6 with api_id
				return FormatTelethon
			}
		}
	}

	if s[0] != '1' || (s[0] == '1' && len(s) > 1) {
		candidate := s
		if payload, err := base64.URLEncoding.DecodeString(padBase64(candidate)); err == nil {
			if len(payload) > 0 && payload[0] == 3 {
				return FormatMtcute
			}
		}

		if payload, err := base64.URLEncoding.DecodeString(padBase64(s)); err == nil && len(payload) == 271 {
			return FormatPyrogram
		}
		if payload, err := base64.URLEncoding.DecodeString(padBase64(s)); err == nil && len(payload) > 27 && payload[0] <= 5 && len(payload) >= 27+256 {
			return FormatGotgExtended
		}
	}

	return FormatUnknown
}

// ---------- decode / encode ----------

// SessionData holds the common fields extracted from or encoded into a
// string session. Not all fields are used by every format.
type SessionData struct {
	DCID          int
	ServerAddress string
	Port          int
	AuthKey       []byte // 256 bytes
	AppID         int32
	TestMode      bool
	UserID        int64
	IsBot         bool
}

// Decode auto-detects the format of the string session and decodes it.
func Decode(s string) (*SessionData, Format, error) {
	if s == "" {
		return nil, FormatUnknown, ErrEmptySession
	}

	f := DetectFormat(s)
	if f != FormatUnknown {
		data, err := decodeFormat(s, f)
		if err == nil {
			return data, f, nil
		}
	}

	for _, f := range []Format{FormatTelethon, FormatGramJS, FormatPyrogram, FormatMtcute, FormatGotgExtended} {
		data, err := decodeFormat(s, f)
		if err == nil {
			return data, f, nil
		}
	}

	return nil, FormatUnknown, ErrUnknownFormat
}

// Encode encodes SessionData into the specified format.
func Encode(data *SessionData, format Format) (string, error) {
	switch format {
	case FormatTelethon:
		return EncodeTelethon(data)
	case FormatPyrogram:
		return EncodePyrogram(data)
	case FormatGramJS:
		return EncodeGramjs(data)
	case FormatMtcute:
		return EncodeMtcute(data)
	default:
		return "", fmt.Errorf("unknown format: %s", format)
	}
}

// ---------- internal ----------

func decodeFormat(s string, f Format) (*SessionData, error) {
	switch f {
	case FormatTelethon:
		return DecodeTelethon(s)
	case FormatPyrogram:
		return DecodePyrogram(s)
	case FormatGramJS:
		return DecodeGramjs(s)
	case FormatMtcute:
		return DecodeMtcute(s)
	case FormatGotgExtended:
		return DecodeGotgExtended(s)
	default:
		return nil, fmt.Errorf("unknown format: %s", f)
	}
}

func padBase64(s string) string {
	switch len(s) % 4 {
	case 2:
		return s + "=="
	case 3:
		return s + "="
	default:
		return s
	}
}

// DecodeGotgExtended decodes the gotg extended session format.
// Format: dc(4 LE) + ver(4 LE) + ip_len(1) + ip(var) + null_pad(var) + port(2 BE) + auth_key(256)
// Uses URL-safe base64 encoding.
func DecodeGotgExtended(s string) (*SessionData, error) {
	payload, err := base64.URLEncoding.DecodeString(padBase64(s))
	if err != nil {
		return nil, fmt.Errorf("gotg_extended: base64 decode: %w", err)
	}
	if len(payload) < 27+256 {
		return nil, fmt.Errorf("gotg_extended: payload too short: %d", len(payload))
	}

	dc := int(int32(binary.LittleEndian.Uint32(payload[0:4])))
	_ = int32(binary.LittleEndian.Uint32(payload[4:8])) // version
	ipLen := int(payload[8])
	if 8+1+ipLen > len(payload) {
		return nil, fmt.Errorf("gotg_extended: ip length %d exceeds payload", ipLen)
	}
	ip := string(payload[9 : 9+ipLen])

	pos := 9 + ipLen
	for pos < len(payload) && payload[pos] == 0 {
		pos++
	}
	if pos+2 > len(payload) {
		return nil, fmt.Errorf("gotg_extended: port out of range")
	}
	port := int(binary.BigEndian.Uint16(payload[pos : pos+2]))
	pos += 2

	if pos+256 > len(payload) {
		return nil, fmt.Errorf("gotg_extended: payload too short for auth_key")
	}
	authKey := make([]byte, 256)
	copy(authKey, payload[pos:pos+256])

	return &SessionData{
		DCID:          dc,
		ServerAddress: ip,
		Port:          port,
		AuthKey:       authKey,
	}, nil
}
