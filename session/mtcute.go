package session

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	errPayloadTooShortForUserID = errors.New("payload too short for user_id")
	errPayloadTooShortForIsBot  = errors.New("payload too short for is_bot")
	errDCOptionTooShortForPort  = errors.New("DC option too short for port")
	errNotEnoughBytesForTLLen   = errors.New("not enough bytes for TL length prefix")
	errPayloadTooShort         = errors.New("payload too short")
)

// mtcute format uses TL serialization primitives.

const (
	mtcuteVersion         = 3
	mtcuteDCOptionVersion = 2

	mtcuteFlagHasSelf  = 1 << 0
	mtcuteFlagHasMedia = 1 << 2

	mtcuteDCFlagIPv6      = 1 << 0
	mtcuteDCFlagMediaOnly = 1 << 1
	mtcuteDCFlagTestMode  = 1 << 2

	tlBoolTrue  = 0xB5757299
	tlBoolFalse = 0x379779BC
)

// EncodeMtcute encodes session data in mtcute string session format.
func EncodeMtcute(data *SessionData) (string, error) {
	if len(data.AuthKey) != 256 {
		return "", fmt.Errorf("auth_key must be 256 bytes, got %d", len(data.AuthKey))
	}

	buf := make([]byte, 0, 512)

	// Version byte.
	buf = append(buf, mtcuteVersion)

	// Flags: hasSelf if userID != 0.
	flags := uint32(0)
	if data.UserID != 0 {
		flags |= mtcuteFlagHasSelf
	}
	flagsBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(flagsBuf, flags)
	buf = append(buf, flagsBuf...)

	// Primary DC option.
	buf = append(buf, tlBytes(serializeDCOption(data.DCID, data.ServerAddress, data.Port, isIPv6(data.ServerAddress), false, data.TestMode))...)

	// If hasSelf: user_id + is_bot.
	if data.UserID != 0 {
		uidBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(uidBuf, uint64(data.UserID))
		buf = append(buf, uidBuf...)
		buf = append(buf, tlBool(data.IsBot)...)
	}

	// Auth key.
	buf = append(buf, tlBytes(data.AuthKey)...)

	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(buf)
	return trimBase64Padding(encoded), nil
}

// DecodeMtcute decodes an mtcute string session.
func DecodeMtcute(s string) (*SessionData, error) {
	raw := trimBase64Padding(s)
	payload, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(raw)
	if err != nil {
		// Fallback: some mtcute implementations emit standard base64.
		payload, err = base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
	}

	if len(payload) < 5 {
		return nil, fmt.Errorf("%w: %d bytes", errPayloadTooShort, len(payload))
	}

	off := 0

	// Version.
	if payload[off] != mtcuteVersion {
		return nil, fmt.Errorf("unexpected version %d (expected %d)", payload[off], mtcuteVersion)
	}
	off++

	// Flags.
	flags := binary.LittleEndian.Uint32(payload[off : off+4])
	off += 4

	hasSelf := flags&mtcuteFlagHasSelf != 0
	hasMedia := flags&mtcuteFlagHasMedia != 0

	// Primary DC option.
	dcData, n, err := readTLBytes(payload, off)
	if err != nil {
		return nil, fmt.Errorf("reading DC option: %w", err)
	}
	off = n

	// Media DC option (skip if present).
	if hasMedia {
		_, n, err := readTLBytes(payload, off)
		if err != nil {
			return nil, fmt.Errorf("reading media DC option: %w", err)
		}
		off = n
	}

	dcID, addr, port, testMode, err := parseDCOption(dcData)
	if err != nil {
		return nil, fmt.Errorf("parsing DC option: %w", err)
	}

	var userID int64
	var isBot bool

	if hasSelf {
		if off+8 > len(payload) {
			return nil, errPayloadTooShortForUserID
		}
		userID = int64(binary.LittleEndian.Uint64(payload[off : off+8]))
		off += 8

		if off+4 > len(payload) {
			return nil, errPayloadTooShortForIsBot
		}
		isBot = readTLBool(payload[off : off+4])
		off += 4
	}

	// Auth key.
	authKeyData, _, err := readTLByteSlice(payload, off)
	if err != nil {
		return nil, fmt.Errorf("reading auth key: %w", err)
	}

	if len(authKeyData) != 256 {
		return nil, fmt.Errorf("auth_key must be 256 bytes, got %d", len(authKeyData))
	}

	authKey := make([]byte, 256)
	copy(authKey, authKeyData)

	return &SessionData{
		DCID:          dcID,
		ServerAddress: addr,
		Port:          port,
		AuthKey:       authKey,
		TestMode:      testMode,
		UserID:        userID,
		IsBot:         isBot,
	}, nil
}

// serializeDCOption packs a DC option in TL format:
// [version=2][dcId][flags] + TL_bytes(ipString) + port[4B LE]
func serializeDCOption(dcID int, ip string, port int, ipv6, mediaOnly, testMode bool) []byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, mtcuteDCOptionVersion) // version
	buf = append(buf, byte(dcID))            // dcId

	var dcFlags byte
	if ipv6 {
		dcFlags |= mtcuteDCFlagIPv6
	}
	if mediaOnly {
		dcFlags |= mtcuteDCFlagMediaOnly
	}
	if testMode {
		dcFlags |= mtcuteDCFlagTestMode
	}
	buf = append(buf, dcFlags)                // flags
	buf = append(buf, tlBytes([]byte(ip))...) // TL_bytes(ip)

	portBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(portBuf, uint32(port))
	buf = append(buf, portBuf...) // port

	return buf
}

// parseDCOption unpacks a DC option.
func parseDCOption(data []byte) (dcID int, addr string, port int, testMode bool, err error) {
	if len(data) < 3 {
		return 0, "", 0, false, fmt.Errorf("DC option too short: %d bytes", len(data))
	}

	off := 0
	if data[off] != mtcuteDCOptionVersion {
		return 0, "", 0, false, fmt.Errorf("unexpected DC option version %d", data[off])
	}
	off++

	dcID = int(data[off])
	off++

	dcFlags := data[off]
	off++
	testMode = dcFlags&mtcuteDCFlagTestMode != 0

	// TL_bytes for IP.
	ipData, n, err := readTLByteSlice(data, off)
	if err != nil {
		return 0, "", 0, false, fmt.Errorf("reading IP: %w", err)
	}
	addr = string(ipData)
	off = n

	if off+4 > len(data) {
		return 0, "", 0, false, errDCOptionTooShortForPort
	}
	port = int(binary.LittleEndian.Uint32(data[off : off+4]))

	return dcID, addr, port, testMode, nil
}

// tlBytes encodes data using TL bytes serialization.
func tlBytes(data []byte) []byte {
	length := len(data)
	buf := make([]byte, 0, length+8)

	if length <= 253 {
		buf = append(buf, byte(length))
	} else {
		buf = append(buf, 0xFE)
		le := make([]byte, 3)
		le[0] = byte(length)
		le[1] = byte(length >> 8)
		le[2] = byte(length >> 16)
		buf = append(buf, le...)
	}

	buf = append(buf, data...)

	// Pad to 4-byte alignment.
	padLen := (4 - (len(buf) % 4)) % 4
	for i := 0; i < padLen; i++ {
		buf = append(buf, 0)
	}

	return buf
}

// readTLByteSlice reads TL-encoded bytes starting at off, returning the decoded
// data and the new offset.
func readTLByteSlice(data []byte, off int) ([]byte, int, error) {
	if off >= len(data) {
		return nil, 0, fmt.Errorf("offset %d out of bounds (len %d)", off, len(data))
	}

	startOff := off

	var length int
	firstByte := data[off]
	if firstByte <= 253 {
		length = int(firstByte)
		off++
	} else {
		off++
		if off+3 > len(data) {
			return nil, 0, errNotEnoughBytesForTLLen
		}
		length = int(data[off]) | int(data[off+1])<<8 | int(data[off+2])<<16
		off += 3
	}

	if off+length > len(data) {
		return nil, 0, fmt.Errorf("TL bytes data extends past payload (need %d, have %d)", length, len(data)-off)
	}

	result := data[off : off+length]
	off += length

	// Advance past padding: the entry (from startOff) must be 4-byte aligned.
	entryEnd := startOff + ((off-startOff)+3)&^3
	if entryEnd > len(data) {
		entryEnd = off // no padding beyond data
	}

	return result, entryEnd, nil
}

// readTLBytes returns the raw data and new offset from a TL bytes field.
func readTLBytes(data []byte, off int) ([]byte, int, error) {
	return readTLByteSlice(data, off)
}

// tlBool encodes a bool as a TL bool.
func tlBool(v bool) []byte {
	buf := make([]byte, 4)
	if v {
		binary.LittleEndian.PutUint32(buf, tlBoolTrue)
	} else {
		binary.LittleEndian.PutUint32(buf, tlBoolFalse)
	}
	return buf
}

// readTLBool reads a TL bool from 4 bytes.
func readTLBool(b []byte) bool {
	val := binary.LittleEndian.Uint32(b)
	return val == tlBoolTrue
}

// isIPv6 checks if the address is an IPv6 address.
func isIPv6(addr string) bool {
	for i := 0; i < len(addr); i++ {
		if addr[i] == ':' {
			return true
		}
	}
	return false
}
