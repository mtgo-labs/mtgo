package telegram

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tgerr"
)

// SendCodeResult holds the outcome of an auth code request sent to a phone number.
// Use the PhoneCodeHash when calling SignIn or SignUp to correlate the code with
// the original request. Inspect NextType and Timeout to determine whether the user
// can request a different delivery method and how long to wait before resending.
//
// Example:
//
//	result, err := client.SendCode(ctx, "+1234567890")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("code hash:", result.PhoneCodeHash)
//	fmt.Println("timeout:", result.Timeout)
type SendCodeResult struct {
	// PhoneCodeHash is the server-generated identifier for this code request.
	// It must be passed verbatim to SignIn or SignUp; without it the server
	// cannot validate the code.
	PhoneCodeHash string
	// Type describes the delivery method used for the current code (e.g. SMS,
	// call, flash call, fragment). Use this to inform the user how the code was sent.
	Type tg.SentCodeTypeClass
	// NextType indicates an alternative code delivery method the user may request
	// via another SendCode call. It is nil when no alternative method is available.
	NextType tg.CodeTypeClass
	// Timeout is the number of seconds the user should wait before requesting
	// a new code. A value of 0 means no timeout was specified by the server.
	Timeout int
}

// SendCode requests Telegram to send a verification code to the given phone number.
// This is the first step in the authentication flow and must be called before SignIn
// or SignUp.
//
// Returns a SendCodeResult containing the phone code hash needed for subsequent calls.
// Returns an error if the request fails due to invalid phone number, rate limiting
// (FloodWait), or network issues. Returns an error wrapping "already authenticated"
// if the session is already authorized.
//
// Example:
//
//	ctx := context.Background()
//	result, err := client.SendCode(ctx, "+1234567890")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("phone code hash:", result.PhoneCodeHash)
func (c *Client) SendCode(ctx context.Context, phoneNumber string) (*SendCodeResult, error) {
	c.Log.Debug("auth: SendCode")
	rpc := c.Raw()
	result, err := rpc.AuthSendCode(ctx, &tg.AuthSendCodeRequest{
		PhoneNumber: phoneNumber,
		APIID:       c.cfg.APIID,
		APIHash:     c.cfg.APIHash,
		Settings:    &tg.CodeSettings{},
	})
	if err != nil {
		c.Log.Warnf("auth: SendCode failed err=%v", err)
		return nil, err
	}
	switch v := result.(type) {
	case *tg.AuthSentCode:
		scr := &SendCodeResult{
			PhoneCodeHash: v.PhoneCodeHash,
			Type:          v.Type,
		}
		if v.NextType != nil {
			scr.NextType = v.NextType
		}
		if v.Timeout != 0 {
			scr.Timeout = int(v.Timeout)
		}
		return scr, nil
	case *tg.AuthSentCodeSuccess:
		return nil, ErrAlreadyAuthed
	default:
		return nil, fmt.Errorf("unexpected send code result type %T", result)
	}
}

// SignIn authenticates a user with the phone number, verification code, and the
// phone code hash returned by SendCode. Call this after the user receives and enters
// the code delivered to their device.
//
// Returns the authenticated User on success.
// Returns an error wrapping "2FA is enabled: use CheckPassword instead" when the
// account has two-factor authentication enabled and CheckPassword must be called
// with the account password.
// Returns an error wrapping "sign up required: use SignUp method" when the phone
// number is not yet registered on Telegram and SignUp must be called instead.
//
// Example:
//
//	user, err := client.SignIn(ctx, "+1234567890", result.PhoneCodeHash, "12345")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("signed in as:", user.FirstName)
func (c *Client) SignIn(ctx context.Context, phoneNumber, phoneCodeHash, phoneCode string) (*types.User, error) {
	c.Log.Debug("auth: SignIn")
	rpc := c.Raw()
	result, err := rpc.AuthSignIn(ctx, &tg.AuthSignInRequest{
		PhoneNumber:   phoneNumber,
		PhoneCodeHash: phoneCodeHash,
		PhoneCode:     phoneCode,
	})
	if err != nil {
		if tgerr.Is(err, tgerr.ErrSessionPasswordNeeded) {
			c.Log.Debug("auth: 2FA required")
			return nil, Err2FARequired
		}
		c.Log.Warnf("auth: SignIn failed err=%v", err)
		return nil, err
	}
	switch v := result.(type) {
	case *tg.AuthAuthorization:
		c.Log.Info("auth: SignIn successful")
		user := types.ParseUser(v.User)
		c.SetMe(user)
		c.saveMeToStorage(user)
		return user, nil
	case *tg.AuthAuthorizationSignUpRequired:
		return nil, ErrSignUpRequired
	default:
		return nil, fmt.Errorf("unexpected sign in result type %T", result)
	}
}

// SignUp registers a new Telegram account for a phone number that is not yet
// associated with any account. This must be called only after SignIn returns a
// "sign up required" error, reusing the same phoneCodeHash from the original
// SendCode call.
//
// lastName is optional; only the first element is used if provided.
//
// Returns the newly created User on success.
// Returns an error if the phone code hash is invalid, the code has expired,
// or the sign-up is otherwise rejected by the server.
//
// Example:
//
//	user, err := client.SignUp(ctx, "+1234567890", result.PhoneCodeHash, "John", "Doe")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("registered as:", user.FirstName, user.LastName)
func (c *Client) SignUp(ctx context.Context, phoneNumber, phoneCodeHash, firstName string, lastName ...string) (*types.User, error) {
	c.Log.Debug("auth: SignUp")
	ln := ""
	if len(lastName) > 0 {
		ln = lastName[0]
	}
	rpc := c.Raw()
	result, err := rpc.AuthSignUp(ctx, &tg.AuthSignUpRequest{
		PhoneNumber:   phoneNumber,
		PhoneCodeHash: phoneCodeHash,
		FirstName:     firstName,
		LastName:      ln,
	})
	if err != nil {
		c.Log.Warnf("auth: SignUp failed err=%v", err)
		return nil, err
	}
	switch v := result.(type) {
	case *tg.AuthAuthorization:
		c.Log.Info("auth: SignUp successful")
		user := types.ParseUser(v.User)
		c.SetMe(user)
		c.saveMeToStorage(user)
		return user, nil
	default:
		return nil, fmt.Errorf("unexpected sign up result type %T", result)
	}
}

// SignOut logs the current session out of Telegram and disconnects the client
// from the server. After a successful sign-out the session is invalidated and
// cannot be reused; a new authentication flow is required to reconnect.
//
// Returns (true, nil) on success.
// Returns (false, err) if the server rejects the log-out request or a network
// error occurs. Note that Disconnect errors are silently ignored since the
// session is already invalidated server-side.
//
// Example:
//
//	ok, err := client.SignOut(ctx)
//	if err != nil {
//	    log.Printf("sign out failed: %v", err)
//	    return
//	}
//	fmt.Println("signed out:", ok)
func (c *Client) SignOut(ctx context.Context) (bool, error) {
	c.Log.Info("auth: SignOut")
	rpc := c.Raw()
	_, err := rpc.AuthLogOut(ctx)
	if err != nil {
		c.Log.Warnf("auth: SignOut failed err=%v", err)
		return false, err
	}
	c.Log.Info("auth: signed out successfully")
	_ = c.Disconnect()
	return true, nil
}

// GetPasswordHint retrieves the optional hint string configured for the account's
// two-factor authentication password. Use this to help the user recall their
// password before calling CheckPassword.
//
// Returns an empty string when no hint has been set by the user.
// Returns an error if the server request fails or the session is not authorized.
func (c *Client) GetPasswordHint(ctx context.Context) (string, error) {
	rpc := c.Raw()
	pwd, err := rpc.AccountGetPassword(ctx)
	if err != nil {
		return "", err
	}
	return pwd.Hint, nil
}

// CheckPassword completes authentication for an account that has two-factor
// authentication (2FA) enabled. Call this only after SignIn returns the
// "2FA is enabled" error.
//
// The password is verified server-side using the Secure Remote Password (SRP)
// protocol; the plaintext password is never sent over the wire.
//
// Returns the authenticated User on success.
// Returns an error if the password is incorrect, the SRP computation fails
// (e.g. unsupported KDF algorithm), or a network error occurs.
//
// Example:
//
//	user, err := client.CheckPassword(ctx, "my_secure_password")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("authenticated as:", user.FirstName)
func (c *Client) CheckPassword(ctx context.Context, password string) (*types.User, error) {
	c.Log.Debug("auth: CheckPassword")
	srp, err := c.computeSRP(ctx, password)
	if err != nil {
		return nil, fmt.Errorf("compute SRP: %w", err)
	}
	rpc := c.Raw()
	result, err := rpc.AuthCheckPassword(ctx, &tg.AuthCheckPasswordRequest{
		Password: srp,
	})
	if err != nil {
		c.Log.Warnf("auth: CheckPassword failed err=%v", err)
		return nil, err
	}
	switch v := result.(type) {
	case *tg.AuthAuthorization:
		c.Log.Info("auth: CheckPassword successful")
		user := types.ParseUser(v.User)
		c.SetMe(user)
		c.saveMeToStorage(user)
		return user, nil
	default:
		return nil, fmt.Errorf("unexpected check password result type %T", result)
	}
}

// RecoverPassword authenticates the user using a recovery code sent to the
// account's recovery email address. This is the fallback path when the user
// cannot remember their 2FA password.
//
// The recovery code must be obtained by first requesting it through the
// Telegram app's password recovery flow.
//
// Returns the authenticated User on success.
// Returns an error if the recovery code is invalid or expired, or if the
// server rejects the request.
func (c *Client) RecoverPassword(ctx context.Context, code string) (*types.User, error) {
	c.Log.Debug("auth: RecoverPassword")
	rpc := c.Raw()
	result, err := rpc.AuthRecoverPassword(ctx, &tg.AuthRecoverPasswordRequest{
		Code: code,
	})
	if err != nil {
		c.Log.Warnf("auth: RecoverPassword failed err=%v", err)
		return nil, err
	}
	switch v := result.(type) {
	case *tg.AuthAuthorization:
		c.Log.Info("auth: RecoverPassword successful")
		user := types.ParseUser(v.User)
		c.SetMe(user)
		c.saveMeToStorage(user)
		return user, nil
	default:
		return nil, fmt.Errorf("unexpected recover password result type %T", result)
	}
}

func (c *Client) computeSRP(ctx context.Context, password string) (tg.InputCheckPasswordSRPClass, error) {
	rpc := c.Raw()
	pwd, err := rpc.AccountGetPassword(ctx)
	if err != nil {
		return nil, fmt.Errorf("get password: %w", err)
	}

	algo, ok := pwd.CurrentAlgo.(*tg.PasswordKdfAlgoSha256sha256pbkdf2hmacsha512iter100000sha256modPow)
	if !ok {
		return nil, fmt.Errorf("unsupported password KDF algorithm: %T", pwd.CurrentAlgo)
	}

	g := big.NewInt(int64(algo.G))
	p := new(big.Int).SetBytes(algo.P)

	salt1 := algo.Salt1
	salt2 := algo.Salt2

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generate random salt: %w", err)
	}
	salt1 = append(salt1, randomBytes...)

	result, err := crypto.ComputeSRP(salt1, salt2, g, p, pwd.SRPB, pwd.SRPID, password)
	if err != nil {
		return nil, fmt.Errorf("compute SRP: %w", err)
	}

	return &tg.InputCheckPasswordSRP{
		SRPID: result.SrpID,
		A:     result.A,
		M1:    result.M1,
	}, nil
}

// Session represents a single active Telegram session with device and network metadata.
// Each entry in ActiveSessions corresponds to a device or application that is currently
// logged into the account. Use the Hash field to identify and terminate sessions via ResetSession.
//
// Example:
//
//	sessions, _ := client.GetActiveSessions(ctx)
//	for _, s := range sessions.Sessions {
//	    fmt.Printf("%s on %s (IP: %s)\n", s.DeviceModel, s.Platform, s.IP)
//	}
type Session struct {
	Hash               int64
	DeviceModel        string
	Platform           string
	SystemVersion      string
	ApplicationName    string
	ApplicationVersion string
	DateCreated        int32
	DateActive         int32
	IP                 string
	Country            string
	Region             string
}

// ActiveSessions holds the list of all active sessions for the authorized account.
// The AuthorizationTTLDays field indicates how long inactive sessions remain valid
// before the server automatically terminates them.
//
// Example:
//
//	result, _ := client.GetActiveSessions(ctx)
//	fmt.Printf("TTL: %d days, active sessions: %d\n",
//	    result.AuthorizationTTLDays, len(result.Sessions))
type ActiveSessions struct {
	AuthorizationTTLDays int
	Sessions             []Session
}

// GetActiveSessions retrieves all active sessions for the currently authorized account.
// The returned list includes the current session along with all other devices that are
// logged in.
//
// Example:
//
//	sessions, err := client.GetActiveSessions(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, s := range sessions.Sessions {
//	    fmt.Printf("%s (%s) - IP: %s\n", s.DeviceModel, s.Platform, s.IP)
//	}
func (c *Client) GetActiveSessions(ctx context.Context) (*ActiveSessions, error) {
	c.Log.Debug("GetActiveSessions")
	rpc := c.Raw()
	result, err := rpc.AccountGetAuthorizations(ctx)
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(result.Authorizations))
	for _, auth := range result.Authorizations {
		a, ok := auth.(*tg.Authorization)
		if !ok {
			continue
		}
		sessions = append(sessions, Session{
			Hash:               a.Hash,
			DeviceModel:        a.DeviceModel,
			Platform:           a.Platform,
			SystemVersion:      a.SystemVersion,
			ApplicationName:    a.AppName,
			ApplicationVersion: a.AppVersion,
			DateCreated:        a.DateCreated,
			DateActive:         a.DateActive,
			IP:                 a.Ip,
			Country:            a.Country,
			Region:             a.Region,
		})
	}

	return &ActiveSessions{
		AuthorizationTTLDays: int(result.AuthorizationTTLDays),
		Sessions:             sessions,
	}, nil
}

// ResetSession terminates a specific session identified by its hash. Obtain the hash
// from the Session.Hash field returned by GetActiveSessions. The terminated session
// is immediately disconnected and its authorization is invalidated.
//
// Example:
//
//	sessions, _ := client.GetActiveSessions(ctx)
//	for _, s := range sessions.Sessions {
//	    if s.IP == "192.168.1.100" {
//	        client.ResetSession(ctx, s.Hash)
//	    }
//	}
func (c *Client) ResetSession(ctx context.Context, hash int64) error {
	c.Log.Debugf("ResetSession hash=%d", hash)
	rpc := c.Raw()
	_, err := rpc.AccountResetAuthorization(ctx, &tg.AccountResetAuthorizationRequest{
		Hash: hash,
	})
	return err
}
