# 项目状态报告

## 当前交付结果

项目目前已经是一套可运行的企业化聊天系统预演工程，包含：

- `auth-service`：负责 JWT access token、refresh token 轮换、Redis 会话管理、单端退出和全端退出。
- `api-service`：负责好友申请、单聊/群聊、群成员管理、置顶、消息历史、已读回执和搜索接口。
- `realtime-service`：负责 WebSocket 在线连接、RabbitMQ 事件广播、Elasticsearch 异步索引。
- `presence-service`：已以内嵌服务模块形式存在，负责在线状态写入和查询。
- `frontend`：负责微信桌面风格工作台，包括会话列表、聊天主区域、搜索和调试抽屉。
- `jaeger`：在 Compose 模式下默认启用，用于查看 trace。

## 当前后端架构

后端已经调整为更清晰的分层结构：

- `internal/controllers`：处理请求绑定、参数校验、响应返回等 HTTP 控制器逻辑。
- `internal/app/*`：处理业务规则，作为 service 层。
- `internal/app/chat` 已按上下文拆为 `user`、`friend`、`conversation`、`message`、`search` 五类服务。
- `internal/app/repository`：处理数据访问、聚合查询和 GORM/SQL 交互。
- `internal/routes`：负责路由注册和中间件装配。
- `pkg/contracts`：负责请求 DTO、响应 DTO 和事件协议。
- `pkg/contracts` 现已细拆为 `envelope`、`requests`、`responses`、`queries`、`events`。
- `internal/platform/apperr`：负责统一业务错误码和 HTTP 状态映射。
- `internal/app/repository/models.go`：负责持久化模型定义。
- `services/*/main.go`：仅作为启动入口和依赖注入层。

如果你要按 MVC 来讲，这一版已经可以这样表达：

- `Controller`：`internal/controllers`
- `Model`：`internal/app/repository/models.go` 以及 repository 层
- `Service`：`internal/app/*`
- `Routes`：`internal/routes`

## 已实现能力

- 基于 bcrypt 的用户注册和登录。
- 基于 JWT access token + refresh token 的鉴权体系。
- 基于 `error_code` 的统一业务错误响应格式。
- 独立的请求 DTO 和事件协议定义，减少控制器匿名结构体。
- 搜索和消息列表查询参数已经收敛到 query DTO。
- 控制器已按资源拆分为 `user/friend/conversation/message/search`。
- 基于 Redis 的会话存储和多端会话管理。
- 好友申请流程：发起、同意、拒绝。
- 单聊复用和群聊创建。
- 群成员添加、移除、退群、群主转让。
- 消息发送、按序号分页拉取历史消息、已读回执推进。
- 会话置顶和聚合后的会话列表查询。
- RabbitMQ 用于实时广播和搜索索引事件。
- WebSocket 用于消息、已读回执、好友申请通知。
- Elasticsearch 用于消息全文检索和会话名检索。
- OpenTelemetry + Jaeger 用于 HTTP、DB、Redis、WebSocket tracing。
- RabbitMQ publish/consume 已透传 trace context，消息链路可跨 MQ 继续串联。
- 在本地无完整中间件时，支持 `MySQL LIKE` 搜索 fallback 和 `Redis Pub/Sub` 消息 fallback。
- 显式 SQL migration 和索引重建命令。
- 微信风格前端工作台和本地 smoke 脚本。
- smoke 已覆盖 `AI Bot` 自动建联、置顶和异步回复校验。
- 管理后台已覆盖 AI 配置、多模型切换、监控总览、消息旅程、会话一致性、AI 审计和补偿队列。
- AI Bot 已支持异步回复、持久化重试作业、批量重试、清理已完成/已耗尽作业和审计联动。
- 图片、文件、引用回复、撤回、会话草稿、公告、免打扰和设备会话管理均已落地。

## 还需要继续改进的地方

- 自动化测试已经覆盖 controller、runtime smoke 和多端/重连关键路径，但仍缺更厚的异常场景与专门 integration 测试矩阵。
- `realtime-service` 里事件消费和网关职责还耦合在一个进程里，后续仍可继续拆分。
- 当前 fallback 方案适合本地演示，正式环境仍建议收敛到 RabbitMQ + Elasticsearch 正式链路。
- AI 补偿已具备作业队列、批量重试、清理和监控指标，但还可以继续演进为独立 worker / 死信队列模型。
- 在线状态已收口为独立 `presence service`，后续仍可继续细化成更完整的在线/离线事件体系。

## 建议的下一步工程化方向

1. 增加更厚的 integration / chaos 场景测试，重点覆盖补偿失败、重试耗尽和搜索漂移修复。
2. 继续细化 AI 补偿体系，例如独立 worker、死信队列、失败告警和更丰富的后台运维动作。
3. 将 RabbitMQ / Elasticsearch fallback 继续抽象为接口，减少 service 层条件分支。
4. 如果你想在面试里更强调 MVC，可以把 `repository` 重命名为 `model` 或 `persistence`，并明确说明使用了 repository pattern。
