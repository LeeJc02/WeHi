# 部署说明

## 本地推荐方式

使用 Docker Compose：

```bash
make start
```

启动后默认地址：

- Frontend: `http://127.0.0.1:25173`
- Auth API: `http://127.0.0.1:28081`
- Chat API: `http://127.0.0.1:28082`
- Realtime WS: `ws://127.0.0.1:28083/ws`
- Jaeger UI: `http://127.0.0.1:28686`

## 启动内容

- `mysql`
- `redis`
- `jaeger`
- `migrate`
- `auth-service`
- `api-service`
- `realtime-service`
- `frontend`

## 核心环境变量

- `MYSQL_DSN`
- `REDIS_ADDR`
- `RABBITMQ_URL`
- `ELASTICSEARCH_URL`
- `JWT_SECRET`
- `AI_CONFIG_PATH`
- `OTEL_EXPORTER`
- `OTEL_EXPORTER_OTLP_ENDPOINT`

## AI 相关配置

AI 统一读取：

- [config/ai.yaml](/Users/lee/GolandProjects/awesomeProject/10/config/ai.yaml)

默认设置：

- 默认 Provider：`zhipu`
- 默认模型：`glm-4.5-air`
- Prompt：空

如果未配置有效 API Key，Bot 仍会走完整异步链路，但会回复兜底消息。

## 管理后台

默认管理员种子账号：

- 用户名：`root`
- 密码：`123456`

首次登录后会被要求修改密码。

## 校验命令

Docker Compose 模式：

```bash
make smoke
```

该 smoke 会验证：

- 注册 / 登录 / 刷新 token
- 好友申请与通过
- 单聊消息发送与实时事件
- 已读回执
- 搜索
- 设备会话
- `AI Bot` 是否自动创建、是否置顶、是否异步回复

## Trace 查看

Compose 默认把 OTel trace 发到 Jaeger。

打开：

- `http://127.0.0.1:28686`

查看服务：

- `auth-service`
- `api-service`
- `realtime-service`
