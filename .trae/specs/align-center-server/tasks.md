# Tasks

- [x] Task 1: 移除 `UpdateQRData` 中的 `qrSuccessClients` 全局清除
  - 修改 `internal/center/state.go` 的 `UpdateQRData` 方法，删除 `s.qrSuccessClients = make(map[string]struct{})` 这一行
  - 验证：`go build ./...` 编译通过
