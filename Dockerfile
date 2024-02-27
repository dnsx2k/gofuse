FROM golang:1.21.3-bullseye AS build_base
RUN go install github.com/psampaz/go-mod-outdated@latest

WORKDIR /src
COPY go.mod .
COPY go.sum .
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go mod download
RUN go get -u github.com/psampaz/go-mod-outdated
RUN go list -u -m -json all | go-mod-outdated -direct -update

FROM build_base as builder
COPY .. .
WORKDIR /src/cmd
RUN go build -ldflags="-w -s" -installsuffix cgo -tags=jsoniter -o /out/gofuse .

FROM debian:bullseye-slim as runner
ENV DEBIAN_FRONTEND noninteractive
RUN adduser --disabled-password --no-create-home --gecos '' appuser
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates net-tools curl \
    && apt-get clean -y \
    && apt-get autoremove -y \
    && rm -rf /tmp/* /var/tmp/* \
    && rm -rf /var/lib/apt/lists/*
RUN mkdir /apps && chown appuser:appuser /apps
WORKDIR /apps
USER appuser

FROM runner
WORKDIR /apps
COPY --from=builder --chown=appuser /out/gofuse .
EXPOSE 8085
ENV GIN_MODE=release
ENV LOG_SEVERITY=debug

ENTRYPOINT ["/apps/gofuse"]
