# 发布与供应链安全 / Release & Supply-chain Security

本文件记录 Warden 的容器加固、CI/发布安全实践，以及**推荐的分支保护设置**。
分支保护属于仓库设置（在 GitHub 仓库 Settings → Branches 配置），无法、也不应在代码中模拟——此处仅作文档化建议。

## 1. 推荐的分支保护 Required Checks

对 `main`（及 `develop`）分支建议启用：

- **Require a pull request before merging**（禁止直接 push）
- **Require approvals**：至少 1 名审阅者
- **Require status checks to pass before merging**，并将以下 CI 作业设为 required：
  - `代码格式检查`（fmt）
  - `代码静态检查`（vet）
  - `代码测试`（test，含 `-race`）
  - `代码质量检查`（golangci-lint）
  - `依赖项安全扫描`（govulncheck）
  - `Docker 构建`（含 Trivy 镜像扫描、SBOM、非 root 校验）
  - `构建检查`
  - `Redis 快照集成测试`
  - `Go quality report`（生成质量报告及英文文档拼写检查）
- **Require branches to be up to date before merging**
- **Require conversation resolution before merging**
- **Do not allow bypassing the above settings**（含管理员）
- （可选）**Require signed commits**

> 上述 check 名称对应 `.github/workflows/ci.yml` 中各 job 的 `name`。修改 job 名称后需同步更新仓库 required checks。

## 2. 容器加固要点

- 运行阶段以非 root 用户运行（uid/gid `10001`）。
- 基础镜像固定到明确版本（`golang:1.27.3-alpine3.24`、`alpine:3.24`）；高保证场景追加 `@sha256:<digest>` 固定不可变镜像。
- 多阶段构建，运行镜像不含编译器、源码、私钥或测试数据。
- 使用 `-trimpath` 去除构建路径信息。
- UPX 默认关闭；如需开启：`make docker ENABLE_UPX=1` 或 `--build-arg ENABLE_UPX=1`。

## 3. CI 安全检查

`.github/workflows/ci.yml` 包含：`go vet`、`golangci-lint`、`go test -race`、`govulncheck`（依赖漏洞）、Docker 构建 + Trivy 镜像扫描（HIGH/CRITICAL 失败）+ SBOM 生成 + 非 root 运行校验。

## 4. 发布安全

`.github/workflows/release.yml`：

- 多平台二进制 + `checksums.txt`（SHA256，覆盖全部产物）。
- 容器镜像 keyless cosign 签名（基于 GitHub OIDC，无长期私钥）。验证：

  ```bash
  cosign verify ghcr.io/<owner>/warden:<tag> \
    --certificate-identity-regexp "^https://github.com/<owner>/warden/" \
    --certificate-oidc-issuer https://token.actions.githubusercontent.com
  ```

- 容器镜像 SBOM（SPDX）随 Release 发布。

## 5. GitHub Actions 固定策略

当前 CI 和发布工作流中的第三方 Action 固定到 commit SHA，旁注标明对应版本；`.github/dependabot.yml` 跟踪后续升级。

更新 Action 时应同时核对 SHA 和版本旁注，保持不可变引用。
Dependabot 支持在 SHA 固定后自动提交升级 PR，兼顾可复现与可维护。

## 6. CHANGELOG 维护

见根目录 `CHANGELOG.md` 顶部"维护流程"。Release 说明会引用该文件。

## 7. 发布范围与验证

`v1.4.4` 基于维护分支，不包含当时 main 上尚未发布的功能；不要只按 tag
日期推断内容。`v1.5.0` 发布 main 上的功能，升级说明见
[英文 Release Notes](releases/v1.5.0.md)。

构建、race、覆盖率和集成验证继续使用 Linux。发布工作流在 Ubuntu 上交叉编译其他
系统的二进制，这不等于已在这些系统上运行验证；其他宿主环境推荐使用 Linux 容器。
分支 CI 按原始语句计数要求至少 95% 覆盖率，发布工作流自身的测试步骤不替代该门禁。
发布前应确认 tag 指向已通过 main CI 的提交，发布后核对工作流、产物校验和、镜像
签名和 SBOM。历史 tag、已发布报告及其原始阈值保持不变。

发布工作流优先读取 tag 内的 `docs/releases/<tag>.md`；旧 tag 没有该文件时保留英文
通用下载说明。版本化说明在删除/重建 Release 前选择，避免把缺少新文档的历史 tag
误当成不可重发。Release Notes 不从当前 main 的文件替代旧 tag 内容。
