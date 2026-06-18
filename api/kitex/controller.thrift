namespace go controller

// Edge 信息
struct EdgeInfo {
    1: string client_id,          // Edge 客户端 ID
    2: string username,           // Edge 用户名 (edge_username)
    3: string addr,               // Edge 注册地址
    4: string registered_at,      // 注册时间
}

// Edge 列表请求
struct ListEdgesRequest {
}

// Edge 列表响应
struct ListEdgesResponse {
    1: list<EdgeInfo> edges,
}

// 通用消息响应
struct MessageResponse {
    1: string message,
}

// 带错误消息的响应
struct ErrorResponse {
    1: string error,
}

// 签到任务
struct Rollcall {
    1: i64 rollcall_id,
    2: string source,
    3: string status,
    4: string course_title,
    5: string rollcall_time,
}

// 暂停状态
struct PauseState {
    1: bool pause,
}

// 设置暂停请求
struct SetPauseRequest {
    1: string edge_username,
    2: bool pause,
}

// 二维码签到请求
struct QRCheckinRequest {
    1: string edge_username,
    2: i64 rollcall_id,
    3: string data,
}

// 数字码签到请求
struct NumberCheckinRequest {
    1: string edge_username,
    2: i64 rollcall_id,
    3: string number_code,
}

// 定位签到请求
struct LocationCheckinRequest {
    1: string edge_username,
    2: i64 rollcall_id,
    3: double lat,
    4: double lon,
}

// 批量二维码签到请求
struct BatchQRCheckinRequest {
    1: string edge_username,
    2: string data,
}

// 签到结果
struct CheckinResult {
    1: i64 rollcall_id,
    2: string status,
    3: optional string error,
}

// 批量签到响应
struct BatchQRCheckinResponse {
    1: list<CheckinResult> results,
}

// 服务异常
exception ServiceException {
    1: i32 status_code,
    2: string error,
}

service ControllerService {
    /// 健康检查
    MessageResponse Health() throws (1: ServiceException ex),

    /// 列出所有在线 Edge
    ListEdgesResponse ListEdges() throws (1: ServiceException ex),

    /// 获取指定 Edge 的签到列表
    list<Rollcall> GetRollcalls(1: string edge_username) throws (1: ServiceException ex),

    /// 获取指定 Edge 的暂停状态
    PauseState GetPauseState(1: string edge_username) throws (1: ServiceException ex),

    /// 设置指定 Edge 的暂停状态
    PauseState SetPause(1: SetPauseRequest req) throws (1: ServiceException ex),

    /// 对指定 Edge 执行 QR 签到
    MessageResponse QRCheckin(1: QRCheckinRequest req) throws (1: ServiceException ex),

    /// 对指定 Edge 执行数字签到
    MessageResponse NumberCheckin(1: NumberCheckinRequest req) throws (1: ServiceException ex),

    /// 对指定 Edge 执行定位签到
    MessageResponse LocationCheckin(1: LocationCheckinRequest req) throws (1: ServiceException ex),

    /// 对指定 Edge 执行批量 QR 签到
    BatchQRCheckinResponse BatchQRCheckin(1: BatchQRCheckinRequest req) throws (1: ServiceException ex),
}
