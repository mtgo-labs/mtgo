package telegram

import (
	"context"
	"fmt"

	"github.com/mtgo-labs/mtgo/tg"
)

// GetWalletState returns the TON wallet state of the logged-in user: address,
// balance, and backup capability.
func (c *WalletMethods) GetWalletState(ctx context.Context) (*tg.WalletState, error) {
	c.Log.Debug("GetWalletState")
	result, err := c.Raw().WalletGetState(ctx)
	if err != nil {
		return nil, fmt.Errorf("get wallet state: %w", err)
	}
	switch v := result.(type) {
	case *tg.WalletState:
		return v, nil
	case *tg.WalletStateEmpty:
		return nil, nil
	default:
		return nil, fmt.Errorf("get wallet state: unexpected result %T", result)
	}
}

// GetWalletUserAddresses resolves wallet addresses for users, or users for
// addresses. When both are empty, the addresses of the logged-in user's
// contacts are listed; force bypasses the cache.
func (c *WalletMethods) GetWalletUserAddresses(ctx context.Context, userIDs []int64, addresses []string, force bool) ([]*tg.WalletUserAddress, error) {
	c.Log.Debugf("GetWalletUserAddresses users=%d addresses=%d", len(userIDs), len(addresses))
	req := &tg.WalletGetUserAddressesRequest{
		Force:     force,
		Addresses: addresses,
	}
	for _, id := range userIDs {
		user, err := resolveUserID(ctx, c.Client, id)
		if err != nil {
			return nil, fmt.Errorf("resolve user: %w", err)
		}
		req.ID = append(req.ID, user)
	}
	result, err := c.Raw().WalletGetUserAddresses(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get wallet user addresses: %w", err)
	}
	return result.Addresses, nil
}

// WalletTransactions is one page of the wallet's transaction history.
type WalletTransactions struct {
	// Balance is the wallet balance in nanoton.
	Balance int64
	// Transactions are the entries on this page.
	Transactions []*tg.WalletTransaction
	// NextOffset fetches the next page; empty on the last page.
	NextOffset string
}

// GetWalletTransactions lists the wallet's transaction history. Pass inbound
// or outbound true to filter by direction; both false lists everything. A
// limit of 0 or less defaults to 25.
func (c *WalletMethods) GetWalletTransactions(ctx context.Context, inbound, outbound bool, offset string, limit int) (*WalletTransactions, error) {
	c.Log.Debugf("GetWalletTransactions inbound=%v outbound=%v", inbound, outbound)
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	result, err := c.Raw().WalletGetTransactions(ctx, &tg.WalletGetTransactionsRequest{
		Inbound:  inbound,
		Outbound: outbound,
		Offset:   offset,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get wallet transactions: %w", err)
	}
	return &WalletTransactions{
		Balance:      result.Balance,
		Transactions: result.Transactions,
		NextOffset:   result.NextOffset,
	}, nil
}

// GetWalletTransactionsByIDs fetches specific wallet transactions by their
// identifiers.
func (c *WalletMethods) GetWalletTransactionsByIDs(ctx context.Context, ids []string) ([]*tg.WalletTransaction, error) {
	c.Log.Debugf("GetWalletTransactionsByIDs n=%d", len(ids))
	result, err := c.Raw().WalletGetTransactionsByIDs(ctx, &tg.WalletGetTransactionsByIDsRequest{
		ID: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("get wallet transactions by ids: %w", err)
	}
	return result.Transactions, nil
}

// GetWalletTransactionsByMessageHash fetches wallet transactions referenced
// by external message hashes.
func (c *WalletMethods) GetWalletTransactionsByMessageHash(ctx context.Context, hashes []string) ([]*tg.WalletTransaction, error) {
	c.Log.Debugf("GetWalletTransactionsByMessageHash n=%d", len(hashes))
	result, err := c.Raw().WalletGetTransactionsByMsgHash(ctx, &tg.WalletGetTransactionsByMsgHashRequest{
		MsgHash: hashes,
	})
	if err != nil {
		return nil, fmt.Errorf("get wallet transactions by message hash: %w", err)
	}
	return result.Transactions, nil
}

// SendWalletTransfer submits a signed TON transfer to the wallet. dataNormal
// carries a regular transfer body, dataGasless a gasless one; exactly one
// must be set. userID is the transfer counterparty for audit trails.
func (c *WalletMethods) SendWalletTransfer(ctx context.Context, userID int64, dataNormal, dataGasless []byte) error {
	c.Log.Debugf("SendWalletTransfer user_id=%d", userID)
	if (dataNormal == nil) == (dataGasless == nil) {
		return fmt.Errorf("exactly one of dataNormal and dataGasless is required")
	}
	req := &tg.WalletSendTransferRequest{
		DataNormal:  dataNormal,
		DataGasless: dataGasless,
		RandomID:    c.RandomID(),
	}
	if userID != 0 {
		user, err := resolveUserID(ctx, c.Client, userID)
		if err != nil {
			return fmt.Errorf("resolve user: %w", err)
		}
		req.UserID = user
	}
	_, err := c.Raw().WalletSendTransfer(ctx, req)
	if err != nil {
		return fmt.Errorf("send wallet transfer: %w", err)
	}
	return nil
}

// WalletNFTs is one page of the wallet's NFT holdings.
type WalletNFTs struct {
	// Items are the NFTs on this page.
	Items []*tg.WalletNftItem
	// NextOffset fetches the next page; empty on the last page.
	NextOffset string
}

// GetWalletNFTs lists the NFTs held by the wallet.
func (c *WalletMethods) GetWalletNFTs(ctx context.Context, offset string, limit int) (*WalletNFTs, error) {
	c.Log.Debugf("GetWalletNFTs offset=%q", offset)
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	result, err := c.Raw().WalletGetNfts(ctx, &tg.WalletGetNftsRequest{
		Offset: offset,
		Limit:  int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("get wallet nfts: %w", err)
	}
	return &WalletNFTs{
		Items:      result.Items,
		NextOffset: result.NextOffset,
	}, nil
}

// GetExistingWalletBalance checks whether the logged-in user already owns a
// TON balance on an external wallet, returning the URL to import it.
func (c *WalletMethods) GetExistingWalletBalance(ctx context.Context) (*tg.WalletExistingBalance, error) {
	c.Log.Debug("GetExistingWalletBalance")
	result, err := c.Raw().WalletGetExistingWaltBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get existing wallet balance: %w", err)
	}
	return result, nil
}

// ExportSecretPhrase begins exporting the wallet's recovery phrase. The
// cloud password (when set) unlocks it; the returned token addresses the
// phrase parts held by the listed DCs, each fetched and decrypted separately
// with [WalletMethods.FetchEncryptedSecretPhrasePart]. The caller performs
// the decryption: parts are sealed to a key only the holder knows.
func (c *WalletMethods) ExportSecretPhrase(ctx context.Context, password string) (*tg.WalletSecretPhraseParts, error) {
	c.Log.Debug("ExportSecretPhrase")
	req := &tg.WalletExportSecretPhraseRequest{}
	if password != "" {
		srp, err := c.computeSRP(ctx, password)
		if err != nil {
			return nil, fmt.Errorf("compute password check: %w", err)
		}
		req.Password = srp
	}
	result, err := c.Raw().WalletExportSecretPhrase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("export secret phrase: %w", err)
	}
	return result, nil
}

// FetchEncryptedSecretPhrasePart fetches one encrypted share of the recovery
// phrase from its holder DC, sealed to publicKey. Decrypting the share is
// the caller's job.
func (c *WalletMethods) FetchEncryptedSecretPhrasePart(ctx context.Context, token string, publicKey []byte) (*tg.WalletEncryptedSecretPhrasePart, error) {
	c.Log.Debug("FetchEncryptedSecretPhrasePart")
	result, err := c.Raw().WalletFetchEncryptedSecretPhrasePart(ctx, &tg.WalletFetchEncryptedSecretPhrasePartRequest{
		Token:     token,
		PublicKey: publicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch encrypted secret phrase part: %w", err)
	}
	return result, nil
}

// ReplaceWallet replaces the wallet with a new one (for example when
// importing an external wallet). The cloud password, when set, must unlock
// the current wallet.
func (c *WalletMethods) ReplaceWallet(ctx context.Context, wallet tg.InputWalletReplacementClass, password string) (*tg.WalletState, error) {
	c.Log.Debug("ReplaceWallet")
	req := &tg.WalletReplaceWalletRequest{Wallet: wallet}
	if password != "" {
		srp, err := c.computeSRP(ctx, password)
		if err != nil {
			return nil, fmt.Errorf("compute password check: %w", err)
		}
		req.Password = srp
	}
	result, err := c.Raw().WalletReplaceWallet(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("replace wallet: %w", err)
	}
	state, ok := result.(*tg.WalletState)
	if !ok {
		return nil, fmt.Errorf("replace wallet: unexpected result %T", result)
	}
	return state, nil
}

// GetBackupHolderDcs lists the DCs holding encrypted wallet-backup shares.
func (c *WalletMethods) GetBackupHolderDcs(ctx context.Context) ([]*tg.WalletHolderDC, error) {
	c.Log.Debug("GetBackupHolderDcs")
	raw, err := c.Raw().WalletGetBackupHolderDcs(ctx)
	if err != nil {
		return nil, fmt.Errorf("get backup holder dcs: %w", err)
	}
	vector, ok := raw.(*tg.GenericVector)
	if !ok {
		return nil, fmt.Errorf("get backup holder dcs: unexpected result %T", raw)
	}
	dcs := make([]*tg.WalletHolderDC, 0, len(vector.Items))
	for _, item := range vector.Items {
		if dc, ok := item.(*tg.WalletHolderDC); ok {
			dcs = append(dcs, dc)
		}
	}
	return dcs, nil
}

// EnableBackup turns on distributed wallet backup with the caller-encrypted
// shares; newPublicKey and proof re-bind the backup when rotating keys.
func (c *WalletMethods) EnableBackup(ctx context.Context, parts [][]byte, newPublicKey []byte, proof *tg.WalletOwnershipProof) (*tg.WalletState, error) {
	c.Log.Debug("EnableBackup")
	result, err := c.Raw().WalletEnableBackup(ctx, &tg.WalletEnableBackupRequest{
		Parts:        parts,
		NewPublicKey: newPublicKey,
		Proof:        proof,
	})
	if err != nil {
		return nil, fmt.Errorf("enable backup: %w", err)
	}
	state, ok := result.(*tg.WalletState)
	if !ok {
		return nil, fmt.Errorf("enable backup: unexpected result %T", result)
	}
	return state, nil
}

// DisableBackup turns off distributed wallet backup. The cloud password,
// when set, must unlock the wallet.
func (c *WalletMethods) DisableBackup(ctx context.Context, password string, newPublicKey []byte, proof *tg.WalletOwnershipProof) (*tg.WalletState, error) {
	c.Log.Debug("DisableBackup")
	req := &tg.WalletDisableBackupRequest{
		NewPublicKey: newPublicKey,
		Proof:        proof,
	}
	if password != "" {
		srp, err := c.computeSRP(ctx, password)
		if err != nil {
			return nil, fmt.Errorf("compute password check: %w", err)
		}
		req.Password = srp
	}
	result, err := c.Raw().WalletDisableBackup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("disable backup: %w", err)
	}
	state, ok := result.(*tg.WalletState)
	if !ok {
		return nil, fmt.Errorf("disable backup: unexpected result %T", result)
	}
	return state, nil
}

// GetProofChallenge fetches the challenge to sign when a wallet operation
// demands proof of ownership.
func (c *WalletMethods) GetProofChallenge(ctx context.Context) (*tg.WalletProofChallenge, error) {
	c.Log.Debug("GetProofChallenge")
	result, err := c.Raw().WalletGetProofChallenge(ctx)
	if err != nil {
		return nil, fmt.Errorf("get proof challenge: %w", err)
	}
	return result, nil
}

// GetGaslessInfo fetches the current gasless-transfer parameters.
func (c *WalletMethods) GetGaslessInfo(ctx context.Context) (tg.UpdatesClass, error) {
	c.Log.Debug("GetGaslessInfo")
	result, err := c.Raw().WalletGetGaslessInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gasless info: %w", err)
	}
	return result, nil
}

// TonConnectCreateSession opens a Ton Connect session for a dApp, identified
// by its client id and manifest URL. Signatures and request bodies for the
// session are produced by the dApp protocol; mtgo transports them.
func (c *WalletMethods) TonConnectCreateSession(ctx context.Context, dappClientID, manifestURL string) (*tg.TonConnectSession, error) {
	c.Log.Debug("TonConnectCreateSession")
	result, err := c.Raw().WalletTonConnectCreateSession(ctx, &tg.WalletTonConnectCreateSessionRequest{
		DappClientID: dappClientID,
		ManifestURL:  manifestURL,
	})
	if err != nil {
		return nil, fmt.Errorf("ton connect create session: %w", err)
	}
	return result, nil
}

// TonConnectRegisterKey registers a signing key with a session; the returned
// challenge must be signed by that key for the connect result.
func (c *WalletMethods) TonConnectRegisterKey(ctx context.Context, sessionID int64, clientID string) (*tg.WalletTonConnectChallenge, error) {
	c.Log.Debug("TonConnectRegisterKey")
	result, err := c.Raw().WalletTonConnectRegisterKey(ctx, &tg.WalletTonConnectRegisterKeyRequest{
		SessionID: sessionID,
		ClientID:  clientID,
	})
	if err != nil {
		return nil, fmt.Errorf("ton connect register key: %w", err)
	}
	return result, nil
}

// TonConnectSubmitConnectResult completes a connect with the signed
// challenge answer and the protocol body; submitErr reports a dApp-side
// failure instead.
func (c *WalletMethods) TonConnectSubmitConnectResult(ctx context.Context, sessionID int64, challengeAnswer, body []byte, traceID string, submitErr bool) error {
	c.Log.Debug("TonConnectSubmitConnectResult")
	req := &tg.WalletTonConnectSubmitConnectResultRequest{
		SessionID:       sessionID,
		ChallengeAnswer: challengeAnswer,
		Body:            body,
		TraceID:         traceID,
	}
	req.Error = submitErr
	_, err := c.Raw().WalletTonConnectSubmitConnectResult(ctx, req)
	if err != nil {
		return fmt.Errorf("ton connect submit connect result: %w", err)
	}
	return nil
}

// TonConnectGetPending lists pending dApp requests, optionally filtered by
// dApp or session.
func (c *WalletMethods) TonConnectGetPending(ctx context.Context, dappClientID string, sessionID int64) (*tg.WalletTonConnectPending, error) {
	c.Log.Debug("TonConnectGetPending")
	req := &tg.WalletTonConnectGetPendingRequest{
		DappClientID: dappClientID,
		SessionID:    sessionID,
	}
	result, err := c.Raw().WalletTonConnectGetPending(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ton connect get pending: %w", err)
	}
	return result, nil
}

// TonConnectClaimRequest claims a dApp request for answering; declined
// rejects it instead.
func (c *WalletMethods) TonConnectClaimRequest(ctx context.Context, sessionID int64, msgID int32, appRequestID string, challengeAnswer []byte, declined bool) error {
	c.Log.Debug("TonConnectClaimRequest")
	req := &tg.WalletTonConnectClaimRequestRequest{
		SessionID:       sessionID,
		MsgID:           msgID,
		AppRequestID:    appRequestID,
		ChallengeAnswer: challengeAnswer,
	}
	req.Declined = declined
	_, err := c.Raw().WalletTonConnectClaimRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("ton connect claim request: %w", err)
	}
	return nil
}

// TonConnectSubmitResponse answers a claimed request with the protocol body.
func (c *WalletMethods) TonConnectSubmitResponse(ctx context.Context, sessionID int64, msgID int32, body []byte, traceID string) error {
	c.Log.Debug("TonConnectSubmitResponse")
	_, err := c.Raw().WalletTonConnectSubmitResponse(ctx, &tg.WalletTonConnectSubmitResponseRequest{
		SessionID: sessionID,
		MsgID:     msgID,
		Body:      body,
		TraceID:   traceID,
	})
	if err != nil {
		return fmt.Errorf("ton connect submit response: %w", err)
	}
	return nil
}

// TonConnectNextEventID returns the next event id to observe for a session.
func (c *WalletMethods) TonConnectNextEventID(ctx context.Context, sessionID int64) (*tg.TonConnectNextEventID, error) {
	c.Log.Debug("TonConnectNextEventID")
	result, err := c.Raw().WalletTonConnectNextEventID(ctx, &tg.WalletTonConnectNextEventIDRequest{
		SessionID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("ton connect next event id: %w", err)
	}
	return result, nil
}

// TonConnectCloseSession closes a session, optionally sending a protocol
// goodbye body.
func (c *WalletMethods) TonConnectCloseSession(ctx context.Context, sessionID int64, body []byte) error {
	c.Log.Debug("TonConnectCloseSession")
	req := &tg.WalletTonConnectCloseSessionRequest{SessionID: sessionID}
	if body != nil {
		req.Body = body
	}
	_, err := c.Raw().WalletTonConnectCloseSession(ctx, req)
	if err != nil {
		return fmt.Errorf("ton connect close session: %w", err)
	}
	return nil
}

// TonConnectGetSessions lists the active Ton Connect sessions.
func (c *WalletMethods) TonConnectGetSessions(ctx context.Context) ([]*tg.TonConnectSession, error) {
	c.Log.Debug("TonConnectGetSessions")
	result, err := c.Raw().WalletTonConnectGetSessions(ctx)
	if err != nil {
		return nil, fmt.Errorf("ton connect get sessions: %w", err)
	}
	return result.Sessions, nil
}
