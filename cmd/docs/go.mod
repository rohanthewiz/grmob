// The docs server is a module of its own so that gkdocs and its transitive
// dependencies — rweb, element, logger, logrus, goldmark, yaml — stay out of
// github.com/rohanthewiz/grmob's go.mod, which every consumer of the framework
// inherits. See main.go.
module github.com/rohanthewiz/grmob/cmd/docs

go 1.26.1

require (
	github.com/rohanthewiz/gkdocs v0.1.4
	github.com/rohanthewiz/rweb v0.1.26
)

require (
	github.com/rohanthewiz/element v0.5.6 // indirect
	github.com/rohanthewiz/logger v1.3.0 // indirect
	github.com/rohanthewiz/serr v1.3.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/sys v0.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
