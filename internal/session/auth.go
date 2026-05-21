package session

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
)

// AuthResult holds the output of a successful DH key exchange: the shared
// authorization key, the server-assigned salt, and the server's reported time.
type AuthResult struct {
	// AuthKey is the 256-byte shared authorization key.
	AuthKey []byte
	// ServerSalt is the salt value negotiated with the server.
	ServerSalt int64
	// ServerTime is the server's current Unix timestamp.
	ServerTime int32
}

type authTransport interface {
	Send(data []byte) error
	Recv() ([]byte, error)
}

// Auth performs the full MTProto DH key exchange to establish an authorization
// key with a Telegram server.
type Auth struct {
	// DC is the data center ID being authenticated. The zero value defaults to
	// DC 2 for compatibility with older callers.
	DC int
	// TestMode indicates that the connection is using Telegram test servers.
	TestMode bool
}

func (a *Auth) innerDataDC() int32 {
	dc := a.DC
	if dc == 0 {
		dc = 2
	}
	if a.TestMode {
		dc += 10000
	}
	return int32(dc)
}

func (a *Auth) serializeTL(obj tg.TLObject) ([]byte, error) {
	var buf bytes.Buffer
	if err := obj.Encode(&buf); err != nil {
		return nil, fmt.Errorf("serializeTL: %w", err)
	}
	return buf.Bytes(), nil
}

func (a *Auth) deserializeTL(data []byte) (tg.TLObject, error) {
	r := tg.NewReader(data)
	defer tg.ReleaseReader(r)
	obj, err := tg.ReadTLObject(r)
	if err != nil {
		return nil, fmt.Errorf("deserializeTL: %w", err)
	}
	return obj, nil
}

func (a *Auth) sendUnencrypted(conn authTransport, obj tg.TLObject) error {
	payload, err := a.serializeTL(obj)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := packUnencrypted(&buf, payload); err != nil {
		return err
	}
	return conn.Send(buf.Bytes())
}

func (a *Auth) recvUnencrypted(conn authTransport) (tg.TLObject, error) {
	data, err := conn.Recv()
	if err != nil {
		return nil, err
	}

	if len(data) == 4 {
		code := int32(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24)
		return nil, fmt.Errorf("server error code: %d", -code)
	}

	payload, err := unpackUnencrypted(data)
	if err != nil {
		return nil, err
	}
	return a.deserializeTL(payload)
}

func knownFingerprints() []int64 {
	fps := make([]int64, 0, len(crypto.ServerPublicKeys))
	for fp := range crypto.ServerPublicKeys {
		fps = append(fps, fp)
	}
	return fps
}

func (a *Auth) findKeyFingerprint(fingerprints []int64) (int64, bool) {
	for _, fp := range fingerprints {
		if _, ok := crypto.GetServerKey(fp); ok {
			return fp, true
		}
	}
	return 0, false
}

func (a *Auth) generateRandomBN(bits int) (*big.Int, error) {
	b := make([]byte, bits/8)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("session: generate random: %w", err)
	}
	b[0] |= 0x80
	return new(big.Int).SetBytes(b), nil
}

// Create performs the complete MTProto key generation handshake over the
// provided transport: PQ request, DH parameter exchange, and key derivation.
// Returns an AuthResult with the established key, salt, and server time, or an
// error describing which step failed.
func (a *Auth) Create(conn authTransport) (*AuthResult, error) {
	nonce, err := randomInt128()
	if err != nil {
		return nil, fmt.Errorf("step 1: generate nonce: %w", err)
	}

	if err := a.sendUnencrypted(conn, &tg.ReqPQMultiRequest{Nonce: nonce}); err != nil {
		return nil, fmt.Errorf("step 2: send ReqPqMulti: %w", err)
	}

	obj, err := a.recvUnencrypted(conn)
	if err != nil {
		return nil, fmt.Errorf("step 3: recv ResPQ: %w", err)
	}
	resPQ, ok := obj.(*tg.ResPQ)
	if !ok {
		return nil, fmt.Errorf("step 3: expected ResPQ, got %T", obj)
	}
	if resPQ.Nonce != nonce {
		return nil, ErrNonceMismatch
	}

	pqBig := new(big.Int).SetBytes([]byte(resPQ.PQ))
	pqInt64 := pqBig.Int64()
	pVal := crypto.Decompose(pqInt64)
	qVal := pqInt64 / pVal
	if pVal > qVal {
		pVal, qVal = qVal, pVal
	}

	pStr := string(big.NewInt(pVal).Bytes())
	qStr := string(big.NewInt(qVal).Bytes())

	fp, ok := a.findKeyFingerprint(resPQ.ServerPublicKeyFingerprints)
	if !ok {
		return nil, fmt.Errorf("step 5: no matching key fingerprint (server=%v, known=%v)", resPQ.ServerPublicKeyFingerprints, knownFingerprints())
	}

	newNonce, err := randomInt256()
	if err != nil {
		return nil, fmt.Errorf("step 6: generate new_nonce: %w", err)
	}

	pqInnerData := &tg.PQInnerDataDC{
		PQ:          resPQ.PQ,
		P:           pStr,
		Q:           qStr,
		Nonce:       nonce,
		ServerNonce: resPQ.ServerNonce,
		NewNonce:    newNonce,
		DC:          a.innerDataDC(),
	}

	pqInnerBytes, err := a.serializeTL(pqInnerData)
	if err != nil {
		return nil, fmt.Errorf("step 6: serialize PQInnerData: %w", err)
	}

	if len(pqInnerBytes) > 144 {
		return nil, fmt.Errorf("step 6: PQInnerData too large (%d bytes)", len(pqInnerBytes))
	}

	encData, err := crypto.RSAEncrypt(pqInnerBytes, fp)
	if err != nil {
		return nil, fmt.Errorf("step 6: RSA encrypt: %w", err)
	}

	if err := a.sendUnencrypted(conn, &tg.ReqDHParamsRequest{
		Nonce:                nonce,
		ServerNonce:          resPQ.ServerNonce,
		P:                    pStr,
		Q:                    qStr,
		PublicKeyFingerprint: fp,
		EncryptedData:        string(encData),
	}); err != nil {
		return nil, fmt.Errorf("step 7: send ReqDHParams: %w", err)
	}

	obj, err = a.recvUnencrypted(conn)
	if err != nil {
		return nil, fmt.Errorf("step 8: recv ServerDHParams: %w", err)
	}

	switch v := obj.(type) {
	case *tg.ServerDHParamsFail:
		return nil, ErrDHParamsFail
	case *tg.ServerDHParamsOk:
		key, iv := computeKeyAndIV(newNonce[:], resPQ.ServerNonce[:])
		decrypted, err := crypto.IGEDecrypt([]byte(v.EncryptedAnswer), key, iv)
			if err != nil {
				return nil, fmt.Errorf("step 8: decrypt DH params: %w", err)
			}
			defer crypto.ReleaseAESBuf(decrypted)

			serverDHData, err := unwrapDataWithHash(decrypted)
		if err != nil {
			return nil, fmt.Errorf("step 8: unwrap decrypted DH params: %w", err)
		}

		serverDHInner, err := a.deserializeTL(serverDHData)
		if err != nil {
			return nil, fmt.Errorf("step 8: deserialize ServerDHInnerData: %w", err)
		}
		dhInner, ok := serverDHInner.(*tg.ServerDHInnerData)
		if !ok {
			return nil, fmt.Errorf("step 8: expected ServerDHInnerData, got %T", serverDHInner)
		}

		if dhInner.Nonce != nonce {
			return nil, ErrDHNonceMismatch
		}

		if dhInner.ServerNonce != resPQ.ServerNonce {
			return nil, ErrServerNonceMismatch
		}

		if v.ServerNonce != resPQ.ServerNonce {
			return nil, ErrServerNonceMismatch
		}

		dhPrime := new(big.Int).SetBytes([]byte(dhInner.DHPrime))
		if dhPrime.BitLen() != 2048 {
			return nil, fmt.Errorf("%w: bit length %d", ErrDHPrimeInvalid, dhPrime.BitLen())
		}
		if dhPrime.Cmp(crypto.CurrentDHPrime) != 0 {
			if !dhPrime.ProbablyPrime(20) {
				return nil, ErrDHPrimeInvalid
			}
		}

		g := big.NewInt(int64(dhInner.G))
		if g.Int64() < 2 || g.Int64() > 7 {
			return nil, fmt.Errorf("session: invalid DH generator %d", g.Int64())
		}

		gA := new(big.Int).SetBytes([]byte(dhInner.GA))
		one := big.NewInt(1)
		if gA.Cmp(one) <= 0 || gA.Cmp(new(big.Int).Sub(dhPrime, one)) >= 0 {
			return nil, ErrGAOutOfRange
		}

		lowerBound := new(big.Int).Lsh(big.NewInt(1), 2048-64)
		upperBound := new(big.Int).Sub(dhPrime, new(big.Int).Lsh(big.NewInt(1), 2048-64))
		if gA.Cmp(lowerBound) < 0 || gA.Cmp(upperBound) > 0 {
			return nil, ErrGAOutOfRange
		}

		// B8: verify gA is in the correct subgroup
		halfPrime := new(big.Int).Rsh(dhPrime, 1)
		subgroupCheck := new(big.Int).Exp(gA, halfPrime, dhPrime)
		if subgroupCheck.Cmp(one) != 0 {
			return nil, ErrGAOutOfRange
		}

		b, err := a.generateRandomBN(2048)
		if err != nil {
			return nil, fmt.Errorf("step 9: generate DH secret: %w", err)
		}
		gB := new(big.Int).Exp(g, b, dhPrime)

		two := big.NewInt(2)
		if gB.Cmp(two) < 0 || gB.Cmp(new(big.Int).Sub(dhPrime, two)) > 0 {
			return nil, ErrGBOutOfRange
		}

		authKey := computeAuthKey(gA, b, dhPrime)

		authKeyAuxHash := func() int64 {
			h := sha1.Sum(authKey)
			return int64(binary.LittleEndian.Uint64(h[0:8]))
		}

		var retryID int64
		const maxRetries = 5
		for attempt := 0; attempt < maxRetries; attempt++ {
			clientDHInner := &tg.ClientDHInnerData{
				Nonce:       nonce,
				ServerNonce: resPQ.ServerNonce,
				RetryID:     retryID,
				GB:          string(gB.Bytes()),
			}

		clientDHBytes, err := a.serializeTL(clientDHInner)
		if err != nil {
			return nil, fmt.Errorf("step 9: serialize ClientDHInnerData: %w", err)
		}

		clientDHWithHash := sha1Hash(clientDHBytes)
		clientDHWithHash = append(clientDHWithHash, clientDHBytes...)
		padLen := 16 - (len(clientDHWithHash) % 16)
		if padLen < 16 {
			pad := make([]byte, padLen)
			clientDHWithHash = append(clientDHWithHash, pad...)
		}

		encClientDH, err := crypto.IGEEncrypt(clientDHWithHash, key, iv)
			if err != nil {
				return nil, fmt.Errorf("step 9: encrypt client DH: %w", err)
			}
			defer crypto.ReleaseAESBuf(encClientDH)

		if err := a.sendUnencrypted(conn, &tg.SetClientDHParamsRequest{
			Nonce:         nonce,
			ServerNonce:   resPQ.ServerNonce,
			EncryptedData: string(encClientDH),
		}); err != nil {
			return nil, fmt.Errorf("step 9: send SetClientDHParams: %w", err)
		}

		obj, err = a.recvUnencrypted(conn)
		if err != nil {
			return nil, fmt.Errorf("step 10: recv DH answer: %w", err)
		}

		switch v := obj.(type) {
		case *tg.DHGenOk:
			if v.Nonce != nonce {
				return nil, ErrNonceMismatch
			}
			if v.ServerNonce != resPQ.ServerNonce {
				return nil, ErrServerNonceMismatch
			}

			expectedHash := computeNewNonceHash1(newNonce[:], authKey)
			if !bytes.Equal(v.NewNonceHash1[:], expectedHash) {
				return nil, ErrNewNonceHashMismatch
			}

			if len(authKey) < 256 {
				padded := make([]byte, 256)
				copy(padded[256-len(authKey):], authKey)
				authKey = padded
			} else {
				authKey = authKey[:256]
			}

			salt := computeServerSalt(newNonce[:], resPQ.ServerNonce[:])

			return &AuthResult{
				AuthKey:    authKey,
				ServerSalt: salt,
				ServerTime: dhInner.ServerTime,
			}, nil

		case *tg.DHGenRetry:
			if v.ServerNonce != resPQ.ServerNonce {
				return nil, ErrServerNonceMismatch
			}
			retryID = authKeyAuxHash()
			var retryErr error
			b, retryErr = a.generateRandomBN(2048)
			if retryErr != nil {
				return nil, fmt.Errorf("step 10: retry DH secret: %w", retryErr)
			}
			gB = new(big.Int).Exp(g, b, dhPrime)
			authKey = computeAuthKey(gA, b, dhPrime)

		case *tg.DHGenFail:
			if v.ServerNonce != resPQ.ServerNonce {
				return nil, ErrServerNonceMismatch
			}
			return nil, ErrDHGenFail

		default:
			return nil, fmt.Errorf("step 10: unexpected DH answer type %T", obj)
		}
		}
		return nil, ErrDHGenRetry

	default:
		return nil, fmt.Errorf("step 8: unexpected ServerDHParams type %T", obj)
	}
}
