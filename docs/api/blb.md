# Package blb

```go
import "github.com/rohanthewiz/grmob/blb"
```

Package blb fetches Bible passages from Blue Letter Bible (blueletterbible.org), for comps.BibleVerse and anything else that wants a verse's text, its canonical reference and a link back to the passage.

	p, err := blb.Fetch(ctx, "John 3:16-18", "ESV")
	// p.Reference "John 3:16-18", p.Translation "ESV",
	// p.Verses [{16 "For God so loved…"} {17 …} {18 …}],
	// p.URL "https://www.blueletterbible.org/esv/jhn/3/16/"

## Where the text comes from

Blue Letter Bible has no open JSON API. What it does publish, for anyone to embed, is the BLB ScriptTagger (blueletterbible.org/webtools): a script a site includes so that every reference on its pages shows the verse in a hover bubble. The bubble's contents arrive from one endpoint,

	GET /remoteExtensions/toolTip/toolTipRemote.cfm?id=KJV.John.3.16&style=par

which answers with a few lines of JavaScript that hand an HTML string to the tagger. This package makes the same request the tagger makes and reads the same HTML out of the reply, so it serves exactly what BLB already serves to every page that embeds its tool, under the same terms: the text links back to Blue Letter Bible (Passage.URL), and comps.BibleVerse draws that link by default.

The endpoint is not a documented API, so its shape can change without notice. The parser is deliberately narrow (the title, the verse markers, the first passage link) and fails with ErrUnexpectedResponse rather than returning half a verse when the shape moves. The fixtures in blb\_test.go are verbatim replies, and the live test (GRMOB\_BLB\_LIVE=1) checks the endpoint itself.

## Where it works

On Android and iOS (Go's net/http under gomobile) and on a server. A browser build cannot call it directly: the reply carries no Access-Control-Allow-Origin header, so the browser refuses to hand it to WebAssembly. A browser app points Client.BaseURL at a same-origin proxy that forwards /remoteExtensions/... to www.blueletterbible.org.

## Translations

The translation is BLB's code, upper-case: KJV, NKJV, NLT, NIV, ESV, CSB, NASB20, NASB95, LSB, AMP, NET, RSV, ASV, YLT and WEB all answer. All but the KJV, ASV, YLT and WEB are under copyright. An app that stores or republishes their text, rather than showing it with the link, should check the publisher's terms first.

## Index

- [Constants](#constants) — `DefaultBaseURL`, `DefaultTranslation`
- [Variables](#variables) — `DefaultClient`, `ErrInvalidReference`, `ErrUnexpectedResponse`
- [`func PassageID`](#func-passageid)
- [`func SearchURL`](#func-searchurl)
- [`type Client`](#type-client)
    - [`func (*Client) Fetch`](#func-client-fetch)
- [`type Passage`](#type-passage)
    - [`func Fetch`](#func-fetch)
    - [`func (Passage) Text`](#func-passage-text)
- [`type Verse`](#type-verse)

## Constants

DefaultBaseURL is Blue Letter Bible's origin.

```go
const DefaultBaseURL = "https://www.blueletterbible.org"
```

<small>[blb/blb.go:65](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L65)</small>

DefaultTranslation is used when neither the call nor the Client names one. The King James is the one translation BLB serves that is in the public domain everywhere it is read (outside the UK's Crown patent), which makes it the safe default for an app that has not thought about licensing yet.

```go
const DefaultTranslation = "KJV"
```

<small>[blb/blb.go:71](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L71)</small>

## Variables

DefaultClient is used by the package-level Fetch.

```go
var DefaultClient = &Client{}
```

<small>[blb/blb.go:139](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L139)</small>

ErrInvalidReference is returned when BLB does not recognise the reference ("Hezekiah 1:1", "John 99:1"). Test for it with errors.Is.

```go
var ErrInvalidReference = errors.New("blb: invalid scripture reference")
```

<small>[blb/blb.go:75](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L75)</small>

ErrUnexpectedResponse is returned when the reply parses as neither a passage nor BLB's error bubble: the endpoint changed shape, or a proxy in front of it answered with something else.

```go
var ErrUnexpectedResponse = errors.New("blb: unexpected response")
```

<small>[blb/blb.go:80](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L80)</small>

## Functions

### func PassageID

```go
func PassageID(ref, translation string) (string, error)
```

PassageID builds the tagger's id for ref: translation, book with its spaces removed, then chapter and verses with "." for ":".

	PassageID("1 John 4:7-8", "esv") == "ESV.1John.4.7-8"
	PassageID("Psalm 23", "")        == "KJV.Psalm.23"

BLB resolves the book name itself, abbreviations included, so this does no book lookup of its own and an unknown book is reported by BLB (ErrInvalidReference from Fetch), not here. An error here means the reference has no chapter number to send.

<small>[blb/blb.go:231](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L231)</small>

### func SearchURL

```go
func SearchURL(ref, translation string) string
```

SearchURL is a link to ref on Blue Letter Bible that needs no fetch first: BLB's own search, which opens the passage when the query is a reference. It is what the ScriptTagger links a reference to, and the fallback for a comps.BibleVerse whose fetch failed.

<small>[blb/blb.go:260](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L260)</small>

## Types

### type Client

```go
type Client struct {
	// HTTPClient makes the requests. Nil uses a client with a 15-second
	// timeout. The timeout is a backstop; a call's context still governs.
	HTTPClient *http.Client

	// BaseURL replaces DefaultBaseURL, for a proxy (see "Where it works")
	// or a test server. No trailing slash is needed.
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

<small>[blb/blb.go:117](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L117)</small>

#### func (*Client) Fetch

```go
func (c *Client) Fetch(ctx context.Context, ref, translation string) (Passage, error)
```

Fetch fetches ref ("John 3:16", "1 John 4:7-8", "Psalm 23", "Rom 8:28,31") in translation ("" for the Client's default).

<small>[blb/blb.go:157](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L157)</small>

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
}
```

Passage is a fetched reference.

<small>[blb/blb.go:89](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L89)</small>

#### func Fetch

```go
func Fetch(ctx context.Context, ref, translation string) (Passage, error)
```

Fetch fetches ref in translation with DefaultClient.

<small>[blb/blb.go:151](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L151)</small>

#### func (Passage) Text

```go
func (p Passage) Text() string
```

Text is the passage as one paragraph, verses joined by a space and their numbers left out: the form for a share sheet or a notification.

<small>[blb/blb.go:108](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L108)</small>

### type Verse

```go
type Verse struct {
	Number int
	Text   string
}
```

Verse is one numbered verse of a passage.

<small>[blb/blb.go:83](https://github.com/rohanthewiz/grmob/blob/master/blb/blb.go#L83)</small>

