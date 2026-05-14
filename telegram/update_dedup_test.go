package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestExtractUpdateMetaAccountPts(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateDeleteMessages{Messages: []int32{10, 11}, PTS: 5, PTSCount: 2})
	if meta.Pts != 5 || meta.PtsCount != 2 || meta.Key == "" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtractUpdateMetaChannelPts(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateDeleteChannelMessages{ChannelID: 100, Messages: []int32{10}, PTS: 9, PTSCount: 1})
	if !meta.IsChannel || meta.ChannelID != 100 || meta.ChannelPts != 9 || meta.PtsCount != 1 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestDedupKeyStable(t *testing.T) {
	a := extractUpdateMeta(&tg.UpdateDeleteMessages{Messages: []int32{10}, PTS: 5, PTSCount: 1})
	b := extractUpdateMeta(&tg.UpdateDeleteMessages{Messages: []int32{10}, PTS: 5, PTSCount: 1})
	if a.Key == "" || a.Key != b.Key {
		t.Fatalf("dedup keys = %q %q", a.Key, b.Key)
	}
}

func TestExtractUpdateMetaNewMessage(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateNewMessage{
		Message:  &tg.Message{ID: 42, PeerID: &tg.PeerUser{UserID: 1}},
		PTS:      10,
		PTSCount: 1,
	})
	if meta.Pts != 10 || meta.PtsCount != 1 || meta.IsChannel {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.Key == "" {
		t.Fatal("expected non-empty dedup key")
	}
}

func TestExtractUpdateMetaNewChannelMessage(t *testing.T) {
	meta := extractUpdateMeta(&tg.UpdateNewChannelMessage{
		Message:  &tg.Message{ID: 7, PeerID: &tg.PeerChannel{ChannelID: 200}},
		PTS:      15,
		PTSCount: 1,
	})
	if !meta.IsChannel || meta.ChannelID != 200 || meta.ChannelPts != 15 {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestDedupKeyDifferentForDifferentUpdates(t *testing.T) {
	a := extractUpdateMeta(&tg.UpdateDeleteMessages{Messages: []int32{10}, PTS: 5, PTSCount: 1})
	b := extractUpdateMeta(&tg.UpdateDeleteMessages{Messages: []int32{11}, PTS: 5, PTSCount: 1})
	if a.Key == b.Key {
		t.Fatalf("different updates should have different keys: %q", a.Key)
	}
}
