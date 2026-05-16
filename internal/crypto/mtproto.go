package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
	tgerr "github.com/mtgo-labs/mtgo/tgerr"
)

// KDF derives the AES key and IV used for message encryption from the
// authorization key and message key. The outgoing parameter determines which
// offset into authKey is used: offset 0 for client-to-server messages and
// offset 8 for server-to-client messages.
//
// Returns the 32-byte aesKey and 32-byte aesIV.
//
// See https://core.telegram.org/mtproto/description#defining-aes-key-and-initialization-vector.
func KDF(authKey, msgKey []byte, outgoing bool) (aesKey, aesIV []byte) {
	x := 0
	if !outgoing {
		x = 8
	}

	tmpA := make([]byte, len(msgKey)+36)
	copy(tmpA, msgKey)
	copy(tmpA[len(msgKey):], authKey[x:x+36])
	sha256A := sha256.Sum256(tmpA)

	tmpB := make([]byte, 36+len(msgKey))
	copy(tmpB, authKey[x+40:x+76])
	copy(tmpB[36:], msgKey)
	sha256B := sha256.Sum256(tmpB)

	aesKey = make([]byte, 0, 32)
	aesKey = append(aesKey, sha256A[:8]...)
	aesKey = append(aesKey, sha256B[8:24]...)
	aesKey = append(aesKey, sha256A[24:32]...)

	aesIV = make([]byte, 0, 32)
	aesIV = append(aesIV, sha256B[:8]...)
	aesIV = append(aesIV, sha256A[8:24]...)
	aesIV = append(aesIV, sha256B[24:32]...)

	return aesKey, aesIV
}

// Pack serializes, pads, and encrypts a message for transmission. It assembles
// the plaintext (salt + sessionID + encoded message + random padding), computes
// msgKey as SHA-256(authKey[88:120] + plaintext)[8:24], derives the AES key/IV
// via KDF, encrypts with AES-IGE, and returns authKeyID + msgKey + ciphertext.
//
// See https://core.telegram.org/mtproto/description#encrypted-message.
func Pack(message *tg.MTProtoMessage, salt int64, sessionID []byte, authKey, authKeyID []byte) []byte {
	var dataBuf bytes.Buffer
	tg.WriteLong(&dataBuf, salt)
	dataBuf.Write(sessionID)

	var msgBuf bytes.Buffer
	if err := message.Encode(&msgBuf); err != nil {
		panic("crypto/mtproto: " + err.Error())
	}
	dataBuf.Write(msgBuf.Bytes())

	paddingLen := (-(len(dataBuf.Bytes())+12)%16 + 12)
	if paddingLen < 12 {
		paddingLen += 16
	}
	padding := make([]byte, paddingLen)
	rand.Read(padding)
	dataBuf.Write(padding)

	data := dataBuf.Bytes()

	tmpKey := make([]byte, 32+len(data))
	copy(tmpKey, authKey[88:120])
	copy(tmpKey[32:], data)
	msgKeyLarge := sha256.Sum256(tmpKey)
	msgKey := msgKeyLarge[8:24]

	aesKey, aesIV := KDF(authKey, msgKey, true)
	encrypted := IGEEncrypt(data, aesKey, aesIV)

	var result bytes.Buffer
	result.Write(authKeyID)
	result.Write(msgKey)
	result.Write(encrypted)

	return result.Bytes()
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

	if !bytes.Equal(data[:8], authKeyID) {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "b.read(8) == auth_key_id"}
	}

	msgKey := data[8:24]
	encrypted := data[24:]

	aesKey, aesIV := KDF(authKey, msgKey, false)

	if len(encrypted) == 0 || len(encrypted)%16 != 0 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "encrypted data not aligned to 16"}
	}

	decrypted := IGEDecrypt(encrypted, aesKey, aesIV)

	if len(decrypted) < 16 {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "decrypted data too short"}
	}

	tmpChk := make([]byte, 32+len(decrypted))
	copy(tmpChk, authKey[96:128])
	copy(tmpChk[32:], decrypted)
	msgKeyCheck := sha256.Sum256(tmpChk)
	if !bytes.Equal(msgKey, msgKeyCheck[8:24]) {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "msg_key == sha256(auth_key[96:128] + data)[8:24]"}
	}

	if !bytes.Equal(decrypted[8:16], sessionID) {
		return nil, nil, &tgerr.SecurityCheckMismatch{Name: "data.read(8) == session_id"}
	}

	message, err := tg.DecodeMTProtoMessage(bytes.NewReader(decrypted[16:]))
	if err != nil {
		return nil, nil, fmt.Errorf("crypto/mtproto: decode message: %w", err)
	}

	// Validate padding
	// Layout: salt(8) + session_id(8) + msg_id(8) + seq_no(4) + body_len(4) + body + padding
	// body_len is at decrypted[28:32]
	if len(decrypted) >= 32 {
		bodyLen := int32(decrypted[28]) | int32(decrypted[29])<<8 |
			int32(decrypted[30])<<16 | int32(decrypted[31])<<24
		paddingLen := len(decrypted) - 32 - int(bodyLen)
		if paddingLen < 12 || paddingLen > 1024 {
			return nil, nil, &tgerr.SecurityCheckMismatch{Name: "12 <= len(padding) <= 1024"}
		}
		if (len(decrypted)-32)%4 != 0 {
			return nil, nil, &tgerr.SecurityCheckMismatch{Name: "len(payload) % 4 == 0"}
		}
	}

	return message, decrypted, nil
}
