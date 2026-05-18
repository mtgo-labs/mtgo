package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// CreateCall creates a group call in the specified peer chat. The optional rtmp
// parameter enables RTMP stream mode when set to true.
//
// Example:
//
//	call, err := client.CreateCall(ctx, chatID)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("group call created:", call)
//
// To create a call with RTMP streaming enabled:
//
//	call, err := client.CreateCall(ctx, chatID, true)
func (c *Client) CreateCall(ctx context.Context, peer int64, rtmp ...bool) (tg.PhoneCallClass, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	resolved, err := resolvePeer(c, peer)
	if err != nil {
		return nil, fmt.Errorf("create call: resolve peer: %w", err)
	}

	var useRtmp bool
	if len(rtmp) > 0 {
		useRtmp = rtmp[0]
	}

	rpc := c.Raw()
	updates, err := rpc.PhoneCreateGroupCall(ctx, &tg.PhoneCreateGroupCallRequest{
		Peer:       resolved,
		RandomID:   int32(c.RandomID()),
		RtmpStream: useRtmp,
	})
	if err != nil {
		return nil, fmt.Errorf("create call: %w", err)
	}

	return extractPhoneCall(updates)
}

// GetActiveCall retrieves the currently active group call for the given chat or channel.
//
// Example:
//
//	call, err := client.GetActiveCall(ctx, chatID)
//	if err != nil {
//		if errors.Is(err, telegram.ErrNoGroupCall) {
//			fmt.Println("no active call in this chat")
//			return
//		}
//		log.Fatal(err)
//	}
//	fmt.Println("active call found:", call)
func (c *Client) GetActiveCall(ctx context.Context, chatID int64) (tg.InputGroupCallClass, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	resolved, err := resolvePeer(c, chatID)
	if err != nil {
		return nil, fmt.Errorf("get active call: resolve peer: %w", err)
	}

	rpc := c.Raw()

	switch p := resolved.(type) {
	case *tg.InputPeerChannel:
		ch := &tg.InputChannel{ChannelID: p.ChannelID, AccessHash: p.AccessHash}
		result, err := rpc.ChannelsGetFullChannel(ctx, &tg.ChannelsGetFullChannelRequest{Channel: ch})
		if err != nil {
			return nil, fmt.Errorf("get active call: get full channel: %w", err)
		}
		if cf, ok := result.(*tg.ChannelFull); ok {
			if cf.Call == nil {
				return nil, ErrNoGroupCall
			}
			return cf.Call, nil
		}
		return nil, fmt.Errorf("get active call: unexpected full chat type %T", result)

	case *tg.InputPeerChat:
		result, err := rpc.MessagesGetFullChat(ctx, &tg.MessagesGetFullChatRequest{ChatID: p.ChatID})
		if err != nil {
			return nil, fmt.Errorf("get active call: get full chat: %w", err)
		}
		if cf, ok := result.(*tg.ChatFull); ok {
			if cf.Call == nil {
				return nil, ErrNoGroupCall
			}
			return cf.Call, nil
		}
		return nil, fmt.Errorf("get active call: unexpected full chat type %T", result)

	default:
		return nil, ErrNotChannel
	}
}

// CallReader reads streaming audio/video chunks from an active group call.
// Use NewCallReader to create one, then call NextChunk in a loop to receive data.
//
// Example:
//
//	reader, err := client.NewCallReader(ctx, chatID)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for {
//		chunk, err := reader.NextChunk(ctx)
//		if err != nil {
//			break
//		}
//		processChunk(chunk)
//	}
type CallReader struct {
	channels   []*tg.GroupCallStreamChannel
	call       tg.InputGroupCallClass
	currentTS  int64
	selectedCh int32
	scale      int32
	client     *Client
}

// SetTimestamp sets the playback timestamp in milliseconds for the next chunk read.
func (r *CallReader) SetTimestamp(ts int64) {
	r.currentTS = ts
}

// Timestamp returns the current playback timestamp in milliseconds.
func (r *CallReader) Timestamp() int64 {
	return r.currentTS
}

// Scale returns the current stream scale factor.
func (r *CallReader) Scale() int32 {
	return r.scale
}

// SetScale sets the stream scale factor for chunk retrieval.
func (r *CallReader) SetScale(scale int32) {
	r.scale = scale
}

// Channels returns the available stream channels for the active group call.
func (r *CallReader) Channels() []*tg.GroupCallStreamChannel {
	return r.channels
}

// SelectedChannel returns the currently selected stream channel, or nil if none is selected.
func (r *CallReader) SelectedChannel() *tg.GroupCallStreamChannel {
	if r.selectedCh < 0 || int(r.selectedCh) >= len(r.channels) {
		return nil
	}
	return r.channels[r.selectedCh]
}

// SelectChannel selects the stream channel at the given index and updates the scale
// and timestamp accordingly.
func (r *CallReader) SelectChannel(idx int32) {
	if idx < 0 || int(idx) >= len(r.channels) {
		return
	}
	r.selectedCh = idx
	r.scale = r.channels[idx].Scale
	r.currentTS = r.channels[idx].LastTimestampMs
}

// NextChunk fetches the next audio/video chunk from the selected stream channel.
func (r *CallReader) NextChunk(ctx context.Context) ([]byte, error) {
	if r.selectedCh < 0 || int(r.selectedCh) >= len(r.channels) {
		return nil, ErrCallNoChannel
	}

	ch := r.SelectedChannel()
	if ch == nil {
		return nil, ErrCallChannelNil
	}

	vch := ch.Channel
	vq := int32(2)
	input := &tg.InputGroupCallStream{
		Call:         r.call,
		TimeMs:       r.currentTS,
		Scale:        r.scale,
		VideoChannel: vch,
		VideoQuality: vq,
	}

	rpc := r.client.Raw()
	result, err := rpc.UploadGetFile(ctx, &tg.UploadGetFileRequest{
		Location: input,
		Offset:   0,
		Limit:    512 * 1024,
	})
	if err != nil {
		return nil, fmt.Errorf("call reader: get file: %w", err)
	}

	switch f := result.(type) {
	case *tg.UploadFile:
		r.currentTS += 1000 >> r.scale
		return f.Bytes, nil
	case *tg.UploadFileCDNRedirect:
		return nil, ErrCallCDNNotSupported
	default:
		return nil, fmt.Errorf("call reader: unexpected result type %T", result)
	}
}

// NewCallReader creates a CallReader for the active group call in the specified chat,
// fetching available stream channels automatically.
//
// Example:
//
//	reader, err := client.NewCallReader(ctx, chatID)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("channels: %d, scale: %d\n", len(reader.Channels()), reader.Scale())
//	chunk, err := reader.NextChunk(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Printf("got chunk: %d bytes\n", len(chunk))
func (c *Client) NewCallReader(ctx context.Context, chatID int64) (*CallReader, error) {
	if err := c.ensureConnected(); err != nil {
		return nil, err
	}

	call, err := c.GetActiveCall(ctx, chatID)
	if err != nil {
		return nil, err
	}

	rpc := c.Raw()
	stream, err := rpc.PhoneGetGroupCallStreamChannels(ctx, &tg.PhoneGetGroupCallStreamChannelsRequest{
		Call: call,
	})
	if err != nil {
		return nil, fmt.Errorf("call reader: get stream channels: %w", err)
	}

	if len(stream.Channels) == 0 {
		return nil, ErrCallNoStreams
	}

	return &CallReader{
		channels:   stream.Channels,
		call:       call,
		scale:      stream.Channels[0].Scale,
		selectedCh: 0,
		client:     c,
	}, nil
}

func extractPhoneCall(updates tg.UpdatesClass) (tg.PhoneCallClass, error) {
	var allUpdates []tg.UpdateClass
	switch u := updates.(type) {
	case *tg.Updates:
		allUpdates = u.Updates
	case *tg.UpdatesCombined:
		allUpdates = u.Updates
	}
	for _, upd := range allUpdates {
		if pc, ok := upd.(*tg.UpdatePhoneCall); ok {
			return pc.PhoneCall, nil
		}
	}
	return nil, ErrCallNotFound
}
