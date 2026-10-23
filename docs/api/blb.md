# Package blb

```go
import "github.com/rohanthewiz/grmob/blb"
```

Package blb fetches Bible passages from Blue Letter Bible (blueletterbible.org), for comps.BibleVerse and anything else that wants a verse's text, its canonical reference and a link back to the passage.

	p, err := blb.Fetch(ctx, "John 3:16-18", "ESV")
	// p.Reference "John 3:16-18", p.Translation "ESV",
	// p.Verses [{16 "For God so loved…"} {17 …} {18 …}],
	// p.URL "https://www.blueletterbible.org/esv/jhn/3/16/"

	p.StudyURL(17, blb.ToolCrossRefs)
	// "https://www.blueletterbible.org/esv/jhn/3/17/t_corr_1000017"

StudyURL links one verse's study tools on BLB (interlinear and Strong's, cross-references, commentaries, dictionaries), the addresses comps.BibleStudy opens when a reader taps a verse to dig deeper.

## Where the text comes from

Blue Letter Bible has no open JSON API. What it does publish, for anyone to embed, is the BLB ScriptTagger (blueletterbible.org/webtools): a script a site includes so that every reference on its pages shows the verse in a hover bubble. The bubble's contents arrive from one endpoint,

	GET /remoteExtensions/toolTip/toolTipRemote.cfm?id=KJV.John.3.16&style=par

which answers with a few lines of JavaScript that hand an HTML string to the tagger. This package makes the same request the tagger makes and reads the same HTML out of the reply, so it serves exactly what BLB already serves to every page that embeds its tool, under the same terms: the text links back to Blue Letter Bible (Passage.URL), and comps.BibleVerse draws that link by default.

The endpoint is not a documented API, so its shape can change without notice. The parser is deliberately narrow (the title, the verse markers, the first passage link) and fails with ErrUnexpectedResponse rather than returning half a verse when the shape moves. The fixtures in blb\_test.go are verbatim replies, and the live test (GRMOB\_BLB\_LIVE=1) checks the endpoint itself.

## Where it works

On Android and iOS (Go's net/http under gomobile) and on a server. A browser build cannot call it directly: the reply carries no Access-Control-Allow-Origin header, so the browser refuses to hand it to WebAssembly. A browser app points Client.BaseURL at a same-origin proxy that forwards EndpointPath to www.blueletterbible.org. Proxy is that handler, for the app's own server; \`go run ./serve\` and a scaffolded app's ./dev.sh mount one, so verses load in a browser during development once BaseURL is the page's origin. A static host (GitHub Pages) has no server to put it on.

## Translations

The translation is BLB's code, upper-case: KJV, NKJV, NLT, NIV, ESV, CSB, NASB20, NASB95, LSB, AMP, NET, RSV, ASV, YLT and WEB all answer. All but the KJV, ASV, YLT and WEB are under copyright. An app that stores or republishes their text, rather than showing it with the link, should check the publisher's terms first.

## Index

- [Constants](#constants) — `DefaultBaseURL`, `DefaultTranslation`, `EndpointPath`, `ToolBibles`, `ToolCommentaries`, `ToolCrossRefs`, `ToolDictionaries`, `ToolInterlinear`, `ToolMisc`
- [Variables](#variables) — `DefaultClient`, `ErrInvalidReference`, `ErrUnexpectedResponse`
- [`func PassageID`](#func-passageid)
- [`func SearchURL`](#func-searchurl)
- [`type Client`](#type-client)
    - [`func (*Client) Fetch`](#func-client-fetch)
- [`type Passage`](#type-passage)
    - [`func Fetch`](#func-fetch)
    - [`func (Passage) StudyURL`](#func-passage-studyurl)
    - [`func (Passage) Text`](#func-passage-text)
- [`type Proxy`](#type-proxy)
    - [`func (*Proxy) Forward`](#func-proxy-forward)
    - [`func (*Proxy) ServeHTTP`](#func-proxy-servehttp)
- [`type ProxyReply`](#type-proxyreply)
- [`type Verse`](#type-verse)

## Constants

The study tools BLB shows under a verse, one tab each on the verse's study page. The values are BLB's own tab keys (the page's data-bible-tool attributes); StudyURL puts them in the address so the page opens on that tab. Untyped string constants, so a StudyURL method value fits a func(int, string) string field such as comps.BibleStudy.StudyURL.

```go
const (
	// ToolInterlinear is the verse word by word against the Hebrew or
	// Greek, each word's Strong's number linking to its lexicon entry. It is
	// the tab BLB opens on when none is named.
	ToolInterlinear = "conc"
	// ToolBibles is the verse in every translation BLB carries.
	ToolBibles = "bibles"
	// ToolCrossRefs is the Treasury of Scripture Knowledge's references.
	ToolCrossRefs = "corr"
	// ToolCommentaries lists the commentaries that discuss the verse.
	ToolCommentaries = "comms"
	// ToolDictionaries lists dictionary and encyclopedia entries that cite
	// the verse. BLB labels the tab "Dictionaries" and keys it "refs";
	// "dict", the guess the label invites, redirects away from the verse.
	ToolDictionaries = "refs"
	// ToolMisc is BLB's miscellany: images, maps, outlines.
	ToolMisc = "misc"
)
```

<small>[blb/blb.go:133](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L133)</small>

DefaultBaseURL is Blue Letter Bible's origin.

```go
const DefaultBaseURL = "https://www.blueletterbible.org"
```

<small>[blb/blb.go:76](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L76)</small>

DefaultTranslation is used when neither the call nor the Client names one. The King James is the one translation BLB serves that is in the public domain everywhere it is read (outside the UK's Crown patent), which makes it the safe default for an app that has not thought about licensing yet.

```go
const DefaultTranslation = "KJV"
```

<small>[blb/blb.go:86](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L86)</small>

EndpointPath is the ScriptTagger endpoint's path, the one Client.Fetch requests under Client.BaseURL. It is also the one path a same-origin proxy for a browser build must answer on (see Proxy).

```go
const EndpointPath = "/remoteExtensions/toolTip/toolTipRemote.cfm"
```

<small>[blb/proxy.go:16](https://github.com/rohanthewiz/grmob/blob/master/blb/proxy.go#L16)</small>

## Variables

DefaultClient is used by the package-level Fetch.

```go
var DefaultClient = &Client{}
```

<small>[blb/blb.go:230](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L230)</small>

ErrInvalidReference is returned when BLB does not recognise the reference ("Hezekiah 1:1", "John 99:1"). Test for it with errors.Is.

```go
var ErrInvalidReference = errors.New("blb: invalid scripture reference")
```

<small>[blb/blb.go:90](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L90)</small>

ErrUnexpectedResponse is returned when the reply parses as neither a passage nor BLB's error bubble: the endpoint changed shape, or a proxy in front of it answered with something else.

```go
var ErrUnexpectedResponse = errors.New("blb: unexpected response")
```

<small>[blb/blb.go:95](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L95)</small>

## Functions

### func PassageID

```go
func PassageID(ref, translation string) (string, error)
```

PassageID builds the tagger's id for ref: translation, book with its spaces removed, then chapter and verses with "." for ":".

	PassageID("1 John 4:7-8", "esv") == "ESV.1John.4.7-8"
	PassageID("Psalm 23", "")        == "KJV.Psalm.23"

BLB resolves the book name itself, abbreviations included, so this does no book lookup of its own and an unknown book is reported by BLB (ErrInvalidReference from Fetch), not here. An error here means the reference has no chapter number to send.

<small>[blb/blb.go:322](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L322)</small>

### func SearchURL

```go
func SearchURL(ref, translation string) string
```

SearchURL is a link to ref on Blue Letter Bible that needs no fetch first: BLB's own search, which opens the passage when the query is a reference. It is what the ScriptTagger links a reference to, and the fallback for a comps.BibleVerse whose fetch failed.

<small>[blb/blb.go:351](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L351)</small>

## Types

### type Client

```go
type Client struct {
	// HTTPClient makes the requests. Nil uses a client with a 15-second
	// timeout. The timeout is a backstop; a call's context still governs.
	HTTPClient *http.Client

	// BaseURL replaces DefaultBaseURL, for a proxy (see "Where it works"
	// and Proxy) or a test server. No trailing slash is needed.
	BaseURL string

	// Translation is the default for calls that pass "". Empty means
	// DefaultTranslation.
	Translation string

	// KeepMarkers keeps the translators' marks BLB leaves in the text: the
	// asterisk after a word that has a footnote (ESV, NKJV), and the square
	// brackets the KJV puts around words the translators supplied
	// ("The LORD [is] my shepherd"). Off, they are removed, which is how a
	// verse is quoted outside a study edition.
	KeepMarkers bool
}
```

Client fetches passages. The zero value is ready to use.

<small>[blb/blb.go:208](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L208)</small>

#### func (*Client) Fetch

```go
func (c *Client) Fetch(ctx context.Context, ref, translation string) (Passage, error)
```

Fetch fetches ref ("John 3:16", "1 John 4:7-8", "Psalm 23", "Rom 8:28,31") in translation ("" for the Client's default).

<small>[blb/blb.go:248](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L248)</small>

### type Passage

```go
type Passage struct {
	// Reference is BLB's own spelling of what was asked for, with the book
	// name expanded: "jn 3:16" comes back as "John 3:16", "Song 2:1" as
	// "Song of Songs 2:1". Show this rather than the caller's input.
	Reference string

	// Translation is the code the text is from, as BLB titled it ("ESV").
	Translation string

	// Verses are in the order BLB returned them, which is reading order.
	Verses []Verse

	// URL is the passage's page on blueletterbible.org, the link the text
	// owes its source. Empty only if BLB's reply carried no passage link.
	URL string

	// ChapterID is BLB's running number for the passage's chapter, counted
	// across the whole Bible: Genesis 1 is 1, John 3 is 1000, Revelation 22
	// is 1189. It is what BLB's study pages key a verse by (see StudyURL),
	// read from the s_ segment of the reply's passage link. Zero when the
	// reply carried no such segment, and StudyURL then has nothing to build.
	ChapterID int
}
```

Passage is a fetched reference.

<small>[blb/blb.go:104](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L104)</small>

#### func Fetch

```go
func Fetch(ctx context.Context, ref, translation string) (Passage, error)
```

Fetch fetches ref in translation with DefaultClient.

<small>[blb/blb.go:242](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L242)</small>

#### func (Passage) StudyURL

```go
func (p Passage) StudyURL(verse int, tool string) string
```

StudyURL is the study page on Blue Letter Bible for one verse of the passage, opened on tool's tab (one of the Tool constants; "" opens BLB's default, the interlinear). The verse is a number within the passage's chapter and need not be one of p.Verses: a passage of John 3:16 can link the study page of 3:17.

	p.StudyURL(17, blb.ToolCrossRefs)
	// "https://www.blueletterbible.org/kjv/jhn/3/17/t_corr_1000017"

The address is the passage page's, with the verse swapped in, followed by a segment naming the tab and BLB's verse id:

	/kjv/jhn/3/17/t_corr_1000017
	 └┬┘ └┬┘ │ └┤   └┬─┘ └──┬──┘
	  │   │  │  │    │      └ ChapterID × 1000 + verse
	  │   │  │  │    └ tab key
	  │   │  │  └ verse
	  │   │  └ chapter
	  │   └ BLB's book code
	  └ translation

BLB's verse id is not something it documents. The ids in every saved reply (Genesis 3:1 is s\_3001, Psalm 117:1 s\_595001, Romans 1:16 s\_1047016) are this product, and the live check in blb\_test.go follows the rule for a verse no reply named.

Empty when the passage has no URL or ChapterID to build from, as for a Passage put together by hand: link the passage itself (URL, or SearchURL) instead.

<small>[blb/blb.go:181](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L181)</small>

#### func (Passage) Text

```go
func (p Passage) Text() string
```

Text is the passage as one paragraph, verses joined by a space and their numbers left out: the form for a share sheet or a notification.

<small>[blb/blb.go:199](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L199)</small>

### type Proxy

```go
type Proxy struct {
	// HTTPClient makes the upstream requests. Nil uses the client
	// Client.Fetch uses, with its 15-second timeout.
	HTTPClient *http.Client

	// Upstream replaces DefaultBaseURL, for a test server. No trailing
	// slash is needed.
	Upstream string
}
```

Proxy forwards Client.Fetch's request to Blue Letter Bible on behalf of a browser build, which cannot make it itself: BLB's reply carries no Access-Control-Allow-Origin header, so the browser withholds it from WebAssembly (see "Where it works"). The app's own server mounts a Proxy at EndpointPath, and the app points its Client at that server:

	// server
	http.Handle(blb.EndpointPath, &blb.Proxy{})

	// wasm app (syscall/js)
	blb.DefaultClient.BaseURL = js.Global().Get("location").Get("origin").String()

The page and the proxy then share an origin, so no CORS header is needed at all. \`go run ./serve\` (and an app's ./dev.sh, which runs the same server) mounts one, so a browser build fetches verses in development with no setup.

#### Not an open proxy

A handler that forwards whatever it is asked to is a liability on any public server, so this one is narrow by construction:

  - The upstream is fixed (DefaultBaseURL, or Upstream for a test) and so is the path: the request's own path is ignored, so mounting it at the wrong route can only ever reach the one endpoint.
  - Only GET and HEAD are answered, and only the three query parameters Fetch sends (id, style, target) are forwarded, each bounded in length. A request without an id is refused before any upstream call.
  - None of the caller's headers travel: no cookies, no Authorization, no client address. The upstream sees the same User-Agent Fetch sends.
  - Only the status, the Content-Type and a body of at most maxReply bytes come back. A longer reply is refused rather than cut, because a cut passage parses as a short one.

rweb's own Server.Proxy was not used for this reason: it forwards every header, method and parameter, which is right for a backend you own and wrong for a third party's endpoint.

The zero value is ready to use.

<small>[blb/proxy.go:57](https://github.com/rohanthewiz/grmob/blob/master/blb/proxy.go#L57)</small>

#### func (*Proxy) Forward

```go
func (p *Proxy) Forward(ctx context.Context, rawQuery string) (ProxyReply, error)
```

Forward makes the upstream request for one proxied call, given the caller's raw query string. It is ServeHTTP without net/http, for a server built on another router (serve mounts it on rweb this way).

The reply is always one to send. A refused query is a 400 and a failed or oversized upstream reply is a 502, each with a one-line text body; the error is returned alongside so the server can log it. An upstream status other than 200 is passed through as it came, since it is BLB's answer and Client.Fetch already reads it as ErrUnexpectedResponse.

<small>[blb/proxy.go:112](https://github.com/rohanthewiz/grmob/blob/master/blb/proxy.go#L112)</small>

#### func (*Proxy) ServeHTTP

```go
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

ServeHTTP answers a browser's Client.Fetch through Forward.

<small>[blb/proxy.go:87](https://github.com/rohanthewiz/grmob/blob/master/blb/proxy.go#L87)</small>

### type ProxyReply

```go
type ProxyReply struct {
	Status      int
	ContentType string
	Body        []byte
}
```

ProxyReply is what Proxy.Forward hands back for the caller to write out: the status, the Content-Type ("" to leave unset) and the body.

<small>[blb/proxy.go:69](https://github.com/rohanthewiz/grmob/blob/master/blb/proxy.go#L69)</small>

### type Verse

```go
type Verse struct {
	Number int
	Text   string
}
```

Verse is one numbered verse of a passage.

<small>[blb/blb.go:98](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L98)</small>

