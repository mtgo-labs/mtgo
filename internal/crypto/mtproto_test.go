package crypto

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestKDF(t *testing.T) {
	authKey := make([]byte, 1024)
	for i := range authKey {
		authKey[i] = byte(i)
	}
	msgKey := make([]byte, 16)
	for i := range msgKey {
		msgKey[i] = byte(255 - i)
	}

	aesKeyOut, aesIVOut := KDF(authKey, msgKey, true)
	aesKeyIn, _ := KDF(authKey, msgKey, false)

	if bytes.Equal(aesKeyOut[:], aesKeyIn[:]) {
		t.Fatal("outgoing and incoming keys should differ")
	}

	// Verify against spec calculation
	x := 0
	sha256A := sha256.Sum256(append(msgKey, authKey[x:x+36]...))
	sha256B := sha256.Sum256(append(authKey[x+40:x+76], msgKey...))

	expectedKey := make([]byte, 0, 32)
	expectedKey = append(expectedKey, sha256A[:8]...)
	expectedKey = append(expectedKey, sha256B[8:24]...)
	expectedKey = append(expectedKey, sha256A[24:32]...)
	expectedIV := make([]byte, 0, 32)
	expectedIV = append(expectedIV, sha256B[:8]...)
	expectedIV = append(expectedIV, sha256A[8:24]...)
	expectedIV = append(expectedIV, sha256B[24:32]...)

	if !bytes.Equal(aesKeyOut[:], expectedKey) {
		t.Fatal("outgoing key mismatch")
	}
	if !bytes.Equal(aesIVOut[:], expectedIV) {
		t.Fatal("outgoing IV mismatch")
	}
}

func TestKDFIncoming(t *testing.T) {
	authKey := make([]byte, 1024)
	msgKey := make([]byte, 16)
	for i := range msgKey {
		msgKey[i] = byte(i)
	}

	key, iv := KDF(authKey, msgKey, false)

	if len(key) != 32 || len(iv) != 32 {
		t.Fatal("KDF should return 32-byte key and IV")
	}
	_ = key
	_ = iv
}

func TestPackUnpackRoundTrip(t *testing.T) {
	authKey := make([]byte, 1024)
	sessionID := make([]byte, 8)
	for i := range sessionID {
		sessionID[i] = byte(0xAB)
	}
	salt := int64(123456789)

	h := sha256.Sum256(authKey)
	authKeyIDBytes := h[:8]

	msg := &tg.MTProtoMessage{
		MsgID: 0x600000000000000B,
		SeqNo: 1,
		Body:  tg.TLBool(true),
	}

	packed, err := Pack(msg, salt, sessionID, authKey, authKeyIDBytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(packed) < 24 {
			t.Fatal("packed data too short")
		}

	if !bytes.Equal(packed[:8], authKeyIDBytes) {
		t.Fatal("auth_key_id mismatch")
	}

	recovered, decrypted, err := Unpack(packed, sessionID, authKey, authKeyIDBytes)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	if recovered.MsgID != msg.MsgID {
		t.Fatalf("MsgID mismatch: got %d, want %d", recovered.MsgID, msg.MsgID)
	}
	if recovered.SeqNo != msg.SeqNo {
		t.Fatalf("SeqNo mismatch: got %d, want %d", recovered.SeqNo, msg.SeqNo)
	}

	if len(decrypted) == 0 {
		t.Fatal("decrypted data should not be empty")
	}
}

func TestPackUnpackSecurityChecks(t *testing.T) {
	authKey := make([]byte, 1024)
	authKeyIDFull := sha256.Sum256(authKey[96:128])
	authKeyIDBytes := authKeyIDFull[:8]
	sessionID := make([]byte, 8)

	msg := &tg.MTProtoMessage{
		MsgID: 0x600000000000000B,
		SeqNo: 1,
		Body:  tg.TLBool(true),
	}

	packed, err := Pack(msg, 0, sessionID, authKey, authKeyIDBytes)
		if err != nil {
			t.Fatal(err)
		}

		tampered := make([]byte, len(packed))
	copy(tampered, packed)
	tampered[0] ^= 0xFF

	_, _, err = Unpack(tampered, sessionID, authKey, authKeyIDBytes)
	if err == nil {
		t.Fatal("expected error on auth_key_id mismatch")
	}
}

func TestPackUnpackSessionIDMismatch(t *testing.T) {
	authKey := make([]byte, 1024)
	authKeyIDFull := sha256.Sum256(authKey[96:128])
	authKeyIDBytes := authKeyIDFull[:8]
	sessionID := make([]byte, 8)

	msg := &tg.MTProtoMessage{
		MsgID: 0x600000000000000B,
		SeqNo: 1,
		Body:  tg.TLBool(true),
	}

	packed, err := Pack(msg, 0, sessionID, authKey, authKeyIDBytes)
		if err != nil {
			t.Fatal(err)
		}

		wrongSessionID := make([]byte, 8)
	wrongSessionID[0] = 0xFF

	_, _, err = Unpack(packed, wrongSessionID, authKey, authKeyIDBytes)
	if err == nil {
		t.Fatal("expected error on session_id mismatch")
	}
}
