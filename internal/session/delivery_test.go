package session

import (
	"errors"
	"testing"
)

func TestDeliveryFailureState(t *testing.T) {
	handle := &CallHandle{}
	err := deliveryFailure(handle, ErrSessionClosed)
	var deliveryErr *DeliveryError
	if !errors.As(err, &deliveryErr) || deliveryErr.State != DeliveryUnknown {
		t.Fatalf("unacked delivery error = %#v", err)
	}
	if !errors.Is(err, ErrSessionClosed) {
		t.Fatal("delivery error did not preserve session error")
	}

	handle.acked.Store(true)
	err = deliveryFailure(handle, ErrSessionClosed)
	if !errors.As(err, &deliveryErr) || deliveryErr.State != DeliveryReceived {
		t.Fatalf("acked delivery error = %#v", err)
	}
}

func TestDeliveryFailurePreservesConfirmedNotReceived(t *testing.T) {
	if err := deliveryFailure(&CallHandle{}, ErrMsgNotReceived); err != ErrMsgNotReceived {
		t.Fatalf("deliveryFailure = %v, want ErrMsgNotReceived", err)
	}
}
