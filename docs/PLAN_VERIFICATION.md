# 计划表核验

## 核验结论

- `Phase 1` 基础清理与多服务骨架：已完成，并通过本机 fallback 运行验证。
- `Phase 2` 鉴权与会话治理：已完成，并通过 smoke 验证。
- `Phase 3` 数据治理与查询性能：已完成基础目标，并通过本机运行验证。
- `Phase 4` 实时通信与异步消息链路：已完成，并通过 WebSocket smoke 验证。
- `Phase 5` 企业 IM 基础业务补齐：已完成当前计划范围。
- `Phase 6` Elasticsearch 搜索：已完成，并通过本机 fallback 和真实索引接口验证。
- `Phase 7` 前端微信化改版：已完成当前计划范围。

## 逐项核对

### 1. 基础清理与多服务骨架

- [x] 删除 SQLite 依赖和旧单体主路径
- [x] 建立 `services/api`、`services/auth`、`services/realtime`
- [x] 建立 `cmd`、`migrations`、`deploy/compose`
- [x] 增加 Docker Compose 编排
- [x] 增加本机一键联调脚本
- [x] 显式 SQL migration 替代 `AutoMigrate`

验证：

- `go test ./cmd/... ./internal/... ./pkg/... ./services/...`
- `make local-up`
- `make local-smoke`
- `make local-down`

### 2. 鉴权与会话治理

- [x] JWT access token
- [x] refresh token 轮换
- [x] Redis 会话存储
- [x] 单端退出 / 全端退出
- [x] 统一鉴权中间件

验证：

- `frontend/scripts/api-smoke.mjs`
- `scripts/runtime_smoke.mjs`

### 3. 数据治理与查询性能

- [x] migration 建表
- [x] 消息序号和游标分页
- [x] 会话列表聚合查询
- [x] 好友申请 / 会话 / 消息核心索引
- [x] `client_msg_id` 字段与消息状态字段

说明：

- 当前已达到计划中的“基础企业化”目标。
- 更激进的性能压测和 SQL 基准测试暂未单独补报表，但核心查询链路已稳定。

### 4. 实时通信与异步链路

- [x] WebSocket 连接入口
- [x] RabbitMQ 事件消费
- [x] Redis 在线状态
- [x] 消息推送、已读回执、好友申请通知
- [x] 本机 fallback：Redis Pub/Sub

验证：

- `make local-smoke`

### 5. 业务完整度

- [x] 好友申请制
- [x] 群聊创建
- [x] 群成员管理
- [x] 群主转让
- [x] 会话置顶
- [x] 文本消息
- [x] 系统消息 / 图片 / 文件类型字段预留
- [x] 已读回执

说明：

- 按原计划假设，本轮不做上传中心，因此 `image/file` 为协议和数据结构预留，视为符合原计划边界。

### 6. Elasticsearch 搜索

- [x] 搜索索引结构
- [x] 搜索事件消费
- [x] 重建索引命令
- [x] 搜索接口
- [x] 前端搜索入口
- [x] MySQL fallback 搜索

验证：

- `make local-smoke`

### 7. 前端微信化改版

- [x] 左侧导航 / 会话列表
- [x] 中间聊天主区域
- [x] 搜索入口
- [x] 调试抽屉
- [x] REST + WebSocket 主链路

验证：

- `cd frontend && npm run lint`
- `cd frontend && npm run build`

## 额外完成项

- [x] `error_code` 统一错误码体系
- [x] `contracts` 拆分为 request/response/query/event
- [x] controller 按资源拆分
- [x] controller / platform / helper 测试
- [x] CI 工作流
- [x] `X-Request-Id`
- [x] 结构化访问日志
- [x] Prometheus `/metrics`
- [x] 独立 `presence service`

## 当前唯一未完成的最终验收项

- [ ] 默认 Docker Compose 主路径的完整容器拉起验收
- [ ] 当前机器上的 fallback 运行链路重放

原因：

- 代码和 Compose 配置本身已通过 `docker compose config`。
- Docker daemon 已可访问。
- 但在实际 `docker compose up -d --build` 过程中，外部镜像拉取命中 Docker Hub 未登录限流 `429 Too Many Requests`，属于环境侧限制，不是仓库代码错误。
- 当前机器上的 fallback 重放也受到本地 MySQL 环境影响：
  - `3307` 上已有 `mysqld` 进程，但当前账号无法通过 TCP 访问。
  - `3306` 上也有 `mysqld`，但 `root` 无密码访问被拒绝。
  - Homebrew `mysqld 9.4.0` 在 `--initialize-insecure` 时会直接崩溃，因此不能依赖“现场起一个全新的本地 MySQL 实例”完成重放。

本轮已完成的修复：

- `scripts/local_up.sh` 已支持 `MYSQL_ROOT_PASSWORD`
- `scripts/local-env.sh` 已改为按 MySQL 端口隔离 `.runtime/mysql-*` 数据目录，避免旧数据目录污染新实例

如果环境解除该限制，建议执行：

```bash
docker compose -f deploy/compose/docker-compose.yml -f deploy/compose/docker-compose.runtime.yml up -d --build
curl http://127.0.0.1:28082/metrics
cd frontend && npm run smoke -- http://127.0.0.1:28081 http://127.0.0.1:28082
docker compose -f deploy/compose/docker-compose.yml -f deploy/compose/docker-compose.runtime.yml down -v
```
