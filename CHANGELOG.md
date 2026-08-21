# Changelog

本项目变更记录遵循 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## 维护流程

- 每次改动在 `Unreleased` 段落记录，按 `Added` / `Changed` / `Fixed` / `Security` / `Removed` 分类。
- 发布时（打 `vX.Y.Z` tag，触发 `.github/workflows/release.yml`）：
  1. 将 `Unreleased` 内容移动到新版本段落并标注日期；
  2. 重新建立空的 `Unreleased` 段落；
  3. 提交后再打 tag。
- Release 说明会引用本文件；请保持条目简洁、面向使用者。

## [Unreleased]

### Security
- 容器：运行阶段以非 root 用户（uid/gid 10001）运行；基础镜像固定到明确版本并说明可选 digest 固定；`-trimpath` 构建去除本地路径信息。
- 容器：UPX 压缩默认关闭，改为显式 build arg `ENABLE_UPX=1`。
- 构建上下文：收紧 `.dockerignore`，排除源码无关文件、密钥、证书、真实数据与已提交产物。
- CI：新增容器镜像漏洞扫描（Trivy，HIGH/CRITICAL 失败）、SBOM（SPDX）生成、非 root 运行校验。
- Release：容器镜像新增 keyless cosign 签名与镜像 SBOM，校验和覆盖全部发布产物。

### Changed
- Docker/Compose 中 Redis 从已 EOL 的 `6.2.4` 升级至受支持的 `7.4-alpine`（Warden 仅使用基础命令，向后兼容）。

### Removed
- 从版本库移除误提交的编译二进制 `example/advanced/mock-api/mock-api`，改由 `make mock-api` 或 Docker 构建生成。

### Added
- `Makefile`：提供 `build`、`mock-api`、`vet`、`test-race`、`govulncheck`、`sbom`、`docker` 等目标。
- `.github/dependabot.yml`：自动更新 Go 依赖、GitHub Actions 与 Docker 基础镜像。
- `docs/RELEASE_SECURITY.md`：发布与分支保护建议。
