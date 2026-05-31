namespace go edge

// ============================================================
// 基础数据结构
// ============================================================

/// 签到任务
struct Rollcall {
    1: i64 rollcall_id,       // 签到任务 ID
    2: string source,         // 签到类型：qr / number / radar
    3: string status,         // 状态：absent（未签）/ on_call（已签）
    4: string course_title,   // 课程名称
    5: string rollcall_time,  // 签到时间（ISO8601 UTC）
}

/// 健康检查响应
struct HealthResponse {
    1: string status,    // 固定 "ok"
    2: string client_id, // 客户端 ID
}

/// 暂停状态
struct PauseState {
    1: bool pause, // 是否暂停
}

/// 设置暂停请求
struct SetPauseRequest {
    1: bool pause,
}

/// 设置暂停响应
struct SetPauseResponse {
    1: string message,
    2: bool pause,
}

/// 二维码签到请求
struct QRCheckinRequest {
    1: i64 rollcall_id,
    2: string data, // 二维码原始数据（支持 URL 格式或纯 hex 格式）
}

/// 数字码签到请求
struct NumberCheckinRequest {
    1: i64 rollcall_id,
    2: string number_code, // 签到数字码
}

/// 定位签到请求
struct LocationCheckinRequest {
    1: i64 rollcall_id,
    2: double lat, // 纬度
    3: double lon, // 经度
}

/// 单次签到结果
struct CheckinResult {
    1: i64 rollcall_id,
    2: string status,  // success / failed
    3: optional string error, // 失败时的错误码
}

/// 批量二维码签到请求
struct BatchQRCheckinRequest {
    1: string data, // 二维码原始数据
}

/// 批量二维码签到响应
struct BatchQRCheckinResponse {
    1: list<CheckinResult> results,
}

/// 通用操作响应
struct OperationResponse {
    1: string message,
}

// ============================================================
// 异常定义
// ============================================================

/// 服务异常
exception ServiceException {
    1: i32 status_code,  // HTTP 状态码
    2: string error,     // 错误信息
}

// ============================================================
// Edge 服务
// ============================================================

service EdgeService {
    /// 服务健康检查
    HealthResponse Health() throws (1: ServiceException ex),

    /// 获取当前所有活跃签到列表
    list<Rollcall> GetRollcalls() throws (1: ServiceException ex),

    /// 获取当前暂停状态
    PauseState GetPauseState() throws (1: ServiceException ex),

    /// 设置暂停/恢复
    SetPauseResponse SetPause(1: SetPauseRequest req) throws (1: ServiceException ex),

    /// 二维码签到（指定签到任务）
    OperationResponse QRCheckin(1: QRCheckinRequest req) throws (1: ServiceException ex),

    /// 数字码签到（指定签到任务）
    OperationResponse NumberCheckin(1: NumberCheckinRequest req) throws (1: ServiceException ex),

    /// 定位签到（指定签到任务）
    OperationResponse LocationCheckin(1: LocationCheckinRequest req) throws (1: ServiceException ex),

    /// 批量二维码签到（对所有未签到的 QR 签到任务执行签到）
    BatchQRCheckinResponse BatchQRCheckin(1: BatchQRCheckinRequest req) throws (1: ServiceException ex),
}
