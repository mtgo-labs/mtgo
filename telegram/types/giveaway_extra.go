package types

// GiveawayCreated indicates that a giveaway was created.
type GiveawayCreated struct{}

// GiveawayCompleted indicates that a giveaway has ended.
type GiveawayCompleted struct{}

// GiveawayPrizeStars represents a Stars prize for a giveaway.
type GiveawayPrizeStars struct {
	Stars int64
}
