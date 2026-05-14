package types

// ChatJoinRequest represents a pending request from a user to join a chat that
// requires admin approval.
type ChatJoinRequest struct {
	ChatID      int64
	UserID      int64
	Date        int32
	Bio         string
	InviteLink  *ChatInviteLink
	binder      Binder
}

func (r *ChatJoinRequest) SetBinder(b Binder) {
	r.binder = b
}

func (r *ChatJoinRequest) Approve() error {
	if r.binder == nil {
		return ErrNoBinder
	}
	return r.binder.BoundApproveJoinRequest(r.ChatID, r.UserID)
}

func (r *ChatJoinRequest) Decline() error {
	if r.binder == nil {
		return ErrNoBinder
	}
	return r.binder.BoundDeclineJoinRequest(r.ChatID, r.UserID)
}
