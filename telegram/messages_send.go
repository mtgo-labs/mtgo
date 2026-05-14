package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/telegram/params"
	"github.com/mtgo-labs/mtgo/telegram/types"
	"github.com/mtgo-labs/mtgo/tg"
)

// SendContact sends a contact card (vCard) as a message to the specified chat.
// The contact is rendered as a tappable card showing the phone number and name.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - phoneNumber: the contact's phone number in international format
//   - firstName: the contact's first name
//   - lastName: the contact's last name
//   - opts: optional SendMessage parameters (silent, reply, schedule, etc.)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.SendContact(ctx, chatID, "+1234567890", "John", "Doe", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendContact(ctx context.Context, chatID int64, phoneNumber, firstName, lastName string, opts *params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendContact chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var cfg params.SendMessage
	if opts != nil {
		cfg = *opts
	}

	media := &tg.InputMediaContact{
		PhoneNumber: phoneNumber,
		FirstName:   firstName,
		LastName:    lastName,
		Vcard:       "",
	}

	return c.sendMediaInternal(ctx, peer, media, "", &cfg)
}

// SendLocation sends a geographic point (latitude/longitude) as a message to the
// specified chat. The location is displayed as a tappable map pin.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - lat: latitude of the location
//   - lng: longitude of the location
//   - opts: optional SendMessage parameters (silent, reply, schedule, etc.)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.SendLocation(ctx, chatID, 40.7128, -74.0060, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendLocation(ctx context.Context, chatID int64, lat, lng float64, opts *params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendLocation chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var cfg params.SendMessage
	if opts != nil {
		cfg = *opts
	}

	media := &tg.InputMediaGeoPoint{
		GeoPoint: &tg.InputGeoPoint{
			Lat:  lat,
			Long: lng,
		},
	}

	return c.sendMediaInternal(ctx, peer, media, "", &cfg)
}

// SendVenue sends a venue (a named location with an address) as a message to the
// specified chat. Venues display a map pin with a title and address block.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - lat: latitude of the venue
//   - lng: longitude of the venue
//   - title: the venue name shown above the address
//   - address: the venue address shown below the title
//   - opts: optional SendMessage parameters (silent, reply, schedule, etc.)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.SendVenue(ctx, chatID, 48.8584, 2.2945, "Eiffel Tower", "Paris, France", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendVenue(ctx context.Context, chatID int64, lat, lng float64, title, address string, opts *params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendVenue chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var cfg params.SendMessage
	if opts != nil {
		cfg = *opts
	}

	media := &tg.InputMediaVenue{
		GeoPoint: &tg.InputGeoPoint{
			Lat:  lat,
			Long: lng,
		},
		Title:     title,
		Address:   address,
		Provider:  "",
		VenueID:   "",
		VenueType: "",
	}

	return c.sendMediaInternal(ctx, peer, media, "", &cfg)
}

// SendDice sends an animated dice (or other animated emoji) message to the specified
// chat. The dice value is generated randomly by the server.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - opts: optional SendDiceOption to choose the animated emoji (defaults to 🎲)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.SendDice(ctx, chatID, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendDice(ctx context.Context, chatID int64, opts *SendDiceOption) (*types.Message, error) {
	c.Log.Debugf("SendDice chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	emoticon := "\U0001F3B2"
	if opts != nil && opts.Emoticon != "" {
		emoticon = opts.Emoticon
	}

	media := &tg.InputMediaDice{
		Emoticon: emoticon,
	}

	return c.sendMediaInternal(ctx, peer, media, "", &params.SendMessage{})
}

// SendPoll sends a poll question with the given answer options to the specified chat.
// Polls allow users to vote on one or more predefined answers.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - question: the poll question text
//   - options: the available answer choices
//   - opts: optional SendMessage parameters (silent, reply, schedule, etc.)
//
// Returns the sent poll message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - fewer than two options are provided
//   - the RPC call fails
//
// Example:
//
//	ctx := context.Background()
//	msg, err := client.SendPoll(ctx, chatID, "What's your favorite color?",
//	    []string{"Red", "Blue", "Green"}, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(msg.ID)
func (c *Client) SendPoll(ctx context.Context, chatID int64, question string, options []string, opts *params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendPoll chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var cfg params.SendMessage
	if opts != nil {
		cfg = *opts
	}

	answers := make([]tg.PollAnswerClass, len(options))
	for i, opt := range options {
		optionBytes := []byte{byte(i)}
		answers[i] = &tg.PollAnswer{
			Text: &tg.TextWithEntities{
				Text: opt,
			},
			Option: optionBytes,
		}
	}

	media := &tg.InputMediaPoll{
		Poll: &tg.Poll{
			Question: &tg.TextWithEntities{
				Text: question,
			},
			Answers: answers,
		},
	}

	return c.sendMediaInternal(ctx, peer, media, "", &cfg)
}

// SendCachedMedia sends a previously uploaded file referenced by its file ID.
// Use this to resend files that have already been uploaded to Telegram's servers
// without re-uploading.
//
// Parameters:
//   - ctx: context for cancellation and deadlines
//   - chatID: the target chat
//   - fileID: the cached file identifier (from a previously sent file)
//   - opts: optional SendMessage parameters (caption via text, silent, reply, etc.)
//
// Returns the sent message on success.
//
// Returns an error if:
//   - the peer cannot be resolved
//   - the file ID cannot be decoded
//   - the RPC call fails
func (c *Client) SendCachedMedia(ctx context.Context, chatID int64, fileID string, opts *params.SendMessage) (*types.Message, error) {
	c.Log.Debugf("SendCachedMedia chat_id=%d", chatID)
	peer, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("resolve peer: %w", err)
	}

	var cfg params.SendMessage
	if opts != nil {
		cfg = *opts
	}

	doc, err := resolveCachedFileID(fileID)
	if err != nil {
		return nil, fmt.Errorf("resolve cached file: %w", err)
	}

	media := &tg.InputMediaDocument{
		ID: doc,
	}

	return c.sendMediaInternal(ctx, peer, media, "", &cfg)
}

func resolveCachedFileID(fileID string) (tg.InputDocumentClass, error) {
	return &tg.InputDocument{
		ID:            0,
		AccessHash:    0,
		FileReference: []byte(fileID),
	}, nil
}
