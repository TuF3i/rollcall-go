# Edge Server 服务发现与 API 调用示例

通过 etcd 发现指定用户的 Edge Server 实例，并调用其 API。

支持 **HTTP** 和 **RPC** 两种模式，自动识别协议。

## 使用前准备

确保 Edge Server 已配置 etcd 并正常运行：

```env
EDGE_ETCD_ENDPOINTS=127.0.0.1:2379
EDGE_ETCD_PREFIX=/rollcall
```

## 运行

```bash
cd example

go run main.go --etcd 127.0.0.1:2379 --username 1697666
```

### 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--etcd` | `127.0.0.1:2379` | etcd 地址，多个用逗号分隔 |
| `--prefix` | `/rollcall` | etcd 键前缀，需与 Edge Server 配置一致 |
| `--username` | (必填) | 要查找的 `edge_username` |

## 流程

```
etcd 注册数据:
  /rollcall/rollcall_edge/<client_id>
  → {"service":"rollcall_edge","edge_username":"1697666","addr":"10.0.0.5:8080",...}

示例程序:
  1. 连接 etcd，按前缀查询所有 Edge 节点
  2. 遍历匹配 edge_username
  3. 根据注册地址自动选择调用方式:
     - ip:port 或 http://ip:port → HTTP API
     - rpc://ip:port               → 提示 Kitex 客户端用法
  4. 调用 API
```

## HTTP 模式输出示例

```
找到 1 个匹配 1697666 的节点:
  http://10.0.0.5:8080

--- HTTP API 调用 ---
[health] status=ok  client_id=abc12345-...
[rollcalls] 当前活跃签到: 2 个
  · [qr] 高等数学A — absent (ID: 123456)
  · [number] 大学英语 — absent (ID: 123457)
[pause_shared] {"pause":false}
```

## RPC 模式输出示例

```
找到 1 个匹配 1697666 的节点:
  rpc://10.0.0.5:8888

--- RPC API 调用 ---
该实例以 RPC 模式运行，地址为 10.0.0.5:8888
需使用 Kitex 客户端通过 Thrift 协议调用
接口定义: api/kitex/edge.thrift

import "github.com/Auto-CQUPT-Plan/rollcall-go/internal/rpc/kitex_gen/edge/edgeservice"

clients, _ := edgeservice.NewClient("rollcall_edge", client.WithHostPorts("10.0.0.5:8888"))
health, _ := clients.Health(context.Background())
rollcalls, _ := clients.GetRollcalls(context.Background())
```
