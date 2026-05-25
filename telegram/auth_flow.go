package telegram

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/mtgo-labs/mtgo/tgerr"
)

const maxAuthRetries = 3

// CodeFunc returns a one-time verification code for the given phone number.
// The phone parameter is the number the code was sent to.
// Return an empty string or error to abort the login.
type CodeFunc func(ctx context.Context, phone string) (string, error)

// PasswordFunc returns the two-factor authentication password.
// hint is the optional hint the user configured for their 2FA.
// Return an empty string or error to abort the login.
type PasswordFunc func(ctx context.Context, hint string) (string, error)

// TerminalCodeFunc reads a verification code from stdin.
func TerminalCodeFunc() CodeFunc {
	return func(_ context.Context, phone string) (string, error) {
		fmt.Printf("Enter the code sent to %s: ", phone)
		code, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read code: %w", err)
		}
		return strings.TrimSpace(code), nil
	}
}

// TerminalPasswordFunc reads a 2FA password from stdin (hidden input).
func TerminalPasswordFunc() PasswordFunc {
	return func(_ context.Context, hint string) (string, error) {
		if hint != "" {
			fmt.Printf("Enter 2FA password (hint: %s): ", hint)
		} else {
			fmt.Print("Enter 2FA password: ")
		}
		pw, err := readPassword()
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return strings.TrimSpace(string(pw)), nil
	}
}

func readPassword() ([]byte, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	newState := *oldState
	newState.Lflag &^= unix.ECHO
	_ = unix.IoctlSetTermios(fd, unix.TCSETS, &newState)
	defer unix.IoctlSetTermios(fd, unix.TCSETS, oldState)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadBytes('\n')
	return line, err
}

// loginUser runs the interactive phone login flow:
//
//	SendCode → code via CodeFunc → SignIn
//	  → if 2FA → password via PasswordFunc → CheckPassword
//	  → if sign-up required → SignUp
//
// It retries up to maxAuthRetries times on PHONE_CODE_INVALID or
// PASSWORD_HASH_INVALID errors. The flow is aborted on any other error.
func (c *Client) loginUser(ctx context.Context) error {
	phone := c.cfg.PhoneNumber
	if phone == "" {
		return fmt.Errorf("telegram: login: PhoneNumber is required")
	}

	codeFn := c.cfg.CodeFunc
	if codeFn == nil {
		codeFn = TerminalCodeFunc()
	}
	pwFn := c.cfg.PasswordFunc
	if pwFn == nil {
		pwFn = TerminalPasswordFunc()
	}

	fmt.Fprintf(os.Stderr, "login: sending code to %s...\n", phone)
	sentCode, err := c.SendCode(ctx, phone)
	if err != nil {
		if errors.Is(err, ErrAlreadyAuthed) {
			return nil
		}
		return fmt.Errorf("send code: %w", err)
	}

	hash := sentCode.PhoneCodeHash

	for attempt := 0; attempt < maxAuthRetries; attempt++ {
		code, err := codeFn(ctx, phone)
		if err != nil {
			return fmt.Errorf("code input: %w", err)
		}

		_, err = c.SignIn(ctx, phone, hash, code)
		if err == nil {
			fmt.Fprintln(os.Stderr, "login: authenticated")
			return nil
		}

		if errors.Is(err, Err2FARequired) {
			return c.loginPassword(ctx, pwFn)
		}
		if errors.Is(err, ErrSignUpRequired) {
			return c.loginSignUp(ctx, phone, hash, codeFn)
		}
		if tgerr.Is(err, "PHONE_CODE_INVALID") || tgerr.Is(err, "PHONE_CODE_EMPTY") {
			fmt.Fprintf(os.Stderr, "login: invalid code (%d/%d)\n", attempt+1, maxAuthRetries)
			continue
		}
		return fmt.Errorf("sign in: %w", err)
	}
	return fmt.Errorf("login: too many invalid code attempts")
}

func (c *Client) loginPassword(ctx context.Context, pwFn PasswordFunc) error {
	fmt.Fprintln(os.Stderr, "login: 2FA required")
	hint, _ := c.GetPasswordHint(ctx)

	for attempt := 0; attempt < maxAuthRetries; attempt++ {
		pw, err := pwFn(ctx, hint)
		if err != nil {
			return fmt.Errorf("password input: %w", err)
		}

		_, err = c.CheckPassword(ctx, pw)
		if err == nil {
			fmt.Fprintln(os.Stderr, "login: 2FA authenticated")
			return nil
		}
		if tgerr.Is(err, "PASSWORD_HASH_INVALID") {
			c.Log.Warnf("login: invalid password (%d/%d)", attempt+1, maxAuthRetries)
			continue
		}
		return fmt.Errorf("check password: %w", err)
	}
	return fmt.Errorf("login: too many invalid password attempts")
}

func (c *Client) loginSignUp(ctx context.Context, phone, hash string, codeFn CodeFunc) error {
	fmt.Print("Phone number not registered. Enter first name to sign up: ")
	firstName, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read first name: %w", err)
	}
	firstName = strings.TrimSpace(firstName)

	fmt.Print("Enter last name (optional): ")
	lastName, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	lastName = strings.TrimSpace(lastName)

	_, err = c.SignUp(ctx, phone, hash, firstName, lastName)
	if err != nil {
		return fmt.Errorf("sign up: %w", err)
	}
	c.Log.Info("login: signed up")
	return nil
}

// isAuthorized checks whether the current session has a valid user ID
// stored in the storage backend, which indicates a previous successful login.
func (c *Client) isAuthorized() bool {
	if c.storage == nil {
		return false
	}
	uid, err := c.storage.UserID()
	if err != nil || uid == 0 {
		return false
	}
	return true
}
