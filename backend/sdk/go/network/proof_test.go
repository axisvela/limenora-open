package network

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// recordingProxy is a minimal forward proxy that records the absolute-form
// request (headers) it receives and answers 200 without contacting an origin.
type recordingProxy struct {
	mu      sync.Mutex
	headers http.Header
	url     string
	srv     *httptest.Server
}

func newRecordingProxy(t *testing.T) *recordingProxy {
	t.Helper()
	p := &recordingProxy{}
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		p.headers = r.Header.Clone()
		p.url = r.URL.String()
		p.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *recordingProxy) get(name string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.headers.Get(name)
}

// fakeCarrier records its calls and returns configured headers.
type fakeCarrier struct {
	mu     sync.Mutex
	method string
	target string
	token  string
	calls  int
	out    map[string]string
	err    error
}

func (c *fakeCarrier) ProofHeaders(method, target, token string) (map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	c.method, c.target, c.token = method, target, token
	if c.err != nil {
		return nil, c.err
	}
	return c.out, nil
}

func TestProxyEgressRequestHeaderCarriage(t *testing.T) {
	proxy := newRecordingProxy(t)
	carrier := &fakeCarrier{out: map[string]string{
		"DPoP":                "DPoP resource-proof",
		"DPoP-Introspection":  "DPoP introspection-proof",
	}}
	egress, err := NewProxyEgress(proxy.srv.URL, NewStaticTokenSource("gw-token"),
		WithProofCarrier(carrier), WithCarriage(CarriageRequestHeader))
	if err != nil {
		t.Fatalf("NewProxyEgress: %v", err)
	}

	res, err := egress.Do(context.Background(), GatewayRequest{
		URL: "http://origin.test/path?q=1", Method: "POST",
		Headers: map[string]string{"Authorization": "Bearer dest"},
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if res.Status != http.StatusOK {
		t.Fatalf("status = %d", res.Status)
	}
	if got := proxy.get("DPoP"); got != "DPoP resource-proof" {
		t.Fatalf("resource proof header = %q", got)
	}
	if got := proxy.get("DPoP-Introspection"); got != "DPoP introspection-proof" {
		t.Fatalf("introspection proof header = %q", got)
	}
	if got := proxy.get("Authorization"); got != "Bearer dest" {
		t.Fatalf("destination Authorization = %q, must be preserved", got)
	}
	if got := proxy.get("Proxy-Authorization"); got != "Bearer gw-token" {
		t.Fatalf("gateway credential = %q", got)
	}
	if carrier.method != "POST" || carrier.target != "http://origin.test/path?q=1" || carrier.token != "gw-token" {
		t.Fatalf("carrier called with (%q,%q,%q)", carrier.method, carrier.target, carrier.token)
	}
}

func TestProxyEgressDisabledCarriageSkipsProofs(t *testing.T) {
	proxy := newRecordingProxy(t)
	carrier := &fakeCarrier{out: map[string]string{"DPoP": "should-not-appear"}}
	egress, err := NewProxyEgress(proxy.srv.URL, NewStaticTokenSource("gw-token"),
		WithProofCarrier(carrier), WithCarriage(CarriageDisabled))
	if err != nil {
		t.Fatalf("NewProxyEgress: %v", err)
	}
	if _, err := egress.Do(context.Background(), GatewayRequest{URL: "http://origin.test/x", Method: "GET"}); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if carrier.calls != 0 {
		t.Fatalf("carrier must not be consulted when carriage is disabled: %d calls", carrier.calls)
	}
	if got := proxy.get("DPoP"); got != "" {
		t.Fatalf("no proof header expected, got %q", got)
	}
}

func TestProxyEgressProofFailureFailsClosed(t *testing.T) {
	proxy := newRecordingProxy(t)
	carrier := &fakeCarrier{err: errors.New("signing unavailable")}
	egress, err := NewProxyEgress(proxy.srv.URL, NewStaticTokenSource("gw-token"),
		WithProofCarrier(carrier), WithCarriage(CarriageRequestHeader))
	if err != nil {
		t.Fatalf("NewProxyEgress: %v", err)
	}
	if _, err := egress.Do(context.Background(), GatewayRequest{URL: "http://origin.test/x", Method: "GET"}); err == nil {
		t.Fatal("a proof-carrier error must fail the request closed")
	}
	if got := proxy.get("Proxy-Authorization"); got != "" {
		t.Fatalf("no request must reach the proxy on a proof failure, got %q", got)
	}
}
