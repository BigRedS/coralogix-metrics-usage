# syntax=docker/dockerfile:1.7

# Build a static webui binary, then ship it on distroless/static.
#
# Build context is the repo root (the webui imports internal/... packages).
# From the repo root:
#   docker build -f webui/Dockerfile -t coralogix-unused-metrics-finder-webui .
#
# Designed to run with `--read-only`:
#   docker run --rm \
#       --read-only \
#       --tmpfs /tmp:rw,size=512m,mode=1777 \
#       --cap-drop=ALL \
#       --security-opt=no-new-privileges \
#       -p 8765:8765 \
#       coralogix-unused-metrics-finder-webui
#
# The webui writes job output under os.MkdirTemp("") (i.e. /tmp), so /tmp
# must be writable — a tmpfs satisfies that without making the rootfs rw.

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH}
RUN go build -trimpath -ldflags "-s -w" -o /out/webui ./webui

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/webui /webui
EXPOSE 8765
USER nonroot:nonroot
ENTRYPOINT ["/webui"]
CMD ["-listen", ":8765"]
