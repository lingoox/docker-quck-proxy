# Docker-Quck-Proxy

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

要发布一个新的版本：

1. **编辑 VERSION 文件**：`echo "1.2.3" > VERSION`
2. **提交并推送到 main**：`git add VERSION && git commit -m "Release v1.2.3" && git push`
3. **等待 GitHub Actions 自动构建**：工作流会自动编译所有平台的二进制
4. **创建 Draft Release**：去 Releases 页面点击 "Draft a new release"，tag 选择 `v1.2.3`，填写描述后保存
5. **发布**：GitHub Actions 会自动检测到 draft release 创建，并将构建好的附件关联到该 release
6. **点击 Publish**：你的 Release 就包含了所有平台的下载包！

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
# 以 Linux x86_64 为例
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
├── docs/                # 文档
├── Dockerfile           # 多阶段构建
├── docker-compose.yml   # 一键部署
├── .github/workflows/   # CI/CD 工作流（多平台自动构建）
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

项目使用 [GitHub Actions](https://github.com/lingoox/docker-quck-proxy/actions) 进行自动化构建：

- **触发事件**：push VERSION tag、PR/pull_request/main、release draft 创建
- **构建产物**：7 个平台的二进制压缩包
- **发布方式**：PR/Push → Artifact（30天保留）；Release Draft → Release 附件自动关联

## License

MIT
