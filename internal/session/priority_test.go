package session

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestRoutePriority_NilReturnsHigh(t *testing.T) {
	if got := RoutePriority(nil); got != PriorityHigh {
		t.Fatalf("nil query: want PriorityHigh, got %d", got)
	}
}

func TestClassifyGoType(t *testing.T) {
	cases := []struct {
		typeStr string
		want    Priority
	}{
		{"*tg.MessagesSendMessageRequest", PriorityHigh},
		{"*tg.MessagesSendMediaRequest", PriorityHigh},
		{"*tg.MessagesEditMessageRequest", PriorityHigh},
		{"*tg.AccountUpdateProfileRequest", PriorityHigh},
		{"*tg.AuthImportBotAuthorizationRequest", PriorityHigh},
		{"*tg.MessagesGetDialogsRequest", PriorityLow},
		{"*tg.MessagesGetHistoryRequest", PriorityLow},
		{"*tg.MessagesSearchRequest", PriorityLow},
		{"*tg.MessagesGetRepliesRequest", PriorityLow},
		{"*tg.UploadGetFileRequest", PriorityLow},
		{"*tg.UploadSaveFilePartRequest", PriorityLow},
		{"*tg.MessagesGetSearchCountersRequest", PriorityLow},
		{"*tg.AccountGetAuthorizationsRequest", PriorityHigh},
		{"*tg.SomeUnknownMethodRequest", PriorityHigh},
	}
	for _, c := range cases {
		if got := classifyGoType(c.typeStr); got != c.want {
			t.Errorf("classifyGoType(%q) = %d, want %d", c.typeStr, got, c.want)
		}
	}
}

func TestRoutePriority_RealTypes(t *testing.T) {
	// Verify classification on actual generated TL types.
	if got := RoutePriority(&tg.UploadGetFileRequest{}); got != PriorityLow {
		t.Errorf("upload.GetFile: want Low, got %d", got)
	}
	if got := RoutePriority(&tg.MessagesSendMessageRequest{}); got != PriorityHigh {
		t.Errorf("messages.SendMessage: want High, got %d", got)
	}
}

func TestRoutePriority_UnwrapsHelperMethods(t *testing.T) {
	low := &tg.InvokeAfterMsgRequest{
		MsgID: 1,
		Query: &tg.InvokeWithLayerRequest{
			Layer: tg.Layer,
			Query: &tg.InitConnectionRequest{
				Query: &tg.UploadSaveFilePartRequest{},
			},
		},
	}
	if got := RoutePriority(low); got != PriorityLow {
		t.Errorf("wrapped upload.SaveFilePart: want Low, got %d", got)
	}

	high := &tg.InvokeWithoutUpdatesRequest{
		Query: &tg.MessagesSendMessageRequest{},
	}
	if got := RoutePriority(high); got != PriorityHigh {
		t.Errorf("wrapped messages.SendMessage: want High, got %d", got)
	}
}
