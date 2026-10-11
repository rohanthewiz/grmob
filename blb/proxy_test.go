package blb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeUpstream stands in for www.blueletterbible.org: it records what the
// proxy sent and answers with a fixture, or with whatever reply overrides.
type fakeUpstream struct {
	*httptest.Server
	got   *http.Request
	calls int
	reply func(w http.ResponseWriter, r *http.Request)
}

func newFakeUpstream(t *testing.T) *fakeUpstream {
	t.Helper()
	f := &fakeUpstream{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.got, f.calls = r, f.calls+1
		if f.reply != nil {
			f.reply(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=UTF-8")
		w.Write([]byte(fixture(t, "KJV.John.3.16-18")))
	}))
	t.Cleanup(f.Close)
	return f
}

// The round trip a browser build makes: Client.Fetch with BaseURL at the
// page's own server, whose Proxy forwards to BLB. The passage arrives whole,
// so the proxy is transparent to the parser.
func TestProxyRoundTripFromAClient(t *testing.T) {
	up := newFakeUpstream(t)
	mux := http.NewServeMux()
	mux.Handle(EndpointPath, &Proxy{Upstream: up.URL, HTTPClient: up.Client()})
	page := httptest.NewServer(mux)
	defer page.Close()

	c := &Client{BaseURL: page.URL, HTTPClient: page.Client()}
	p, err := c.Fetch(context.Background(), "John 3:16-18", "KJV")
	if err != nil {
		t.Fatal(err)
	}
	if p.Reference != "John 3:16-18" || len(p.Verses) != 3 {
		t.Errorf("through the proxy: %+v", p)
	}
	if up.got.URL.Path != EndpointPath || up.got.URL.Query().Get("id") != "KJV.John.3.16-18" {
		t.Errorf("upstream saw %s", up.got.URL)
	}
}

// Only the three parameters Fetch sends travel, and none of the caller's
// headers do: a cookie or an Authorization header for the app's own origin
// must never reach a third party.
func TestProxyForwardsOnlyFetchsParametersAndNoHeaders(t *testing.T) {
	up := newFakeUpstream(t)
	px := &Proxy{Upstream: up.URL, HTTPClient: up.Client()}
	req := httptest.NewRequest(http.MethodGet,
		EndpointPath+"?id=KJV.John.3.16&style=par&target=true&callback=evil&url=http://169.254.169.254/", nil)
	req.Header.Set("Cookie", "session=secret")
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Forwarded-For", "10.0.0.7")
	rec := httptest.NewRecorder()
	px.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	q := up.got.URL.Query()
	if len(q) != 3 || q.Get("id") != "KJV.John.3.16" || q.Get("style") != "par" || q.Get("target") != "true" {
		t.Errorf("upstream query = %v, want exactly id, style and target", q)
	}
	for _, h := range []string{"Cookie", "Authorization", "X-Forwarded-For"} {
		if v := up.got.Header.Get(h); v != "" {
			t.Errorf("upstream received %s: %q", h, v)
		}
	}
	if ua := up.got.Header.Get("User-Agent"); ua != userAgent {
		t.Errorf("upstream User-Agent = %q, want Fetch's %q", ua, userAgent)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/javascript; charset=UTF-8" {
		t.Errorf("Content-Type = %q, want the upstream's", ct)
	}
}

// The request's own path is ignored: a Proxy mounted at a catch-all route
// still reaches only the one endpoint.
func TestProxyIgnoresTheRequestPath(t *testing.T) {
	up := newFakeUpstream(t)
	px := &Proxy{Upstream: up.URL, HTTPClient: up.Client()}
	rec := httptest.NewRecorder()
	px.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/../../etc/passwd?id=KJV.John.3.16", nil))
	if up.got == nil || up.got.URL.Path != EndpointPath {
		t.Fatalf("upstream path = %v, want %s", up.got, EndpointPath)
	}
}

// Refusals happen before any upstream call: a write method, a request with
// no passage id, an oversized parameter and a malformed query.
func TestProxyRefusesWithoutCallingUpstream(t *testing.T) {
	up := newFakeUpstream(t)
	px := &Proxy{Upstream: up.URL, HTTPClient: up.Client()}
	for _, c := range []struct {
		name, method, target string
		want                 int
	}{
		{"post", http.MethodPost, EndpointPath + "?id=KJV.John.3.16", http.StatusMethodNotAllowed},
		{"no id", http.MethodGet, EndpointPath + "?style=par", http.StatusBadRequest},
		{"empty id", http.MethodGet, EndpointPath + "?id=", http.StatusBadRequest},
		{"long id", http.MethodGet, EndpointPath + "?id=" + strings.Repeat("a", maxParam+1), http.StatusBadRequest},
		{"bad escape", http.MethodGet, EndpointPath + "?id=%zz", http.StatusBadRequest},
	} {
		rec := httptest.NewRecorder()
		px.ServeHTTP(rec, httptest.NewRequest(c.method, c.target, nil))
		if rec.Code != c.want {
			t.Errorf("%s: status %d, want %d", c.name, rec.Code, c.want)
		}
	}
	if up.calls != 0 {
		t.Errorf("upstream was called %d times for refused requests", up.calls)
	}
	rec := httptest.NewRecorder()
	px.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, EndpointPath, nil))
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("405 Allow = %q", allow)
	}
}

// HEAD answers the GET's headers with no body.
func TestProxyHead(t *testing.T) {
	up := newFakeUpstream(t)
	px := &Proxy{Upstream: up.URL, HTTPClient: up.Client()}
	rec := httptest.NewRecorder()
	px.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, EndpointPath+"?id=KJV.John.3.16", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 || rec.Header().Get("Content-Type") == "" {
		t.Errorf("HEAD: status %d, %d body bytes, Content-Type %q", rec.Code, rec.Body.Len(), rec.Header().Get("Content-Type"))
	}
}

// Upstream trouble is a 502 with the reason, and an over-long reply is
// refused rather than cut: a cut passage would parse as a shorter one. A
// non-200 from BLB itself passes through, since it is BLB's answer.
func TestProxyUpstreamFailures(t *testing.T) {
	up := newFakeUpstream(t)
	px := &Proxy{Upstream: up.URL, HTTPClient: up.Client()}

	up.reply = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", maxReply+1)))
	}
	reply, err := px.Forward(context.Background(), "id=KJV.Psa.119")
	if reply.Status != http.StatusBadGateway || err == nil {
		t.Errorf("oversized reply: status %d, err %v", reply.Status, err)
	}

	up.reply = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Repeat("x", maxReply)))
	}
	if reply, err := px.Forward(context.Background(), "id=KJV.Psa.119"); reply.Status != http.StatusOK || err != nil || len(reply.Body) != maxReply {
		t.Errorf("a reply of exactly maxReply: status %d, %d bytes, err %v", reply.Status, len(reply.Body), err)
	}

	up.reply = func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "maintenance", http.StatusServiceUnavailable)
	}
	if reply, _ := px.Forward(context.Background(), "id=KJV.John.3.16"); reply.Status != http.StatusServiceUnavailable {
		t.Errorf("upstream 503 came back as %d", reply.Status)
	}

	up.Close()
	reply, err = px.Forward(context.Background(), "id=KJV.John.3.16")
	if reply.Status != http.StatusBadGateway || err == nil {
		t.Errorf("upstream down: status %d, err %v", reply.Status, err)
	}
	if !strings.Contains(string(reply.Body), "upstream") {
		t.Errorf("502 body %q does not say why", reply.Body)
	}
}
