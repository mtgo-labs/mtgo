package types

import (
	"github.com/mtgo-labs/mtgo/tg"
)

type MessageReactions struct {
	Reactions            []Reaction
	AreTags              bool
	PaidReactors         []*PaidReactor
	CanGetAddedReactions bool
}

type ReactionList struct {
	Hash      int32
	Reactions []AvailableReaction
	Modified  bool
}

type AvailableReaction struct {
	Emoji    string
	Title    string
	Inactive bool
	Premium  bool
}

type PeerReaction struct {
	Reaction Reaction
	ChatID   int64
	Date     int32
	IsBig    bool
	IsUnread bool
	IsMine   bool
}

type PeerReactionList struct {
	Count      int32
	Reactions  []PeerReaction
	NextOffset string
}

func ParseAvailableReactions(raw tg.AvailableReactionsClass) *ReactionList {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case *tg.MessagesAvailableReactions:
		out := &ReactionList{Hash: v.Hash, Modified: true}
		for _, r := range v.Reactions {
			if r != nil {
				out.Reactions = append(out.Reactions, AvailableReaction{
					Emoji:    r.Reaction,
					Title:    r.Title,
					Inactive: r.Inactive,
					Premium:  r.Premium,
				})
			}
		}
		return out
	case *tg.MessagesAvailableReactionsNotModified:
		return &ReactionList{Modified: false}
	}
	return nil
}

type StarsTransaction struct {
	ID          string
	Amount      int64
	Nanostars   int32
	Date        int32
	IsRefund    bool
	IsPending   bool
	IsFailed    bool
	IsGift      bool
	IsReaction  bool
	Title       string
	Description string
	MsgID       int32
	ChatID      int64
}

type StarsStatus struct {
	Balance          int64
	NanostarsBalance int32
	Transactions     []StarsTransaction
	NextOffset       string
	Subscriptions    []StarsSubscription
	SubsNextOffset   string
}

type StarsSubscription struct {
	ID          string
	Title       string
	ChatInvite  string
	UntilDate   int32
	IsCanceled  bool
	IsMissing   bool
	ChatID      int64
}

func ParseStarsStatus(raw *tg.PaymentsStarsStatus) *StarsStatus {
	if raw == nil {
		return nil
	}
	out := &StarsStatus{
		NextOffset:     raw.NextOffset,
		SubsNextOffset: raw.SubscriptionsNextOffset,
	}
	if raw.Balance != nil {
		if amt, ok := raw.Balance.(*tg.StarsAmount); ok {
			out.Balance = amt.Amount
			out.NanostarsBalance = amt.Nanos
		}
	}
	for _, tx := range raw.History {
		if tx == nil {
			continue
		}
		st := StarsTransaction{
			ID:          tx.ID,
			Date:        tx.Date,
			IsRefund:    tx.Refund,
			IsPending:   tx.Pending,
			IsFailed:    tx.Failed,
			IsGift:      tx.Gift,
			IsReaction:  tx.Reaction,
			Title:       tx.Title,
			Description: tx.Description,
			MsgID:       tx.MsgID,
		}
		if tx.Amount != nil {
			if amt, ok := tx.Amount.(*tg.StarsAmount); ok {
				st.Amount = amt.Amount
				st.Nanostars = amt.Nanos
			}
		}
		if tx.Peer != nil {
			st.ChatID = extractStarsPeerID(tx.Peer)
		}
		out.Transactions = append(out.Transactions, st)
	}
	for _, sub := range raw.Subscriptions {
		if sub == nil {
			continue
		}
		ss := StarsSubscription{
			ID:         sub.ID,
			Title:      sub.Title,
			ChatInvite: sub.ChatInviteHash,
			UntilDate:  sub.UntilDate,
			IsCanceled: sub.Canceled,
			IsMissing:  sub.MissingBalance,
		}
		if sub.Peer != nil {
			ss.ChatID = ExtractChatID(sub.Peer)
		}
		out.Subscriptions = append(out.Subscriptions, ss)
	}
	return out
}

func extractStarsPeerID(peer tg.StarsTransactionPeerClass) int64 {
	switch v := peer.(type) {
	case *tg.StarsTransactionPeer:
		return ExtractChatID(v.Peer)
	case *tg.StarsTransactionPeerAPI:
		return 0
	case *tg.StarsTransactionPeerPremiumBot:
		return 0
	case *tg.StarsTransactionPeerAppStore:
		return 0
	case *tg.StarsTransactionPeerPlayMarket:
		return 0
	case *tg.StarsTransactionPeerFragment:
		return 0
	case *tg.StarsTransactionPeerAds:
		return 0
	}
	return 0
}

func ExtractChatID(peer tg.PeerClass) int64 {
	switch v := peer.(type) {
	case *tg.PeerUser:
		return v.UserID
	case *tg.PeerChat:
		return int64(-v.ChatID)
	case *tg.PeerChannel:
		return int64(-100 - v.ChannelID)
	}
	return 0
}
