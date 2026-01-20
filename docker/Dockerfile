FROM golang:1.25-alpine3.22 AS Builder
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.22/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.22/community"    >> /etc/apk/repositories && \
    echo "Asia/Shanghai" > /etc/timezone
RUN apk add git bash gcc musl-dev upx
ENV TZ=Asia/Shanghai
RUN apk add tzdata && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
WORKDIR /app
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

# 定义构建参数，提供默认值以支持本地构建
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE

# 如果没有提供 BUILD_DATE，则使用当前时间
RUN BUILD_DATE=${BUILD_DATE:-$(date +%FT%T%z)} && \
    GO111MODULE=on GOARCH=amd64 go build -ldflags "-w -s -X 'github.com/soulteary/warden/internal/version.Version=$VERSION' -X 'github.com/soulteary/warden/internal/version.Commit=$COMMIT' -X 'github.com/soulteary/warden/internal/version.BuildDate=$BUILD_DATE' " -o warden main.go
RUN upx -9 -o warden.minify warden && mv warden.minify warden


FROM alpine:3.22
COPY --from=Builder /app/warden /bin/warden
ENV TZ=Asia/Shanghai
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add tzdata curl && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone && \
    rm -rf /var/cache/apk/*
WORKDIR /app
CMD ["warden"]