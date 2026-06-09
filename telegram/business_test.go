package telegram

import (
	"context"
	"testing"
)

func TestGetBusinessConnection_EmptyConnectionID(t *testing.T) {
	c := &Client{testResolver: &mockPeerResolver{}}
	_, err := c.GetBusinessConnection(context.TODO(), "")
	if err == nil {
		t.Fatal("expected error for empty connection ID")
	}
}
