# 项目状态报告

## 当前交付结果

项目目前已经是一套可运行的企业化聊天系统预演工程，包含：

- `auth-service`：负责 JWT access token、refresh token 轮换、Redis 会话管理、单端退出和全端退出。
- `api-service`：负责好友申请、单聊/群聊、群成员管理、置顶、消息历史、已读回执和搜索接口。
- `realtime-service`：负责 WebSocket 在线连接、RabbitMQ 事件广播、Elasticsearch 异步索引。
- `presence-service`：已以内嵌服务模块形式存在，负责在线状态写入和查询。
- `frontend`：负责微信桌面风格工作台，包括会话列表、聊天主区域、搜索和调试抽屉。

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
- 在本地无完整中间件时，支持 `MySQL LIKE` 搜索 fallback 和 `Redis Pub/Sub` 消息 fallback。
- 显式 SQL migration 和索引重建命令。
- 微信风格前端工作台和本地 smoke 脚本。

## 还需要继续改进的地方

- 当前已经有基础错误码体系，并已补基础错误码文档，但还缺全接口覆盖清单和对外 API 文档同步。
- 当前 DTO 已拆成 request / response / query / event，但还能继续补响应模型规范和分页基类。
- `realtime-service` 里事件消费和网关职责还耦合在一个进程里，可以继续拆分。
- 当前 fallback 方案适合本地演示，但正式环境应优先只走 RabbitMQ + Elasticsearch 正式链路。
- 自动化测试目前偏 smoke test，还缺针对权限边界、异常场景和幂等场景的更细粒度测试。
- 已补基础 controller 测试和平台层单测，但仍缺更完整的业务集成测试矩阵。
- `image/file` 类型消息目前只在协议和数据结构层面预留，尚未接入上传/对象存储能力。
- 在线状态和已读状态目前可用，但还没有抽成独立的 presence 子系统。
- 在线状态已收口为独立 `presence service`，但后续仍可继续细化成更完整的在线/离线事件体系。

## 建议的下一步工程化方向

1. 补 response DTO、query DTO 和错误码文档清单。
2. 增加 service 单测、controller 集成测试和更完整的端到端测试。
3. 补结构化日志、监控指标和链路追踪。
4. 将 RabbitMQ / Elasticsearch fallback 继续抽象为接口，减少 service 层条件分支。
5. 如果你想在面试里更强调 MVC，可以把 `repository` 重命名为 `model` 或 `persistence`，并明确说明使用了 repository pattern。
