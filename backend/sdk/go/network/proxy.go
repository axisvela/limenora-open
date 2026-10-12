package network

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultMaxResponseBytes bounds how much of a proxied response body is read.
const defaultMaxResponseBytes = 10 << 20 // 10 MiB

// ProxyEgress routes outbound requests through a LIMENORA forward proxy using a
// proxy-aware HTTP transport:
//
//   - http:// targets are sent to the proxy in absolute-form and never
//     concatenated with the proxy base URL;
//   - https:// targets use a CONNECT tunnel to the proxy, then TLS to the origin;
//   - the gateway credential is presented to the proxy via Proxy-Authorization
//     (Bearer <token>), so it is never conflated with the destination's
//     Authorization header, which is preserved and forwarded to the origin.
//
// A missing credential fails closed (ErrNoAgentToken); a proxy/gateway failure
// is returned as-is with no local/application fallback. Cloud metadata
// destinations are denied before any egress (denyMetadataDestination).
//
// Credential hygiene: a caller-supplied Proxy-Authorization is stripped from the
// destination-bound request; a redirect that crosses an origin (scheme+authority)
// boundary drops both Authorization and Proxy-Authorization, while a same-origin
// redirect preserves the destination Authorization. Because Go pools CONNECT
// tunnels without a credential dimension, idle connections are not retained, so
// every request re-presents the current gateway credential to the proxy.
type ProxyEgress struct {
	client           *http.Client
	tokens           AgentTokenSource
	maxResponseBytes int64
}

// ProxyOption configures a ProxyEgress.
type ProxyOption func(*proxyOptions)

type proxyOptions struct {
	tlsConfig        *tls.Config
	maxResponseBytes int64
}

// WithTLSConfig sets the TLS configuration used for HTTPS destinations reached
// through the proxy tunnel (for example, a custom RootCAs pool).
func WithTLSConfig(cfg *tls.Config) ProxyOption {
	return func(o *proxyOptions) { o.tlsConfig = cfg }
}

// WithMaxResponseBytes bounds how much of a proxied response body is read. A
// response larger than n fails closed rather than being silently truncated.
func WithMaxResponseBytes(n int64) ProxyOption {
	return func(o *proxyOptions) { o.maxResponseBytes = n }
}

// sameOriginURL reports whether a and b share scheme and authority (host+port).
func sameOriginURL(a, b *url.URL) bool {
	return a != nil && b != nil &&
		strings.EqualFold(a.Scheme, b.Scheme) &&
		strings.EqualFold(a.Host, b.Host)
}

// proxyTokenCtxKey carries the per-request gateway token to the transport so the
// token is fetched exactly once, at the Do boundary.
type proxyTokenCtxKey struct{}

func proxyTokenFromContext(ctx context.Context) (string, bool) {
	tok, ok := ctx.Value(proxyTokenCtxKey{}).(string)
	return tok, ok
}

// NewProxyEgress builds a proxy-aware NetworkGateway for proxyBase. The gateway
// credential is resolved per request from tokens.
func NewProxyEgress(proxyBase string, tokens AgentTokenSource, opts ...ProxyOption) (*ProxyEgress, error) {
	if tokens == nil {
		return nil, fmt.Errorf("network: proxy egress: nil token source")
	}
	u, err := url.Parse(proxyBase)
	if err != nil {
		return nil, fmt.Errorf("network: proxy egress: parse proxy URL: %w", err)
	}
	if u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("network: proxy egress: proxy URL must be absolute http(s): %q", proxyBase)
	}

	cfg := proxyOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.maxResponseBytes <= 0 {
		cfg.maxResponseBytes = defaultMaxResponseBytes
	}

	transport := &http.Transport{
		Proxy:           http.ProxyURL(u),
		TLSClientConfig: cfg.tlsConfig,
		// Go pools proxy CONNECT tunnels by connectMethodKey (proxy/scheme/target),
		// with no credential dimension. Without this, a tunnel authenticated with
		// token A could be reused for a request bearing a rotated token B, so the
		// proxy would never re-validate the current credential. Refusing to retain
		// idle connections forces a fresh, re-authenticated CONNECT per request.
		MaxIdleConnsPerHost: -1,
	}
	// CONNECT path (HTTPS destinations): the credential is presented only on the
	// proxy-bound CONNECT request, never on the tunneled origin request.
	transport.GetProxyConnectHeader = func(ctx context.Context, _ *url.URL, _ string) (http.Header, error) {
		tok, ok := proxyTokenFromContext(ctx)
		if !ok || tok == "" {
			return nil, ErrNoAgentToken
		}
		return http.Header{"Proxy-Authorization": []string{"Bearer " + tok}}, nil
	}

	client := &http.Client{
		Transport: &proxyAuthTransport{base: transport},
		Timeout:   30 * time.Second,
		// Enforce redirect hygiene explicitly rather than relying on Go defaults:
		// credentials never cross an origin (scheme+authority) boundary, while the
		// destination Authorization is preserved for same-origin redirects.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("network: proxy egress: stopped after 10 redirects")
			}
			if len(via) > 0 && !sameOriginURL(via[0].URL, req.URL) {
				req.Header.Del("Authorization")
				req.Header.Del("Proxy-Authorization")
			}
			return nil
		},
	}

	return &ProxyEgress{
		tokens:           tokens,
		maxResponseBytes: cfg.maxResponseBytes,
		client:           client,
	}, nil
}

// proxyAuthTransport adds Proxy-Authorization to proxy-bound HTTP (absolute-form)
// requests without touching the destination Authorization header. HTTPS
// destinations take the CONNECT path, where the credential is supplied via
// Transport.GetProxyConnectHeader instead.
type proxyAuthTransport struct {
	base http.RoundTripper
}

func (t *proxyAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	if out.URL.Scheme == "http" {
		tok, ok := proxyTokenFromContext(out.Context())
		if !ok || tok == "" {
			return nil, ErrNoAgentToken
		}
		out.Header.Set("Proxy-Authorization", "Bearer "+tok)
	}
	return t.base.RoundTrip(out)
}

// Do proxies req through the configured proxy. The target URL is used as-is.
func (p *ProxyEgress) Do(ctx context.Context, req GatewayRequest) (*GatewayResult, error) {
	// Deny cloud metadata endpoints before any egress; the denylist must travel
	// with the production egress, not only the local/dev gateway.
	if err := denyMetadataDestination(req); err != nil {
		return nil, err
	}
	tok, err := p.tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("network: proxy egress: %w", err)
	}
	if tok == "" {
		return nil, fmt.Errorf("network: proxy egress: %w", ErrNoAgentToken)
	}
	ctx = context.WithValue(ctx, proxyTokenCtxKey{}, tok)

	var body io.Reader
	if req.Body != nil {
		body = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, body)
	if err != nil {
		return nil, fmt.Errorf("network: proxy egress: build request: %w", err)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}
	// Proxy-Authorization is a proxy credential, never a destination header. Drop
	// any caller-supplied value on both paths; the transport sets its own
	// credential for the proxy (HTTP) or the CONNECT request (HTTPS) only.
	httpReq.Header.Del("Proxy-Authorization")
	if req.Ctx.TraceID != "" {
		httpReq.Header.Set("X-Trace-ID", req.Ctx.TraceID)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network: proxy egress: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, p.maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("network: proxy egress: read response: %w", err)
	}
	if int64(len(raw)) > p.maxResponseBytes {
		return nil, fmt.Errorf("network: proxy egress: response exceeds %d bytes", p.maxResponseBytes)
	}
	headers := make(map[string]string, len(resp.Header))
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	return &GatewayResult{Status: resp.StatusCode, Body: raw, Headers: headers}, nil
}
