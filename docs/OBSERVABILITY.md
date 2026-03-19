# 可观测性说明

## 已覆盖能力

- Prometheus 指标
  - HTTP 请求总量
  - HTTP 延迟
  - WebSocket 在线连接数
- OpenTelemetry tracing
  - Gin HTTP 请求
  - GORM / MySQL
  - Redis
  - WebSocket 连接生命周期
- 结构化日志
  - `request_id`
  - `trace_id`
  - `span_id`
- 后台观测页
  - 监控总览
  - AI 审计
  - 消息旅程
  - 会话一致性

## 查看方式

### 1. Prometheus 风格指标

每个 Go 服务暴露：

- `/metrics`

### 2. 后台监控

- `/admin/monitor`

可查看：

- 服务健康状态
- 总请求量
- 4xx / 5xx
- 平均延迟
- WebSocket 连接数

### 3. Jaeger Trace

Compose 模式默认启用 Jaeger：

- `http://127.0.0.1:28686`

建议重点看：

- 登录请求
- 消息发送请求
- WebSocket 建连
- Redis 在线状态写入

### 4. 后台诊断

- `/admin/messages/:id`
  - 单条消息旅程
- `/admin/conversations/:id`
  - 会话一致性与事件时间线
- `/admin/audit`
  - AI 调用耗时、错误和摘要

## 当前限制

- 还没有把 RabbitMQ publish/consume 做成 OTel span
- 搜索 fallback 走 MySQL 时，没有独立 search service trace
- trace 目前主要用于排障与演示，还没有做按 trace_id 的后台全文检索
