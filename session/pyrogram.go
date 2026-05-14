package session

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// EncodePyrogram encodes session data in Pyrogram string session format:
// base64url(dc_id[1B] + app_id[4B BE] + test_mode[1B] + auth_key[256B] + user_id[8B BE] + is_bot[1B])
// No version prefix. Total payload: 271 bytes.
func EncodePyrogram(data *SessionData) (string, error) {
	if len(data.AuthKey) != 256 {
		return "", fmt.Errorf("auth_key must be 256 bytes, got %d", len(data.AuthKey))
	}

	buf := make([]byte, 0, 271)
	buf = append(buf, byte(data.DCID))

	appID := make([]byte, 4)
	binary.BigEndian.PutUint32(appID, uint32(data.AppID))
	buf = append(buf, appID...)

	testMode := byte(0)
	if data.TestMode {
		testMode = 1
	}
	buf = append(buf, testMode)
	buf = append(buf, data.AuthKey...)

	userID := make([]byte, 8)
	binary.BigEndian.PutUint64(userID, uint64(data.UserID))
	buf = append(buf, userID...)

	isBot := byte(0)
	if data.IsBot {
		isBot = 1
	}
	buf = append(buf, isBot)

	encoded := base64.URLEncoding.EncodeToString(buf)
	// Strip trailing = padding.
	return trimBase64Padding(encoded), nil
}

// DecodePyrogram decodes a Pyrogram string session.
func DecodePyrogram(s string) (*SessionData, error) {
	payload, err := base64.URLEncoding.DecodeString(padBase64(s))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	if len(payload) != 271 {
		return nil, fmt.Errorf("unexpected payload length %d (expected 271)", len(payload))
	}

	dcID := int(payload[0])
	appID := int32(binary.BigEndian.Uint32(payload[1:5]))
	testMode := payload[5] != 0
	authKey := make([]byte, 256)
	copy(authKey, payload[6:262])
	userID := int64(binary.BigEndian.Uint64(payload[262:270]))
	isBot := payload[270] != 0

	return &SessionData{
		DCID:    dcID,
		AppID:   appID,
		TestMode: testMode,
		AuthKey: authKey,
		UserID:  userID,
		IsBot:   isBot,
	}, nil
}

// trimBase64Padding removes trailing = characters from a base64 encoded string.
func trimBase64Padding(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}
