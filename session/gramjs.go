package session

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

var errNotGramJS = errors.New("not a GramJS session string")

// EncodeGramjs encodes session data in GramJS string session format:
// "1" + base64_std(dc_id[1B] + addr_len[2B BE] + addr_bytes[variable] + port[2B BE] + auth_key[256B])
// addr_bytes is the raw string bytes of the IP address (not packed IP).
func EncodeGramjs(data *SessionData) (string, error) {
	if len(data.AuthKey) != 256 {
		return "", fmt.Errorf("auth_key must be 256 bytes, got %d", len(data.AuthKey))
	}

	addrBytes := []byte(data.ServerAddress)
	if len(addrBytes) > 253 {
		return "", fmt.Errorf("address too long: %d bytes", len(addrBytes))
	}

	buf := make([]byte, 0, 1+2+len(addrBytes)+2+256)
	buf = append(buf, byte(data.DCID))

	addrLen := make([]byte, 2)
	binary.BigEndian.PutUint16(addrLen, uint16(len(addrBytes)))
	buf = append(buf, addrLen...)
	buf = append(buf, addrBytes...)

	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(data.Port))
	buf = append(buf, portBytes...)
	buf = append(buf, data.AuthKey...)

	return "1" + base64.StdEncoding.EncodeToString(buf), nil
}

// DecodeGramjs decodes a GramJS string session.
func DecodeGramjs(s string) (*SessionData, error) {
	if len(s) < 2 || s[0] != '1' {
		return nil, errNotGramJS
	}

	payload, err := base64.StdEncoding.DecodeString(s[1:])
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	if len(payload) < 5+256 {
		return nil, fmt.Errorf("payload too short: %d bytes", len(payload))
	}

	dcID := int(payload[0])
	addrLen := int(binary.BigEndian.Uint16(payload[1:3]))

	if len(payload) < 3+addrLen+2+256 {
		return nil, fmt.Errorf("payload too short for address length %d", addrLen)
	}

	addr := string(payload[3 : 3+addrLen])
	port := int(binary.BigEndian.Uint16(payload[3+addrLen : 3+addrLen+2]))
	authKey := make([]byte, 256)
	copy(authKey, payload[3+addrLen+2:])

	return &SessionData{
		DCID:          dcID,
		ServerAddress: addr,
		Port:          port,
		AuthKey:       authKey,
	}, nil
}
