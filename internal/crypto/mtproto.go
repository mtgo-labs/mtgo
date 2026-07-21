package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"hash"
	"sync"

	"github.com/mtgo-labs/mtgo/tg"
	tgerr "github.com/mtgo-labs/mtgo/tgerr"
)

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// KDF derives the AES key and IV used for message encryption from the
// authorization key and message key. The outgoing parameter determines which
// offset into authKey is used: offset 0 for client-to-server messages and
// offset 8 for server-to-client messages.
//
// Returns the 32-byte aesKey and 32-byte aesIV.
//
// See https://core.telegram.org/mtproto/description#defining-aes-key-and-initialization-vector.
func KDF(authKey, msgKey []byte, outgoing bool) (aesKey, aesIV [32]byte) {
	x := 0
	if !outgoing {
		x = 8
	}

	var tmpA [52]byte
	copy(tmpA[:], msgKey)
	copy(tmpA[len(msgKey):], authKey[x:x+36])
	sha256A := sha256.Sum256(tmpA[:])

	var tmpB [52]byte
	copy(tmpB[:], authKey[x+40:x+76])
	copy(tmpB[36:], msgKey)
	sha256B := sha256.Sum256(tmpB[:])

	copy(aesKey[0:8], sha256A[:8])
	copy(aesKey[8:24], sha256B[8:24])
	copy(aesKey[24:32], sha256A[24:32])

	copy(aesIV[0:8], sha256B[:8])
	copy(aesIV[8:24], sha256A[8:24])
	copy(aesIV[24:32], sha256B[24:32])

	return aesKey, aesIV
}

// encryptedPaddingLen returns the padding length for a plaintext of length
// bufLen: at least 12 bytes, 16-aligned, plus a random 0..15 extra 16-byte
// blocks so the ciphertext length is not a deterministic client fingerprint.
// Max is 27+240=267 bytes, within the MTProto-2.0 12..1024 bound.
func encryptedPaddingLen(bufLen int, randByte byte) int {
	pad := (16 - (bufLen % 16)) % 16
	if pad < 12 {
		pad += 16
	}
	pad += int(randByte&0x0F) * 16
	return pad
}

const maxEncryptedPadding = 267

// Pack serializes, pads, and encrypts a message for transmission. It assembles
// the plaintext (salt + sessionID + encoded message + random padding), computes
// msgKey as SHA-256(authKey[88:120] + plaintext)[8:24], derives the AES key/IV
// via KDF, encrypts with AES-IGE, and returns authKeyID + msgKey + ciphertext.
//
// See https://core.telegram.org/mtproto/description#encrypted-message.
func Pack(message *tg.MTProtoMessage, salt int64, sessionID []byte, authKey, authKeyID []byte) ([]byte, error) {
	dataBuf := bufPool.Get().(*bytes.Buffer)
	dataBuf.Reset()
	defer bufPool.Put(dataBuf)

	tg.WriteLong(dataBuf, salt)
	dataBuf.Write(sessionID)

	msgBuf := bufPool.Get().(*bytes.Buffer)
	msgBuf.Reset()
	if err := message.Encode(msgBuf); err != nil {
		bufPool.Put(msgBuf)
		return nil, fmt.Errorf("crypto/mtproto: %w", err)
	}
	dataBuf.Write(msgBuf.Bytes())
	bufPool.Put(msgBuf)

	var jb [1]byte
	if _, err := rand.Read(jb[:]); err != nil {
		return nil, fmt.Errorf("crypto/mtproto: padding random: %w", err)
	}
	var padding [maxEncryptedPadding]byte
	paddingLen := encryptedPaddingLen(dataBuf.Len(), jb[0])
	if _, err := rand.Read(padding[:paddingLen]); err != nil {
		return nil, fmt.Errorf("crypto/mtproto: padding random: %w", err)
	}
	dataBuf.Write(padding[:paddingLen])

	data := dataBuf.Bytes()

	hk := sha256.New()
	hk.Write(authKey[88:120])
	hk.Write(data)
	var msgKeyLarge [32]byte
	hk.Sum(msgKeyLarge[:0])
	msgKey := msgKeyLarge[8:24]

	keyArr, ivArr := KDF(authKey, msgKey, true)
	encrypted, err := IGEEncrypt(data, keyArr[:], ivArr[:])
	if err != nil {
		return nil, err
	}

	result := bufPool.Get().(*bytes.Buffer)
	result.Reset()
	result.Write(authKeyID)
	result.Write(msgKey)
	result.Write(encrypted)
	ReleaseAESBuf(encrypted)

	out := make([]byte, result.Len())
	copy(out, result.Bytes())
	bufPool.Put(result)
	return out, nil
}

// PackRaw is like [Pack] but accepts pre-serialized body bytes instead of a
// *tg.MTProtoMessage. It manually writes the MTProto envelope (msgID, seqNo,
// body length prefix) followed by bodyBytes, then applies padding, msgKey
// computation, and AES-IGE encryption identically to [Pack].
func PackRaw(msgID int64, seqNo uint32, bodyBytes []byte, salt int64, sessionID, authKey, authKeyID []byte) ([]byte, error) {
	dataBuf := bufPool.Get().(*bytes.Buffer)
	dataBuf.Reset()
	defer bufPool.Put(dataBuf)

	tg.WriteLong(dataBuf, salt)
	dataBuf.Write(sessionID)
	tg.WriteLong(dataBuf, msgID)
	tg.WriteInt(dataBuf, seqNo)
	tg.WriteInt(dataBuf, uint32(len(bodyBytes)))
	dataBuf.Write(bodyBytes)

	var jb [1]byte
	if _, err := rand.Read(jb[:]); err != nil {
		return nil, fmt.Errorf("crypto/mtproto: padding random: %w", err)
	}
	var padding [maxEncryptedPadding]byte
	paddingLen := encryptedPaddingLen(dataBuf.Len(), jb[0])
	if _, err := rand.Read(padding[:paddingLen]); err != nil {
		return nil, fmt.Errorf("crypto/mtproto: padding random: %w", err)
	}
	dataBuf.Write(padding[:paddingLen])

	data := dataBuf.Bytes()

	hk := sha256.New()
	hk.Write(authKey[88:120])
	hk.Write(data)
	var msgKeyLarge [32]byte
	hk.Sum(msgKeyLarge[:0])
	msgKey := msgKeyLarge[8:24]

	keyArr, ivArr := KDF(authKey, msgKey, true)
	encrypted, err := IGEEncrypt(data, keyArr[:], ivArr[:])
	if err != nil {
		return nil, err
	}

	result := bufPool.Get().(*bytes.Buffer)
	result.Reset()
	result.Write(authKeyID)
	result.Write(msgKey)
	result.Write(encrypted)
	ReleaseAESBuf(encrypted)

	out := make([]byte, result.Len())
	copy(out, result.Bytes())
	bufPool.Put(result)
	return out, nil
}

// Unpack decrypts and decodes an incoming encrypted message. It verifies
// authKeyID, decrypts the payload with AES-IGE using KDF-derived keys, validates
// msgKey against SHA-256(authKey[96:128] + decrypted)[8:24], checks sessionID
// and padding constraints, and returns the decoded Message along with the raw
// decrypted bytes.
//
// Returns a *tgerr.SecurityCheckMismatch error on any integrity failure.
//
// See https://core.telegram.org/mtproto/description#encrypted-message.
func Unpack(data []byte, sessionID, authKey, authKeyID []byte) (*tg.MTProtoMessage, []byte, error) {
	// Declare variables before any goto to avoid jumping over declarations
	// (required by Go spec for goto safety).
	var earlyErr *tgerr.SecurityCheckMismatch
	var decrypted []byte
	var msgKey []byte
	var keyArr, ivArr [32]byte
	var err error
	var message *tg.MTProtoMessage
	var decErr error
	var bodyLen int32
	var paddingLen int
	var msgKeyCheck [32]byte
	var hc hash.Hash

	if len(data) < 24 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "data too short"}
	}

	if subtle.ConstantTimeCompare(data[:8], authKeyID) != 1 {
		earlyErr = &tgerr.SecurityCheckMismatch{Name: "b.read(8) == auth_key_id"}
		goto verifyMsgKey
	}

	msgKey = data[8:24]
	keyArr, ivArr = KDF(authKey, msgKey, false)

	if len(data[24:]) == 0 || len(data[24:])%16 != 0 {
		earlyErr = &tgerr.SecurityCheckMismatch{Name: "encrypted data not aligned to 16"}
		goto verifyMsgKey
	}

	decrypted, err = IGEDecrypt(data[24:], keyArr[:], ivArr[:])
	if err != nil {
		earlyErr = &tgerr.SecurityCheckMismatch{Name: "invalid message"}
		goto verifyMsgKey
	}

	if len(decrypted) < 16 {
		earlyErr = &tgerr.SecurityCheckMismatch{Name: "decrypted data too short"}
		goto verifyMsgKey
	}

	hc = sha256.New()
	hc.Write(authKey[96:128])
	hc.Write(decrypted)
	hc.Sum(msgKeyCheck[:0])
	if subtle.ConstantTimeCompare(msgKey, msgKeyCheck[8:24]) != 1 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "msg_key check failed"}
	}

	if subtle.ConstantTimeCompare(decrypted[8:16], sessionID) != 1 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "data.read(8) == session_id"}
	}

	// Validate padding BEFORE decode to prevent OOM from corrupted body length.
	if len(decrypted) < 32 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "decrypted data too short for header"}
	}
	bodyLen = int32(decrypted[28]) | int32(decrypted[29])<<8 |
		int32(decrypted[30])<<16 | int32(decrypted[31])<<24
	paddingLen = len(decrypted) - 32 - int(bodyLen)
	if paddingLen < 12 || paddingLen > 1024 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "12 <= len(padding) <= 1024"}
	}
	if (len(decrypted)-32)%4 != 0 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "len(payload) % 4 == 0"}
	}

	message, decErr = func() (*tg.MTProtoMessage, error) {
		r := tg.NewReader(decrypted[16:])
		defer tg.ReleaseReader(r)
		return tg.DecodeMTProtoMessage(r)
	}()
	if decErr != nil {
		ReleaseAESBuf(decrypted)
		return nil, nil, fmt.Errorf("crypto/mtproto: decode message: %w", decErr)
	}

	return message, decrypted, nil

verifyMsgKey:
	// Run msg_key computation + constant-time comparison before returning
	// any error, so an attacker cannot use timing to distinguish which check
	// failed. Per MTProto security guidelines: if an error is encountered
	// before the msg_key check could be performed, the client MUST perform
	// the msg_key check anyway before returning any result.
	if len(data) >= 24 {
		mk := data[8:24]
		hc := sha256.New()
		hc.Write(authKey[96:128])
		if decrypted != nil {
			hc.Write(decrypted)
		}
		var mkCheck [32]byte
		hc.Sum(mkCheck[:0])
		subtle.ConstantTimeCompare(mk, mkCheck[8:24])
	}
	if decrypted != nil {
		ReleaseAESBuf(decrypted)
	}
	return nil, nil, earlyErr
}

// UnpackEnvelope decrypts and validates an incoming MTProto message but only
// decodes the envelope (msgID, seqNo, raw body bytes) without TL deserialization.
func UnpackEnvelope(data []byte, sessionID, authKey, authKeyID []byte) (*tg.MTProtoMessageRaw, []byte, error) {
	if len(data) < 24 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "data too short"}
	}

	if subtle.ConstantTimeCompare(data[:8], authKeyID) != 1 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "b.read(8) == auth_key_id"}
	}

	msgKey := data[8:24]
	encrypted := data[24:]

	keyArr, ivArr := KDF(authKey, msgKey, false)

	if len(encrypted) == 0 || len(encrypted)%16 != 0 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "encrypted data not aligned to 16"}
	}

	decrypted, err := IGEDecrypt(encrypted, keyArr[:], ivArr[:])
	if err != nil {
		return nil, nil, err
	}

	if len(decrypted) < 16 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "decrypted data too short"}
	}

	hc := sha256.New()
	hc.Write(authKey[96:128])
	hc.Write(decrypted)
	var msgKeyCheck [32]byte
	hc.Sum(msgKeyCheck[:0])
	if subtle.ConstantTimeCompare(msgKey, msgKeyCheck[8:24]) != 1 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "msg_key check failed"}
	}

	if subtle.ConstantTimeCompare(decrypted[8:16], sessionID) != 1 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "session_id mismatch"}
	}

	if len(decrypted) < 32 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "decrypted data too short for header"}
	}
	bodyLen := int32(decrypted[28]) | int32(decrypted[29])<<8 |
		int32(decrypted[30])<<16 | int32(decrypted[31])<<24
	paddingLen := len(decrypted) - 32 - int(bodyLen)
	if paddingLen < 12 || paddingLen > 1024 {
		ReleaseAESBuf(decrypted)
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "padding length"}
	}

	r := tg.NewReader(decrypted[16:])
	raw, err := tg.DecodeMTProtoMessageRaw(r)
	if err != nil {
		tg.ReleaseReader(r)
		ReleaseAESBuf(decrypted)
		return nil, nil, fmt.Errorf("crypto/mtproto: decode envelope: %w", err)
	}

	// Copy BodyRaw before releasing the pooled reader — the sub-slice
	// references the reader's buffer and would be clobbered on reuse.
	if len(raw.BodyRaw) > 0 {
		cp := make([]byte, len(raw.BodyRaw))
		copy(cp, raw.BodyRaw)
		raw.BodyRaw = cp
	}
	tg.ReleaseReader(r)

	return raw, decrypted, nil
}
