# Docker-Quck-Proxy

[![Release](https://img.shields.io/github/v/release/lingoox/docker-quck-proxy)](https://github.com/lingoox/docker-quck-proxy/releases)
[![Build](https://github.com/lingoox/docker-quck-proxy/actions/workflows/release.yml/badge.svg)](https://github.com/lingoox/docker-quck-proxy/actions)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

轻量级的 Docker Hub 镜像加速代理服务，纯 Go 编写，容器化部署。

## 📥 二进制下载

预编译的二进制包可通过以下方式获取：

### GitHub Release（最新发布）

每次更新版本号后，GitHub Actions 会自动构建所有平台的压缩包并作为 Release 附件发布：

| 平台 | 文件名 |
|------|--------|
| Linux x86_64 | `proxy-linux-amd64.tar.gz` |
| Linux ARM64 | `proxy-linux-arm64.tar.gz` |
| Linux ARMv7 (Armbian) | `proxy-armv7hf-linux.tar.gz` |
| Windows x86_64 | `proxy-windows-amd64.zip` |
| macOS Intel | `proxy-mac-intel.tar.gz` |
| macOS ARM64 (M1/M2) | `proxy-mac-arm64.tar.gz` |
| FreeBSD x86_64 | `proxy-freebsd-amd64.tar.gz` |

点击项目右上角的 **[Releases](https://github.com/lingoox/docker-quck-proxy/releases)** 页面查看最新版本。

### 本地自行构建

如果想自己编译，确保 Go 环境已安装（Go 1.21+）：

```bash
# 编译所有平台
./build-all.sh  # （未来可添加此脚本）

# 或使用单个平台
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o proxy-linux-amd64 .
```

### GitHub Actions Artifact

在 PR 或推送 VERSION 文件时，GitHub Actions 会自动上传构建产物（Artifact），可在 Actions 页面下载。

## 🚀 发布新版本

要发布一个新的版本，只需两步：

```bash
# 1. 更新版本号（例如发布 1.0.0）
echo "1.0.0" > VERSION

# 2. 提交并推送到 main
git add VERSION && git commit -m "Release v1.0.0" && git push
```

然后 GitHub Actions 会自动完成一切：
- ✅ 编译 7 个平台的二进制
- ✅ 打包为 `.tar.gz` / `.zip`
- ✅ 自动创建 Release `v1.0.0`
- ✅ 上传所有平台包到 Release Asset

去 **[Releases](https://github.com/lingoox/docker-quck-proxy/releases)** 页面就能看到下载链接！🎉

---

```
Docker Client ──► :5000 (our proxy) ──► registry-1.docker.io (upstream)
                     │
                     ├─ 重写 manifest 中的 blob URL → 指向自己
                     ├─ 透明传递 Bearer token 认证
                     └─ 透传所有非 manifest 请求（blob/health/auth）
```

## 快速开始

### 直接运行

```bash
# 使用默认配置
go run .

# 或指定上游和监听地址
UPSTREAM=https://registry-1.docker.io LISTEN_ADDR=:5000 go run .
```

> 访问 `http://localhost:5000/health` 可检查服务状态。

### 命令行参数（推荐）

编译后可直接使用命令行标志配置：

```bash
# 示例：指定上游、监听端口、日志目录和启用日志
./proxy \
  --upstream https://registry-1.docker.io \
  --port :5000 \          # 可简写为 -p :5000（注意：使用 flag.String("p", ...）定义短形式）
  --logdir ./logs \
  --log-enabled true

# 查看所有可用参数
./proxy --help
```

**参数对照表：**

| 长形式 | 默认值 | 说明 |
|--------|--------|------|
| `--upstream` | (空，用 default) | 上游 Docker Registry |
| `--port` | (空，用 default :5000) | 监听地址 |
| `--logdir` | (空) | 日志目录（空则输出到 stdout） |
| `--http-proxy` | (空) | **上游** HTTP 代理（经代理访问 Docker Hub 时使用） |
| `--https-proxy` | (空) | **上游** HTTPS 代理 |
| `--log-enabled` | `false` | 是否启用日志输出 |

⚠️ 注意：`HTTP_PROXY`/`HTTPS_PROXY` 是 **docker-quck-proxy 自身访问上游时的代理**，与下游 Docker Client 的配置无关。下游仍然直接使用 `localhost:5000` 作为镜像源。

也可以通过环境变量设置（命令行参数的优先级高于环境变量）：
- `UPSTREAM` → `--upstream`
- `LISTEN_ADDR` → `--port`
- `LOG_DIR` → `--logdir`
- `HTTP_PROXY` / `HTTPS_PROXY` → 上游代理
- `LOG_ENABLED` → `--log-enabled`（支持 `true`/`false` / `1`/`0` / `y`/`n`）

### 下载并安装二进制（Linux/FreeBSD/macOS）

```bash
# 查看最新版本号，替换下方 vX.X.X 为实际版本
# 以 Linux x86_64 为例：
curl -L https://github.com/lingoox/docker-quck-proxy/releases/download/vX.X.X/proxy-linux-amd64.tar.gz | tar xz
chmod +x proxy-linux-amd64
sudo mv proxy-linux-amd64 /usr/local/bin/proxy

# 启动（使用命令行参数）
proxy \
  --upstream https://registry-1.docker.io \
  --port :5000 \
  --log-enabled false
```

### Windows 下载

下载 `proxy-windows-amd64.zip`，解压后重命名为 `proxy.exe`，放入系统 PATH 即可使用。通过命令行参数配置方式同 Linux。

### Docker 部署

```bash
docker build -t docker-quck-proxy .
docker run -d -p 5000:5000 \
  --env UPSTREAM=https://registry-1.docker.io \
  --name proxy \
  docker-quck-proxy
```

### Docker Compose

```bash
# 使用默认配置
docker compose up -d

# 或通过 environment 设置
HOST_PORT=5050 UPSTREAM=https://registry-1.docker.io docker compose up -d
```

### 📋 Linux 管理脚本（推荐）

项目提供了一个全功能的管理脚本，支持一键安装、配置和维护：

```bash
# 下载管理脚本
curl -O https://raw.githubusercontent.com/lingoox/docker-quck-proxy/main/scripts/docker-quck-proxy-manager.sh

# 赋予执行权限
chmod +x docker-quck-proxy-manager.sh

# 运行管理面板（会自动检测并安装合适平台的版本）
./docker-quck-proxy-manager.sh
```

**脚本功能菜单：**

| 选项 | 功能 | 说明 |
|------|------|------|
| `1` | 📥 安装 | 自动检测平台，下载最新版本并安装为 systemd 服务 |
| `2` | 🔄 重启 | 重启代理服务 |
| `3` | 📊 查看状态 | 显示服务运行状态、当前配置、最近日志 |
| `4` | ⏹ 停止 | 停止代理服务 |
| `5` | ⚙️ 切换配置 | 交互式修改上游地址、监听端口、日志开关等参数 |
| `6` | 📋 查看配置 | 显示当前配置文件内容 |
| `7` | 🗑 卸载 | 完全移除二进制、配置、日志和 systemd 服务 |
| `0` | 🚪 退出 | 退出管理面板 |

> ⚠️ 脚本会自动提权到 root（systemctl 需要管理员权限）

### Docker 客户端配置

#### 方法一：daemon.json（推荐）

编辑 `/etc/docker/daemon.json`：

```json
{
  "registry-mirrors": ["http://localhost:5000"]
}
```

重启 Docker：`systemctl restart docker`

#### 方法二：手动 pull

```bash
docker pull localhost:5000/library/alpine
# 或
docker tag alpine localhost:5000/library/alpine
docker pull localhost:5000/library/alpine
```

## 项目结构

```
├── main.go              # 入口点（支持命令行参数）
├── cmd/                 # CLI 子命令（预留 cobra 扩展）
├── internal/
│   ├── config/          # Config struct & defaults
│   └── proxy/
│       ├── config.go    # 配置解析
│       └── handler.go   # 反向代理 + manifest 重写
├── scripts/
│   └── docker-quck-proxy-manager.sh  # 🔧 Linux 管理脚本（安装/配置/启停）
├── docs/                # 文档
├── Dockerfile           # 多阶段构建
├── docker-compose.yml   # 一键部署
├── VERSION              # 📌 版本号文件
├── .github/workflows/   # CI/CD 工作流（多平台自动构建 + Release）
└── README.md
```

## 支持的 API

| 端点 | 状态 | 说明 |
|------|------|------|
| `GET /v2/` | ✅ | Registry 健康检查 |
| `GET /v2/<name>/manifests/<tag>` | ✅ | Manifest 获取 + URL 重写 |
| `GET /v2/<name>/blobs/<digest>` | ✅ | Blob 下载（直通透传） |
| `GET /v2/<name>/tags/list` | ✅ | 标签列表 |
| `POST /v2/<name>/blobs/uploads` | ⏸️ | Blob 上传（push，后续支持） |
| `GET /tokens?service=...` | ✅ | Token 认证（已修复） |
| `GET /health` | ✅ | 健康检查端点 |

## 持续集成

项目使用 [GitHub Actions](https://github.com/lingoox/docker-quck-proxy/actions) 进行自动化构建和发布：

- **触发机制**：推送 `VERSION` 文件修改 ✅
- **构建平台**：7 个平台（Linux/Windows/macOS/FreeBSD/Armbian）
- **输出形式**：`.tar.gz`（Unix）和 `.zip`（Windows）压缩包
- **发布方式**：自动创建 Release 并上传所有资产 🚀
- **版本管理**：通过 `VERSION` 文件集中控制版本号

## License

MIT
