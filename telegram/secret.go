package telegram

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"

	"github.com/mtgo-labs/mtgo/internal/crypto"
	"github.com/mtgo-labs/mtgo/tg"
	"github.com/mtgo-labs/mtgo/tg/e2e"
)

func (c *Client) initSecretChats() {
	c.mu.Lock()
	if c.handlerDispatcher != nil {
		c.handlerDispatcher.AddHandler(&encryptionUpdateHandler{client: c}, 0)
	}
	c.mu.Unlock()
}

func (c *Client) RequestSecretChat(ctx context.Context, userID tg.InputUserClass) (*SecretChat, error) {
	if c.secretChats == nil {
		c.mu.Lock()
		c.secretChats = NewSecretChatManager()
		c.mu.Unlock()
	}

	dhConfig, err := c.RPC().MessagesGetDHConfig(ctx, &tg.MessagesGetDHConfigRequest{
		Version:      0,
		RandomLength: 256,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: get dh config: %w", err)
	}

	var dhPrime *big.Int
	var g int32
	var dhVersion int32

	switch cfg := dhConfig.(type) {
	case *tg.MessagesDHConfig:
		dhPrime = new(big.Int).SetBytes(cfg.P)
		g = cfg.G
		dhVersion = cfg.Version
		if dhPrime.BitLen() < 2048 || !dhPrime.ProbablyPrime(20) {
			return nil, fmt.Errorf("telegram: server sent invalid DH prime")
		}
		if g < 2 || g > 7 {
			return nil, fmt.Errorf("telegram: invalid DH generator: %d", g)
		}
	case *tg.MessagesDHConfigNotModified:
		dhPrime = crypto.CurrentDHPrime
		g = 2
		dhVersion = 0
	default:
		return nil, fmt.Errorf("telegram: unexpected dh config type %T", dhConfig)
	}

	mySecret, err := crypto.GenerateDHSecret(dhPrime, 2048)
	if err != nil {
		return nil, fmt.Errorf("telegram: generate dh secret: %w", err)
	}

	ga := crypto.ComputeGA(g, mySecret, dhPrime)
	gaBytes := ga.Bytes()

	randomID := int32(0)
	b := make([]byte, 4)
	rand.Read(b)
	randomID = int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16 | int32(b[3])<<24

	result, err := c.RPC().MessagesRequestEncryption(ctx, &tg.MessagesRequestEncryptionRequest{
		UserID:   userID,
		RandomID: randomID,
		GA:       gaBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: request encryption: %w", err)
	}

	var chat *SecretChat
	switch r := result.(type) {
	case *tg.EncryptedChatWaiting:
		chat = &SecretChat{
			ID:              r.ID,
			AccessHash:      r.AccessHash,
			AdminID:         r.AdminID,
			PartID:          r.ParticipantID,
			State:           SecretChatStateWaiting,
			Outgoing:        true,
			Layer:           crypto.SecretChatLayer,
			DHPrime:         dhPrime,
			G:               g,
			MySecret:        mySecret,
			GA:              ga,
			DHConfigVersion: dhVersion,
			RandomID:        randomID,
		}
	default:
		return nil, fmt.Errorf("telegram: unexpected result type %T", result)
	}

	c.secretChats.Put(chat)
	return chat, nil
}

func (c *Client) AcceptSecretChat(ctx context.Context, chatID int32) (*SecretChat, error) {
	if c.secretChats == nil {
		return nil, ErrNoSecretChats
	}

	chat, ok := c.secretChats.Get(chatID)
	if !ok {
		return nil, fmt.Errorf("telegram: secret chat %d not found", chatID)
	}
	if chat.GetState() != SecretChatStateWaiting {
		return nil, fmt.Errorf("telegram: secret chat %d not in waiting state", chatID)
	}

	dhConfig, err := c.RPC().MessagesGetDHConfig(ctx, &tg.MessagesGetDHConfigRequest{
		Version:      chat.DHConfigVersion,
		RandomLength: 256,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: get dh config: %w", err)
	}

	switch cfg := dhConfig.(type) {
	case *tg.MessagesDHConfig:
		chat.DHPrime = new(big.Int).SetBytes(cfg.P)
		chat.G = cfg.G
		chat.DHConfigVersion = cfg.Version
		if chat.DHPrime.BitLen() < 2048 || !chat.DHPrime.ProbablyPrime(20) {
			return nil, fmt.Errorf("telegram: server sent invalid DH prime")
		}
		if chat.G < 2 || chat.G > 7 {
			return nil, fmt.Errorf("telegram: invalid DH generator: %d", chat.G)
		}
	case *tg.MessagesDHConfigNotModified:
	}

	mySecret, err := crypto.GenerateDHSecret(chat.DHPrime, 2048)
	if err != nil {
		return nil, fmt.Errorf("telegram: generate dh secret: %w", err)
	}

	gb := crypto.ComputeGA(chat.G, mySecret, chat.DHPrime)
	gbBytes := gb.Bytes()

	authKey := crypto.ComputeSharedKey(chat.GA, mySecret, chat.DHPrime)
	fp := crypto.KeyFingerprint(authKey)

	_, err = c.RPC().MessagesAcceptEncryption(ctx, &tg.MessagesAcceptEncryptionRequest{
		Peer:           chat.InputPeer(),
		GB:             gbBytes,
		KeyFingerprint: fp,
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: accept encryption: %w", err)
	}

	chat.mu.Lock()
	chat.AuthKey = authKey
	chat.KeyFingerprint = fp
	chat.MySecret = mySecret
	chat.GB = gbBytes
	chat.State = SecretChatStateReady
	chat.mu.Unlock()

	c.secretChats.Put(chat)
	return chat, nil
}

func (c *Client) SendSecretMessage(ctx context.Context, chatID int32, text string) (int64, error) {
	chat, ok := c.secretChats.Get(chatID)
	if !ok {
		return 0, fmt.Errorf("telegram: secret chat %d not found", chatID)
	}
	if chat.GetState() != SecretChatStateReady {
		return 0, fmt.Errorf("telegram: secret chat %d not ready", chatID)
	}

	outSeq := chat.NextOutSeqNo()
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)

	innerMsg := &e2e.DecryptedMessage{
		RandomID: generateRandomInt64(),
		Message:  text,
	}

	msgLayer := &e2e.DecryptedMessageLayer{
		RandomBytes: randomBytes,
		Layer:       chat.Layer,
		InSeqNo:     chat.InSeqNo,
		OutSeqNo:    outSeq,
		Message:     innerMsg,
	}

	var msgBuf bytes.Buffer
	if err := msgLayer.Encode(&msgBuf); err != nil {
		return 0, fmt.Errorf("telegram: encode message: %w", err)
	}

	encrypted, err := crypto.SecretEncrypt(msgBuf.Bytes(), chat.AuthKey, true)
	if err != nil {
		return 0, fmt.Errorf("telegram: encrypt message: %w", err)
	}

	randomID := generateRandomInt64()

	_, err = c.RPC().MessagesSendEncrypted(ctx, &tg.MessagesSendEncryptedRequest{
		Peer:     chat.InputPeer(),
		RandomID: randomID,
		Data:     encrypted,
	})
	if err != nil {
		return 0, fmt.Errorf("telegram: send encrypted: %w", err)
	}

	return randomID, nil
}

func (c *Client) DecryptSecretMessage(chatID int32, ciphertext []byte) (*e2e.DecryptedMessageLayer, error) {
	chat, ok := c.secretChats.Get(chatID)
	if !ok {
		return nil, fmt.Errorf("telegram: secret chat %d not found", chatID)
	}
	if chat.AuthKey == nil {
		return nil, fmt.Errorf("telegram: secret chat %d has no auth key", chatID)
	}

	plaintext, err := crypto.SecretDecrypt(ciphertext, chat.AuthKey, false)
	if err != nil {
		return nil, fmt.Errorf("telegram: decrypt message: %w", err)
	}

	obj, err := e2e.ReadE2ETLObject(tg.NewReader(plaintext))
	if err != nil {
		return nil, fmt.Errorf("telegram: decode decrypted message: %w", err)
	}

	layer, ok := obj.(*e2e.DecryptedMessageLayer)
	if !ok {
		return nil, fmt.Errorf("telegram: unexpected decrypted type %T", obj)
	}

	chat.NextInSeqNo()

	if layer.Layer > chat.Layer {
		chat.mu.Lock()
		chat.Layer = layer.Layer
		chat.mu.Unlock()
	}

	return layer, nil
}

func (c *Client) DiscardSecretChat(ctx context.Context, chatID int32, deleteHistory bool) error {
	_, err := c.RPC().MessagesDiscardEncryption(ctx, &tg.MessagesDiscardEncryptionRequest{
		ChatID:        chatID,
		DeleteHistory: deleteHistory,
	})
	if err != nil {
		return fmt.Errorf("telegram: discard encryption: %w", err)
	}

	if c.secretChats != nil {
		if chat, ok := c.secretChats.Get(chatID); ok {
			chat.SetState(SecretChatStateDiscarded)
			c.secretChats.Remove(chatID)
		}
	}
	return nil
}

func (c *Client) GetSecretChat(chatID int32) (*SecretChat, bool) {
	if c.secretChats == nil {
		return nil, false
	}
	return c.secretChats.Get(chatID)
}

func (c *Client) SecretChatVisualization(chatID int32) []string {
	chat, ok := c.GetSecretChat(chatID)
	if !ok {
		return nil
	}
	return chat.Visualization()
}

// SecretMessageHandler is a callback invoked when a decrypted secret message is received.
//
// Example:
//
//	client.OnSecretMessage(func(chat *telegram.SecretChat, layer *e2e.DecryptedMessageLayer) {
//		msg := layer.Message.(*e2e.DecryptedMessage)
//		fmt.Printf("secret chat %d: %s\n", chat.ID, msg.Message)
//	})
type SecretMessageHandler func(chat *SecretChat, layer *e2e.DecryptedMessageLayer)

func (c *Client) OnSecretMessage(handler SecretMessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.secretMsgHandlers == nil {
		c.secretMsgHandlers = make([]SecretMessageHandler, 0, 4)
	}
	c.secretMsgHandlers = append(c.secretMsgHandlers, handler)
}

func (c *Client) fireSecretMessage(chat *SecretChat, layer *e2e.DecryptedMessageLayer) {
	c.mu.RLock()
	handlers := c.secretMsgHandlers
	c.mu.RUnlock()
	for _, h := range handlers {
		h(chat, layer)
	}
}

// SecretChatRequestHandler is a callback invoked when a new secret chat is requested.
// Return false to reject the chat.
//
// Example:
//
//	client.OnSecretChatRequest(func(chat *telegram.SecretChat) bool {
//		fmt.Printf("secret chat request from user %d\n", chat.AdminID)
//		_, err := client.AcceptSecretChat(ctx, chat.ID)
//		return err == nil
//	})
type SecretChatRequestHandler func(chat *SecretChat) bool

func (c *Client) OnSecretChatRequest(handler SecretChatRequestHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.secretChatReqHandlers == nil {
		c.secretChatReqHandlers = make([]SecretChatRequestHandler, 0, 4)
	}
	c.secretChatReqHandlers = append(c.secretChatReqHandlers, handler)
}

func (c *Client) fireSecretChatRequest(chat *SecretChat) bool {
	c.mu.RLock()
	handlers := c.secretChatReqHandlers
	c.mu.RUnlock()
	for _, h := range handlers {
		if !h(chat) {
			return false
		}
	}
	return true
}

type encryptionUpdateHandler struct {
	client *Client
}

func (h *encryptionUpdateHandler) Check(u *Update) bool {
	if u.Raw == nil {
		return false
	}
	switch u.Raw.(type) {
	case *tg.UpdateEncryption:
		return true
	case *tg.UpdateNewEncryptedMessage:
		return true
	}
	return false
}

func (h *encryptionUpdateHandler) Handle(ctx *Context) {
	if ctx.Client == nil {
		return
	}
	c := ctx.Client

	switch raw := ctx.Update.Raw.(type) {
	case *tg.UpdateEncryption:
		h.handleEncryptionUpdate(c, ctx, raw)
	case *tg.UpdateNewEncryptedMessage:
		h.handleEncryptedMessage(c, ctx, raw)
	}
}

func (h *encryptionUpdateHandler) handleEncryptionUpdate(c *Client, ctx *Context, upd *tg.UpdateEncryption) {
	if c.secretChats == nil {
		c.mu.Lock()
		c.secretChats = NewSecretChatManager()
		c.mu.Unlock()
	}

	switch chat := upd.Chat.(type) {
	case *tg.EncryptedChatRequested:
		dhPrime := crypto.CurrentDHPrime
		g := int32(2)
		dhVersion := int32(0)

		ga := new(big.Int).SetBytes(chat.GA)
		if !crypto.ValidateGA(ga, dhPrime) {
			return
		}

		sc := &SecretChat{
			ID:              chat.ID,
			AccessHash:      chat.AccessHash,
			AdminID:         chat.AdminID,
			PartID:          chat.ParticipantID,
			State:           SecretChatStateWaiting,
			Outgoing:        false,
			Layer:           crypto.SecretChatLayer,
			DHPrime:         dhPrime,
			G:               g,
			GA:              ga,
			DHConfigVersion: dhVersion,
		}

		c.secretChats.Put(sc)
		ctx.Update.SecretChat = sc
		ctx.SecretChat = sc
		c.fireSecretChatRequest(sc)

	case *tg.EncryptedChat:
		sc, ok := c.secretChats.Get(chat.ID)
		if !ok || !sc.Outgoing {
			return
		}

		gb := new(big.Int).SetBytes(chat.GAOrB)
		if !crypto.ValidateGA(gb, sc.DHPrime) {
			if err := c.DiscardSecretChat(context.Background(), chat.ID, false); err != nil {
				c.Log.Warnf("discard secret chat on GA validation failure: %v", err)
			}
			return
		}

		authKey := crypto.ComputeSharedKey(gb, sc.MySecret, sc.DHPrime)
		fp := crypto.KeyFingerprint(authKey)

		if fp != chat.KeyFingerprint {
			if err := c.DiscardSecretChat(context.Background(), chat.ID, false); err != nil {
				c.Log.Warnf("discard secret chat on fingerprint mismatch: %v", err)
			}
			return
		}

		sc.mu.Lock()
		sc.AuthKey = authKey
		sc.KeyFingerprint = fp
		sc.GB = chat.GAOrB
		sc.State = SecretChatStateReady
		sc.mu.Unlock()
		ctx.Update.SecretChat = sc
		ctx.SecretChat = sc

	case *tg.EncryptedChatDiscarded:
		if sc, ok := c.secretChats.Get(chat.ID); ok {
			sc.SetState(SecretChatStateDiscarded)
			c.secretChats.Remove(chat.ID)
			ctx.Update.SecretChat = sc
			ctx.SecretChat = sc
		}
	}
}

func (h *encryptionUpdateHandler) handleEncryptedMessage(c *Client, ctx *Context, upd *tg.UpdateNewEncryptedMessage) {
	var chatID int32
	var ciphertext []byte

	switch msg := upd.Message.(type) {
	case *tg.EncryptedMessage:
		chatID = msg.ChatID
		ciphertext = msg.Bytes
	case *tg.EncryptedMessageService:
		chatID = msg.ChatID
		ciphertext = msg.Bytes
	default:
		return
	}

	sc, ok := c.secretChats.Get(chatID)
	if !ok || sc.GetState() != SecretChatStateReady {
		return
	}

	layer, err := c.DecryptSecretMessage(chatID, ciphertext)
	if err != nil {
		return
	}

	c.fireSecretMessage(sc, layer)

	ctx.Update.SecretChat = sc
	ctx.Update.SecretMessage = layer
	ctx.SecretChat = sc
	ctx.SecretMessage = layer
}

func (c *Client) SendLayerNotification(ctx context.Context, chatID int32) error {
	chat, ok := c.secretChats.Get(chatID)
	if !ok {
		return fmt.Errorf("telegram: secret chat %d not found", chatID)
	}
	if chat.GetState() != SecretChatStateReady {
		return fmt.Errorf("telegram: secret chat %d not ready", chatID)
	}

	outSeq := chat.NextOutSeqNo()
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)

	serviceMsg := &e2e.DecryptedMessageService{
		RandomID: generateRandomInt64(),
		Action:   &e2e.DecryptedMessageActionNotifyLayer{Layer: crypto.SecretChatLayer},
	}

	msgLayer := &e2e.DecryptedMessageLayer{
		RandomBytes: randomBytes,
		Layer:       chat.Layer,
		InSeqNo:     chat.InSeqNo,
		OutSeqNo:    outSeq,
		Message:     serviceMsg,
	}

	var buf bytes.Buffer
	if err := msgLayer.Encode(&buf); err != nil {
		return fmt.Errorf("telegram: encode layer notification: %w", err)
	}

	encrypted, err := crypto.SecretEncrypt(buf.Bytes(), chat.AuthKey, true)
	if err != nil {
		return fmt.Errorf("telegram: encrypt layer notification: %w", err)
	}

	_, err = c.RPC().MessagesSendEncryptedService(ctx, &tg.MessagesSendEncryptedServiceRequest{
		Peer:     chat.InputPeer(),
		RandomID: generateRandomInt64(),
		Data:     encrypted,
	})
	if err != nil {
		return fmt.Errorf("telegram: send layer notification: %w", err)
	}
	return nil
}

// SecretFileOptions holds optional parameters for sending an encrypted file in a secret chat.
//
// Example:
//
//	f, _ := os.Open("photo.jpg")
//	defer f.Close()
//	info, _ := f.Stat()
//	opts := &telegram.SecretFileOptions{
//		Caption:  "Here is the photo",
//		TTL:      60,
//		MimeType: "image/jpeg",
//	}
//	err := client.SendSecretFile(ctx, chatID, f, "photo.jpg", info.Size(), opts)
type SecretFileOptions struct {
	Caption  string
	TTL      int32
	MimeType string
	FileKey  []byte
	FileIV   []byte
}

func (c *Client) SendSecretFile(ctx context.Context, chatID int32, reader io.Reader, fileName string, fileSize int64, opts *SecretFileOptions) error {
	if opts == nil {
		opts = &SecretFileOptions{}
	}

	chat, ok := c.secretChats.Get(chatID)
	if !ok {
		return fmt.Errorf("telegram: secret chat %d not found", chatID)
	}
	if chat.GetState() != SecretChatStateReady {
		return fmt.Errorf("telegram: secret chat %d not ready", chatID)
	}

	fileData, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("telegram: read file data: %w", err)
	}

	var fileKey, fileIV []byte
	if len(opts.FileKey) == 32 && len(opts.FileIV) == 32 {
		fileKey = opts.FileKey
		fileIV = opts.FileIV
	} else {
		fileKey, fileIV, err = crypto.GenerateFileKeyIV()
		if err != nil {
			return fmt.Errorf("telegram: generate file key: %w", err)
		}
	}

	keyFP := crypto.FileKeyFingerprint(fileKey, fileIV)

	encryptedFileData := crypto.EncryptFile(fileData, fileKey, fileIV)

	uploadResult, err := c.UploadFile(ctx, bytes.NewReader(encryptedFileData), fileName, int64(len(encryptedFileData)), nil)
	if err != nil {
		return fmt.Errorf("telegram: upload encrypted file: %w", err)
	}

	var inputFile tg.InputEncryptedFileClass
	switch f := uploadResult.File.(type) {
	case *tg.InputFileBig:
		inputFile = &tg.InputEncryptedFileBigUploaded{
			ID:             f.ID,
			Parts:          f.Parts,
			KeyFingerprint: keyFP,
		}
	case *tg.InputFile:
		inputFile = &tg.InputEncryptedFileUploaded{
			ID:             f.ID,
			Parts:          f.Parts,
			MD5Checksum:    f.MD5Checksum,
			KeyFingerprint: keyFP,
		}
	default:
		return fmt.Errorf("telegram: unexpected upload result type %T", uploadResult.File)
	}

	outSeq := chat.NextOutSeqNo()
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)

	innerMsg := &e2e.DecryptedMessage{
		RandomID: generateRandomInt64(),
		TTL:      opts.TTL,
		Message:  opts.Caption,
	}

	msgLayer := &e2e.DecryptedMessageLayer{
		RandomBytes: randomBytes,
		Layer:       chat.Layer,
		InSeqNo:     chat.InSeqNo,
		OutSeqNo:    outSeq,
		Message:     innerMsg,
	}

	var msgBuf bytes.Buffer
	if err := msgLayer.Encode(&msgBuf); err != nil {
		return fmt.Errorf("telegram: encode message: %w", err)
	}

	encrypted, err := crypto.SecretEncrypt(msgBuf.Bytes(), chat.AuthKey, true)
	if err != nil {
		return fmt.Errorf("telegram: encrypt message: %w", err)
	}

	_, err = c.RPC().MessagesSendEncryptedFile(ctx, &tg.MessagesSendEncryptedFileRequest{
		Peer:     chat.InputPeer(),
		RandomID: generateRandomInt64(),
		Data:     encrypted,
		File:     inputFile,
	})
	if err != nil {
		return fmt.Errorf("telegram: send encrypted file: %w", err)
	}
	return nil
}

func (c *Client) DecryptSecretFile(ctx context.Context, file *tg.EncryptedFile, fileKey, fileIV []byte) ([]byte, error) {
	location := &tg.InputEncryptedFileLocation{
		ID:         file.ID,
		AccessHash: file.AccessHash,
	}

	data, err := c.DownloadFile(ctx, location, file.Size, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram: download encrypted file: %w", err)
	}

	decrypted, err := crypto.DecryptFile(data, fileKey, fileIV)
	if err != nil {
		return nil, fmt.Errorf("telegram: decrypt file: %w", err)
	}
	return decrypted, nil
}

func generateRandomInt64() int64 {
	b := make([]byte, 8)
	rand.Read(b)
	return int64(b[0]) | int64(b[1])<<8 | int64(b[2])<<16 | int64(b[3])<<24 |
		int64(b[4])<<32 | int64(b[5])<<40 | int64(b[6])<<48 | int64(b[7])<<56
}
