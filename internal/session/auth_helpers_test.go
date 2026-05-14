package session

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"math/big"
	"testing"
)

func TestAuthSHA1(t *testing.T) {
	data := []byte("hello world")
	got := sha1Hash(data)
	expected := sha1.Sum(data)
	if !bytes.Equal(got, expected[:]) {
		t.Errorf("sha1Hash mismatch")
	}
}

func TestAuthXORBytes(t *testing.T) {
	a := []byte{0x0f, 0xff, 0x00, 0x0a}
	b := []byte{0xf0, 0x0f, 0x00, 0xa0}
	got := xorBytes(a, b)
	want := []byte{0xff, 0xf0, 0x00, 0xaa}
	if !bytes.Equal(got, want) {
		t.Errorf("xorBytes = %x, want %x", got, want)
	}
}

func TestAuthXORBytesDifferentLengths(t *testing.T) {
	a := []byte{0x01, 0x02, 0x03}
	b := []byte{0xff}
	got := xorBytes(a, b)
	if len(got) != 1 {
		t.Fatalf("xorBytes length = %d, want 1", len(got))
	}
	if got[0] != 0xfe {
		t.Errorf("xorBytes[0] = %x, want 0xfe", got[0])
	}
}

func TestAuthBigEndianInt128(t *testing.T) {
	val, err := randomInt128()
	if err != nil {
		t.Fatalf("randomInt128 error: %v", err)
	}
	if val == [16]byte{} {
		t.Error("randomInt128 returned zero")
	}
	val2, err := randomInt128()
	if err != nil {
		t.Fatalf("randomInt128 error: %v", err)
	}
	if val == val2 {
		t.Error("two randomInt128 calls returned same value")
	}
}

func TestAuthBigEndianInt256(t *testing.T) {
	val, err := randomInt256()
	if err != nil {
		t.Fatalf("randomInt256 error: %v", err)
	}
	if val == [32]byte{} {
		t.Error("randomInt256 returned zero")
	}
}

func TestAuthComputeKeyAndIV(t *testing.T) {
	newNonce := make([]byte, 32)
	serverNonce := make([]byte, 16)
	for i := range newNonce {
		newNonce[i] = byte(i)
	}
	for i := range serverNonce {
		serverNonce[i] = byte(i + 32)
	}

	key, iv := computeKeyAndIV(newNonce, serverNonce)

	if len(key) != 32 {
		t.Errorf("key length = %d, want 32", len(key))
	}
	if len(iv) != 32 {
		t.Errorf("iv length = %d, want 32", len(iv))
	}

	hash1 := sha1.Sum(append(newNonce, serverNonce...))
	hash2 := sha1.Sum(append(serverNonce, newNonce...))
	hash3 := sha1.Sum(append(newNonce, newNonce...))

	var expectedKey [32]byte
	copy(expectedKey[0:20], hash1[:])
	copy(expectedKey[20:32], hash2[0:12])

	if !bytes.Equal(key, expectedKey[:]) {
		t.Errorf("key mismatch\ngot:  %x\nwant: %x", key, expectedKey)
	}

	var expectedIV [32]byte
	copy(expectedIV[0:8], hash2[12:20])
	copy(expectedIV[8:28], hash3[:])
	copy(expectedIV[28:32], newNonce[0:4])

	if !bytes.Equal(iv, expectedIV[:]) {
		t.Errorf("iv mismatch\ngot:  %x\nwant: %x", iv, expectedIV)
	}
}

func TestAuthComputeNewNonceHash1(t *testing.T) {
	newNonce := make([]byte, 32)
	for i := range newNonce {
		newNonce[i] = byte(i)
	}
	authKey := make([]byte, 256)
	for i := range authKey {
		authKey[i] = byte(i)
	}

	got := computeNewNonceHash1(newNonce, authKey)

	authKeyHash := sha1.Sum(authKey)
	buf := make([]byte, len(newNonce)+1+8)
	copy(buf, newNonce)
	buf[len(newNonce)] = 1
	copy(buf[len(newNonce)+1:], authKeyHash[:8])
	h := sha1.Sum(buf)
	want := h[4:20]

	if !bytes.Equal(got, want) {
		t.Errorf("computeNewNonceHash1 = %x, want %x", got, want)
	}
	if len(got) != 16 {
		t.Errorf("computeNewNonceHash1 length = %d, want 16", len(got))
	}
}

func TestAuthUnwrapDataWithHash(t *testing.T) {
	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	hash := sha1.Sum(payload)
	dataWithHash := append(hash[:], payload...)
	dataWithHash = append(dataWithHash, []byte{0, 1, 2, 3, 4}...)

	got, err := unwrapDataWithHash(dataWithHash)
	if err != nil {
		t.Fatalf("unwrapDataWithHash error: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unwrapDataWithHash = %x, want %x", got, payload)
	}
}

func TestAuthUnwrapDataWithHashRejectsMismatch(t *testing.T) {
	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	hash := sha1.Sum([]byte("different"))
	dataWithHash := append(hash[:], payload...)

	if _, err := unwrapDataWithHash(dataWithHash); err == nil {
		t.Fatal("expected unwrapDataWithHash to reject mismatched hash")
	}
}

func TestAuthComputeServerSalt(t *testing.T) {
	newNonce := make([]byte, 8)
	serverNonce := make([]byte, 8)
	binary.LittleEndian.PutUint64(newNonce, 0x0102030405060708)
	binary.LittleEndian.PutUint64(serverNonce, 0x0a0b0c0d0e0f1011)

	salt := computeServerSalt(newNonce, serverNonce)

	nonceVal := int64(0x0102030405060708)
	serverVal := int64(0x0a0b0c0d0e0f1011)
	expected := nonceVal ^ serverVal
	if salt != expected {
		t.Errorf("serverSalt = %x, want %x", salt, expected)
	}
}

func TestAuthComputeAuthKey(t *testing.T) {
	dhPrime, _ := new(big.Int).SetString("0x00", 0)
	dhPrime.SetString("C71CAEB9C6B1C9048E6C522F70F13F73980D40238E3E21C14934D037563D930F48198A0AA7C14058229493D22530F4DBFA336F6E0AC925139543AED44CCE7C3720FD51F69458705AC68CD4FE6B6B13ABDC9746512969328454F18FAF8C595F642477FE96BB2A941D5BCD1D4AC8CC49880708FA9B378E3C4F3A9060BEE67CF9A4A4A695811051907E162753B56B0F6B410DBA74D8A84B2A14B3144E0EF1284754FD17ED950D5965B4B9DD46582DB1178D169C6DC4D0A9E5F8B4514139CB95B06891C35F2B87E3E04B7A3F0D0C3A515F7DBEA0E1ADFD3C08EEC0E0A", 16)

	gA := big.NewInt(3)
	b := big.NewInt(5)

	result := computeAuthKey(gA, b, dhPrime)

	expected := new(big.Int).Exp(gA, b, dhPrime)
	if !bytes.Equal(result, expected.Bytes()) {
		t.Errorf("computeAuthKey mismatch")
	}
}

func TestAuthInnerDataDC(t *testing.T) {
	tests := []struct {
		name string
		auth Auth
		want int32
	}{
		{
			name: "zero value defaults to production DC2",
			auth: Auth{},
			want: 2,
		},
		{
			name: "production DC",
			auth: Auth{DC: 4},
			want: 4,
		},
		{
			name: "test DC",
			auth: Auth{DC: 2, TestMode: true},
			want: 10002,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.auth.innerDataDC(); got != tt.want {
				t.Fatalf("innerDataDC() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAuthPackUnpackUnencrypted(t *testing.T) {
	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	var buf bytes.Buffer

	if err := packUnencrypted(&buf, payload); err != nil {
		t.Fatalf("packUnencrypted error: %v", err)
	}

	data := buf.Bytes()
	if len(data) != 20+len(payload) {
		t.Fatalf("packed length = %d, want %d", len(data), 20+len(payload))
	}

	authKeyID := binary.LittleEndian.Uint64(data[0:8])
	if authKeyID != 0 {
		t.Errorf("auth_key_id = %d, want 0", authKeyID)
	}

	msgLen := binary.LittleEndian.Uint32(data[16:20])
	if msgLen != uint32(len(payload)) {
		t.Errorf("msg_len = %d, want %d", msgLen, len(payload))
	}

	unpacked, err := unpackUnencrypted(data)
	if err != nil {
		t.Fatalf("unpackUnencrypted error: %v", err)
	}
	if !bytes.Equal(unpacked, payload) {
		t.Errorf("unpacked = %x, want %x", unpacked, payload)
	}
}

func TestAuthUnpackTooShort(t *testing.T) {
	_, err := unpackUnencrypted([]byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for short packet")
	}
}

func TestAuthUnpackNonZeroAuthKeyID(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint64(data[0:8], 42)
	_, err := unpackUnencrypted(data)
	if err == nil {
		t.Error("expected error for non-zero auth_key_id")
	}
}
