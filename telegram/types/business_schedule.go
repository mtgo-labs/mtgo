package types

// BusinessSchedule represents the operating schedule type for a Telegram business account.
type BusinessSchedule string

// Business schedule types.
const (
	// BusinessScheduleAlwaysOpen indicates the business is always open.
	BusinessScheduleAlwaysOpen BusinessSchedule = "always_open"
	// BusinessScheduleByDate indicates the business operates on specific dates.
	BusinessScheduleByDate BusinessSchedule = "by_date"
	// BusinessScheduleCustom indicates a custom schedule.
	BusinessScheduleCustom BusinessSchedule = "custom"
	// BusinessScheduleOutside indicates outside business hours.
	BusinessScheduleOutside BusinessSchedule = "outside"
)

// String returns the string representation of the BusinessSchedule.
func (b BusinessSchedule) String() string { return string(b) }
