# syntax=docker/dockerfile:1
#
# 演示用 Mock API：多阶段构建，运行阶段以非 root 用户运行。
# 二进制在构建期生成，不再提交到版本库（见仓库 .gitignore）。
FROM golang:1.27.0-alpine3.23 AS builder

WORKDIR /app

# 使用已有的 go.mod（无需 go mod init），复制源码后构建静态二进制
COPY go.mod .
COPY main.go .

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o mock-api main.go

# 运行阶段
FROM alpine:3.23

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 10001 -S mockapi && \
    adduser -u 10001 -S -G mockapi -H -s /sbin/nologin mockapi

WORKDIR /app

COPY --from=builder /app/mock-api .

EXPOSE 8080

# data.json 通过卷挂载提供（见 example/advanced/docker-compose.yml）
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

USER 10001:10001
CMD ["./mock-api"]
