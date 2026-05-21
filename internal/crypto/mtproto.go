package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
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

// Pack serializes, pads, and encrypts a message for transmission. It assembles
// the plaintext (salt + sessionID + encoded message + random padding), computes
// msgKey as SHA-256(authKey[88:120] + plaintext)[8:24], derives the AES key/IV
// via KDF, encrypts with AES-IGE, and returns authKeyID + msgKey + ciphertext.
//
// See https://core.telegram.org/mtproto/description#encrypted-message.
func Pack(message *tg.MTProtoMessage, salt int64, sessionID []byte, authKey, authKeyID []byte) []byte {
	dataBuf := bufPool.Get().(*bytes.Buffer)
	dataBuf.Reset()
	defer bufPool.Put(dataBuf)

	tg.WriteLong(dataBuf, salt)
	dataBuf.Write(sessionID)

	msgBuf := bufPool.Get().(*bytes.Buffer)
	msgBuf.Reset()
	if err := message.Encode(msgBuf); err != nil {
		bufPool.Put(msgBuf)
		panic("crypto/mtproto: " + err.Error())
	}
	dataBuf.Write(msgBuf.Bytes())
	bufPool.Put(msgBuf)

	paddingLen := (-(dataBuf.Len()+12)%16 + 12)
	if paddingLen < 12 {
		paddingLen += 16
	}
	var padding [28]byte
	rand.Read(padding[:paddingLen])
	dataBuf.Write(padding[:paddingLen])

	data := dataBuf.Bytes()

	hk := sha256.New()
	hk.Write(authKey[88:120])
	hk.Write(data)
	var msgKeyLarge [32]byte
	hk.Sum(msgKeyLarge[:0])
	msgKey := msgKeyLarge[8:24]

	keyArr, ivArr := KDF(authKey, msgKey, true)
	encrypted := IGEEncrypt(data, keyArr[:], ivArr[:])

	result := bufPool.Get().(*bytes.Buffer)
	result.Reset()
	result.Write(authKeyID)
	result.Write(msgKey)
	result.Write(encrypted)
	ReleaseAESBuf(encrypted)

	out := make([]byte, result.Len())
	copy(out, result.Bytes())
	bufPool.Put(result)
	return out
}

// PackRaw is like [Pack] but accepts pre-serialized body bytes instead of a
// *tg.MTProtoMessage. It manually writes the MTProto envelope (msgID, seqNo,
// body length prefix) followed by bodyBytes, then applies padding, msgKey
// computation, and AES-IGE encryption identically to [Pack].
func PackRaw(msgID int64, seqNo uint32, bodyBytes []byte, salt int64, sessionID, authKey, authKeyID []byte) []byte {
	dataBuf := bufPool.Get().(*bytes.Buffer)
	dataBuf.Reset()
	defer bufPool.Put(dataBuf)

	tg.WriteLong(dataBuf, salt)
	dataBuf.Write(sessionID)
	tg.WriteLong(dataBuf, msgID)
	tg.WriteInt(dataBuf, seqNo)
	tg.WriteInt(dataBuf, uint32(len(bodyBytes)))
	dataBuf.Write(bodyBytes)

	paddingLen := (-(dataBuf.Len()+12)%16 + 12)
	if paddingLen < 12 {
		paddingLen += 16
	}
	var padding [28]byte
	rand.Read(padding[:paddingLen])
	dataBuf.Write(padding[:paddingLen])

	data := dataBuf.Bytes()

	hk := sha256.New()
	hk.Write(authKey[88:120])
	hk.Write(data)
	var msgKeyLarge [32]byte
	hk.Sum(msgKeyLarge[:0])
	msgKey := msgKeyLarge[8:24]

	keyArr, ivArr := KDF(authKey, msgKey, true)
	encrypted := IGEEncrypt(data, keyArr[:], ivArr[:])

	result := bufPool.Get().(*bytes.Buffer)
	result.Reset()
	result.Write(authKeyID)
	result.Write(msgKey)
	result.Write(encrypted)
	ReleaseAESBuf(encrypted)

	out := make([]byte, result.Len())
	copy(out, result.Bytes())
	bufPool.Put(result)
	return out
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

	decrypted := IGEDecrypt(encrypted, keyArr[:], ivArr[:])

	if len(decrypted) < 16 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "decrypted data too short"}
	}

	hc := sha256.New()
	hc.Write(authKey[96:128])
	hc.Write(decrypted)
	var msgKeyCheck [32]byte
	hc.Sum(msgKeyCheck[:0])
	if subtle.ConstantTimeCompare(msgKey, msgKeyCheck[8:24]) != 1 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "msg_key == sha256(auth_key[96:128] + data)[8:24]"}
	}

	if subtle.ConstantTimeCompare(decrypted[8:16], sessionID) != 1 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "data.read(8) == session_id"}
	}

	// Validate padding BEFORE decode to prevent OOM from corrupted body length.
	if len(decrypted) < 32 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "decrypted data too short for header"}
	}
	bodyLen := int32(decrypted[28]) | int32(decrypted[29])<<8 |
		int32(decrypted[30])<<16 | int32(decrypted[31])<<24
	paddingLen := len(decrypted) - 32 - int(bodyLen)
	if paddingLen < 12 || paddingLen > 1024 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "12 <= len(padding) <= 1024"}
	}
	if (len(decrypted)-32)%4 != 0 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "len(payload) % 4 == 0"}
	}

	message, err := func() (*tg.MTProtoMessage, error) {
		r := tg.NewReader(decrypted[16:])
		defer tg.ReleaseReader(r)
		return tg.DecodeMTProtoMessage(r)
	}()
	if err != nil {
		return nil, nil, fmt.Errorf("crypto/mtproto: decode message: %w", err)
	}

	return message, decrypted, nil
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

	decrypted := IGEDecrypt(encrypted, keyArr[:], ivArr[:])

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
