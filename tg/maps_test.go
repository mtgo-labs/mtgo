package tg

import (
	"testing"
)

func TestNamesMapContainsKnownFunctions(t *testing.T) {
	id, ok := NamesMap["messages.sendMessage"]
	if !ok {
		t.Fatal("messages.sendMessage not found in NamesMap")
	}
	if id != MessagesSendMessageTypeID {
		t.Fatalf("expected 0x%08x, got 0x%08x", MessagesSendMessageTypeID, id)
	}
}

func TestNamesMapContainsTypes(t *testing.T) {
	id, ok := NamesMap["user"]
	if !ok {
		t.Fatal("user not found in NamesMap")
	}
	if id != UserTypeID {
		t.Fatalf("expected 0x%08x, got 0x%08x", UserTypeID, id)
	}
}

func TestConstructorMapCreatesCorrectType(t *testing.T) {
	factory, ok := ConstructorMap[UserEmptyTypeID]
	if !ok {
		t.Fatal("UserEmptyTypeID not found in ConstructorMap")
	}
	obj := factory()
	empty, ok := obj.(*UserEmpty)
	if !ok {
		t.Fatalf("expected *UserEmpty, got %T", obj)
	}
	if empty.ConstructorID() != UserEmptyTypeID {
		t.Fatalf("expected constructor ID 0x%08x", UserEmptyTypeID)
	}
}

func TestFunctionsMapCreatesCorrectRequest(t *testing.T) {
	factory, ok := FunctionsMap[MessagesSendMessageTypeID]
	if !ok {
		t.Fatal("MessagesSendMessageTypeID not found in FunctionsMap")
	}
	obj := factory()
	req, ok := obj.(*MessagesSendMessageRequest)
	if !ok {
		t.Fatalf("expected *MessagesSendMessageRequest, got %T", obj)
	}
	if req.ConstructorID() != MessagesSendMessageTypeID {
		t.Fatalf("expected constructor ID 0x%08x", MessagesSendMessageTypeID)
	}
}
