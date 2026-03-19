# 架构说明

## 服务划分

- `auth-service`
  - 用户注册、登录、刷新令牌
  - Redis 会话管理
  - 设备会话与退出其他设备
- `api-service`
  - 好友、会话、群聊、消息、搜索、AI Bot、后台 API
  - sync event 生产
  - 审计、配置、诊断接口
- `realtime-service`
  - WebSocket 连接管理
  - 在线状态写入
  - 事件广播到在线端

## 核心依赖

- `MySQL`
  - 用户、好友、会话、消息、会话设置、审计日志
- `Redis`
  - 会话、在线状态
  - 本地演示模式下的 Pub/Sub 事件总线 fallback
- `RabbitMQ`
  - 正式事件总线抽象位
- `Elasticsearch`
  - 消息与会话搜索
  - 当前 Compose 默认使用 MySQL fallback
- `Jaeger / OpenTelemetry`
  - HTTP、DB、Redis、WebSocket tracing

## 代码分层

- `internal/controllers`
  - 参数绑定、鉴权入口、HTTP 响应
- `internal/app`
  - 业务服务层
- `internal/app/repository`
  - 数据访问和聚合查询
- `internal/platform`
  - DB、Redis、RabbitMQ、HTTP 中间件、Tracing
- `pkg/contracts`
  - DTO、事件协议、后台诊断模型

## 消息主链路

1. 客户端调用 `POST /conversations/:id/messages`
2. `api-service` 进行鉴权、幂等校验、消息落库
3. 写入 `message.accepted / message.persisted / message.delivered / message.read` 事件
4. `realtime-service` 消费事件并广播给在线端
5. 离线端通过 `sync cursor + sync_events` 追平

## AI Bot 链路

1. 用户拉取好友或会话列表时，后端幂等确保 `AI Bot` 用户、好友关系和私聊会话存在
2. 用户给 Bot 发消息后，消息先正常落库并进入实时链路
3. 后端异步调用 AI Provider
4. 成功时以 Bot 身份回一条普通消息
5. 失败时回一条系统兜底消息
6. AI 调用写入 `ai_audit_logs`

## 诊断与观测

- 后台监控页
  - 服务健康、HTTP 请求量、错误数、平均延迟、WS 连接数
- 消息旅程页
  - 查看消息从 accepted 到 delivered/read 的服务端阶段
- 会话一致性页
  - 查看成员已读游标、未读数、当前 sync cursor、在线状态
- Jaeger
  - 查看 HTTP、DB、Redis、WebSocket span
