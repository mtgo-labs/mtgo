package telegram

import (
	"context"
	"testing"

	"github.com/mtgo-labs/mtgo/tg"
)

func TestGetWalletState(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletGetStateTypeID, &tg.WalletState{
		Address: "UQ...",
		Balance: 1500000000,
	})

	state, err := c.GetWalletState(context.Background())
	if err != nil {
		t.Fatalf("GetWalletState() error: %v", err)
	}
	if state.Address != "UQ..." || state.Balance != 1500000000 {
		t.Fatalf("state = %#v", state)
	}
}

func TestGetWalletState_Empty(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletGetStateTypeID, &tg.WalletStateEmpty{})

	state, err := c.GetWalletState(context.Background())
	if err != nil {
		t.Fatalf("GetWalletState() error: %v", err)
	}
	if state != nil {
		t.Fatalf("state = %#v, want nil for empty wallet", state)
	}
}

func TestGetWalletUserAddresses(t *testing.T) {
	c, inv := newClientWithMock(t)
	c.CachePeer(10, &tg.InputPeerUser{UserID: 10, AccessHash: 20})
	inv.setResult(tg.WalletGetUserAddressesTypeID, &tg.WalletUserAddresses{
		Addresses: []*tg.WalletUserAddress{{UserID: 10, Address: "UQabc"}},
	})

	addresses, err := c.GetWalletUserAddresses(context.Background(), []int64{10}, nil, false)
	if err != nil {
		t.Fatalf("GetWalletUserAddresses() error: %v", err)
	}
	if len(addresses) != 1 || addresses[0].Address != "UQabc" {
		t.Fatalf("addresses = %#v, want one", addresses)
	}
	req, ok := inv.lastCall().(*tg.WalletGetUserAddressesRequest)
	if !ok {
		t.Fatalf("expected WalletGetUserAddressesRequest, got %T", inv.lastCall())
	}
	if len(req.ID) != 1 || req.Force {
		t.Errorf("request = %#v, want one user without force", req)
	}
}

func TestGetWalletTransactions(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletGetTransactionsTypeID, &tg.WalletTransactions{
		Balance:      2000000000,
		Transactions: []*tg.WalletTransaction{{ID: "tx1", Amount: -100}},
		NextOffset:   "page2",
	})

	page, err := c.GetWalletTransactions(context.Background(), false, true, "", 0)
	if err != nil {
		t.Fatalf("GetWalletTransactions() error: %v", err)
	}
	if page.Balance != 2000000000 || len(page.Transactions) != 1 || page.NextOffset != "page2" {
		t.Fatalf("page = %#v, want balance and one tx", page)
	}
	req, ok := inv.lastCall().(*tg.WalletGetTransactionsRequest)
	if !ok || !req.Outbound || req.Limit != 25 {
		t.Fatalf("request = %#v, want outbound default limit", inv.lastCall())
	}
}

func TestGetWalletTransactionsByIDs(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletGetTransactionsByIDsTypeID, &tg.WalletTransactions{
		Transactions: []*tg.WalletTransaction{{ID: "tx7"}},
	})

	txs, err := c.GetWalletTransactionsByIDs(context.Background(), []string{"tx7"})
	if err != nil {
		t.Fatalf("GetWalletTransactionsByIDs() error: %v", err)
	}
	if len(txs) != 1 || txs[0].ID != "tx7" {
		t.Fatalf("transactions = %#v", txs)
	}
}

func TestSendWalletTransfer_RequiresExactlyOnePayload(t *testing.T) {
	c, _ := newClientWithMock(t)

	if err := c.SendWalletTransfer(context.Background(), 0, nil, nil); err == nil {
		t.Error("expected error with no payload")
	}
	if err := c.SendWalletTransfer(context.Background(), 0, []byte("a"), []byte("b")); err == nil {
		t.Error("expected error with both payloads")
	}
}

func TestSendWalletTransfer(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletSendTransferTypeID, &tg.Updates{})

	if err := c.SendWalletTransfer(context.Background(), 0, []byte("b64body"), nil); err != nil {
		t.Fatalf("SendWalletTransfer() error: %v", err)
	}
	req, ok := inv.lastCall().(*tg.WalletSendTransferRequest)
	if !ok {
		t.Fatalf("expected WalletSendTransferRequest, got %T", inv.lastCall())
	}
	if string(req.DataNormal) != "b64body" || req.DataGasless != nil {
		t.Errorf("request = %#v, want normal payload only", req)
	}
}

func TestGetWalletNFTs(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletGetNftsTypeID, &tg.WalletNftItems{
		Items:      []*tg.WalletNftItem{{Address: "UQnft", Name: "Gift #1"}},
		NextOffset: "n2",
	})

	nfts, err := c.GetWalletNFTs(context.Background(), "", 0)
	if err != nil {
		t.Fatalf("GetWalletNFTs() error: %v", err)
	}
	if len(nfts.Items) != 1 || nfts.Items[0].Name != "Gift #1" || nfts.NextOffset != "n2" {
		t.Fatalf("nfts = %#v, want one item", nfts)
	}
}

func TestExportSecretPhraseAndPart(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletExportSecretPhraseTypeID, &tg.WalletSecretPhraseParts{
		Token: "tok",
		Dcs:   []int32{1, 2},
	})
	inv.setResult(tg.WalletFetchEncryptedSecretPhrasePartTypeID, &tg.WalletEncryptedSecretPhrasePart{
		Data: []byte("share"),
	})

	parts, err := c.ExportSecretPhrase(t.Context(), "")
	if err != nil || parts.Token != "tok" || len(parts.Dcs) != 2 {
		t.Fatalf("ExportSecretPhrase = %+v, %v", parts, err)
	}
	// Without a password the request carries no SRP input.
	req, ok := inv.lastCall().(*tg.WalletExportSecretPhraseRequest)
	if !ok || req.Password != nil {
		t.Fatalf("request = %#v", inv.lastCall())
	}

	part, err := c.FetchEncryptedSecretPhrasePart(t.Context(), parts.Token, []byte("pk"))
	if err != nil || string(part.Data) != "share" {
		t.Fatalf("FetchEncryptedSecretPhrasePart = %+v, %v", part, err)
	}
}

func TestTonConnectSessionLifecycle(t *testing.T) {
	c, inv := newClientWithMock(t)
	inv.setResult(tg.WalletTonConnectCreateSessionTypeID, &tg.TonConnectSession{ID: 5})
	session, err := c.TonConnectCreateSession(t.Context(), "dapp-1", "https://dapp/manifest.json")
	if err != nil || session.ID != 5 {
		t.Fatalf("TonConnectCreateSession = %+v, %v", session, err)
	}
	req, ok := inv.lastCall().(*tg.WalletTonConnectCreateSessionRequest)
	if !ok || req.DappClientID != "dapp-1" {
		t.Fatalf("request = %#v", inv.lastCall())
	}

	inv.setResult(tg.WalletTonConnectGetSessionsTypeID, &tg.WalletTonConnectSessions{
		Sessions: []*tg.TonConnectSession{{ID: 5}},
	})
	sessions, err := c.TonConnectGetSessions(t.Context())
	if err != nil || len(sessions) != 1 {
		t.Fatalf("TonConnectGetSessions = %+v, %v", sessions, err)
	}
}
