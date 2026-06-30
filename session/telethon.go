package session

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

var errNotTelethon = errors.New("not a Telethon session string")

// EncodeTelethon encodes session data in Telethon string session format with
// api_id included (version "2"):
// "2" + base64url(dc_id[1B] + ip[4B|16B] + port[2B BE] + api_id[4B BE] + auth_key[256B])
//
// Falls back to legacy format ("1" prefix) when api_id is zero.
func EncodeTelethon(data *SessionData) (string, error) {
	if len(data.AuthKey) != 256 {
		return "", fmt.Errorf("auth_key must be 256 bytes, got %d", len(data.AuthKey))
	}

	ip := net.ParseIP(data.ServerAddress)
	if ip == nil {
		ip = net.IPv4(0, 0, 0, 0)
	}

	ip4 := ip.To4()
	var ipBytes []byte
	if ip4 != nil {
		ipBytes = ip4
	} else {
		ipBytes = ip.To16()
	}

	if data.AppID > 0 {
		buf := make([]byte, 0, 1+16+2+4+256)
		buf = append(buf, byte(data.DCID))
		buf = append(buf, ipBytes...)
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, uint16(data.Port))
		buf = append(buf, portBytes...)
		apiIDBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(apiIDBytes, uint32(data.AppID))
		buf = append(buf, apiIDBytes...)
		buf = append(buf, data.AuthKey...)

		return "2" + base64.URLEncoding.EncodeToString(buf), nil
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

// DecodeTelethon decodes a Telethon string session (both v1 and v2 formats).
func DecodeTelethon(s string) (*SessionData, error) {
	if len(s) < 2 || (s[0] != '1' && s[0] != '2') {
		return nil, errNotTelethon
	}

	version := s[0]
	payload, err := base64.URLEncoding.DecodeString(padBase64(s[1:]))
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	dcID := int(payload[0])

	switch version {
	case '2':
		return decodeTelethonV2(payload, dcID)
	default:
		return decodeTelethonV1(payload, dcID)
	}
}

func decodeTelethonV1(payload []byte, dcID int) (*SessionData, error) {
	if len(payload) < 1+4+2+256 {
		return nil, fmt.Errorf("%w: %d bytes", errPayloadTooShort, len(payload))
	}

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

// decodeTelethonV2 decodes: dc_id(1B) + ip(4B|16B) + port(2B) + api_id(4B) + auth_key(256B)
func decodeTelethonV2(payload []byte, dcID int) (*SessionData, error) {
	if len(payload) < 1+4+2+4+256 {
		return nil, fmt.Errorf("%w: %d bytes", errPayloadTooShort, len(payload))
	}

	var ipLen int
	switch len(payload) {
	case 267: // 1 + 4 (IPv4) + 2 + 4 + 256
		ipLen = 4
	case 279: // 1 + 16 (IPv6) + 2 + 4 + 256
		ipLen = 16
	default:
		return nil, fmt.Errorf("unexpected payload length %d (expected 267 or 279)", len(payload))
	}

	ipBytes := payload[1 : 1+ipLen]
	addr := net.IP(ipBytes).String()

	port := int(binary.BigEndian.Uint16(payload[1+ipLen : 1+ipLen+2]))
	apiID := int32(binary.BigEndian.Uint32(payload[1+ipLen+2 : 1+ipLen+2+4]))
	authKey := make([]byte, 256)
	copy(authKey, payload[1+ipLen+2+4:])

	return &SessionData{
		DCID:          dcID,
		ServerAddress: addr,
		Port:          port,
		AppID:         apiID,
		AuthKey:       authKey,
	}, nil
}
