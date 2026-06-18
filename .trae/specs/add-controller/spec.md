# Controller 聚合控制中心 Spec

## Why
当前项目中 Edge Server 各自独立运行并注册到 etcd，但没有统一的控制平面来发现和管理这些 Edge 实例。需要一个 Controller 端，可以从 etcd 发现所有在线 Edge，并按 `edge_username` 定向调用 RPC/HTTP 方法执行签到命令。

## What Changes
- **新增** `cmd/controller/main.go` — Controller 入口
- **新增** `internal/controller/` 包 — controller 核心逻辑
  - `discovery.go` — 从 etcd 发现和追踪在线 Edge
  - `server.go` — HTTP API 路由（chi router）
  - `handler.go` — HTTP 请求处理（代理调用 Edge）
- **新增** `api/kitex/controller.thrift` — ControllerService IDL（RPC 模式用）
- **新增** `internal/controller/rpc_handler.go` — Kitex RPC 服务端实现
- **新增** `internal/controller/rpc_server.go` — Kitex RPC 服务器
- **修改** `internal/config/config.go` — 新增 controller 相关配置项
- **新增** `Dockerfile.controller` — Controller Docker 镜像构建

## Impact
- Affected specs: `controller-discovery`, `controller-api`
- Affected code: `internal/config/config.go`, 新增 `internal/controller/`, `cmd/controller/`

## ADDED Requirements

### Requirement: Edge 实例发现
Controller SHALL 通过 etcd watch 机制实时发现所有注册的 Edge 实例，并维护一个本地缓存映射 `edge_username → ServiceInfo`。

#### Scenario: 发现新 Edge 上线
- **WHEN** 一个 Edge Server 注册到 etcd（写入 key `{prefix}/rollcall_edge/{clientID}`）
- **THEN** Controller 自动发现该 Edge，解析 `edge_username` 和 `addr`，加入本地缓存

#### Scenario: 检测 Edge 下线
- **WHEN** 一个 Edge 的 etcd 租约过期或主动注销
- **THEN** Controller 从本地缓存中移除该 Edge

### Requirement: 列出在线 Edge
Controller SHALL 提供 API 返回当前所有在线 Edge 的列表，包含 `client_id`、`edge_username`、`addr`、`registered_at`。

#### Scenario: GET /edges
- **WHEN** 收到 `GET /edges` 请求
- **THEN** 返回当前所有在线 Edge 的信息列表

### Requirement: 按 username 定向执行命令
Controller SHALL 根据 `edge_username` 查找对应 Edge，先验证其在线，再通过该 Edge 暴露的 API（HTTP 或 Kitex RPC，与 Edge 模式一致）调用方法，并将结果返回。

#### Scenario: 调用存在的 Edge
- **WHEN** 收到 `POST /edges/{username}/health` 请求
- **THEN** 先在本地缓存中查找该 username 对应的 Edge 地址，若存在则向该 Edge 发起 Health 调用并返回结果；若不存在则返回 404

#### Scenario: 调用不存在的 Edge
- **WHEN** 收到 `POST /edges/{username}/rollcalls` 但该 username 不在线
- **THEN** 返回 `404` 错误信息 `{"error": "edge not found"}`

### Requirement: 与 Edge 相同的 API 模式
Controller 对外的 API 暴露方式 SHALL 与 Edge Server 保持一致：通过 `CONTROLLER_API_MODE` 配置选择 `http`（默认，chi router）或 `rpc`（Kitex gRPC）。调用 Edge 时也使用对应的通信方式。

### Requirement: Controller API 端点
Controller SHALL 暴露以下 HTTP 端点：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | Controller 自身健康检查 |
| GET | `/edges` | 列出所有在线 Edge |
| POST | `/edges/{username}/health` | 对指定 Edge 执行 Health |
| POST | `/edges/{username}/rollcalls` | 获取指定 Edge 的签到列表 |
| POST | `/edges/{username}/pause` | 获取/设置暂停状态（Body 含 `pause` 字段时设置） |
| POST | `/edges/{username}/qr-checkin` | 对指定 Edge 执行 QR 签到 |
| POST | `/edges/{username}/number-checkin` | 对指定 Edge 执行数字签到 |
| POST | `/edges/{username}/location-checkin` | 对指定 Edge 执行定位签到 |
| POST | `/edges/{username}/batch-qr-checkin` | 对指定 Edge 执行批量 QR 签到 |

### Requirement: 配置项
Controller SHALL 复用项目现有配置结构，新增以下配置：

| 变量 | 必填 | 说明 |
|------|------|------|
| `CONTROLLER_API_MODE` | 否 | API 模式：`http`（默认）或 `rpc` |
| `CONTROLLER_HTTP_PORT` | 否 | HTTP 端口（默认 8082） |
| `CONTROLLER_RPC_PORT` | 否 | RPC 端口（默认 8889） |
| `EDGE_ETCD_ENDPOINTS` | 是 | etcd 连接地址（已有，复用） |
| `EDGE_ETCD_PREFIX` | 否 | etcd 前缀（已有，复用，默认 `/rollcall`） |

## REMOVED Requirements
无
