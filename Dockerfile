FROM oven/bun:latest AS builder

WORKDIR /build
COPY web/package.json .
COPY web/bun.lock .
RUN bun install
COPY ./web .
COPY ./VERSION .
RUN DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

FROM golang:alpine AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=builder /build/dist ./web/dist
RUN go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

FROM debian:bookworm-slim

ARG TARGETOS
ARG TARGETARCH
ENV HYDRA_VERSION=v25.4.0

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Download hydra binary from GitHub releases (glibc compatible)
RUN ARCH=$(case "${TARGETARCH:-amd64}" in \
        amd64) echo "64bit" ;; \
        arm64) echo "arm64" ;; \
        *) echo "64bit" ;; \
    esac) && \
    wget -q "https://github.com/ory/hydra/releases/download/${HYDRA_VERSION}/hydra_${HYDRA_VERSION#v}-linux_${ARCH}.tar.gz" -O /tmp/hydra.tar.gz && \
    tar -xzf /tmp/hydra.tar.gz -C /usr/bin hydra && \
    chmod +x /usr/bin/hydra && \
    rm /tmp/hydra.tar.gz

COPY --from=builder2 /build/new-api /
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/entrypoint.sh"]
