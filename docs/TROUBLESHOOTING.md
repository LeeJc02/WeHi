# 故障排查

## 1. AI Bot 不回复

检查顺序：

1. 后台 `/admin/ai` 中 Provider 是否启用
2. [config/ai.yaml](/Users/lee/GolandProjects/awesomeProject/10/config/ai.yaml) 的 `api_key` 是否为空
3. 后台 `/admin/audit` 是否有错误记录
4. 会话里是否落了系统兜底消息

## 2. 消息实时到了，但搜索搜不到

优先检查：

1. `realtime-service` 是否健康
2. `search.message.index` 事件是否被消费
3. 当前是否处于 `mock://` 搜索 fallback 模式
4. 后台首页触发“重建搜索索引”

## 3. 会话未读数不一致

检查顺序：

1. 后台 `/admin/conversations/:id`
2. 查看成员 `last_read_seq`
3. 查看最近 `message.read / message.persisted` 事件
4. 查看客户端最后 `sync cursor`

## 4. 重发后出现重复消息

检查顺序：

1. 是否使用相同 `client_msg_id`
2. 后台首页用 `client_msg_id` 快速定位消息
3. 检查定位结果是否始终指向同一 `message_id`

## 5. WebSocket 在线但没有收到事件

检查顺序：

1. `/ws` 建连是否成功
2. 是否收到 `auth.ok`
3. `realtime-service` 是否消费到对应 MQ 事件
4. `/admin/messages/:id` 查看消息旅程是否已进入 delivered/read

## 6. Trace 看不到

检查顺序：

1. Compose 中 `jaeger` 是否健康
2. `OTEL_EXPORTER=otlp` 是否生效
3. `OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318` 是否正确
4. 打开 `http://127.0.0.1:28686`
