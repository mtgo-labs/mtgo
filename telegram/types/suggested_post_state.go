package types

// SuggestedPostState represents the lifecycle state of a suggested post.
// Suggested posts allow channel admins to propose content that the channel
// owner can approve, decline, or post.
type SuggestedPostState string

const (
	// SuggestedPostStatePending indicates the suggested post is awaiting review
	// by the channel owner.
	SuggestedPostStatePending SuggestedPostState = "pending"
	// SuggestedPostStateApproved indicates the channel owner has approved the
	// suggested post for publication.
	SuggestedPostStateApproved SuggestedPostState = "approved"
	// SuggestedPostStateDeclined indicates the channel owner has rejected the
	// suggested post.
	SuggestedPostStateDeclined SuggestedPostState = "declined"
	// SuggestedPostStatePosted indicates the suggested post has been published
	// to the channel.
	SuggestedPostStatePosted SuggestedPostState = "posted"
)

// String returns the string representation of the SuggestedPostState.
func (s SuggestedPostState) String() string { return string(s) }
