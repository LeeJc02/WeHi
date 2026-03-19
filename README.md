# Chat Enterprise Workspace

这是一个 Go + Next.js 的聊天系统工程，当前默认运行方式是纯 Docker Compose 隔离模式：

- 后端服务：`auth-service`、`api-service`、`realtime-service`
- 前端：`frontend/`（Next.js App Router）
- 基础设施：MySQL、Redis
- 可观测：OpenTelemetry + Jaeger
- 默认不会依赖宿主机上的 MySQL / Redis / Go / Node 进程
- 默认 Docker 栈使用仓库内置 fallback：事件链路走 Redis Pub/Sub，搜索走 MySQL fallback

## 目录结构

```text
.
├── cmd/                 # migrate / reindex
├── deploy/              # Docker Compose 及部署材料
├── docs/                # 设计和交付文档
├── frontend/            # Next.js 前端
├── internal/            # 后端业务与平台代码
├── migrations/          # SQL migrations
├── pkg/                 # 跨服务 contracts
├── scripts/             # 本地启动 / 刷新 / 停止脚本
└── services/            # auth / api / realtime 入口
```

## 现在只保留 3 个命令

启动项目：

```bash
make start
```

更新前后端资源并重启服务：

```bash
make refresh
```

停止项目：

```bash
make stop
```

`make start` 会完成以下工作：

- 构建并启动整套 Docker Compose 服务
- 在容器内启动 MySQL、Redis
- 执行数据库迁移
- 启动 3 个 Go 服务和前端 Next.js 服务
- 等待健康检查通过

`make refresh` 适合你修改前后端代码之后执行，它会：

- 重新构建相关镜像
- 重新创建受影响的容器
- 保留 Docker volume 中的数据
- 等待服务重新健康

`make stop` 会停止并移除当前项目的 Docker 容器，但保留 Docker volume 数据。

可选复检命令：

```bash
make smoke
```

`make smoke` 会在 Docker 容器内执行 API 与实时链路 smoke，不依赖宿主机的 Node 环境。

附加入口：

- 管理后台：`/admin`
- 默认管理员：`root / 123456`
- Jaeger：`http://127.0.0.1:28686`

## 默认端口

- Auth: `http://127.0.0.1:28081`
- API: `http://127.0.0.1:28082`
- Realtime: `ws://127.0.0.1:28083/ws`
- Frontend: `http://127.0.0.1:25173`
- Jaeger UI: `http://127.0.0.1:28686`

说明：

- 如果这组端口被占用，启动脚本会自动切到下一组隔离端口，并在终端输出实际端口。
- MySQL / Redis 仅在 Docker 网络内部暴露，不映射到宿主机端口。
- 如需调试基础设施，使用 `docker compose exec ...` 进入容器，而不是依赖宿主机连接。

## 清理说明

以下内容属于可再生运行产物，不应提交：

- `.runtime/`
- `frontend/.next/`
- `frontend/dist/`
- `frontend/tsconfig.tsbuildinfo`

## 补充文档

- [架构说明](/Users/lee/GolandProjects/awesomeProject/10/docs/ARCHITECTURE.md)
- [部署说明](/Users/lee/GolandProjects/awesomeProject/10/docs/DEPLOYMENT.md)
- [可观测性说明](/Users/lee/GolandProjects/awesomeProject/10/docs/OBSERVABILITY.md)
- [关键时序](/Users/lee/GolandProjects/awesomeProject/10/docs/SEQUENCES.md)
- [故障排查](/Users/lee/GolandProjects/awesomeProject/10/docs/TROUBLESHOOTING.md)
