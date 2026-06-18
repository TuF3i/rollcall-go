# Tasks

- [x] Task 1: 新增 Controller 配置项
  - 在 `internal/config/config.go` 的 `Config` 结构体中新增 `ControllerAPIMode`、`ControllerHTTPPort`、`ControllerRPCPort` 字段
  - 在 `applyEnvOverrides()` 中添加对应环境变量解析（`CONTROLLER_API_MODE`、`CONTROLLER_HTTP_PORT`、`CONTROLLER_RPC_PORT`）
  - 验证：`go build ./...` 编译通过

- [x] Task 2: 实现 Edge 发现模块
  - 新增 `internal/controller/discovery.go`
  - 实现 `Discovery` 结构体：连接 etcd、watch `{prefix}/rollcall_edge/` 前缀
  - 维护 `sync.RWMutex` 保护的 `map[string]*registry.ServiceInfo`（key: `edge_username`，value: ServiceInfo），支持多个 clientID 对应同一 username
  - 实现 `ListEdges()` 返回所有在线 Edge
  - 实现 `Resolve(username string) (*registry.ServiceInfo, bool)` 查找指定 username 的 Edge
  - 验证：`go build ./...` 编译通过

- [x] Task 3: 实现 Controller HTTP Server
  - 新增 `internal/controller/server.go` — chi router 设置和端点注册
  - 新增 `internal/controller/handler.go` — 各端点的请求处理逻辑：
    - `GET /health` → 返回 Controller 自身状态
    - `GET /edges` → 返回 discovery 中所有在线 Edge 列表
    - `POST /edges/{username}/health` → 代理调用 Edge HTTP API
    - `POST /edges/{username}/rollcalls` → 代理调用 Edge
    - `POST /edges/{username}/pause` → 获取/设置暂停（GET/POST 合并）
    - `POST /edges/{username}/qr-checkin` → 代理 QR 签到
    - `POST /edges/{username}/number-checkin` → 代理数字签到
    - `POST /edges/{username}/location-checkin` → 代理定位签到
    - `POST /edges/{username}/batch-qr-checkin` → 代理批量 QR 签到
  - 每个端点先检查 username 是否存在，不存在返回 404
  - 向 Edge 发起 HTTP 请求时使用 `http.Client`，超时 30s
  - 验证：`go build ./...` 编译通过

- [x] Task 4: 实现 Controller 入口
  - 新增 `cmd/controller/main.go`
  - 参考 `cmd/edge/main.go` 的结构：日志初始化 → 配置加载 → 创建 Discovery → HTTP/RPC 服务器 → 优雅关闭
  - 使用 `config.Cfg.ControllerAPIMode` 选择 HTTP 或 RPC 模式
  - 验证：`go build ./cmd/controller/` 编译通过

- [x] Task 5: 新增 Dockerfile 和 Helm 支持
  - 新增 `Dockerfile.controller`
  - 在 `deployment/helm/values.yaml` 中新增 controller 配置段
  - 在 `deployment/helm/templates/` 中新增 `deployment-controller.yaml` 和 `service-controller.yaml`
  - 验证：`helm template ./deployment/helm/` 渲染通过

- [x] Task 6: 新增 Controller Thrift IDL（RPC 模式预留）
  - 新增 `api/kitex/controller.thrift`，定义 `ControllerService`
  - 方法包括：Health、ListEdges、GetRollcalls、GetPauseState、SetPause、QRCheckin、NumberCheckin、LocationCheckin、BatchQRCheckin（每个都需要 `edge_username` 参数）
  - 注意：需要运行 `kitex -module` 生成代码后才能启用 RPC 模式

# Task Dependencies
- Task 2 依赖 Task 1（需要配置中的 etcd 地址）
- Task 3 依赖 Task 2（handler 需要调用 discovery）
- Task 4 依赖 Task 2 和 Task 3
- Task 5 依赖 Task 4
- Task 6 独立，可并行
