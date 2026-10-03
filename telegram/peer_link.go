package telegram

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/mtgo-labs/mtgo/telegram/peers"
)

// linkKind classifies a parsed peer reference string.
type linkKind int

const (
	linkInvalid linkKind = iota
	linkUsername
	linkPhone
	linkID
	linkInviteHash
)

// link is the uniform parse result for every peer reference string accepted
// by the library. One parser produces it; ChatRefFrom and parseJoinLink both
// consume it so the two entry points can never disagree about what a string
// means.
type link struct {
	kind     linkKind
	username string // public username without '@'
	phone    string // phone without '+' or '00'
	id       int64  // numeric peer ID (raw or marked)
	hash     string // invite hash without '+'
}

// parseLink parses a peer reference string. Accepted forms:
//
//   - usernames: "durov", "@durov"
//   - phone numbers: "+1234567890", "001234567890"
//   - numeric IDs: "123456789", "-1000000001234"
//   - t.me links (http(s):// or schemeless; t.me, telegram.me, telegram.dog):
//     "/username", "/@username", "/+hash" (invite), "/joinchat/<hash>"
//     (legacy invite), "/c/<channelID>/<messageID>" (private channel post)
//   - tg:// deep links: "tg://resolve?domain=<username>",
//     "tg://join?invite=<hash>"
//
// Bare strings that are not phone numbers or numeric IDs are classified as
// usernames; callers applying stricter rules (e.g. the invite-hash heuristic
// for bare hashes) do so on top of this result.
func parseLink(s string) link {
	s = strings.TrimSpace(s)
	if s == "" {
		return link{}
	}

	if strings.HasPrefix(s, "tg://") {
		if u, err := url.Parse(s); err == nil {
			switch strings.ToLower(u.Host) {
			case "resolve":
				if d := u.Query().Get("domain"); d != "" {
					return link{kind: linkUsername, username: strings.TrimPrefix(d, "@")}
				}
			case "join":
				if h := u.Query().Get("invite"); h != "" {
					return link{kind: linkInviteHash, hash: strings.TrimPrefix(h, "+")}
				}
			}
		}
		return link{}
	}

	// HTTP(S) URLs: only t.me-family hosts carry peer references.
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		if u, err := url.Parse(s); err == nil {
			if target, ok := parseTmePath(u.Host, u.Path); ok {
				return target
			}
		}
		return link{}
	}

	// Schemeless t.me links, e.g. "t.me/username".
	if idx := strings.Index(s, "/"); idx > 0 {
		if host := strings.ToLower(s[:idx]); isTmeHost(host) {
			if target, ok := parseTmePath(host, s[idx:]); ok {
				return target
			}
		}
	}

	// Phone numbers: "+..." or "00..." prefixes.
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "00") {
		return link{kind: linkPhone, phone: peers.NormalizePhone(s)}
	}

	// Numeric IDs (raw or marked).
	if id, err := strconv.ParseInt(s, 10, 64); err == nil {
		return link{kind: linkID, id: id}
	}

	// Bare username, possibly with '@'.
	return link{kind: linkUsername, username: strings.TrimPrefix(s, "@")}
}

func isTmeHost(host string) bool {
	return host == "t.me" || host == "telegram.me" || host == "telegram.dog"
}

// parseTmePath parses the path of a t.me-family link.
func parseTmePath(host, path string) (link, bool) {
	if !isTmeHost(strings.ToLower(host)) {
		return link{}, false
	}
	p := strings.Trim(path, "/")
	p = strings.TrimSpace(p)
	if p == "" {
		return link{}, false
	}
	if strings.HasPrefix(p, "+") {
		return link{kind: linkInviteHash, hash: p[1:]}, true
	}
	if strings.HasPrefix(p, "joinchat/") {
		h := strings.TrimPrefix(p, "joinchat/")
		if h != "" {
			return link{kind: linkInviteHash, hash: h}, true
		}
		return link{}, false
	}
	if strings.HasPrefix(p, "c/") {
		rest := strings.TrimPrefix(p, "c/")
		channelID := rest
		if idx := strings.Index(rest, "/"); idx >= 0 {
			channelID = rest[:idx]
		}
		if id, err := strconv.ParseInt(channelID, 10, 64); err == nil {
			return link{kind: linkID, id: id}, true
		}
		return link{}, false
	}
	// Username: only the first path segment, without any query string
	// (schemeless links are not pre-parsed by net/url).
	if idx := strings.IndexAny(p, "/?"); idx >= 0 {
		p = p[:idx]
	}
	return link{kind: linkUsername, username: strings.TrimPrefix(p, "@")}, true
}
