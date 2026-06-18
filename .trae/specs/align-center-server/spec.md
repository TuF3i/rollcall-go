# Center Server 对齐 Python 版 Spec

## Why
对比 Python 版 `center_server.py` 与 Go 版 `internal/center/`，发现 Go 的 `SharedState.UpdateQRData` 在更新新 QR 时全局清除了 `qrSuccessClients`，而 Python 不这样做。Python 采用按客户端粒度单独清除的策略。需要对齐到 Python 的行为。

## What Changes
- 移除 `SharedState.UpdateQRData` 中 `qrSuccessClients = make(map[string]struct{})` 的重置逻辑

## Impact
- Affected specs: `center-server-state`
- Affected code: `internal/center/state.go` (`UpdateQRData` 方法)

## MODIFIED Requirements

### Requirement: QR Success Client Tracking
系统在接收到新 QR 数据并更新 `latestQRData` 时，**不应当**全局清除 `qrSuccessClients` 集合。已成功签到的客户端状态应保持，直到该客户端在下一次 `rollcall_tasks` 消息中重新报告需要 QR 签到时，才由 `SetQRNeedingClient` 单独清除该客户端的 success 状态。

#### Scenario: 新 QR 更新不清除 success 列表
- **WHEN** Center 收到一条新的有效 QR 数据并成功更新 `latestQRData`
- **THEN** `qrSuccessClients` 集合保持不变，不做清空操作
