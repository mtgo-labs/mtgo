package telegram

import (
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

// extractBotUpdateMetaKey returns just the dedup key for a bot update, to keep
// the regression tests below compact.
func extractBotUpdateMetaKey(u tg.UpdateClass) string {
	return extractUpdateMeta(u).Key
}

// Regression (G1.1): distinct qts-bearing bot/participant updates must NOT
// collapse onto the default "type:<id>" dedup key — otherwise only the first of
// each type survives (the rest are dedup-dropped). They must each carry qts
// (for gap-recovery) and a qts-unique key.
func TestExtractQtsUpdatesUniqueKeys(t *testing.T) {
	cases := []struct {
		name string
		a, b tg.UpdateClass
	}{
		{
			"bot_reaction",
			&tg.UpdateBotMessageReaction{Qts: 11},
			&tg.UpdateBotMessageReaction{Qts: 12},
		},
		{
			"channel_participant",
			&tg.UpdateChannelParticipant{Qts: 21},
			&tg.UpdateChannelParticipant{Qts: 22},
		},
		{
			"chat_boost",
			&tg.UpdateBotChatBoost{Qts: 31},
			&tg.UpdateBotChatBoost{Qts: 32},
		},
		{
			"poll_vote",
			&tg.UpdateMessagePollVote{Qts: 41},
			&tg.UpdateMessagePollVote{Qts: 42},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ka, kb := extractBotUpdateMetaKey(tc.a), extractBotUpdateMetaKey(tc.b)
			if ka == "" || kb == "" {
				t.Fatalf("empty dedup key: %q / %q", ka, kb)
			}
			if ka == kb {
				t.Fatalf("distinct updates collapsed to same key %q (dedup-drop bug)", ka)
			}
			if ma, mb := extractUpdateMeta(tc.a).Qts, extractUpdateMeta(tc.b).Qts; ma == 0 || mb == 0 {
				t.Fatalf("qts not extracted: %d / %d (no gap-recovery)", ma, mb)
			}
		})
	}
}

// Regression (G1.1): query updates (no qts) must dedup on their unique query_id,
// not the shared type-id key. Two different callback/shipping/pre-checkout
// queries must have different keys; the same query redelivered must keep one key.
func TestExtractQueryUpdatesDedupByQueryID(t *testing.T) {
	type pair = struct{ a, b tg.UpdateClass }
	cases := []struct {
		name     string
		distinct pair
		same     tg.UpdateClass
	}{
		{"callback", pair{
			&tg.UpdateBotCallbackQuery{QueryID: 100},
			&tg.UpdateBotCallbackQuery{QueryID: 101},
		}, &tg.UpdateBotCallbackQuery{QueryID: 100}},
		{"shipping", pair{
			&tg.UpdateBotShippingQuery{QueryID: 200},
			&tg.UpdateBotShippingQuery{QueryID: 201},
		}, &tg.UpdateBotShippingQuery{QueryID: 200}},
		{"precheckout", pair{
			&tg.UpdateBotPrecheckoutQuery{QueryID: 300},
			&tg.UpdateBotPrecheckoutQuery{QueryID: 301},
		}, &tg.UpdateBotPrecheckoutQuery{QueryID: 300}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ka, kb := extractBotUpdateMetaKey(tc.distinct.a), extractBotUpdateMetaKey(tc.distinct.b)
			if ka == "" || ka == kb {
				t.Fatalf("distinct queries: keys %q / %q must differ (dedup-drop bug)", ka, kb)
			}
			// No qts for query updates — they don't join the qts sequence.
			if q := extractUpdateMeta(tc.same).Qts; q != 0 {
				t.Fatalf("query update unexpectedly carries qts %d", q)
			}
		})
	}
}
