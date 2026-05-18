package session

import "errors"

// Session connection and authentication errors.
//
// These errors are returned during MTProto session establishment (key
// exchange, transport setup, and the DH handshake).
var (
	// ErrAuthKeyNotSet is returned when an operation requires an
	// authorization key but none has been generated or loaded yet.
	ErrAuthKeyNotSet = errors.New("session: auth key not set")
	// ErrTransportNotSet is returned when an operation requires a transport
	// (TCP, WebSocket, etc.) but none has been configured.
	ErrTransportNotSet = errors.New("session: transport not set")
	// ErrSendTimeout is returned when sending a message to the server
	// exceeds the configured write deadline.
	ErrSendTimeout = errors.New("session: send timeout")
	// ErrSessionClosed is returned when the session has been stopped and
	// pending operations are cancelled.
	ErrSessionClosed = errors.New("session: closed")
	// ErrConnectNoAuthKey is returned when Connect is called before an
	// authorization key has been generated via key generation.
	ErrConnectNoAuthKey = errors.New("session: connect: no auth key")

	// ErrSHA1Mismatch is returned during the DH key exchange when the SHA1
	// hash of the received data does not match the expected value.
	ErrSHA1Mismatch = errors.New("session: sha1 hash mismatch")
	// ErrNonceMismatch is returned during key exchange step 3 when the
	// server's response nonce does not match the nonce sent by the client.
	ErrNonceMismatch = errors.New("step 3: nonce mismatch")
	// ErrDHParamsFail is returned during key exchange step 8 when the server
	// responds with a DH parameter failure.
	ErrDHParamsFail = errors.New("step 8: server dh params fail")
	// ErrDHNonceMismatch is returned during key exchange step 8 when the
	// nonce in the DH inner data does not match the expected value.
	ErrDHNonceMismatch = errors.New("step 8: nonce mismatch in dh inner data")
	// ErrNewNonceHashMismatch is returned during key exchange step 10 when
	// the new_nonce_hash1 value does not match the expected hash.
	ErrNewNonceHashMismatch = errors.New("step 10: new_nonce_hash1 mismatch")
	// ErrDHGenRetry is returned during key exchange step 10 when the server
	// responds with dh_gen_retry, indicating the client should retry with a
	// new nonce.
	ErrDHGenRetry = errors.New("step 10: dh_gen_retry")
	// ErrDHGenFail is returned during key exchange step 10 when the server
	// responds with dh_gen_fail, indicating the DH key generation failed.
	ErrDHGenFail = errors.New("step 10: dh_gen_fail")
)
