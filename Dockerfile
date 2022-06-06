ARG BASEIMAGE
ARG GOVERSION
ARG LDFLAGS
ARG PKGNAME

FROM --platform=$BUILDPLATFORM golang:${GOVERSION} as builder

WORKDIR /go/src/github.com/clusternet/sample-controller
COPY pkg pkg/
COPY cmd cmd/
COPY go.mod go.mod
COPY go.sum go.sum

ARG LDFLAGS
ARG PKGNAME
ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="${LDFLAGS}" -a -o ${PKGNAME} /go/src/github.com/clusternet/sample-controller/cmd/${PKGNAME}

FROM ${BASEIMAGE}
WORKDIR /root
RUN apk add gcompat
ARG PKGNAME
COPY --from=builder /go/src/github.com/clusternet/sample-controller/${PKGNAME} /usr/local/bin/