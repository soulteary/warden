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
- **Require branches to be up to date before merging**
- **Require conversation resolution before merging**
- **Do not allow bypassing the above settings**（含管理员）
- （可选）**Require signed commits**

> 上述 check 名称对应 `.github/workflows/ci.yml` 中各 job 的 `name`。修改 job 名称后需同步更新仓库 required checks。

## 2. 容器加固要点

- 运行阶段以非 root 用户运行（uid/gid `10001`）。
- 基础镜像固定到明确版本（`golang:1.27.0-alpine3.23`、`alpine:3.23`）；高保证场景追加 `@sha256:<digest>` 固定不可变镜像。
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

当前第三方 Action 固定到主版本 tag，并由 `.github/dependabot.yml` 自动升级。

**高保证建议**：将第三方 Action 进一步固定到 commit SHA（例如 `uses: aquasecurity/trivy-action@<40 位 SHA> # v0.35.0`）。
Dependabot 支持在 SHA 固定后自动提交升级 PR，兼顾可复现与可维护。

## 6. CHANGELOG 维护

见根目录 `CHANGELOG.md` 顶部"维护流程"。Release 说明会引用该文件。
