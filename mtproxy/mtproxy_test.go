package mtproxy

import (
	"testing"
)

func TestParseSecretSimple(t *testing.T) {
	hex := "52a493bdfb90eea55739eabff2d92a14"
	s, err := ParseSecret(hex)
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != SecretSimple {
		t.Errorf("type = %d, want SecretSimple", s.Type)
	}
	if len(s.Secret) != 16 {
		t.Errorf("secret len = %d, want 16", len(s.Secret))
	}
	if s.NeedsFakeTLS() {
		t.Error("simple secret should not need fake TLS")
	}
}

func TestParseSecretDD(t *testing.T) {
	hex := "ddf05fb7acb549be047a7c585116581418"
	s, err := ParseSecret(hex)
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != SecretSecured {
		t.Errorf("type = %d, want SecretSecured", s.Type)
	}
	if s.Tag != 0xdd {
		t.Errorf("tag = 0x%02x, want 0xdd", s.Tag)
	}
	if s.Codec() != 0xdd {
		t.Errorf("codec = 0x%02x, want 0xdd", s.Codec())
	}
	if s.NeedsFakeTLS() {
		t.Error("dd secret should not need fake TLS")
	}
}

func TestParseSecretEE(t *testing.T) {
	hex := "ee852380f362a09343efb4690c4e17862e676f6f676c652e636f6d"
	s, err := ParseSecret(hex)
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != SecretTLS {
		t.Errorf("type = %d, want SecretTLS", s.Type)
	}
	if s.Tag != 0xee {
		t.Errorf("tag = 0x%02x, want 0xee", s.Tag)
	}
	if s.Domain != "google.com" {
		t.Errorf("domain = %q, want %q", s.Domain, "google.com")
	}
	if !s.NeedsFakeTLS() {
		t.Error("ee secret should need fake TLS")
	}
}

func TestParseSecretBytes(t *testing.T) {
	raw := append([]byte{0xdd}, make([]byte, 16)...)
	s, err := ParseSecretBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	if s.Type != SecretSecured {
		t.Errorf("type = %d, want SecretSecured", s.Type)
	}
	if s.Tag != 0xdd {
		t.Errorf("tag = 0x%02x, want 0xdd", s.Tag)
	}
}

func TestParseSecretInvalid(t *testing.T) {
	_, err := ParseSecret("zzzz")
	if err == nil {
		t.Error("expected error for invalid hex")
	}

	_, err = ParseSecretBytes([]byte{0x01, 0x02})
	if err == nil {
		t.Error("expected error for too-short secret")
	}
}

func TestBuildClientHello(t *testing.T) {
	secret := make([]byte, 16)
	for i := range secret {
		secret[i] = byte(i)
	}
	hello, clientRandom, err := buildClientHello(secret, "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(hello) != 517 {
		t.Errorf("hello len = %d, want 517", len(hello))
	}
	if len(clientRandom) != 32 {
		t.Errorf("clientRandom len = %d, want 32", len(clientRandom))
	}
	if hello[0] != 0x16 {
		t.Errorf("hello[0] = 0x%02x, want 0x16 (handshake)", hello[0])
	}
	if hello[5] != 0x01 {
		t.Errorf("hello[5] = 0x%02x, want 0x01 (ClientHello)", hello[5])
	}
	if hello[9] != 0x03 || hello[10] != 0x03 {
		t.Errorf("version = %x, want 0303 (TLS 1.2)", hello[9:11])
	}
}

func TestGenerateFakeKeyShare(t *testing.T) {
	key := generateFakeKeyShare()
	if len(key) != 32 {
		t.Errorf("key share len = %d, want 32", len(key))
	}
}
