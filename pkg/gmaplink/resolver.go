package gmaplink

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

const (
	defaultTimeout       = 5 * time.Second
	defaultMaxRedirects  = 3
	defaultUserAgent     = "AccessPath/1.0 (+https://accesspath.example)"
	maxResponseBodyBytes = 64 * 1024
)

// Resolver expande enlaces cortos (maps.app.goo.gl) siguiendo como maximo
// maxRedirects saltos y solo hacia hosts de Google. El resto de enlaces se
// parsean sin red.
type Resolver struct {
	client       *http.Client
	maxRedirects int
}

// NewResolver construye el resolver. timeout y maxRedirects <= 0 usan los
// valores por defecto (5s, 3 saltos).
func NewResolver(timeout time.Duration, maxRedirects int) *Resolver {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if maxRedirects <= 0 {
		maxRedirects = defaultMaxRedirects
	}

	r := &Resolver{maxRedirects: maxRedirects}

	dialer := &net.Dialer{
		Timeout: timeout,
		// Defensa en profundidad contra SSRF: aunque el allowlist de hosts ya
		// limita el destino a Google, se rechaza cualquier conexion a una IP
		// no publica (rebinding de DNS incluido).
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil || !isPublicIP(ip) {
				return ErrBlockedAddress
			}
			return nil
		},
	}

	r.client = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   timeout,
			ResponseHeaderTimeout: timeout,
			DisableKeepAlives:     true,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > r.maxRedirects {
				return ErrTooManyRedirect
			}
			if !AllowedHost(req.URL.Hostname()) {
				return ErrHostNotAllowed
			}
			if req.URL.Scheme != "https" {
				return ErrUnsupportedURL
			}
			return nil
		},
	}
	return r
}

// Resolve devuelve los datos del enlace. Si es un acortador, sigue la
// redireccion y parsea la URL final.
func (r *Resolver) Resolve(ctx context.Context, raw string) (*Link, error) {
	link, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	if !link.IsShort() {
		return link, nil
	}

	final, err := r.expand(ctx, link.URL)
	if err != nil {
		return nil, err
	}
	return Parse(final)
}

func (r *Resolver) expand(ctx context.Context, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrInvalidURL
	}
	u.Scheme = "https"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("gmaplink: build request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gmaplink: expand: %w", err)
	}
	defer resp.Body.Close()

	// No nos interesa el HTML: solo la URL final. Aun asi drenamos un trozo
	// acotado para que la conexion se cierre limpia.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBodyBytes))

	if resp.Request == nil || resp.Request.URL == nil {
		return "", ErrInvalidURL
	}
	return resp.Request.URL.String(), nil
}

// isPublicIP descarta direcciones loopback, privadas, link-local, multicast,
// CGNAT y sin especificar.
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 0 {
			return false
		}
		// 100.64.0.0/10 (carrier-grade NAT)
		if ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127 {
			return false
		}
	}
	return true
}
