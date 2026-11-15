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
