// Command docs serves grmob's documentation site — the narrative pages under
// docs/ and the generated reference under docs/api/ — with gkdocs.
//
//	./docs.sh                     from the repository root
//	cd cmd/docs && go run .       the same thing, said longer
//	PORT=9000 ./docs.sh           somewhere else
//
// # Why this is a module of its own
//
// gkdocs brings rweb, element, logger, logrus, goldmark and yaml with it. None
// of that belongs in grmob's go.mod: a mobile framework's dependency graph is
// something its users inherit, and they should not inherit a web server and a
// markdown renderer because the framework's authors wanted to read their own
// docs in a browser. A nested module keeps the whole set behind a go.mod that
// nothing importing grmob ever reads.
//
// The cost is that `go build ./...` and `go test ./...` at the repository root
// do not reach this directory. CI builds it as a step of its own for that
// reason; without it this file could stop compiling and nothing would say so.
//
// # Why not GitHub Pages
//
// gkdocs renders on request rather than building a static tree, so there is
// nothing to upload. The repository's Pages deployment is already taken by the
// interactive tutorial's WASM build (.github/workflows/site.yml), which is a
// genuinely static artifact. Hosting this is a matter of running the binary —
// locally, or in the container the Dockerfile next to this file builds.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rohanthewiz/gkdocs"
	"github.com/rohanthewiz/rweb"
)

const (
	defaultPort = "8000"

	// The site's own config, which gkdocs reads for the nav, the site name and
	// docs_dir. It lives at the repository root next to docs/.
	configName = "mkdocs.yml"

	// The embedded theme's CSS and JS are the only responses cached. Markdown
	// is deliberately left uncached so that an edit shows up on the next
	// reload, which is the whole point of running this locally.
	assetMaxAge = time.Hour
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "docs:", err)
		os.Exit(1)
	}
}

func run() error {
	configPath, err := findConfig()
	if err != nil {
		return err
	}

	port := envOr("PORT", defaultPort)

	// KeepTrailingSlashes is not optional here: it is the precondition for
	// gkdocs' StrictSlashes below, which is what gives the site mkdocs'
	// trailing-slash semantics — `/api/` and `/api` are different URLs, and
	// the canonical one is reached by a 301 rather than by both working
	// silently. Getting this pair out of step is the one misconfiguration that
	// produces a site that looks correct and links wrong.
	s := rweb.NewServer(rweb.ServerOptions{
		Address:    ":" + port,
		Verbose:    true,
		URLOptions: rweb.URLOptions{KeepTrailingSlashes: true},
	})
	s.Use(rweb.RequestInfo)

	// For a container health check, and outside the docs mount so that it
	// cannot be shadowed by a page called "health".
	s.Get("/health", func(ctx rweb.Context) error { return ctx.WriteString("ok") })

	h, err := gkdocs.New(gkdocs.Options{
		ConfigPath: configPath,
		// The root: this process serves nothing but the docs. A host that
		// mounts the site alongside an application would set "/docs" here
		// instead — that is the shape gkdocs is designed for, and the two
		// lines below are the whole of the integration.
		MountPath:     "",
		StrictSlashes: true,
		AssetMaxAge:   assetMaxAge,
	})
	if err != nil {
		return err
	}
	h.RegisterRoutes(s)

	fmt.Printf("Serving %q from %s\n", h.SiteName(), configPath)
	fmt.Printf("  http://localhost:%s/\n", port)
	fmt.Printf("  http://localhost:%s/api/    (generated reference)\n", port)

	return s.Run()
}

// findConfig locates mkdocs.yml.
//
// MKDOCS_CONFIG wins, for the container and for anyone serving a docs tree from
// somewhere else. Otherwise the search walks up from the working directory,
// which is what lets this be started from the repository root, from cmd/docs,
// or from anywhere in between without an argument — and the walk is also why
// the Dockerfile can simply copy the tree in and set a workdir.
func findConfig() (string, error) {
	if p := os.Getenv("MKDOCS_CONFIG"); p != "" {
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("MKDOCS_CONFIG=%s: %w", p, err)
		}
		return p, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	start := dir
	for {
		candidate := filepath.Join(dir, configName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %s or any parent — "+
				"run this from inside the repository, or set MKDOCS_CONFIG",
				configName, start)
		}
		dir = parent
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
