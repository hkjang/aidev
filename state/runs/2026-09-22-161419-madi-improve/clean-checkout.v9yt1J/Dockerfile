# Build once on a connected network. Runtime assets are bundled; only explicitly
# configured integrations need access to their administrator-approved endpoints.
FROM node:22-alpine AS web
WORKDIR /build/web
COPY web/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.8-alpine AS server
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=web /build/web/dist ./web/dist
ARG VERSION
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    MADI_VERSION="${VERSION:-$(tr -d '\r\n' < VERSION)}" && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${MADI_VERSION}" -o /out/madi ./cmd/madi

FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS runtime-base
COPY deploy/runtime-apk.lock /runtime-apk.lock
RUN sed '/^#/d; /^$/d' /runtime-apk.lock | xargs apk add --no-cache && \
    cp /lib/apk/db/installed /runtime-apk-installed

# Corresponding sources are part of this SAME image, not an extra release asset.
# This connected build stage verifies official APKBUILD archive checksums.
FROM node:22-alpine3.24 AS runtime-sources
RUN apk add --no-cache abuild curl git
COPY --from=runtime-base /runtime-apk-installed /runtime-apk-installed
COPY --from=runtime-base /usr/share/tessdata/eng.traineddata /runtime-tessdata/eng.traineddata
COPY --from=runtime-base /usr/share/tessdata/kor.traineddata /runtime-tessdata/kor.traineddata
COPY --from=web /build/web/node_modules/pdfjs-dist/package.json /runtime-pdfjs/package.json
COPY --from=web /build/web/node_modules/pdfjs-dist/standard_fonts /runtime-pdfjs/standard_fonts
COPY scripts/bundle-alpine-sources.mjs /build/bundle-alpine-sources.mjs
COPY scripts/bundle-pdf-font-sources.mjs /build/bundle-pdf-font-sources.mjs
RUN --mount=type=cache,target=/madi-source-cache,sharing=locked \
    DISTFILES_MIRROR=https://distfiles.alpinelinux.org/distfiles/v3.24 \
    node /build/bundle-alpine-sources.mjs /runtime-apk-installed /build/madi-sources

FROM runtime-base
LABEL org.opencontainers.image.title="madi" \
      org.opencontainers.image.source="https://github.com/hkjang/madi" \
      org.opencontainers.image.description="Self-hosted Korean knowledge and AI workspace"
RUN addgroup -g 10001 madi && adduser -D -H -u 10001 -G madi madi && \
    mkdir -p /var/lib/madi/attachments && chown -R madi:madi /var/lib/madi
COPY --from=server /out/madi /usr/local/bin/madi
COPY --from=runtime-sources /build/madi-sources /usr/share/madi/sources
USER 10001:10001
WORKDIR /var/lib/madi
EXPOSE 8080
VOLUME ["/var/lib/madi"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/madi"]
