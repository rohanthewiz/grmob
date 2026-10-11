package blb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// EndpointPath is the ScriptTagger endpoint's path, the one Client.Fetch
// requests under Client.BaseURL. It is also the one path a same-origin proxy
// for a browser build must answer on (see Proxy).
const EndpointPath = "/remoteExtensions/toolTip/toolTipRemote.cfm"

// Proxy forwards Client.Fetch's request to Blue Letter Bible on behalf of a
// browser build, which cannot make it itself: BLB's reply carries no
// Access-Control-Allow-Origin header, so the browser withholds it from
// WebAssembly (see "Where it works"). The app's own server mounts a Proxy at
// EndpointPath, and the app points its Client at that server:
//
//	// server
//	http.Handle(blb.EndpointPath, &blb.Proxy{})
//
//	// wasm app (syscall/js)
//	blb.DefaultClient.BaseURL = js.Global().Get("location").Get("origin").String()
//
// The page and the proxy then share an origin, so no CORS header is needed
// at all. `go run ./serve` (and an app's ./dev.sh, which runs the same
// server) mounts one, so a browser build fetches verses in development with
// no setup.
//
// # Not an open proxy
//
// A handler that forwards whatever it is asked to is a liability on any
// public server, so this one is narrow by construction:
//
//   - The upstream is fixed (DefaultBaseURL, or Upstream for a test) and so
//     is the path: the request's own path is ignored, so mounting it at the
//     wrong route can only ever reach the one endpoint.
//   - Only GET and HEAD are answered, and only the three query parameters
//     Fetch sends (id, style, target) are forwarded, each bounded in length.
//     A request without an id is refused before any upstream call.
//   - None of the caller's headers travel: no cookies, no Authorization, no
//     client address. The upstream sees the same User-Agent Fetch sends.
//   - Only the status, the Content-Type and a body of at most maxReply bytes
//     come back. A longer reply is refused rather than cut, because a cut
//     passage parses as a short one.
//
// rweb's own Server.Proxy was not used for this reason: it forwards every
// header, method and parameter, which is right for a backend you own and
// wrong for a third party's endpoint.
//
// The zero value is ready to use.
type Proxy struct {
	// HTTPClient makes the upstream requests. Nil uses the client
	// Client.Fetch uses, with its 15-second timeout.
	HTTPClient *http.Client

	// Upstream replaces DefaultBaseURL, for a test server. No trailing
	// slash is needed.
	Upstream string
}

// ProxyReply is what Proxy.Forward hands back for the caller to write out:
// the status, the Content-Type ("" to leave unset) and the body.
type ProxyReply struct {
	Status      int
	ContentType string
	Body        []byte
}

// proxiedParams are the query parameters Forward passes upstream: exactly
// the ones Client.Fetch sends. Anything else a caller adds is dropped, not
// refused, so a cache-busting parameter from a page does no harm.
var proxiedParams = []string{"id", "style", "target"}

// maxParam bounds each forwarded parameter's length. The longest id Fetch
// builds is a translation, a book name with its spaces removed and a verse
// list ("NASB20.SongofSongs.2.1-3,5,7-9"); 256 is several times that, and
// short enough that the proxy cannot be used to send BLB arbitrary payloads.
const maxParam = 256

// ServeHTTP answers a browser's Client.Fetch through Forward.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "blb proxy: GET only", http.StatusMethodNotAllowed)
		return
	}
	reply, _ := p.Forward(r.Context(), r.URL.RawQuery)
	if reply.ContentType != "" {
		w.Header().Set("Content-Type", reply.ContentType)
	}
	w.WriteHeader(reply.Status)
	if r.Method != http.MethodHead {
		_, _ = w.Write(reply.Body)
	}
}

// Forward makes the upstream request for one proxied call, given the
// caller's raw query string. It is ServeHTTP without net/http, for a server
// built on another router (serve mounts it on rweb this way).
//
// The reply is always one to send. A refused query is a 400 and a failed or
// oversized upstream reply is a 502, each with a one-line text body; the
// error is returned alongside so the server can log it. An upstream status
// other than 200 is passed through as it came, since it is BLB's answer and
// Client.Fetch already reads it as ErrUnexpectedResponse.
func (p *Proxy) Forward(ctx context.Context, rawQuery string) (ProxyReply, error) {
	in, err := url.ParseQuery(rawQuery)
	if err != nil {
		return refused(http.StatusBadRequest, fmt.Errorf("blb proxy: query: %w", err))
	}
	out := url.Values{}
	for _, k := range proxiedParams {
		v := in.Get(k)
		if v == "" {
			continue
		}
		if len(v) > maxParam {
			return refused(http.StatusBadRequest, fmt.Errorf("blb proxy: %s is %d bytes, over %d", k, len(v), maxParam))
		}
		out.Set(k, v)
	}
	if out.Get("id") == "" {
		return refused(http.StatusBadRequest, errors.New("blb proxy: no passage id"))
	}

	base := p.Upstream
	if base == "" {
		base = DefaultBaseURL
	}
	endpoint := strings.TrimRight(base, "/") + EndpointPath + "?" + out.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return refused(http.StatusBadGateway, fmt.Errorf("blb proxy: building request: %w", err))
	}
	req.Header.Set("User-Agent", userAgent)

	hc := p.HTTPClient
	if hc == nil {
		hc = defaultHTTPClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return refused(http.StatusBadGateway, fmt.Errorf("blb proxy: upstream: %w", err))
	}
	defer resp.Body.Close()

	// One byte past the cap is read so that a reply of exactly maxReply is
	// told apart from a longer one.
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReply+1))
	if err != nil {
		return refused(http.StatusBadGateway, fmt.Errorf("blb proxy: reading upstream: %w", err))
	}
	if len(body) > maxReply {
		return refused(http.StatusBadGateway, fmt.Errorf("blb proxy: upstream reply over %d bytes", maxReply))
	}
	return ProxyReply{Status: resp.StatusCode, ContentType: resp.Header.Get("Content-Type"), Body: body}, nil
}

// refused is the reply for a request the proxy will not, or could not,
// complete: the status, a plain-text line naming why, and the same error for
// the caller's log.
func refused(status int, err error) (ProxyReply, error) {
	return ProxyReply{
		Status:      status,
		ContentType: "text/plain; charset=utf-8",
		Body:        []byte(err.Error() + "\n"),
	}, err
}
