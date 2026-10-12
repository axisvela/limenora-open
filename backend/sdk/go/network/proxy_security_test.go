package network

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// errTokenSource always fails; emptyTokenSource succeeds with no token.
type errTokenSource struct{ err error }

func (e errTokenSource) Token(context.Context) (string, error) { return "", e.err }

type emptyTokenSource struct{}

func (emptyTokenSource) Token(context.Context) (string, error) { return "", nil }

func TestProxyEgress_StripsCallerProxyAuthorizationHTTP(t *testing.T) {
	var mu sync.Mutex
	var gotProxyAuth string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotProxyAuth = r.Header.Get("Proxy-Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok-real"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Do(context.Background(), GatewayRequest{
		URL:     "http://example.invalid/x",
		Method:  http.MethodGet,
		Headers: map[string]string{"Proxy-Authorization": "Bearer CALLER-LEAK"},
	}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotProxyAuth != "Bearer tok-real" {
		t.Errorf("proxy Proxy-Authorization = %q, want the transport credential (caller value must be dropped)", gotProxyAuth)
	}
}

func TestProxyEgress_StripsCallerProxyAuthorizationConnect(t *testing.T) {
	var originProxyAuth, originAuth atomic.Value
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originProxyAuth.Store(r.Header.Get("Proxy-Authorization"))
		originAuth.Store(r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()

	cap := &connectCapture{}
	proxy := newTunnelProxy(t, cap)

	pool := x509.NewCertPool()
	pool.AddCert(origin.Certificate())
	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok-real"), WithTLSConfig(&tls.Config{RootCAs: pool}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Do(context.Background(), GatewayRequest{
		URL:    origin.URL + "/x",
		Method: http.MethodGet,
		Headers: map[string]string{
			"Proxy-Authorization": "Bearer CALLER-LEAK",
			"Authorization":       "Bearer dest",
		},
	}); err != nil {
		t.Fatal(err)
	}

	if _, connAuth, seen := cap.get(); !seen {
		t.Fatal("proxy did not receive a CONNECT")
	} else if connAuth != "Bearer tok-real" {
		t.Errorf("CONNECT Proxy-Authorization = %q, want the transport credential", connAuth)
	}
	if v := originProxyAuth.Load(); v == nil || v.(string) != "" {
		t.Errorf("origin Proxy-Authorization = %v, want empty (caller value must not reach the origin)", v)
	}
	if v := originAuth.Load(); v == nil || v.(string) != "Bearer dest" {
		t.Errorf("origin Authorization = %v, want preserved Bearer dest", v)
	}
}

func TestProxyEgress_CrossOriginRedirectDropsCredentials(t *testing.T) {
	var mu sync.Mutex
	var gotAuth, gotProxyAuth string
	originB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		gotProxyAuth = r.Header.Get("Proxy-Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer originB.Close()

	originA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, originB.URL+"/next", http.StatusFound)
	}))
	defer originA.Close()

	proxy := newForwardProxy(t)
	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Do(context.Background(), GatewayRequest{
		URL:     originA.URL + "/start",
		Method:  http.MethodGet,
		Headers: map[string]string{"Authorization": "Bearer dest"},
	}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "" {
		t.Errorf("cross-origin redirect leaked Authorization %q to the new origin", gotAuth)
	}
	if gotProxyAuth != "" {
		t.Errorf("cross-origin redirect leaked Proxy-Authorization %q to the new origin", gotProxyAuth)
	}
}

func TestProxyEgress_SameOriginRedirectKeepsDestinationAuthorization(t *testing.T) {
	var mu sync.Mutex
	var gotAuth string
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/next", http.StatusFound)
			return
		}
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()

	proxy := newForwardProxy(t)
	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Do(context.Background(), GatewayRequest{
		URL:     origin.URL + "/start",
		Method:  http.MethodGet,
		Headers: map[string]string{"Authorization": "Bearer dest"},
	}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "Bearer dest" {
		t.Errorf("same-origin redirect Authorization = %q, want preserved Bearer dest", gotAuth)
	}
}

func TestProxyEgress_TokenChangePresentedOnNextRequest(t *testing.T) {
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()

	cap := &connectCapture{}
	proxy := newTunnelProxy(t, cap)

	pool := x509.NewCertPool()
	pool.AddCert(origin.Certificate())
	gw, err := NewProxyEgress(proxy.URL, &rotatingTokenSource{tokens: []string{"tok-A", "tok-B"}}, WithTLSConfig(&tls.Config{RootCAs: pool}))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := gw.Do(context.Background(), GatewayRequest{URL: origin.URL + "/x", Method: http.MethodGet}); err != nil {
			t.Fatal(err)
		}
	}

	h := cap.history()
	if len(h) != 2 {
		t.Fatalf("CONNECT count = %d, want 2 (a rotated token must be re-presented, not a reused tunnel)", len(h))
	}
	if h[0].auth != "Bearer tok-A" || h[1].auth != "Bearer tok-B" {
		t.Errorf("CONNECT credentials = %q, %q; want Bearer tok-A then Bearer tok-B", h[0].auth, h[1].auth)
	}
}

func TestProxyEgress_BoundsResponseBody(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, strings.Repeat("x", 1024))
	}))
	defer proxy.Close()

	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok"), WithMaxResponseBytes(64))
	if err != nil {
		t.Fatal(err)
	}
	_, err = gw.Do(context.Background(), GatewayRequest{URL: "http://example.invalid/big", Method: http.MethodGet})
	if err == nil {
		t.Fatal("expected error when the response exceeds the configured bound")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("err = %v, want an 'exceeds' bound error", err)
	}
}

func TestProxyEgress_PropagatesTraceID(t *testing.T) {
	var got atomic.Value
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.Store(r.Header.Get("X-Trace-ID"))
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gw.Do(context.Background(), GatewayRequest{
		URL:    "http://example.invalid/x",
		Method: http.MethodGet,
		Ctx:    AuthzContext{TraceID: "trace-1"},
	}); err != nil {
		t.Fatal(err)
	}
	if v := got.Load(); v == nil || v.(string) != "trace-1" {
		t.Errorf("X-Trace-ID = %v, want trace-1", v)
	}
}

func TestProxyEgress_DeniesCloudMetadata(t *testing.T) {
	var hits int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer proxy.Close()

	gw, err := NewProxyEgress(proxy.URL, NewStaticTokenSource("tok"))
	if err != nil {
		t.Fatal(err)
	}
	targets := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
		"http://169.254.170.2/v2/metadata",
		"http://168.63.129.16/",
	}
	for _, u := range targets {
		_, err := gw.Do(context.Background(), GatewayRequest{URL: u, Method: http.MethodGet})
		if err == nil {
			t.Errorf("Do(%q): expected a metadata deny error", u)
			continue
		}
		if !strings.Contains(err.Error(), "denied") {
			t.Errorf("Do(%q): err = %v, want a 'denied' error", u, err)
		}
	}
	if n := atomic.LoadInt64(&hits); n != 0 {
		t.Errorf("proxy hits = %d, want 0 (metadata must be denied before egress)", n)
	}
}
