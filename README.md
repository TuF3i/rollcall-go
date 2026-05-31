# CQUPT Rollcall Go

重庆邮电大学自动签到系统 — Go 重写版，增强长时间后台运行稳定性。

## 功能

- **自动定位签到** — 根据课表匹配教学楼坐标，自动提交位置签到
- **自动数字签到** — 检测到有同学已签到成功后，自动提交签到码完成签到
- **扫码/数字签到共享** — 通过 Center 服务器实时广播，一人扫码全体签到
- **课表感知轮询** — 仅在上课时段轮询，课前自动开启
- **Telegram 通知** — 关键事件（签到成功/失败、轮询状态）实时推送
- **指数退避重连** — WebSocket 断线自动恢复，1s → 60s 递增
- **日志去重** — 连续重复的日志自动合并，末尾标注重复次数
- **优雅关闭** — SIGINT/SIGTERM 信号安全退出

## 快速开始

### Docker（推荐）

```bash
# 1. 复制配置
cp .env.example .env
# 编辑 .env 填入你的账号信息

# 2. 启动
docker compose up -d

# 3. 查看日志
docker compose logs -f
```

### 使用 GHCR 镜像

```bash
docker run -d --restart unless-stopped \
  --env-file .env \
  ghcr.io/lulaide/rollcall-go/edge:latest
```

### 本地编译

```bash
go build -o edge ./cmd/edge

# 使用环境变量启动
export EDGE_USERNAME=your_id
export EDGE_PASSWORD=your_pwd
./edge
```

### Kubernetes (Helm)

```bash
# 1. 构建镜像
docker build -f Dockerfile.edge -t your-registry/rollcall-go:latest .
docker push your-registry/rollcall-go:latest

# 2. 创建配置
cat > my-values.yaml << 'EOF'
image:
  registry: your-registry
config:
  edgeUsername: "你的学号"
  edgePassword: "你的密码"
  curriculumApi: "https://cqupt.ishub.top/api/curriculum/学号/curriculum.json"
EOF

# 3. 一键部署
helm upgrade --install rollcall ./deployment/helm \
  -f my-values.yaml \
  --namespace rollcall --create-namespace

# 4. 查看日志
kubectl logs -n rollcall -l app.kubernetes.io/name=rollcall-go -f
```

## 环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `EDGE_USERNAME` | 是 | IDS 账号 |
| `EDGE_PASSWORD` | 是 | IDS 密码 |
| `EDGE_CURRICULUM_API` | 否 | 课表 API 地址 |
| `EDGE_CURRICULUM_PRE_MINUTES` | 否 | 课前提前轮询分钟数（默认 10） |
| `EDGE_HTTP_PORT` | 否 | HTTP 端口（留空禁用） |
| `EDGE_CENTER_SERVER_URL` | 否 | Center WebSocket 地址 |
| `EDGE_CENTER_SERVER_SECRET` | 否 | Center 认证密钥 |
| `EDGE_AUTO_LOCATION_CHECKIN` | 否 | 自动定位签到（默认 true） |
| `EDGE_AUTO_NUMBER_CHECKIN` | 否 | 自动数字签到（默认 true） |
| `TG_BOT_TOKEN` | 否 | Telegram Bot Token |
| `TG_CHAT_ID` | 否 | Telegram Chat ID |

## 架构

```
Edge Server                          Center Server
┌──────────────┐    WebSocket     ┌──────────────┐
│  LMS 轮询    │◄──────────────►│  广播签到码   │
│  自动签到    │   rollcall_*    │  状态管理     │
│  HTTP API    │                 │  密钥验证     │
│  TG 通知     │                 │              │
└──────────────┘                 └──────────────┘
```

- **Edge** — 运行在用户侧，轮询 LMS、执行签到、接收共享
- **Center** — 中转站，广播签到码实现多节点共享

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

获取当前共享签到的暂停状态。

**请求**：无参数

**响应**：
```json
{"pause": false}
```

### POST /pause_shared

设置共享签到的暂停/恢复。

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
{"data": "https://xxx/j?p=...!3~xxxxxxxxxxxx!4~..."}
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

## 相比 Python 版的改进

- goroutine + context 生命周期管理
- WebSocket 指数退避重连（1s → 60s）
- 每连接写锁保证并发安全
- panic recovery 防止单点崩溃
- 单二进制部署，内存占用更低
- Telegram 实时通知
- 日志去重，避免重复刷屏

## License

MIT
