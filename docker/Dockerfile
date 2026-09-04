# syntax=docker/dockerfile:1
#
# 生产镜像：多阶段构建，运行阶段以非 root 用户运行，仅包含静态二进制与 CA 证书。
# 基础镜像已固定到明确版本；高保证场景可进一步固定 digest（见下方注释）。

# 高保证：将 tag 追加 @sha256:<digest> 以固定不可变镜像，例如：
#   FROM golang:1.27.0-alpine3.23@sha256:<digest> AS builder
FROM golang:1.27.1-alpine3.23 AS builder

# 仅安装构建期需要的工具；upx 仅在显式开启时使用（见 ENABLE_UPX）。
RUN apk add --no-cache git bash gcc musl-dev upx
WORKDIR /app
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

# 定义构建参数，提供默认值以支持本地构建
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE

# UPX 默认关闭：压缩会增加冷启动解压开销、干扰部分安全扫描/调试，且对静态 Go 二进制收益有限。
# 需要时显式开启：--build-arg ENABLE_UPX=1
ARG ENABLE_UPX=0

# 多平台：TARGETOS/TARGETARCH 由 Docker Buildx 注入；本地 build 时用当前内核架构，避免 exec format error（如 ARM Mac）
ARG TARGETOS
ARG TARGETARCH
RUN BUILD_DATE=${BUILD_DATE:-$(date +%FT%T%z)} && \
    ARCH=${TARGETARCH:-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/armv7l/arm/')} && \
    OS=${TARGETOS:-linux} && \
    GO111MODULE=on GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-w -s -X github.com/soulteary/version-kit/v2.Version=$VERSION -X github.com/soulteary/version-kit/v2.Commit=$COMMIT -X github.com/soulteary/version-kit/v2.BuildDate=$BUILD_DATE" \
    -o warden .

# UPX 压缩仅在 ENABLE_UPX=1 时执行；失败则跳过以保证多平台构建兼容性。
RUN if [ "$ENABLE_UPX" = "1" ] && command -v upx > /dev/null 2>&1; then \
      upx -9 -o warden.minify warden 2>/dev/null && mv warden.minify warden || true; \
    fi


# 高保证：可将 tag 追加 @sha256:<digest> 固定运行时基础镜像。
FROM alpine:3.23

# Refresh packages from the pinned Alpine release before adding runtime tools. This
# keeps the image on the latest patched package revision (including OpenSSL fixes)
# without changing the selected Alpine release.
RUN apk upgrade --no-cache && \
    apk add --no-cache curl ca-certificates

# 运行阶段以非 root 用户运行，降低容器逃逸/文件篡改风险。
# 固定 uid/gid 便于卷挂载权限管理与审计。
RUN addgroup -g 10001 -S warden && \
    adduser -u 10001 -S -G warden -H -s /sbin/nologin warden

COPY --from=builder /app/warden /bin/warden

WORKDIR /app
USER 10001:10001
CMD ["warden"]
