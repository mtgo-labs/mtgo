package session

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
)

// EncodeTelethon encodes session data in Telethon string session format:
// "1" + base64url(dc_id[1B] + ip[4B|16B] + port[2B BE] + auth_key[256B])
func EncodeTelethon(data *SessionData) (string, error) {
	if len(data.AuthKey) != 256 {
		return "", fmt.Errorf("auth_key must be 256 bytes, got %d", len(data.AuthKey))
	}

	ip := net.ParseIP(data.ServerAddress)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", data.ServerAddress)
	}

	ip4 := ip.To4()
	var ipBytes []byte
	if ip4 != nil {
		ipBytes = ip4
	} else {
		ipBytes = ip.To16()
	}

	buf := make([]byte, 0, 1+16+2+256)
	buf = append(buf, byte(data.DCID))
	buf = append(buf, ipBytes...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(data.Port))
	buf = append(buf, portBytes...)
	buf = append(buf, data.AuthKey...)

	return "1" + base64.URLEncoding.EncodeToString(buf), nil
}

// DecodeTelethon decodes a Telethon string session.
func DecodeTelethon(s string) (*SessionData, error) {
	if len(s) < 2 || s[0] != '1' {
		return nil, fmt.Errorf("not a Telethon session string")
	}

	payload, err := base64.URLEncoding.DecodeString(padBase64(s[1:]))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	if len(payload) < 1+4+2+256 {
		return nil, fmt.Errorf("payload too short: %d bytes", len(payload))
	}

	dcID := int(payload[0])

	var ipLen int
	switch len(payload) {
	case 263: // 1 + 4 (IPv4) + 2 + 256
		ipLen = 4
	case 275: // 1 + 16 (IPv6) + 2 + 256
		ipLen = 16
	default:
		return nil, fmt.Errorf("unexpected payload length %d (expected 263 or 275)", len(payload))
	}

	ipBytes := payload[1 : 1+ipLen]
	addr := net.IP(ipBytes).String()

	port := int(binary.BigEndian.Uint16(payload[1+ipLen : 1+ipLen+2]))
	authKey := make([]byte, 256)
	copy(authKey, payload[1+ipLen+2:])

	return &SessionData{
		DCID:          dcID,
		ServerAddress: addr,
		Port:          port,
		AuthKey:       authKey,
	}, nil
}
