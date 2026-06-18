# CQUPT Rollcall Go

重庆邮电大学自动签到系统 — Go 重写版，增强长时间后台运行稳定性。

## 功能

- **自动定位签到** — 根据课表匹配教学楼坐标，自动提交位置签到
- **自动数字签到** — 检测到有同学已签到成功后，自动提交签到码完成签到
- **扫码/数字签到共享** — 通过 Center 服务器实时广播，一人扫码全体签到
- **课表感知轮询** — 仅在上课时段轮询，课前自动开启，轮询间隔可配置
- **Controller 聚合控制中心** — 通过 etcd 发现所有在线 Edge，按用户名定向执行签到命令
- **Kitex RPC 服务** — 支持 HTTP 和 Kitex Thrift RPC 双模式，通过环境变量切换
- **Etcd 服务注册** — 注册服务地址到 etcd，方便其他服务发现和调用
- **暂停模式** — 一键暂停所有自动签到和共享接收，手动签到不受影响
- **Telegram 通知** — 关键事件（签到成功/失败、轮询状态）实时推送
- **指数退避重连** — WebSocket 断线自动恢复，1s → 60s 递增
- **彩色结构化日志** — 基于 `fatih/color` 的可读性增强日志输出
- **日志去重** — 连续重复的日志行合并输出并计数，避免刷屏
- **优雅关闭** — SIGINT/SIGTERM 信号安全退出

## 快速开始

### Docker（推荐）

```bash
# 1. 复制配置
cp deployment/docker-compose/.env.example deployment/docker-compose/.env
# 编辑 .env 填入你的账号信息

# 2. 启动
docker compose -f deployment/docker-compose/docker-compose.yml up -d

# 3. 查看日志
docker compose -f deployment/docker-compose/docker-compose.yml logs -f
```

> 构建镜像时若想使用国内加速，请使用 `Dockerfile.edge.cn` 进行构建

### 本地编译

```bash
go build -o edge ./cmd/edge
go build -o controller ./cmd/controller

# 启动 Edge Server
export EDGE_USERNAME=your_id
export EDGE_PASSWORD=your_pwd
./edge

# 启动 Controller（需先启动 etcd）
export EDGE_ETCD_ENDPOINTS=localhost:2379
./controller
```

### Kubernetes (Helm)

支持多实例部署，每个实例使用独立的配置。

```bash
# 1. 构建镜像
docker build -f Dockerfile.edge -t your-registry/rollcall-go:latest .
docker build -f Dockerfile.controller -t your-registry/rollcall-go:latest .
docker push your-registry/rollcall-go:latest

# 2. 创建配置（多实例）
cat > my-values.yaml << 'EOF'
image:
  registry: your-registry

controller:
  etcdEndpoints: "etcd:2379"

edge:
  instances:
    - name: "student-a"
      config:
        edgeUsername: "2024001"
        edgePassword: "pass123"
        curriculumApi: "https://cqupt.ishub.top/api/curriculum/2024001/curriculum.json"
        centerServerUrl: "wss://cqupt.ishub.top/api/rollcall/ws"
        etcdEndpoints: "etcd:2379"

    - name: "student-b"
      config:
        edgeUsername: "2024002"
        edgePassword: "pass456"
        centerServerUrl: "wss://cqupt.ishub.top/api/rollcall/ws"
        etcdEndpoints: "etcd:2379"
EOF

# 3. 一键部署
helm upgrade --install rollcall ./deployment/helm \
  -f my-values.yaml \
  --namespace rollcall --create-namespace

# 4. 查看日志
kubectl logs -n rollcall -l app.kubernetes.io/name=rollcall-go -f
```

#### 增删实例

直接修改 `my-values.yaml` 中 `edge.instances` 列表，执行 `helm upgrade` 即可。新实例会自动创建对应的 Deployment、Service、ConfigMap、Secret、PVC；移除的实例会自动清理。

## 环境变量

### Edge Server

| 变量 | 必填 | 说明 |
|------|------|------|
| `EDGE_USERNAME` | 是 | IDS 账号 |
| `EDGE_PASSWORD` | 是 | IDS 密码 |
| `EDGE_CURRICULUM_API` | 否 | 课表 API 地址 |
| `EDGE_CURRICULUM_PRE_MINUTES` | 否 | 课前提前轮询分钟数（默认 10） |
| `EDGE_HTTP_PORT` | 否 | HTTP 端口（默认 8080，设为空禁用） |
| `EDGE_CENTER_SERVER_URL` | 否 | Center WebSocket 地址 |
| `EDGE_CENTER_SERVER_SECRET` | 否 | Center 认证密钥 |
| `EDGE_AUTO_LOCATION_CHECKIN` | 否 | 自动定位签到（默认 true） |
| `EDGE_AUTO_NUMBER_CHECKIN` | 否 | 自动数字签到（默认 true） |
| `EDGE_ETCD_ENDPOINTS` | 否 | Etcd 地址，用于服务注册（留空禁用） |
| `EDGE_ETCD_PREFIX` | 否 | Etcd 键前缀（默认 /rollcall） |
| `EDGE_API_MODE` | 否 | API 服务模式：`http` 或 `rpc`（默认 http） |
| `EDGE_RPC_PORT` | 否 | Kitex RPC 端口（默认 8888） |
| `POLL_DELAY` | 否 | 轮询间隔秒数（默认 30） |
| `TG_BOT_TOKEN` | 否 | Telegram Bot Token |
| `TG_CHAT_ID` | 否 | Telegram Chat ID |

### Controller

| 变量 | 必填 | 说明 |
|------|------|------|
| `EDGE_ETCD_ENDPOINTS` | 是 | Etcd 地址，用于发现 Edge 实例 |
| `EDGE_ETCD_PREFIX` | 否 | Etcd 键前缀（默认 /rollcall） |
| `CONTROLLER_HTTP_PORT` | 否 | HTTP 端口（默认 8082） |

## 架构

```
Edge Server                          Center Server
┌──────────────┐    WebSocket     ┌──────────────┐
│  LMS 轮询    │◄──────────────►│  广播签到码   │
│  自动签到    │   rollcall_*    │  状态管理     │
│  HTTP/RPC API│                 │  密钥验证     │
│  TG 通知     │                 └──────────────┘
│  Etcd 注册   │
└──────┬───────┘
       │ 注册 /rollcall/rollcall_edge/<id>
       │
   ┌───▼──────────┐     ┌──────────────────────┐
   │     Etcd     │◄───►│  Controller 控制中心  │
   │  服务注册中心  │     │  etcd watch 发现 Edge │
   └──────────────┘     │  按用户名定向调用      │
                        │  HTTP JSON API        │
                        └──────────────────────┘
```

- **Edge** — 运行在用户侧，轮询 LMS、执行签到、接收共享、注册到 etcd
- **Center** — 中转站，广播签到码实现多节点共享
- **Etcd** — 服务注册中心，记录所有在线 Edge 的地址和用户名
- **Controller** — 聚合控制中心，从 etcd 实时发现 Edge，提供统一管理接口

## Edge Server HTTP API

Edge Server 提供 HTTP 接口，可用于手动触发签到、查看状态等。基础地址为 `http://<host>:<port>`（默认端口 `8080`）。

### GET /health

服务健康检查。

**请求**：无参数

**响应**：
```json
{
    "status": "ok",
    "client_id": "abc12345-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}
```

### GET /rollcalls

获取当前所有活跃签到列表（从 LMS 实时拉取）。

**请求**：无参数

**响应**：
```json
[
    {
        "rollcall_id": 123456,
        "source": "qr",
        "status": "absent",
        "course_title": "高等数学A",
        "rollcall_time": "2026-05-30T10:00:00Z"
    }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| rollcall_id | int | 签到任务 ID |
| source | string | 签到类型：`qr` / `number` / `radar` |
| status | string | 状态：`absent`（未签） / `on_call`（已签） |
| course_title | string | 课程名称 |
| rollcall_time | string | 签到时间（ISO8601 UTC） |

### GET /pause_shared

获取当前暂停状态。

**请求**：无参数

**响应**：
```json
{"pause": false}
```

### POST /pause_shared

暂停/恢复。暂停后同时停止：接收共享签到、本地自动定位签到、本地自动数字签到，但通过 HTTP API 手动签到不受影响。

**请求**：
```json
{"pause": true}
```

**响应**：
```json
{"message": "success", "pause": true}
```

### POST /rollcall/{rollcallID}/qr

对指定签到任务进行二维码签到。

**路径参数**：`rollcallID` — 签到任务 ID

**请求**：
```json
{"data": "...!3~xxxxxxxxxxxx!4~..."}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| data | string | 二维码原始数据，支持 URL 格式或纯 hex 格式 |

**成功**（200）：`{"message": "success"}`

**失败**（400）：
```json
{"error": "invalid or expired QR data"}
```

| 错误码 | 说明 |
|--------|------|
| invalid or expired QR data | 二维码无效或过期 |
| ERR_ROLLCALL_NOT_FOUND | 签到任务不存在 |
| ERR_NOT_STARTED | 签到尚未开始 |
| ERR_ALREADY_ANSWERED | 已签到过 |

### POST /rollcall/{rollcallID}/number

对指定签到任务进行数字码签到。

**路径参数**：`rollcallID` — 签到任务 ID

**请求**：
```json
{"numberCode": "1234"}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| numberCode | string / int | 签到数字码 |

**成功**（200）：`{"message": "success"}`

**失败**（400）：
```json
{"error": "ERR_INVALID_NUMBER"}
```

| 错误码 | 说明 |
|--------|------|
| ERR_INVALID_NUMBER | 数字码错误 |
| ERR_ROLLCALL_NOT_FOUND | 签到任务不存在 |
| ERR_NOT_STARTED | 签到尚未开始 |
| ERR_ALREADY_ANSWERED | 已签到过 |

### POST /rollcall/{rollcallID}/location

对指定签到任务进行定位签到（雷达签到）。

**路径参数**：`rollcallID` — 签到任务 ID

**请求**：
```json
{"lat": 29.5382, "lon": 106.6050}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| lat | float | 纬度 |
| lon | float | 经度 |

**成功**（200）：`{"message": "success"}`

**失败**（400）：
```json
{"error": "ERR_OUT_OF_RANGE"}
```

### POST /rollcallqr

批量二维码签到。对所有未签到的 QR 类型签到任务自动执行签到。

**请求**：
```json
{"data": "https://xxx/j?p=...!3~xxxxxxxxxxxx!4~..."}
```

**响应**：
```json
{
    "results": [
        {"rollcall_id": 123456, "status": "success"},
        {"rollcall_id": 123458, "status": "failed", "error": "ERR_ROLLCALL_NOT_FOUND"}
    ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| results[].rollcall_id | int | 签到任务 ID |
| results[].status | string | `success` / `failed` |
| results[].error | string | 失败时的错误码 |

### 通用说明

- 所有接口均使用 `Content-Type: application/json`
- 签到成功后会触发**立即轮询**以获取最新状态，并**同步到 Center 服务器**（如有配置）
- 二维码 `data` 支持两种格式：纯 hex 字符串（32 位），或 `/j?p=...!3~<hex>!4~...` URL 格式（自动提取 hex 部分）
- 可通过设置 `http_port: null` 或环境变量 `EDGE_HTTP_PORT=` 完全禁用手动签到 HTTP 接口

## Controller API

Controller 提供聚合控制 API，通过 etcd 发现所有在线 Edge，按用户名定向执行命令。基础地址为 `http://<host>:<port>`（默认端口 `8082`）。

### GET /health

Controller 自身健康检查。

**请求**：无参数

**响应**：
```json
{
    "status": "ok",
    "edges_count": 3
}
```

### GET /edges

获取当前所有在线 Edge 列表。

**请求**：无参数

**响应**：
```json
[
    {
        "service": "rollcall_edge",
        "client_id": "abc12345-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
        "edge_username": "2024001",
        "addr": "http://10.0.0.5:8080",
        "registered_at": "2026-05-28T10:00:00Z"
    }
]
```

### POST /edges/{username}/health

对指定 Edge 进行健康检查。

**路径参数**：`username` — Edge 用户名（学号）

**成功**（200）：转发 Edge 的 health 响应

**失败**（404）：
```json
{"error": "edge not found"}
```

### POST /edges/{username}/rollcalls

获取指定 Edge 的签到列表。

**路径参数**：`username` — Edge 用户名（学号）

**成功**（200）：转发 Edge 的签到列表

**失败**（404）：
```json
{"error": "edge not found"}
```

### POST /edges/{username}/pause

获取或设置指定 Edge 的暂停状态。

**路径参数**：`username` — Edge 用户名（学号）

**获取**（不带 body）：返回当前暂停状态

**设置**（带 body）：
```json
{"pause": true}
```

### POST /edges/{username}/qr-checkin

对指定 Edge 执行二维码签到。

```json
{"rollcall_id": 123456, "data": "...!3~xxxxxxxxxxxx!4~..."}
```

### POST /edges/{username}/number-checkin

对指定 Edge 执行数字码签到。

```json
{"rollcall_id": 123456, "number_code": "1234"}
```

### POST /edges/{username}/location-checkin

对指定 Edge 执行定位签到。

```json
{"rollcall_id": 123456, "lat": 29.5382, "lon": 106.6050}
```

### POST /edges/{username}/batch-qr-checkin

对指定 Edge 执行批量二维码签到。

```json
{"data": "https://xxx/j?p=...!3~xxxxxxxxxxxx!4~..."}
```

## API 模式

Edge Server 支持两种 API 服务模式，通过环境变量 `EDGE_API_MODE` 切换：

### HTTP 模式（默认）

```env
EDGE_API_MODE=http
EDGE_HTTP_PORT=8080
```

提供标准的 RESTful HTTP JSON 接口，文档见上方。

### Kitex RPC 模式

```env
EDGE_API_MODE=rpc
EDGE_RPC_PORT=8888
```

使用 Kitex Thrift 协议提供 RPC 服务，接口定义在 `api/kitex/edge.thrift`：

```thrift
service EdgeService {
    HealthResponse       Health()
    list<Rollcall>       GetRollcalls()
    PauseState           GetPauseState()
    SetPauseResponse     SetPause(SetPauseRequest)
    OperationResponse    QRCheckin(QRCheckinRequest)
    OperationResponse    NumberCheckin(NumberCheckinRequest)
    OperationResponse    LocationCheckin(LocationCheckinRequest)
    BatchQRCheckinResponse BatchQRCheckin(BatchQRCheckinRequest)
}
```

RPC 模式下 etcd 注册地址格式为 `rpc://ip:port`，便于客户端区分协议。

Controller 也预留了 RPC 模式（`api/kitex/controller.thrift`），当前默认使用 HTTP。

## Etcd 服务注册

配置 `EDGE_ETCD_ENDPOINTS` 后，Edge 启动时自动注册到 etcd：

- **键路径**：`/<prefix>/rollcall_edge/<client_id>`
- **注册信息**：JSON 格式，含 `service`、`client_id`、`edge_username`、`addr`、`registered_at`
- **地址格式**：HTTP 模式 `ip:port`，RPC 模式 `rpc://ip:port`
- **租约保活**：10 秒 TTL，goroutine 持续续期
- **自动注销**：程序退出时撤销租约，键自动清理

Controller 通过 etcd watch 机制实时发现所有在线 Edge，无需轮询。

### 服务发现（Go SDK）

```go
import "github.com/Auto-CQUPT-Plan/rollcall-go/internal/registry"

// 按用户名查找 Edge 节点地址
addrs, err := registry.DiscoverByUsername("127.0.0.1:2379", "/rollcall", "1697666")
// addrs[0] == "http://10.0.0.5:8080"

// 获取所有注册节点
services, err := registry.DiscoverAll("127.0.0.1:2379", "/rollcall")
for _, s := range services {
    fmt.Println(s.Username, s.Addr)
}
```

## 相比 Python 版的改进

- goroutine + context 生命周期管理
- WebSocket 指数退避重连（1s → 60s）
- 每连接写锁保证并发安全
- panic recovery 防止单点崩溃
- 单二进制部署，内存占用更低
- 日志去重，避免重复刷屏
- Telegram 实时通知
- 彩色结构化日志（基于 `fatih/color`）
- Etcd 服务注册，支持多实例管理
- Controller 聚合控制中心，按用户名定向管理
- Helm 多实例部署，增删实例只需编辑配置文件

## License

MIT
