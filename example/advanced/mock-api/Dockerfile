FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制源代码
COPY main.go .
COPY data.json .

# 构建应用
RUN go mod init mock-api && \
    go build -o mock-api main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件和数据
COPY --from=builder /app/mock-api .
COPY --from=builder /app/data.json .

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# 运行服务
CMD ["./mock-api"]

