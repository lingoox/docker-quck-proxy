# Docker-Quck-Proxy

轻量级的 Docker Hub 镜像加速代理服务，纯 Go 编写，容器化部署。

## 工作原理

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

### 环境变量配置

项目通过环境变量覆盖默认值，可创建 `.env` 文件配置：

```bash
# .env.example
UPSTREAM=https://registry-1.docker.io
LISTEN_ADDR=:5000
LOG_DIR=./logs
HOST_PORT=5000
```

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `UPSTREAM` | `https://registry-1.docker.io` | 上游 Docker Registry |
| `LISTEN_ADDR` | `:5000` | 监听地址 |
| `LOG_DIR` | (空) | 日志目录（设为空则输出到 stdout） |
| `HOST_PORT` | `5000` | Docker Compose 宿主端口 |

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

# 或使用自定义 .env 文件（项目根目录）
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
├── main.go              # 入口点
├── cmd/                 # CLI 子命令（预留 cobra 扩展）
├── internal/
│   ├── config/          # 配置解析（预留 YAML 文件支持）
│   └── proxy/
│       ├── config.go    # Config struct & defaults
│       └── handler.go   # 反向代理 + manifest 重写
├── docs/                # 文档
├── Dockerfile           # 多阶段构建
├── docker-compose.yml   # 一键部署
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

## License

MIT
