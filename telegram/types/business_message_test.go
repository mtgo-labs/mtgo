package types

import (
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestParseBusinessMessage_Greeting(t *testing.T) {
	raw := &tg.BusinessGreetingMessage{
		ShortcutID:     10,
		NoActivityDays: 7,
		Recipients: &tg.BusinessRecipients{
			Users: []int64{1},
		},
	}
	users := map[int64]tg.UserClass{
		1: &tg.User{ID: 1, FirstName: "Ada"},
	}

	message := ParseBusinessMessage(raw, users)
	if message == nil {
		t.Fatal("ParseBusinessMessage returned nil")
	}
	if message.ShortcutID != 10 {
		t.Fatalf("ShortcutID = %d, want 10", message.ShortcutID)
	}
	if !message.IsGreeting {
		t.Fatal("IsGreeting = false, want true")
	}
	if message.IsAway {
		t.Fatal("IsAway = true, want false")
	}
	if message.NoActivityDays != 7 {
		t.Fatalf("NoActivityDays = %d, want 7", message.NoActivityDays)
	}
	if len(message.Recipients) != 1 || message.Recipients[0].ID != 1 {
		t.Fatalf("Recipients = %+v, want user 1", message.Recipients)
	}
}

func TestParseBusinessMessage_AwayCustomSchedule(t *testing.T) {
	raw := &tg.BusinessAwayMessage{
		OfflineOnly: true,
		ShortcutID:  20,
		Schedule: &tg.BusinessAwayMessageScheduleCustom{
			StartDate: 1700000000,
			EndDate:   1700003600,
		},
		Recipients: &tg.BusinessRecipients{
			Users: []int64{2},
		},
	}
	users := map[int64]tg.UserClass{
		2: &tg.User{ID: 2, FirstName: "Grace"},
	}

	message := ParseBusinessMessage(raw, users)
	if message == nil {
		t.Fatal("ParseBusinessMessage returned nil")
	}
	if message.ShortcutID != 20 {
		t.Fatalf("ShortcutID = %d, want 20", message.ShortcutID)
	}
	if !message.IsAway {
		t.Fatal("IsAway = false, want true")
	}
	if message.IsGreeting {
		t.Fatal("IsGreeting = true, want false")
	}
	if !message.OfflineOnly {
		t.Fatal("OfflineOnly = false, want true")
	}
	if message.Schedule != BusinessScheduleCustom {
		t.Fatalf("Schedule = %q, want %q", message.Schedule, BusinessScheduleCustom)
	}
	if !message.StartDate.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("StartDate = %v, want %v", message.StartDate, time.Unix(1700000000, 0))
	}
	if !message.EndDate.Equal(time.Unix(1700003600, 0)) {
		t.Fatalf("EndDate = %v, want %v", message.EndDate, time.Unix(1700003600, 0))
	}
	if len(message.Recipients) != 1 || message.Recipients[0].ID != 2 {
		t.Fatalf("Recipients = %+v, want user 2", message.Recipients)
	}
}

func TestParseBusinessMessage_AwayOutsideWorkHours(t *testing.T) {
	message := ParseBusinessMessage(&tg.BusinessAwayMessage{
		Schedule: &tg.BusinessAwayMessageScheduleOutsideWorkHours{},
	}, nil)

	if message.Schedule != BusinessScheduleOutside {
		t.Fatalf("Schedule = %q, want %q", message.Schedule, BusinessScheduleOutside)
	}
}

func TestParseBusinessMessage_Nil(t *testing.T) {
	if message := ParseBusinessMessage(nil, nil); message != nil {
		t.Fatalf("ParseBusinessMessage(nil) = %+v, want nil", message)
	}
}
