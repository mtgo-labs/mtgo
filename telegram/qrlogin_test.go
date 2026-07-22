package telegram

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

type qrImportSuccessInvoker struct {
	input tg.TLObject
}

func (i *qrImportSuccessInvoker) RPCInvoke(_ context.Context, input tg.TLObject, _ func(*tg.Reader) (tg.TLObject, error)) (tg.TLObject, error) {
	i.input = input
	return &tg.AuthLoginTokenSuccess{Authorization: &tg.AuthAuthorization{
		User: &tg.User{ID: 42, FirstName: "QR"},
	}}, nil
}

func (*qrImportSuccessInvoker) RPCInvokeRaw(context.Context, tg.TLObject) ([]byte, error) {
	return nil, errors.New("unexpected raw invoke")
}

func TestGetQRCodeLoginToken_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.GetQRCodeLoginToken(context.Background())
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestCheckQRCodeLoginToken_NotConnected(t *testing.T) {
	c, _ := NewClient(1, "h", nil)
	_, err := c.CheckQRCodeLoginToken(context.Background(), []byte("token"))
	if !errors.Is(err, ErrNotConnected) {
		t.Errorf("expected ErrNotConnected, got %v", err)
	}
}

func TestQRLoginMigrationImportsRedirectToken(t *testing.T) {
	c, _ := NewClient(1, "h", &Config{DC: 4})
	st := NewMemoryStorage()
	if err := st.SetDCID(4); err != nil {
		t.Fatal(err)
	}
	if err := st.SetAuthKey(bytes.Repeat([]byte{0x44}, 256)); err != nil {
		t.Fatal(err)
	}
	c.storage = st
	c.state.SetConnecting(4)
	c.state.SetConnected()
	invoker := &qrImportSuccessInvoker{}
	c.testInvoker = invoker
	redirectToken := []byte("redirect-token")

	result, err := c.qrLoginMigrateToDC(context.Background(), 4, redirectToken)
	if err != nil {
		t.Fatalf("qrLoginMigrateToDC() = %v", err)
	}
	if result == nil || result.User == nil || result.User.ID != 42 {
		t.Fatalf("migration result = %#v, want authorized user 42", result)
	}
	req, ok := invoker.input.(*tg.AuthImportLoginTokenRequest)
	if !ok {
		t.Fatalf("migration RPC = %T, want AuthImportLoginTokenRequest", invoker.input)
	}
	if !bytes.Equal(req.Token, redirectToken) {
		t.Fatalf("import token = %q, want %q", req.Token, redirectToken)
	}
	if got, _ := st.UserID(); got != 42 {
		t.Fatalf("stored user ID = %d, want 42", got)
	}
}
