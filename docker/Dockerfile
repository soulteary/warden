FROM golang:1.19.0-alpine3.16 AS Builder
RUN echo '' > /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.16/main"         >> /etc/apk/repositories && \
    echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.16/community"    >> /etc/apk/repositories && \
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

RUN VERSION=$(cat RELEASE_VERSION) && \
    COMMIT=$(cat RELEASE_COMMIT) && \
    BUILD_DATE=`date +%FT%T%z` && \
    GO111MODULE=on GOARCH=amd64 go build -ldflags "-w -s -X 'soulteary.com/soulteary/warden/internal/version.Version=$VERSION' -X 'soulteary.com/soulteary/warden/internal/version.Commit=$COMMIT' -X 'soulteary.com/soulteary/warden/internal/version.BuildDate=$BUILD_DATE' " -o warden main.go
RUN upx -9 -o warden.minify warden && mv warden.minify warden


FROM alpine:3.16
COPY --from=Builder /app/warden /bin/warden
ENV TZ=Asia/Shanghai
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories
RUN apk add tzdata curl && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone && \
    rm -rf /var/cache/apk/*
WORKDIR /app
CMD ["warden"]