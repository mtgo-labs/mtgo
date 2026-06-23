package tlgen

// fieldDescriptions is a curated overlay mapping TL constructor.field names to
// human-readable descriptions. The TL schema source (api.tl) carries no field
// descriptions — they live only on core.telegram.org HTML pages, and even
// MadelineProto's scraped overlay (extracted.json) lacks critical unit
// annotations (e.g. "in seconds" for edit_time_limit).
//
// This overlay covers the high-value fields where units or semantics are not
// derivable from the field name alone. Descriptions are phrased to read
// naturally after the Go field name (e.g. "EditTimeLimit is the maximum age
// in seconds..."), so they start with "is" or a verb.
//
// Keys are "constructorName.fieldName" using the TL (snake_case) names.
var fieldDescriptions = map[string]string{
	// ---- config#cc1a241e — units and semantics that matter to callers ----
	"config.edit_time_limit":            "is the maximum age (in seconds) of a message that may still be edited; messages older than this cannot be edited.",
	"config.revoke_time_limit":          "is the maximum age (in seconds) of a channel or supergroup message that may be deleted.",
	"config.revoke_pm_time_limit":       "is the maximum age (in seconds) of a private message that may be deleted for both participants.",
	"config.channels_read_media_period": "is the period (in seconds) after which round videos and voice messages in channels must be marked as read.",
	"config.online_update_period_ms":    "is the interval (in milliseconds) at which the client should call account.updateStatus.",
	"config.offline_blur_timeout_ms":    "is the delay (in milliseconds) before the offline status should be sent to the server.",
	"config.offline_idle_timeout_ms":    "is the time (in milliseconds) without user activity after which the client is treated as offline.",
	"config.online_cloud_timeout_ms":    "is the time window (in milliseconds) during which being online on another client delays the offline notification.",
	"config.notify_cloud_delay_ms":      "is the delay (in milliseconds) for the offline notification when online from another client.",
	"config.notify_default_delay_ms":    "is the delay (in milliseconds) for notifications when another client is online.",
	"config.push_chat_period_ms":        "is the push notification period (in milliseconds) for chat messages. Not for client use.",
	"config.push_chat_limit":            "is the maximum number of push chat notifications. Not for client use.",
	"config.call_receive_timeout_ms":    "is the maximum outgoing ring time (in milliseconds) for VoIP calls before hanging up.",
	"config.call_ring_timeout_ms":       "is the maximum incoming ring time (in milliseconds) for VoIP calls before auto-refusing.",
	"config.call_connect_timeout_ms":    "is the VoIP connection timeout (in milliseconds) before aborting the call.",
	"config.call_packet_timeout_ms":     "is the maximum time (in milliseconds) without receiving a VoIP packet before aborting the call.",
	"config.message_length_max":         "is the maximum message length in UTF-8 codepoints (not bytes).",
	"config.caption_length_max":         "is the maximum media caption length in UTF-8 codepoints (not bytes).",
	"config.chat_size_max":              "is the maximum member count for normal (basic) groups.",
	"config.megagroup_size_max":         "is the maximum member count for supergroups.",
	"config.forwarded_count_max":        "is the maximum number of messages that can be forwarded at once.",
	"config.stickers_recent_limit":      "is the maximum number of recent stickers retained.",
	"config.rating_e_decay":             "is the exponential decay rate for computing top peer rating.",
	"config.tmp_sessions":               "is the number of temporary Telegram Passport sessions.",
	"config.webfile_dc_id":              "is the data center ID to use for downloading webfiles.",
	"config.dc_txt_domain_name":         "is the domain name for fetching the encrypted DC list from DNS TXT records.",
	"config.me_url_prefix":              "is the domain prefix used to parse tg:// and t.me deep links.",
	"config.autologin_token":            "is the token for seamless Telegram Login URL authorization.",
	"config.expires":                    "is the expiration timestamp of this config; refetch via help.getConfig when it expires.",
	"config.date":                       "is the current date at the server (Unix timestamp).",
	"config.test_mode":                  "indicates whether the client is connected to the test data centers.",
	"config.this_dc":                    "is the ID of the data center that returned this config.",
	"config.dc_options":                 "is the list of data center connection options.",
	"config.suggested_lang_code":        "is the suggested language pack code for the client.",
	"config.lang_pack_version":          "is the version of the suggested language pack.",
	"config.base_lang_pack_version":     "is the version of the base language pack.",

	// ---- dcOption#18b7a10d — endpoint flags that affect connection logic ----
	"dcOption.ipv6":           "indicates whether this endpoint uses an IPv6 address.",
	"dcOption.media_only":     "indicates that this endpoint should only be used for media transfers.",
	"dcOption.tcpo_only":      "indicates that this endpoint only supports TCP+TLS (fake transport), not plain MTProto over TCP.",
	"dcOption.cdn":            "indicates that this is a CDN redirect endpoint for file downloads.",
	"dcOption.static":         "indicates that this is a static, non-changing endpoint.",
	"dcOption.this_port_only": "indicates that only this specific port should be used.",
	"dcOption.secret":         "is the MTProxy secret bytes for this endpoint, if applicable.",
}

// fieldDescription looks up the curated description for a TL field.
// Returns "" if no description is available (the common case — most fields
// don't need annotation).
func fieldDescription(qualName, fieldName string) string {
	return fieldDescriptions[qualName+"."+fieldName]
}

// structDescriptions is a curated overlay mapping TL constructor names to
// additional struct-level documentation. This is used for constructors where
// usage semantics need clarification beyond the auto-generated boilerplate.
var structDescriptions = map[string]string{
	"config": "The library does not auto-apply any fields from Config. Callers must read and use values (edit_time_limit, dc_options, etc.) as needed for their application layer.",
}

// structDescription looks up additional struct-level documentation.
// Returns "" if none is available.
func structDescription(qualName string) string {
	return structDescriptions[qualName]
}
