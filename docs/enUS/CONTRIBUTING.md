# Contributing Guide

> 🌐 **Language / 语言**: [English](CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Thank you for your interest in the Warden project! We welcome all forms of contributions.

## 📋 Table of Contents

- [How to Contribute](#how-to-contribute)
- [Development Environment Setup](#development-environment-setup)
- [Code Standards](#code-standards)
- [Translation Policy](#translation-policy)
- [Commit Standards](#commit-standards)
- [Pull Request Process](#pull-request-process)
- [Bug Reports and Feature Requests](#bug-reports-and-feature-requests)

## 🚀 How to Contribute

You can contribute in the following ways:

- **Report Bugs**: Report issues in GitHub Issues
- **Suggest Features**: Propose new feature ideas in GitHub Issues
- **Submit Code**: Submit code improvements via Pull Requests
- **Improve Documentation**: Help improve project documentation
- **Answer Questions**: Help other users in Issues

When participating in this project, please respect all contributors, accept constructive criticism, and focus on what's best for the project.

## 🛠️ Development Environment Setup

### Prerequisites

- Go 1.27 or higher
- Redis (for testing)
- Git

### Quick Start

```bash
# 1. Fork and clone the project
git clone https://github.com/your-username/warden.git
cd warden

# 2. Add upstream repository
git remote add upstream https://github.com/soulteary/warden.git

# 3. Install dependencies
go mod download

# 4. Run tests
go test ./...

# 5. Start local service (ensure Redis is running)
go run .
```

## 📝 Code Standards

Please follow these code standards:

1. **Follow Go Official Code Standards**: [Effective Go](https://go.dev/doc/effective_go)
2. **Format Code**: Run `go fmt ./...`
3. **Code Checking**: Use `golangci-lint` or `go vet ./...`
4. **Write Tests**: New features must include tests
5. **Add Comments**: Public functions and types must have documentation comments
6. **Constant Naming**: All constants must use `ALL_CAPS` (UPPER_SNAKE_CASE) naming style

For detailed code style guidelines, please refer to [CODE_STYLE.md](CODE_STYLE.md).

## 🌐 Translation Policy

Warden ships documentation and runtime messages in seven languages. They are **not** all
maintained to the same standard, and pretending otherwise is how five locales silently
drifted 18 translation keys and several documents behind.

**Tiers**

| Tier | Languages | Expectation |
| --- | --- | --- |
| Authoritative | English (`enUS`), Simplified Chinese (`zhCN`) | Updated in the same pull request as the change. A PR that changes behavior without updating both is incomplete. |
| Best effort | `deDE`, `frFR`, `itIT`, `jaJP`, `koKR` | May lag. Each lagging document carries a banner pointing readers at the authoritative versions. |

**Runtime strings are not best effort.** `locales/*.json` is enforced by
`go test ./locales/`, which fails when any locale:

- is missing a key that `en.json` defines (or defines one it does not),
- has a different sequence of `printf` verbs (`%s`, `%d`) than the English source, or
- has a value byte-identical to English (an untranslated string).

Missing keys fall back to English at runtime, so nothing breaks visibly — which is exactly
why the check exists. When adding a user-facing message, add the key to **all seven**
locale files in the same commit. If a value is legitimately identical to English (a loan
word, a protocol name), add it to `intentionallyIdentical` in `locales/locales_test.go`
with a comment rather than deleting the check.

Run `make docs-parity` to see how far each translated document has drifted from `enUS`.

## 📦 Commit Standards

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) standard:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code format adjustment (doesn't affect code execution)
- `refactor`: Code refactoring
- `perf`: Performance optimization
- `test`: Test related
- `chore`: Build process or auxiliary tool changes

### Examples

```
feat(cache): Add Redis cache support

Implemented Redis-based distributed cache, supporting data persistence and multi-instance sharing.

Closes #123
```

```
fix(router): Fix pagination parameter validation issue

Fixed the issue where incorrect status code was returned when page_size exceeds maximum value.

Fixes #456
```

## 🔄 Pull Request Process

### Create Pull Request

```bash
# 1. Create feature branch
git checkout -b feature/your-feature-name

# 2. Make changes and commit
git add .
git commit -m "feat: Add new feature"

# 3. Sync upstream code
git fetch upstream
git rebase upstream/main

# 4. Push branch and create PR
git push origin feature/your-feature-name
```

### Pull Request Checklist

Before submitting a Pull Request, please ensure:

- [ ] Code follows project code standards
- [ ] All tests pass (`go test ./...`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] Necessary tests are added
- [ ] Related documentation is updated
- [ ] Commit message follows [Commit Standards](#commit-standards)
- [ ] Code passes lint checks

All Pull Requests require code review. Please respond to review comments promptly.

## 🐛 Bug Reports and Feature Requests

Before creating an Issue, please search existing Issues to confirm the problem or feature hasn't been reported.

### Bug Report Template

```markdown
**Description**
Clearly and concisely describe the bug.

**Reproduction Steps**
1. Execute '...'
2. See error

**Expected Behavior**
Clearly and concisely describe what you expected to happen.

**Actual Behavior**
Clearly and concisely describe what actually happened.

**Environment Information**
- OS: [e.g. macOS 12.0]
- Go Version: [e.g. 1.27]
- Redis Version: [e.g. 7.0]
```

### Feature Request Template

```markdown
**Feature Description**
Clearly and concisely describe the feature you want.

**Problem Description**
What problem does this feature solve? Why is it needed?

**Proposed Solution**
Clearly and concisely describe how you hope to implement this feature.
```

## 🎯 Getting Started

If you want to contribute but don't know where to start, you can focus on:

- Issues labeled `good first issue`
- Issues labeled `help wanted`
- `TODO` comments in code
- Documentation improvements (fix typos, improve clarity, add examples)

If you have questions, please check existing Issues and Pull Requests, or ask in relevant Issues.

---

Thank you again for contributing to the Warden project! 🎉
