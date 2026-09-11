# syntax=docker/dockerfile:1.7
# AI Game Manager Panel Server Edition 多架构镜像。
# Web/Linux/Docker 是完整产品目标：镜像必须同时包含 AGMP Core + XiaoYu Runtime。
FROM node:24-alpine AS frontend-builder
WORKDIR /src
RUN corepack enable
COPY frontend ./frontend
WORKDIR /src/frontend
RUN pnpm install --no-frozen-lockfile && pnpm run build

FROM --platform=$TARGETPLATFORM rust:1.85-alpine AS xiaoyu-builder
WORKDIR /src
RUN apk add --no-cache musl-dev
COPY rust ./rust
WORKDIR /src/rust
RUN cargo build --release -p xiaoyu-core

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go-builder
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /src/frontend/dist /src/cmd/aigame-manager-web/web
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/ai-game-manager-panel-server ./cmd/aigame-manager-web

FROM alpine:3.22
RUN addgroup -S agmp && adduser -S -G agmp -h /var/lib/ai-game-manager-panel agmp \
    && apk add --no-cache ca-certificates tzdata
WORKDIR /var/lib/ai-game-manager-panel
COPY --from=go-builder /out/ai-game-manager-panel-server /usr/local/bin/ai-game-manager-panel-server
COPY --from=xiaoyu-builder /src/rust/target/release/xiaoyu /usr/local/lib/agmp/AI-Game-Manager-XiaoYu
COPY configs /var/lib/ai-game-manager-panel/configs
RUN chmod 0755 /usr/local/lib/agmp/AI-Game-Manager-XiaoYu \
    && mkdir -p runtime/data runtime/log runtime/backups runtime/instances runtime/temp runtime/exports runtime/plugins runtime/cache \
    && chown -R agmp:agmp /var/lib/ai-game-manager-panel
USER agmp
ENV AGMP_ROOT=/var/lib/ai-game-manager-panel \
    AGMP_XIAOYU_RUNTIME=/usr/local/lib/agmp/AI-Game-Manager-XiaoYu
EXPOSE 17890
ENTRYPOINT ["/usr/local/bin/ai-game-manager-panel-server"]
# 容器网络要求监听 0.0.0.0，因此此处是显式 remote opt-in；宿主示例默认仍只映射到 127.0.0.1。
CMD ["-listen", "0.0.0.0:17890", "-allow-remote-web"]
