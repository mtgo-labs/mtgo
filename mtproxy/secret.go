package mtproxy

import (
	"encoding/hex"
	"fmt"
)

type SecretType int

const (
	SecretSimple  SecretType = iota
	SecretSecured            // dd-prefix: tag(1) + secret(16)
	SecretTLS                // ee-prefix: tag(1) + secret(16) + domain
)

type Secret struct {
	Type   SecretType
	Secret []byte // 16-byte secret
	Tag    byte   // protocol tag (0xdd, 0xee, 0xef)
	Domain string // SNI domain (ee-secrets only)
}

func ParseSecret(s string) (Secret, error) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return Secret{}, fmt.Errorf("mtproxy: invalid hex secret: %w", err)
	}
	return ParseSecretBytes(raw)
}

func ParseSecretBytes(raw []byte) (Secret, error) {
	switch {
	case len(raw) == 16:
		return Secret{Type: SecretSimple, Secret: raw}, nil

	case len(raw) == 17:
		tag := raw[0]
		secret := make([]byte, 16)
		copy(secret, raw[1:17])
		return Secret{Type: SecretSecured, Secret: secret, Tag: tag}, nil

	case len(raw) > 17:
		tag := raw[0]
		secret := make([]byte, 16)
		copy(secret, raw[1:17])
		domain := string(raw[17:])
		return Secret{Type: SecretTLS, Secret: secret, Tag: tag, Domain: domain}, nil

	default:
		return Secret{}, ErrInvalidSecretLen
	}
}

func (s Secret) Codec() byte {
	switch s.Type {
	case SecretSimple:
		return 0xee
	case SecretSecured:
		return s.Tag
	case SecretTLS:
		if s.Tag != 0 {
			return s.Tag
		}
		return 0xee
	default:
		return 0xee
	}
}

func (s Secret) NeedsFakeTLS() bool {
	return s.Type == SecretTLS
}
