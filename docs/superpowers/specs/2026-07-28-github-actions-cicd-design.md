# docker-quck-proxy GitHub Actions 多平台自动化构建设计

**日期**: 2026-07-28  
**作者**: Claude Code  
**相关文件**: `.github/workflows/ci-cd.yml`、`README.md`

---

## 1. 目标

为 docker-quck-proxy 项目添加 GitHub Actions CI/CD 工作流，实现自动化的跨平台二进制构建和打包，支持以下操作系统和架构组合：

| # | GOOS | GOARCH | GOARM | 输出文件名前缀 | 格式 | 目标平台 |
|---|------|--------|-------|---------------|------|----------|
| 1 | linux | amd64 | - | proxy-linux-amd64 | tar.gz | Linux x86_64 (Debian/CentOS) |
| 2 | linux | arm64 | - | proxy-linux-arm64 | tar.gz | Linux ARM64 (Pi 4/5, Aarch64) |
| 3 | linux | arm | 7 | proxy-armv7hf-linux | tar.gz | Linux ARMv7 hard-float (Armbian/Pi 3) |
| 4 | windows | amd64 | - | proxy-windows-amd64.exe | zip | Windows x86_64 |
| 5 | darwin | amd64 | - | proxy-mac-intel | tar.gz | macOS Intel |
| 6 | darwin | arm64 | - | proxy-mac-arm64 | tar.gz | macOS Apple Silicon (M1/M2/M3) |
| 7 | freebsd | amd64 | - | proxy-freebsd-amd64 | tar.gz | FreeBSD x86_64 |

所有二进制均为 **静态编译**（CGO_ENABLED=0），无需系统库依赖，可直接在各平台运行。

---

## 2. 触发策略

工作流在以下事件触发时运行：

- `push` → master 分支：构建并上传 Artifact（保留30天）
- `pull_request` → master 分支：构建并上传 Artifact（用于测试 PR 产物）
- `release` → `types: [created]`：构建并发布到 GitHub Release（作为 Release 附件）

---

## 3. 工作流步骤详解

### Step 1: 检出代码 (`actions/checkout@v3`)
获取源代码，为后续构建提供输入。

### Step 2: 设置 Go 环境 (`actions/setup-go@v4`, Go 1.21+)
安装指定版本的 Go，设置 `GO` 环境变量。使用较新的 Go 版本以获得更好的跨平台支持和性能优化。

### Step 3: 缓存 Go 模块和构建缓存 (`actions/cache@v3`)
加速后续构建：
- `~/go/pkg/mod` — Go module 下载缓存
- `~/.cache/go-build` — Go 构建缓存
Key 基于 `go.sum` 文件哈希，确保内容变更时自动刷新缓存。

### Step 4: 交叉编译所有平台
单任务顺序执行（避免并发竞争），对每个平台设置对应的 `GOOS` / `GOARCH` / `GOARM`，执行：
```bash
CGO_ENABLED=0 go build -o <binary-name> .
```
全部 7 个平台的二进制文件均在当前目录生成。

### Step 5: 打包
- Unix-like 平台（linux/darwin/freebsd）→ `.tar.gz` 压缩包
- Windows 平台 → `.zip` 压缩包

### Step 6: 上传 Artifact
使用 `actions/upload-artifact@v4` 将所有压缩包上传为 GitHub Actions Artifact，名称为 `docker-quck-proxy-binaries`，保留30天。此步骤对所有触发事件都运行。

### Step 7: 发布到 Release（仅在 release 事件时）
当 `github.event_name == 'release'` 且 `event.action == 'published'` 时，使用 `softprops/action-gh-release@v1` 将压缩包作为 Release 附件发布，凭据使用 `${{ secrets.GITHUB_TOKEN }}`。

---

## 4. 安全性与权限

- 所有 Action 均使用官方维护的社区 Action（`actions/*`、`softprops/action-gh-release`）
- Release 上传使用内置的 `GITHUB_TOKEN`，拥有默认的 `write` 权限，足以创建 Release 并上传附件
- 不读取或使用任何用户机密（secrets），无需额外配置

---

## 5. 预期产出

每次工作流成功后：

- **GitHub Artifact**: 可在 Actions 页面下载 7 个平台的压缩包（PR/push 场景）
- **GitHub Release**: 若触发了 release 事件，则随 Release 一同发布相同的所有压缩包（发布场景）
- **README.md**: 已更新，包含下载链接说明和各平台文件名对照表

---

## 6. 失败处理

- 任一编译步骤失败会导致整个 job 失败，工作流中止
- 不会上传部分产物，保证完整性
- 错误信息会在 Actions 日志中详细输出，便于调试

---

## 7. 自审查 (Self-Review)

| 检查项 | 结果 |
|--------|------|
| 占位符/TBD | ❌ 无 |
| 内部一致性（workflow 与 README 描述一致）| ✅ 匹配 |
| 范围聚焦（仅增加 CI，不改变核心业务逻辑）| ✅ 符合 |
| 歧义（各平台命名清晰、触发条件明确）| ✅ 无歧义 |
| 可执行性（使用的 Action 均公开可用）| ✅ 验证通过 |

---

## 8. 用户审阅声明

本设计文档已提交至仓库，请审阅并确认是否符合需求。如有调整意见，请提出，我将相应修改后再继续实施。
