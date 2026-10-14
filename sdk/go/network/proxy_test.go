package network

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// connectRecord is a single observed CONNECT.
type connectRecord struct {
	host string
	auth string
}

// connectCapture records every CONNECT authority + Proxy-Authorization a test
// proxy observed, in a race-safe way (the tunnel goroutines outlive Do).
type connectCapture struct {
	mu  sync.Mutex
	all []connectRecord
}

func (c *connectCapture) set(host, auth string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.all = append(c.all, connectRecord{host: host, auth: auth})
}

func (c *connectCapture) get() (host, auth string, seen bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.all) == 0 {
		return "", "", false
	}
	last := c.all[len(c.all)-1]
	return last.host, last.auth, true
}

func (c *connectCapture) history() []connectRecord {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]connectRecord, len(c.all))
	copy(out, c.all)
	return out
}

// newTunnelProxy is a minimal forward proxy that tunnels CONNECT requests and
// records the CONNECT authority + Proxy-Authorization header.
func newTunnelProxy(t *testing.T, cap *connectCapture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "not a CONNECT", http.StatusMethodNotAllowed)
			return
		}
		target := r.Host
		if target == "" {
			target = r.URL.Host
		}
		cap.set(target, r.Header.Get("Proxy-Authorization"))

		dest, err := net.Dial("tcp", target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer dest.Close()

		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack unsupported", http.StatusInternalServerError)
			return
		}
		clientConn, _, err := hj.Hijack()
		if err != nil {
			return
		}
		defer clientConn.Close()
		if _, err := clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
			return
		}
		done := make(chan struct{}, 2)
		go func() { io.Copy(dest, clientConn); done <- struct{}{} }()
		go func() { io.Copy(clientConn, dest); done <- struct{}{} }()
		<-done
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestProxyEgress_HTTPAbsoluteFormViaProxy(t *testing.T) {
	var mu sync.Mutex
	var got struct{ url, host, method, proxyAuth, destAuth string }

	originHits := int64(0)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&originHits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		got.url = r.URL.String()
		got.host = r.Host
		got.method = r.Method
		got.proxyAuth = r.Header.Get("Proxy-Authorization")
		got.destAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.Header().Set("X-Proxy", "yes")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "via-proxy")
	}))
	defer proxy.Close()

	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok-http"))
	if err != nil {
		t.Fatal(err)
	}

	want := origin.URL + "/path?q=1"
	res, err := gw.Do(context.Background(), GatewayRequest{
		URL:     want,
		Method:  http.MethodPost,
		Headers: map[string]string{"Authorization": "Bearer dest-secret", "X-Custom": "v"},
		Body:    []byte(`{"x":1}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", res.Status)
	}
	if string(res.Body) != "via-proxy" {
		t.Errorf("body = %q, want via-proxy", res.Body)
	}
	if res.Headers["X-Proxy"] != "yes" {
		t.Errorf("result header X-Proxy = %q, want yes", res.Headers["X-Proxy"])
	}

	mu.Lock()
	defer mu.Unlock()
	if got.url != want {
		t.Errorf("proxy saw URL %q, want absolute-form %q", got.url, want)
	}
	if !strings.HasPrefix(got.url, "http://") || got.host == "" {
		t.Errorf("request was not sent in absolute-form (url=%q host=%q)", got.url, got.host)
	}
	if got.method != http.MethodPost {
		t.Errorf("proxy saw method %q, want POST", got.method)
	}
	if got.proxyAuth != "Bearer tok-http" {
		t.Errorf("Proxy-Authorization = %q, want Bearer tok-http", got.proxyAuth)
	}
	if got.destAuth != "Bearer dest-secret" {
		t.Errorf("destination Authorization = %q, want preserved Bearer dest-secret", got.destAuth)
	}
	if n := atomic.LoadInt64(&originHits); n != 0 {
		t.Errorf("origin contacted directly %d times, want 0 (must go via proxy)", n)
	}
}

func TestProxyEgress_HTTPSUsesConnect(t *testing.T) {
	var originAuth atomic.Value
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originAuth.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "secure")
	}))
	defer origin.Close()

	cap := &connectCapture{}
	proxy := newTunnelProxy(t, cap)

	pool := x509.NewCertPool()
	pool.AddCert(origin.Certificate())
	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok-ssl"), WithTLSConfig(&tls.Config{RootCAs: pool}))
	if err != nil {
		t.Fatal(err)
	}

	res, err := gw.Do(context.Background(), GatewayRequest{
		URL:     origin.URL + "/secure",
		Method:  http.MethodGet,
		Headers: map[string]string{"Authorization": "Bearer dest-ssl"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusOK {
		t.Errorf("status = %d, want 200", res.Status)
	}
	if string(res.Body) != "secure" {
		t.Errorf("body = %q, want secure", res.Body)
	}

	host, auth, seen := cap.get()
	if !seen {
		t.Fatal("proxy did not receive a CONNECT")
	}
	wantHost := strings.TrimPrefix(origin.URL, "https://")
	if host != wantHost {
		t.Errorf("CONNECT authority = %q, want %q", host, wantHost)
	}
	if auth != "Bearer tok-ssl" {
		t.Errorf("CONNECT Proxy-Authorization = %q, want Bearer tok-ssl", auth)
	}
	if v := originAuth.Load(); v == nil || v.(string) != "Bearer dest-ssl" {
		t.Errorf("origin Authorization = %v, want preserved Bearer dest-ssl", v)
	}
}

func TestProxyEgress_DoesNotConcatenateProxyBase(t *testing.T) {
	var gotURL atomic.Value
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL.Store(r.URL.String())
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()

	gw, err := NewProxyEgress(proxy.URL+"/gateway/base", NewStaticTokenSource("tok"))
	if err != nil {
		t.Fatal(err)
	}
	want := origin.URL + "/real/path"
	if _, err := gw.Do(context.Background(), GatewayRequest{URL: want, Method: http.MethodGet}); err != nil {
		t.Fatal(err)
	}
	if got := gotURL.Load().(string); got != want {
		t.Errorf("target URL = %q, want %q (proxy base must not be concatenated)", got, want)
	}
}

func TestProxyEgress_TokenErrorFailsClosed(t *testing.T) {
	var hits int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	sentinel := errors.New("token unavailable")
	gw, err := NewProxyEgress(proxy.URL, errTokenSource{err: sentinel})
	if err != nil {
		t.Fatal(err)
	}
	_, err = gw.Do(context.Background(), GatewayRequest{URL: "http://example.invalid/x", Method: http.MethodGet})
	if err == nil {
		t.Fatal("expected error when token source fails")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want to wrap sentinel", err)
	}
	if n := atomic.LoadInt64(&hits); n != 0 {
		t.Errorf("proxy hits = %d, want 0 (no egress on token failure)", n)
	}
}

func TestProxyEgress_EmptyTokenFailsClosed(t *testing.T) {
	var hits int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	gw, err := NewProxyEgress(proxy.URL, emptyTokenSource{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = gw.Do(context.Background(), GatewayRequest{URL: "http://example.invalid/x", Method: http.MethodGet})
	if err == nil {
		t.Fatal("expected error when token source returns empty token")
	}
	if !errors.Is(err, ErrNoAgentToken) {
		t.Errorf("err = %v, want ErrNoAgentToken", err)
	}
	if n := atomic.LoadInt64(&hits); n != 0 {
		t.Errorf("proxy hits = %d, want 0 (no unauthenticated egress)", n)
	}
}

// rotatingTokenSource returns each configured token once, in order.
type rotatingTokenSource struct {
	mu     sync.Mutex
	tokens []string
	i      int
}

func (r *rotatingTokenSource) Token(context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.tokens) == 0 {
		return "", ErrNoAgentToken
	}
	tok := r.tokens[min(r.i, len(r.tokens)-1)]
	r.i++
	return tok, nil
}

// newForwardProxy is a minimal forward HTTP proxy that relays absolute-form
// requests to their destination, stripping hop-by-hop proxy credentials as a
// compliant proxy would.
func newForwardProxy(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for k, vv := range r.Header {
			if http.CanonicalHeaderKey(k) == "Proxy-Authorization" {
				continue
			}
			for _, v := range vv {
				outReq.Header.Add(k, v)
			}
		}
		resp, err := http.DefaultTransport.RoundTrip(outReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	t.Cleanup(srv.Close)
	return srv
}
