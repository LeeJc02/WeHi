# WeHi

WeHi 是一个面向企业 IM 场景的分布式消息系统项目，支持单聊、群聊、消息撤回、已读回执、在线状态同步、历史消息查询、断线重连与离线补偿。项目采用多服务拆分的方式构建消息链路，通过 WebSocket 建立长连接，结合 Redis 提升消息效率和可扩展性，基于 Kafka 构建事件总线并实现断线后的增量追平和状态一致性。

项目技术栈采用 `Go + Gin + MySQL + Redis + Kafka + WebSocket + Elasticsearch + Docker + OpenTelemetry + Kubernetes`。整体上围绕认证服务、业务 API 服务、实时网关服务三类职责拆分，通过缓存、消息队列、全文搜索、链路追踪和容器化部署能力，形成一套完整的实时通信后端骨架。

## 一句话架构

- `auth-service`：负责注册、登录、刷新令牌、设备会话治理。
- `api-service`：负责好友、会话、消息、搜索、上传、后台管理接口。
- `realtime-service`：负责 WebSocket 建连、在线状态更新、事件消费与广播。
- `MySQL`：负责用户、关系、会话、消息、补偿事件、审计数据落库。
- `Redis`：负责会话、在线态、网关实例路由、resume 恢复信息等高频临时状态。
- `Kafka`：负责消息发送、送达、已读、索引更新、补偿通知等异步事件总线。
- `Elasticsearch`：负责消息全文搜索与会话名搜索。

## 这个项目真正体现的能力

- 它不是“发消息接口 + 查历史接口”的简单拼装，而是围绕消息生命周期做了拆分：
  - 发送
  - 持久化
  - 送达
  - 已读
  - 撤回
  - 离线补偿
- 它不是“单体里顺手加个 WebSocket”，而是把实时网关单独拆成了服务。
- 它不是“只有数据库”，而是明确把强一致状态和高频临时状态分层：
  - 强一致数据进 MySQL
  - 高频在线态、会话态、网关路由态进 Redis
- 它不是“只讲消息”，而是把搜索、观测、后台诊断、幂等、补偿都纳入了同一套工程体系。

## 核心链路

### 1. 登录与多端会话

客户端先走 `auth-service` 完成注册、登录和 refresh token 轮换。登录成功后，服务端把 session 状态写入 Redis，并把 access token 中携带的 `uid + sid` 用于后续 API 鉴权和 WebSocket 建连。

对应代码：

- `services/auth/main.go`
- `internal/app/auth/service.go`
- `internal/routes/auth.go`

### 2. 消息发送与实时投递

客户端调用 `POST /api/v1/conversations/:id/messages` 发送消息。`api-service` 先完成鉴权、会话成员校验、消息幂等和落库，再写入 Kafka 事件。`realtime-service` 消费事件后，把变更广播给在线连接；发送方则根据 `accepted / delivered / read` 回执更新本地消息状态。

对应代码：

- `internal/app/chat/message_service.go`
- `internal/app/repository/repository.go`
- `services/realtime/main.go`
- `internal/realtime/hub.go`
- `frontend/lib/chat-store.tsx`

### 3. 断线重连与离线补偿

在线链路只解决“此刻在线的人”，不能解决“刚刚掉线的人”。因此项目额外实现了 `sync_events + cursor` 的补偿路径：所有关键状态变更除了走 Kafka 广播，还会按用户写入 `sync_events`；客户端重连后先取 `current cursor`，再拉 `cursor` 之后的增量事件，按事件类型把本地状态追平。

对应代码：

- `internal/app/chat/sync.go`
- `internal/app/sync/service.go`
- `internal/app/repository/repository.go`
- `frontend/lib/chat-store.tsx`

## 技术亮点

### 1. 消息生命周期做成了完整闭环

- 发送前做成员校验
- 发送时做 `client_msg_id` 幂等
- 落库后发持久化事件
- 在线时走实时广播
- 离线时走补偿追平
- 已读后反向回写发送方状态

### 2. 状态拆层比较清楚

- `MySQL` 存最终事实
- `Redis` 存在线态、会话态、网关实例路由态、resume 恢复态
- `sync_events` 存用户视角的增量变更

### 3. 不是只做功能，也做了工程闭环

- migration
- smoke
- health check
- metrics
- tracing
- 后台诊断页

## 目录速览

```text
.
├── cmd/                 # migrate / reindex
├── deploy/              # 部署与编排
├── docs/                # 项目文档、面试材料、简历映射
├── frontend/            # Next.js 前端
├── internal/            # 后端核心代码
├── migrations/          # SQL migrations
├── pkg/contracts        # DTO / Query / Event / Response
├── scripts/             # 启停与 smoke
└── services/            # auth / api / realtime 入口
```

## 部署与启动

本地开发：

```bash
make start
```

刷新：

```bash
make refresh
```

停止：

```bash
make stop
```

可选复检：

```bash
make smoke
```

生产环境：

- 服务通过 Docker 镜像交付
- 基于 Kubernetes 完成服务注册发现与多服务部署
- 结合健康检查、可观测性与弹性扩缩能力增强系统可维护性

## 镜像发布与拉取运行

如果要让其他机器直接 `docker pull` 后运行，推荐使用预构建镜像而不是在目标机器本地构建。仓库已经补好了两部分：

- GitHub Actions 发布工作流：[.github/workflows/release-images.yml](/Users/lee/GolandProjects/awesomeProject/WeHi/.github/workflows/release-images.yml)
- 基于镜像运行的 Compose 文件：[deploy/compose/docker-compose.release.yml](/Users/lee/GolandProjects/awesomeProject/WeHi/deploy/compose/docker-compose.release.yml)

默认发布目标是 `GHCR`，镜像前缀规则为：

```text
ghcr.io/<github-owner>/<repo-lowercase>
```

发布内容包括：

- `migrate`
- `auth-service`
- `api-service`
- `realtime-service`
- `frontend`

触发方式：

- 推送到 `main`：发布 `latest` 和 `sha-<commit>`
- 推送标签 `v*`：额外发布对应版本标签
- 手动触发：可自定义 `image_tag`

远端机器拉取运行：

```bash
export IMAGE_PREFIX=ghcr.io/<owner>/<repo>
export IMAGE_TAG=latest
make release-up
```

停止：

```bash
make release-down
```

如果不使用 `make`，也可以直接：

```bash
export IMAGE_PREFIX=ghcr.io/<owner>/<repo>
export IMAGE_TAG=latest
docker compose -f deploy/compose/docker-compose.release.yml pull
docker compose -f deploy/compose/docker-compose.release.yml up -d --wait
```

这条路径的价值是：

- 目标机器不需要 Go / Node 构建环境
- 目标机器只做 `pull + run`
- 多服务架构保持不变，更适合面试展示和后续扩展

默认端口：

- Auth：`http://127.0.0.1:28081`
- API：`http://127.0.0.1:28082`
- Realtime：`ws://127.0.0.1:28083/ws`
- Frontend：`http://127.0.0.1:25173`
- Jaeger：`http://127.0.0.1:28686`

## 文档索引

- `docs/01-系统架构与代码地图.md`
- `docs/02-消息链路与实时通信.md`
- `docs/03-离线补偿与状态一致性.md`
- `docs/04-存储、缓存与搜索设计.md`
- `docs/05-工程化、部署与可观测性.md`
- `docs/06-简历表述与项目亮点.md`
- `docs/07-面试追问与深挖清单.md`
