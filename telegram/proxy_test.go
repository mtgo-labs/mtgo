package telegram

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

var (
	_ transport.ContextDialer = (*socksDialer)(nil)
	_ transport.ContextDialer = (*httpProxyDialer)(nil)
)

func TestProxyFromURL(t *testing.T) {
	t.Run("socks5 with auth", func(t *testing.T) {
		got, err := ProxyFromURL("socks5://user:pass@10.0.0.1:1080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p, ok := got.(*Proxy)
		if !ok {
			t.Fatalf("expected *Proxy, got %T", got)
		}
		if p.Addr != "10.0.0.1:1080" {
			t.Errorf("Addr = %q, want 10.0.0.1:1080", p.Addr)
		}
		if p.Protocol != "socks5" {
			t.Errorf("Protocol = %q, want socks5", p.Protocol)
		}
		if p.Username != "user" || p.Password != "pass" {
			t.Errorf("auth = %q/%q, want user/pass", p.Username, p.Password)
		}
	})

	t.Run("socks5h normalizes to socks5", func(t *testing.T) {
		got, err := ProxyFromURL("socks5h://10.0.0.1:1080")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := got.(*Proxy)
		if p.Protocol != "socks5" {
			t.Errorf("Protocol = %q, want socks5", p.Protocol)
		}
		if p.Username != "" || p.Password != "" {
			t.Errorf("expected empty auth, got %q/%q", p.Username, p.Password)
		}
	})

	t.Run("socks4 defaults port to 1080", func(t *testing.T) {
		got, err := ProxyFromURL("socks4://10.0.0.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := got.(*Proxy)
		if p.Addr != "10.0.0.1:1080" {
			t.Errorf("Addr = %q, want 10.0.0.1:1080", p.Addr)
		}
		if p.Protocol != "socks4" {
			t.Errorf("Protocol = %q, want socks4", p.Protocol)
		}
	})

	t.Run("http defaults port to 8080", func(t *testing.T) {
		got, err := ProxyFromURL("http://proxy.local")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := got.(*Proxy)
		if p.Addr != "proxy.local:8080" || p.Protocol != "http" {
			t.Errorf("got %+v", p)
		}
	})

	t.Run("https defaults port to 443", func(t *testing.T) {
		got, err := ProxyFromURL("https://proxy.local")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := got.(*Proxy)
		if p.Addr != "proxy.local:443" || p.Protocol != "https" {
			t.Errorf("got %+v", p)
		}
	})

	t.Run("mtproxy secret from userinfo", func(t *testing.T) {
		got, err := ProxyFromURL("mtproxy://deadbeef@proxy.local:443")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := got.(*MTProxyConfig)
		if !ok {
			t.Fatalf("expected *MTProxyConfig, got %T", got)
		}
		if m.Addr != "proxy.local:443" || m.Secret != "deadbeef" {
			t.Errorf("got %+v", m)
		}
	})

	t.Run("mtproxy secret from query, default port", func(t *testing.T) {
		got, err := ProxyFromURL("mtproxy://proxy.local?secret=ab12")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := got.(*MTProxyConfig)
		if m.Addr != "proxy.local:443" || m.Secret != "ab12" {
			t.Errorf("got %+v", m)
		}
	})

	t.Run("mtproxy missing secret errors", func(t *testing.T) {
		_, err := ProxyFromURL("mtproxy://proxy.local:443")
		if !errors.Is(err, ErrMTProxySecretRequired) {
			t.Fatalf("expected ErrMTProxySecretRequired, got %v", err)
		}
	})

	t.Run("tg proxy valid", func(t *testing.T) {
		got, err := ProxyFromURL("tg://proxy?server=proxy.local&port=1080&secret=zz")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := got.(*MTProxyConfig)
		if !ok {
			t.Fatalf("expected *MTProxyConfig, got %T", got)
		}
		if m.Addr != "proxy.local:1080" || m.Secret != "zz" {
			t.Errorf("got %+v", m)
		}
	})

	t.Run("tg proxy missing field errors", func(t *testing.T) {
		_, err := ProxyFromURL("tg://proxy?server=proxy.local&secret=zz")
		if !errors.Is(err, ErrProxyParamsRequired) {
			t.Fatalf("expected ErrProxyParamsRequired, got %v", err)
		}
	})

	t.Run("tg proxy invalid port errors", func(t *testing.T) {
		_, err := ProxyFromURL("tg://proxy?server=proxy.local&port=abc&secret=zz")
		if err == nil || !strings.Contains(err.Error(), "invalid port") {
			t.Fatalf("expected invalid-port error, got %v", err)
		}
	})

	t.Run("unsupported scheme errors", func(t *testing.T) {
		_, err := ProxyFromURL("ftp://proxy.local")
		if err == nil || !strings.Contains(err.Error(), "unsupported proxy scheme") {
			t.Fatalf("expected unsupported-scheme error, got %v", err)
		}
	})
}

func TestBasicAuth(t *testing.T) {
	cases := []struct{ user, pass string }{
		{"", ""},
		{"a", ""},
		{"ab", ""},
		{"abc", ""},
		{"user", "password"},
		{"admin", "P@ss w0rd!"},
		{strings.Repeat("x", 50), strings.Repeat("y", 17)},
	}
	for _, c := range cases {
		want := base64.StdEncoding.EncodeToString([]byte(c.user + ":" + c.pass))
		got := basicAuth(c.user, c.pass)
		if got != want {
			t.Errorf("basicAuth(%q,%q) = %q, want %q", c.user, c.pass, got, want)
		}
	}
}

func TestBytesEndsWith(t *testing.T) {
	cases := []struct {
		name string
		data string
		suf  string
		want bool
	}{
		{"exact", "HTTP/1.1 200 OK\r\n\r\n", "\r\n\r\n", true},
		{"partial", "HTTP/1.1 200 OK\r\n", "\r\n\r\n", false},
		{"empty data", "", "\r\n\r\n", false},
		{"empty suffix", "abc", "", true},
		{"equal", "abc", "abc", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := bytesEndsWith([]byte(c.data), []byte(c.suf)); got != c.want {
				t.Errorf("bytesEndsWith(%q,%q) = %v, want %v", c.data, c.suf, got, c.want)
			}
		})
	}
}

func TestDialDeadline(t *testing.T) {
	ctxDeadline := time.Now().Add(50 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), ctxDeadline)
	defer cancel()

	deadline, ok := dialDeadline(ctx, time.Second)
	if !ok {
		t.Fatal("dialDeadline should report a deadline")
	}
	if !deadline.Equal(ctxDeadline) {
		t.Fatalf("deadline = %v, want context deadline %v", deadline, ctxDeadline)
	}

	deadline, ok = dialDeadline(context.Background(), time.Millisecond)
	if !ok {
		t.Fatal("dialDeadline with timeout should report a deadline")
	}
	if time.Until(deadline) > time.Second {
		t.Fatalf("deadline too far in future: %v", deadline)
	}

	_, ok = dialDeadline(context.Background(), 0)
	if ok {
		t.Fatal("dialDeadline without timeout or context deadline should report no deadline")
	}
}

func TestNewProxyDialer(t *testing.T) {
	cases := []struct {
		name     string
		protocol string
		wantType string
	}{
		{"socks5", "socks5", "*telegram.socksDialer"},
		{"socks5h", "socks5h", "*telegram.socksDialer"},
		{"socks4", "socks4", "*telegram.socksDialer"},
		{"http", "http", "*telegram.httpProxyDialer"},
		{"https", "https", "*telegram.httpProxyDialer"},
		{"empty defaults to socks5", "", "*telegram.socksDialer"},
		{"unknown defaults to socks5", "garbage", "*telegram.socksDialer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var forward transport.Dialer // unused for type dispatch
			d := newProxyDialer(&Proxy{Addr: "x:1", Protocol: c.protocol}, forward)
			if got := typeName(d); got != c.wantType {
				t.Errorf("protocol %q: got %s, want %s", c.protocol, got, c.wantType)
			}
		})
	}
}

func typeName(v any) string {
	switch v.(type) {
	case *socksDialer:
		return "*telegram.socksDialer"
	case *httpProxyDialer:
		return "*telegram.httpProxyDialer"
	default:
		return "other"
	}
}
