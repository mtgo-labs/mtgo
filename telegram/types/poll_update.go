package types

import "github.com/mtgo-labs/mtgo/tg"

// PollUpdated carries the data needed to build a Bot API "poll" update when a
// poll's state changes (results, closure). Built from updateMessagePoll.
type PollUpdated struct {
	PollID    int64
	ChatID    int64
	MessageID int64
	Poll      *tg.Poll        // full poll definition (may be nil if only results changed)
	Results   *tg.PollResults // current results (vote counts per option)
}

// ParsePollUpdated converts a TL updateMessagePoll into a PollUpdated.
// Returns nil if raw is nil.
func ParsePollUpdated(raw *tg.UpdateMessagePoll) *PollUpdated {
	if raw == nil {
		return nil
	}
	return &PollUpdated{
		PollID:    raw.PollID,
		ChatID:    peerToChatID(raw.Peer),
		MessageID: int64(raw.MsgID),
		Poll:      raw.Poll,
		Results:   raw.Results,
	}
}

// PollAnswerUpdate carries the data for a Bot API "poll_answer" update when a
// user votes. Built from updateMessagePollVote. The Options are the raw option
// byte identifiers the user voted for; mtgo-bot-api maps them to option indices.
type PollAnswerUpdate struct {
	PollID  int64
	ChatID  int64
	UserID  int64
	Options [][]byte // raw option identifiers voted
}

// ParsePollAnswerUpdate converts a TL updateMessagePollVote into a PollAnswerUpdate.
// Returns nil if raw is nil.
func ParsePollAnswerUpdate(raw *tg.UpdateMessagePollVote) *PollAnswerUpdate {
	if raw == nil {
		return nil
	}
	return &PollAnswerUpdate{
		PollID:  raw.PollID,
		ChatID:  peerToChatID(raw.Peer),
		UserID:  peerToUserID(raw.Peer),
		Options: raw.Options,
	}
}
