# syntax=docker/dockerfile:1.7
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --ignore-scripts && node node_modules/esbuild/install.js
COPY web/ ./
RUN npm run build

FROM golang:1.24-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
RUN rm -rf cmd/seaton/webdist && mkdir -p cmd/seaton/webdist
COPY --from=web /src/web/dist/ cmd/seaton/webdist/
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT_AT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.builtAt=${BUILT_AT}" \
    -o /out/seaton ./cmd/seaton

FROM debian:bookworm-slim
LABEL org.opencontainers.image.title="SeatOn" \
      org.opencontainers.image.description="Offline-ready smart office seat management" \
      org.opencontainers.image.source="https://github.com/hkjang/seaton"
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata poppler-utils \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 seaton \
    && useradd --system --uid 10001 --gid seaton --home-dir /var/lib/seaton seaton \
    && install -d -o seaton -g seaton -m 0700 /var/lib/seaton
COPY --from=backend --chown=seaton:seaton /out/seaton /usr/local/bin/seaton
USER 10001:10001
EXPOSE 8080
VOLUME ["/var/lib/seaton"]
ENTRYPOINT ["/usr/local/bin/seaton"]
