module github.com/rohanthewiz/grmob

go 1.26.1

require github.com/rohanthewiz/bytdb v0.11.0

require (
	github.com/rohanthewiz/btypedb v0.7.0 // indirect
	github.com/rohanthewiz/serr v1.4.0 // indirect
	github.com/tidwall/btype v0.3.0 // indirect
	// x/mobile and friends are not imported by any Go file; they pin the
	// gomobile toolchain (do not let `go mod tidy` drop them).
	golang.org/x/mobile v0.0.0-20251021151156-188f512ec823 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
)
