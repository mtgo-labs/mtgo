package types

// MediaAreaType enumerates the types of interactive areas that can be placed on a media item.
type MediaAreaType string

// Media area type constants representing the different kinds of interactive media areas.
const (
	MediaAreaTypeVenue           MediaAreaType = "venue"
	MediaAreaTypeGeoPoint        MediaAreaType = "geo_point"
	MediaAreaTypePreviouslyMedia MediaAreaType = "previously_media"
	MediaAreaTypeValue           MediaAreaType = "value"
	MediaAreaTypePrice           MediaAreaType = "price"
	MediaAreaTypeChannelPost     MediaAreaType = "channel_post"
	MediaAreaTypeURL             MediaAreaType = "url"
	MediaAreaTypeWeather         MediaAreaType = "weather"
)

// String returns the string representation of the MediaAreaType.
func (m MediaAreaType) String() string { return string(m) }
