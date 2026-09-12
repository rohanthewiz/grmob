# The docs site as a container.
#
# The build context is the REPOSITORY ROOT, not this directory — the image needs
# mkdocs.yml and the whole docs/ tree, which live there:
#
#   docker build -f cmd/docs/Dockerfile -t grmob-docs .
#   docker run --rm -p 8000:8000 grmob-docs
#
# Nothing is generated at image build time. docs/api/ is committed (see
# internal/apidoc), so the image needs no Go toolchain beyond the one that
# compiles the server, and a page it serves is byte for byte the page in the
# repository.

FROM golang:1.26 AS build

WORKDIR /src

# The manifests first, on their own layer: dependencies change far less often
# than the docs do, so an edit to a markdown file reuses the download.
COPY cmd/docs/go.mod cmd/docs/go.sum ./cmd/docs/
RUN cd cmd/docs && go mod download

COPY . .

# CGO off and a static link, because the runtime stage below has no libc.
RUN cd cmd/docs && CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/docs .

# Distroless static: the server reads markdown off the filesystem and serves it.
# It needs no shell, no package manager and no certificate store — every asset
# it serves is either embedded in the binary or copied in below.
FROM gcr.io/distroless/static-debian12

# The server finds its config by walking up from the working directory, so the
# config and the docs tree are laid out here the way they are in the repository
# and no MKDOCS_CONFIG is needed.
WORKDIR /app
COPY --from=build /out/docs /usr/local/bin/docs
COPY mkdocs.yml ./mkdocs.yml
COPY docs ./docs

# Matches the server's own default. PORT overrides it.
EXPOSE 8000

# No HEALTHCHECK: a distroless image has no shell and no curl to run one with.
# The endpoint is there for whatever orchestrator is in front of it —
# GET /health returns "ok".
ENTRYPOINT ["/usr/local/bin/docs"]
