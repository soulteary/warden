# Warden Makefile
# 构建、校验与安全扫描相关目标。生成物（二进制）不提交到版本库。

SHELL := /bin/bash

# 版本信息（用于 -ldflags 注入 version-kit）
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%FT%T%z)

LDFLAGS := -s -w \
	-X github.com/soulteary/version-kit.Version=$(VERSION) \
	-X github.com/soulteary/version-kit.Commit=$(COMMIT) \
	-X github.com/soulteary/version-kit.BuildDate=$(BUILD_DATE)

# Docker 镜像构建参数
IMAGE ?= warden:dev
# UPX 默认关闭；如需开启：make docker ENABLE_UPX=1
ENABLE_UPX ?= 0

MOCK_API_DIR := example/advanced/mock-api

.PHONY: help build build-mock-api mock-api vet fmt fmt-check test test-race \
	govulncheck sbom docker docker-manual clean

help: ## 显示可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

build: ## 构建 warden 主二进制（静态链接）
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o warden .

## 生成 mock-api 演示二进制（此前误提交的二进制现由此目标生成）
mock-api build-mock-api:
	cd $(MOCK_API_DIR) && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o mock-api main.go

fmt: ## 格式化代码
	gofmt -s -w .

fmt-check: ## 检查代码格式（CI 用）
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "以下文件格式不正确:"; gofmt -s -d .; exit 1; \
	fi

vet: ## 运行 go vet 静态检查
	go vet ./...

test: ## 运行测试
	go test ./...

test-race: ## 运行竞态检测测试
	go test -race -coverprofile=coverage.out ./...

govulncheck: ## 运行依赖漏洞扫描（需先安装 golang.org/x/vuln/cmd/govulncheck）
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

sbom: build ## 生成 SBOM（需安装 anchore/syft）
	@command -v syft >/dev/null 2>&1 || { echo "请先安装 syft: https://github.com/anchore/syft"; exit 1; }
	syft dir:. -o spdx-json=sbom.spdx.json

docker: ## 构建生产镜像（UPX 默认关闭，ENABLE_UPX=1 开启）
	docker build -f docker/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg ENABLE_UPX=$(ENABLE_UPX) \
		-t $(IMAGE) .

docker-manual: ## 使用国内镜像源构建生产镜像
	docker build -f docker/Dockerfile.manual \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		--build-arg ENABLE_UPX=$(ENABLE_UPX) \
		-t $(IMAGE) .

clean: ## 清理生成物
	rm -f warden warden.exe coverage.out coverage.html sbom.spdx.json
	rm -f $(MOCK_API_DIR)/mock-api
	rm -rf dist/
