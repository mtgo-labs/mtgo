package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SetGameScore sets the game score for the specified user in a game message.
// chatID identifies the chat containing the game message, messageID is the message ID,
// userID is the player's ID, and score is the new score value. If force is true, the score
// is set even if it's lower than the current one. If noForward is true, the score update
// message won't be forwarded. Returns the edited Message or an error.
func (c *Client) SetGameScore(ctx context.Context, chatID int64, messageID int64, userID int64, score int, force, noForward bool) (*types.Message, error) {
	c.Log.Debugf("SetGameScore chat_id=%d user_id=%d score=%d", chatID, userID, score)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	var flags tg.Fields
	if force {
		flags.Set(1)
	}
	if noForward {
		flags.Set(2)
	}
	flags.Set(0)

	rpc := c.Raw()
	result, err := rpc.MessagesSetGameScore(ctx, &tg.MessagesSetGameScoreRequest{
		Flags:       flags,
		EditMessage: true,
		Force:       force,
		Peer:        peer,
		ID:          int32(messageID),
		UserID:      user,
		Score:       int32(score),
	})
	if err != nil {
		return nil, err
	}
	return extractSingleMessage(result, c)
}

// GetGameHighScores returns the high score table for a game message.
// chatID identifies the chat containing the game, messageID is the message ID,
// and userID is the player whose scores to retrieve. Returns a slice of HighScore
// entries or an error if the peer or user cannot be resolved.
func (c *Client) GetGameHighScores(ctx context.Context, chatID int64, messageID int64, userID int64) ([]*tg.HighScore, error) {
	c.Log.Debugf("GetGameHighScores chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	user, err := resolveUserID(c, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve user: %w", err)
	}

	rpc := c.Raw()
	result, err := rpc.MessagesGetGameHighScores(ctx, &tg.MessagesGetGameHighScoresRequest{
		Peer:   peer,
		ID:     int32(messageID),
		UserID: user,
	})
	if err != nil {
		return nil, err
	}
	return result.Scores, nil
}

// SendGame sends a game message to the specified chat. Only bots can send games,
// and the chat must be a private (1-on-1) conversation with a user.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - chatID: target private chat identifier
//   - gameShortName: short name of the game registered with @BotFather
//   - opts: optional [params.SendMessage] overrides (reply markup, scheduling, etc.)
//
// Returns the sent message or an error if the peer cannot be resolved or the
// RPC call fails.
//
// Example:
//
//	msg, err := client.SendGame(ctx, chatID, "myawesomegame")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Game sent, message ID: %d\n", msg.ID)
func (c *Client) SendGame(ctx context.Context, chatID int64, gameShortName string, opts ...*params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendGame chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}
	opt := params.GetOptDef(&params.SendMessage{}, opts...)
	media := &tg.InputMediaGame{
		ID: &tg.InputGameShortName{ShortName: gameShortName},
	}
	return c.sendMediaInternal(ctx, peer, media, "", opt)
}
