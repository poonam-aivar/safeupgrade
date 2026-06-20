FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown

WORKDIR /build

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.Commit=${COMMIT}" \
    -o /safeupgrade .

FROM alpine:3.19

RUN apk add --no-cache \
    git \
    nodejs \
    npm \
    python3 \
    ca-certificates \
    bash \
    && python3 -m venv /opt/venv

ENV PATH="/opt/venv/bin:$PATH"
RUN pip install --no-cache-dir pip --upgrade

RUN addgroup -g 1000 safeupgrade && \
    adduser -D -u 1000 -G safeupgrade safeupgrade && \
    mkdir -p /workspace && chown safeupgrade:safeupgrade /workspace

COPY --from=builder /safeupgrade /usr/local/bin/safeupgrade
COPY --chown=safeupgrade:safeupgrade --from=builder /build/configs /etc/safeupgrade/

WORKDIR /workspace
USER safeupgrade

ENTRYPOINT ["safeupgrade"]
CMD ["--help"]
