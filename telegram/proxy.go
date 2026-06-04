package telegram

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mtgo-labs/mtgo/internal/transport"
)

// ProxyFromURL parses a proxy URL and returns a *Proxy or *MTProxyConfig.
//
// Supported formats:
//
//	socks5://[user:pass@]host:port
//	socks4://[userid@]host:port
//	http://[user:pass@]host:port
//	https://[user:pass@]host:port
//	mtproxy://secret@host:port
//	tg://proxy?server=host&port=port&secret=secret
func ProxyFromURL(raw string) (interface{}, error) {
	if strings.HasPrefix(raw, "tg://proxy?") {
		return parseTgProxy(raw)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("telegram: parse proxy url: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "socks5", "socks5h":
		return parseSOCKS(u, "socks5")
	case "socks4":
		return parseSOCKS(u, "socks4")
	case "http", "https":
		return parseHTTP(u)
	case "mtproxy":
		return parseMTProxy(u)
	default:
		return nil, fmt.Errorf("telegram: unsupported proxy scheme %q", scheme)
	}
}

func parseSOCKS(u *url.URL, version string) (*Proxy, error) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "1080"
	}
	p := &Proxy{
		Addr:     net.JoinHostPort(host, port),
		Protocol: version,
	}
	if u.User != nil {
		p.Username = u.User.Username()
		p.Password, _ = u.User.Password()
	}
	return p, nil
}

func parseHTTP(u *url.URL) (*Proxy, error) {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "8080"
		}
	}
	p := &Proxy{
		Addr:     net.JoinHostPort(host, port),
		Protocol: u.Scheme,
	}
	if u.User != nil {
		p.Username = u.User.Username()
		p.Password, _ = u.User.Password()
	}
	return p, nil
}

func parseMTProxy(u *url.URL) (*MTProxyConfig, error) {
	secret := u.User.Username()
	if secret == "" {
		if s := u.Query().Get("secret"); s != "" {
			secret = s
		}
	}
	if secret == "" {
		return nil, ErrMTProxySecretRequired
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return &MTProxyConfig{
		Addr:   net.JoinHostPort(host, port),
		Secret: secret,
	}, nil
}

func parseTgProxy(raw string) (*MTProxyConfig, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("telegram: parse tg proxy url: %w", err)
	}
	server := u.Query().Get("server")
	portStr := u.Query().Get("port")
	secret := u.Query().Get("secret")
	if server == "" || portStr == "" || secret == "" {
		return nil, ErrProxyParamsRequired
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("telegram: tg://proxy invalid port %q", portStr)
	}
	return &MTProxyConfig{
		Addr:   net.JoinHostPort(server, strconv.Itoa(port)),
		Secret: secret,
	}, nil
}

// socksDialer implements transport.Dialer via a SOCKS4/5 proxy.
type socksDialer struct {
	proxyAddr string
	username  string
	password  string
	version   string
	forward   transport.Dialer
}

func (d *socksDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	conn, err := d.forward.Dial(network, d.proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("socks connect to %s: %w", d.proxyAddr, err)
	}
	if d.version == "socks4" {
		return d.socks4Handshake(conn, address)
	}
	return d.socks5Handshake(conn, address)
}

func (d *socksDialer) socks5Handshake(conn net.Conn, address string) (net.Conn, error) {
	var authMethods []byte
	if d.username != "" {
		authMethods = []byte{0x05, 0x02, 0x00, 0x02}
	} else {
		authMethods = []byte{0x05, 0x01, 0x00}
	}
	if _, err := conn.Write(authMethods); err != nil {
		conn.Close()
		return nil, err
	}
	buf := make([]byte, 2)
	if _, err := readFull(conn, buf); err != nil {
		conn.Close()
		return nil, err
	}
	if buf[0] != 0x05 {
		conn.Close()
		return nil, fmt.Errorf("socks5: unexpected version 0x%02x", buf[0])
	}
	if buf[1] == 0x02 {
		if _, err := conn.Write([]byte{0x01, byte(len(d.username))}); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err := conn.Write([]byte(d.username)); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err := conn.Write([]byte{byte(len(d.password))}); err != nil {
			conn.Close()
			return nil, err
		}
		if _, err := conn.Write([]byte(d.password)); err != nil {
			conn.Close()
			return nil, err
		}
		authBuf := make([]byte, 2)
		if _, err := readFull(conn, authBuf); err != nil {
			conn.Close()
			return nil, err
		}
		if authBuf[1] != 0x00 {
			conn.Close()
			return nil, fmt.Errorf("socks5: auth failed (0x%02x)", authBuf[1])
		}
	}
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		conn.Close()
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		conn.Close()
		return nil, err
	}
	req := []byte{0x05, 0x01, 0x00}
	ip := net.ParseIP(host)
	if ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			req = append(req, 0x01)
			req = append(req, ip4...)
		} else {
			req = append(req, 0x04)
			req = append(req, ip.To16()...)
		}
	} else {
		req = append(req, 0x03, byte(len(host)))
		req = append(req, []byte(host)...)
	}
	req = append(req, byte(port>>8), byte(port))
	if _, err := conn.Write(req); err != nil {
		conn.Close()
		return nil, err
	}
	resp := make([]byte, 4)
	if _, err := readFull(conn, resp); err != nil {
		conn.Close()
		return nil, err
	}
	if resp[1] != 0x00 {
		conn.Close()
		return nil, fmt.Errorf("socks5: connect failed (0x%02x)", resp[1])
	}
	var skip int
	switch resp[3] {
	case 0x01:
		skip = 4
	case 0x03:
		b := make([]byte, 1)
		if _, err := readFull(conn, b); err != nil {
			conn.Close()
			return nil, err
		}
		skip = int(b[0])
	case 0x04:
		skip = 16
	}
	if skip > 0 {
		if _, err := readFull(conn, make([]byte, skip+2)); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return conn, nil
}

func (d *socksDialer) socks4Handshake(conn net.Conn, address string) (net.Conn, error) {
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		conn.Close()
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		conn.Close()
		return nil, err
	}
	ip := net.ParseIP(host)

	if ip == nil {
		// SOCKS4a: domain name support — use 0.0.0.1 as placeholder IP
		// and append the domain as a null-terminated string after the user ID.
		req := []byte{0x04, 0x01, byte(port >> 8), byte(port), 0x00, 0x00, 0x00, 0x01}
		req = append(req, []byte(d.username)...)
		req = append(req, 0x00)
		req = append(req, []byte(host)...)
		req = append(req, 0x00)
		if _, err := conn.Write(req); err != nil {
			conn.Close()
			return nil, err
		}
	} else if ip4 := ip.To4(); ip4 == nil {
		// IPv6: SOCKS4 doesn't support IPv6 addresses. Resolve to IPv4
		// via DNS first and use the resolved address.
		ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("socks4: resolve ipv6 host %s: %w", host, err)
		}
		var ip4Addr net.IP
		for _, addr := range ips {
			if v4 := addr.IP.To4(); v4 != nil {
				ip4Addr = v4
				break
			}
		}
		if ip4Addr == nil {
			conn.Close()
			return nil, fmt.Errorf("socks4: no ipv4 address found for host %s", host)
		}
		req := []byte{0x04, 0x01, byte(port >> 8), byte(port)}
		req = append(req, ip4Addr...)
		req = append(req, []byte(d.username)...)
		req = append(req, 0x00)
		if _, err := conn.Write(req); err != nil {
			conn.Close()
			return nil, err
		}
	} else {
		req := []byte{0x04, 0x01, byte(port >> 8), byte(port)}
		req = append(req, ip4...)
		req = append(req, []byte(d.username)...)
		req = append(req, 0x00)
		if _, err := conn.Write(req); err != nil {
			conn.Close()
			return nil, err
		}
	}
	resp := make([]byte, 8)
	if _, err := readFull(conn, resp); err != nil {
		conn.Close()
		return nil, err
	}
	if resp[1] != 0x5a {
		conn.Close()
		return nil, fmt.Errorf("socks4: rejected (0x%02x)", resp[1])
	}
	return conn, nil
}

// httpProxyDialer implements transport.Dialer via an HTTP CONNECT proxy.
type httpProxyDialer struct {
	proxyAddr string
	username  string
	password  string
	forward   transport.Dialer
}

func (d *httpProxyDialer) Dial(network, address string, timeout time.Duration) (net.Conn, error) {
	conn, err := d.forward.Dial(network, d.proxyAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("http proxy connect to %s: %w", d.proxyAddr, err)
	}
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", address, address)
	if d.username != "" {
		req += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n",
			basicAuth(d.username, d.password))
	}
	req += "\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		conn.Close()
		return nil, err
	}
	buf := make([]byte, 4096)
	n, err := readUntil(conn, buf, []byte("\r\n\r\n"))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("http proxy response: %w", err)
	}
	resp := string(buf[:n])
	if !strings.Contains(resp, " 200 ") {
		conn.Close()
		return nil, fmt.Errorf("http proxy: %s", strings.Split(resp, "\r\n")[0])
	}
	return conn, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func readUntil(conn net.Conn, buf []byte, delimiter []byte) (int, error) {
	br := bufio.NewReader(conn)
	total := 0
	for total < len(buf) {
		b, err := br.ReadByte()
		if err != nil {
			return total, err
		}
		buf[total] = b
		total++
		if total >= len(delimiter) && bytesEndsWith(buf[:total], delimiter) {
			return total, nil
		}
	}
	return total, ErrProxyResponseTooLarge
}

func bytesEndsWith(data, suffix []byte) bool {
	if len(data) < len(suffix) {
		return false
	}
	return string(data[len(data)-len(suffix):]) == string(suffix)
}

func basicAuth(user, pass string) string {
	s := user + ":" + pass
	dst := make([]byte, ((len(s)+2)/3)*4)
	j := 0
	for i := 0; i < len(s); i += 3 {
		var buf [3]byte
		n := copy(buf[:], s[i:])
		_ = n
		_ = j
	}
	v := []byte(s)
	for i := 0; i < len(v); i += 3 {
		b := uint32(v[i]) << 16
		if i+1 < len(v) {
			b |= uint32(v[i+1]) << 8
		}
		if i+2 < len(v) {
			b |= uint32(v[i+2])
		}
		const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		dst[j] = enc[b>>18&0x3F]
		dst[j+1] = enc[b>>12&0x3F]
		dst[j+2] = enc[b>>6&0x3F]
		dst[j+3] = enc[b&0x3F]
		j += 4
	}
	rem := len(v) % 3
	switch rem {
	case 1:
		dst[j-2] = '='
		dst[j-1] = '='
	case 2:
		dst[j-1] = '='
	}
	return string(dst)
}

func newProxyDialer(p *Proxy, forward transport.Dialer) transport.Dialer {
	proto := p.Protocol
	if proto == "" {
		proto = "socks5"
	}
	switch proto {
	case "socks4":
		return &socksDialer{proxyAddr: p.Addr, username: p.Username, password: p.Password, version: "socks4", forward: forward}
	case "socks5", "socks5h":
		return &socksDialer{proxyAddr: p.Addr, username: p.Username, password: p.Password, version: "socks5", forward: forward}
	case "http", "https":
		return &httpProxyDialer{proxyAddr: p.Addr, username: p.Username, password: p.Password, forward: forward}
	default:
		return &socksDialer{proxyAddr: p.Addr, username: p.Username, password: p.Password, version: "socks5", forward: forward}
	}
}
