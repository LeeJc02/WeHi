[English](./README.md)

<p align="center">
  <img src="./frontend/public/wehi-logo.png" alt="WeHi" width="560" />
</p>



一个面向实时通信场景的分布式 IM 项目，提供桌面风格的 Web 客户端，以及可独立部署的认证、业务 API 与实时网关服务。
<p align="center">
  <a href="https://github.com/LeeJc02/WeHi/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/LeeJc02/WeHi/ci.yml?branch=main&label=CI" alt="CI" /></a>
  <a href="https://github.com/LeeJc02/WeHi/actions/workflows/release-images.yml"><img src="https://img.shields.io/github/actions/workflow/status/LeeJc02/WeHi/release-images.yml?branch=main&label=GHCR" alt="GHCR" /></a>
  <a href="https://github.com/LeeJc02/WeHi/blob/main/LICENSE"><img src="https://img.shields.io/github/license/LeeJc02/WeHi" alt="License" /></a>
  <a href="https://github.com/LeeJc02/WeHi/pkgs/container/wehi%2Fauth-service"><img src="https://img.shields.io/badge/GHCR-multi--arch-07c160" alt="GHCR multi-arch" /></a>
  <a href="https://github.com/LeeJc02/WeHi/stargazers"><img src="https://img.shields.io/github/stars/LeeJc02/WeHi?style=social" alt="GitHub stars" /></a>
</p>


## 项目概览

WeHi 关注的不是单一发消息接口，而是一条完整的实时消息链路：

- 认证与多端会话管理
- WebSocket 长连接与在线状态同步
- 单聊、群聊、消息已读、消息撤回
- 历史消息查询与断线后的增量追平
- 容器化部署、健康检查与链路观测

仓库包含两部分：

- `backend/`: Go 后端服务与运行配置
- `frontend/`: Next.js App Router 客户端

## 为什么值得看

- 清晰的服务边界：认证、业务 API、实时网关分离
- 真实的状态拆层：MySQL 存事实，Redis 承接高频状态
- 补偿链路完整：通过 `sync_events + cursor` 追平离线期间状态
- 可交付：本地 Compose 启动、CI 校验、GHCR 多架构镜像

## 技术栈

- Backend: Go, Gin, GORM
- Frontend: Next.js 16, React 19, TypeScript, Tailwind CSS
- Data: MySQL, Redis
- Realtime: WebSocket
- Observability: OpenTelemetry, Jaeger
- Delivery: Docker, GitHub Actions, GHCR

## 架构概览

- `auth-service`: 注册、登录、刷新令牌、会话管理
- `api-service`: 好友、会话、消息、搜索、上传、后台接口
- `realtime-service`: WebSocket 建连、心跳、在线态更新、事件推送
- `sync_events`: 用户维度的增量事件日志，支撑断线补偿与重连追平

## 快速开始

本地开发：

```bash
make start
make smoke
```

停止：

```bash
make stop
```

默认端口：

- Frontend: `http://127.0.0.1:25173`
- Auth: `http://127.0.0.1:28081`
- API: `http://127.0.0.1:28082`
- Realtime: `ws://127.0.0.1:28083/ws`
- Jaeger: `http://127.0.0.1:28686`

## GHCR 运行

仓库已配置多架构镜像发布，支持 `linux/amd64` 与 `linux/arm64`：

```bash
export IMAGE_PREFIX=ghcr.io/leejc02/wehi
export IMAGE_TAG=latest
make release-up
```

停止：

```bash
make release-down
```

## 仓库结构

```text
.
├── backend/           # Go 后端服务、配置、迁移与模块代码
├── deploy/compose/    # 本地与发布环境的 Compose 编排
├── frontend/          # Next.js 客户端
├── scripts/           # 启停、环境装配与 smoke 脚本
└── .github/workflows/ # CI 与镜像发布
```

对外展示所需的品牌资源现在统一放在 `frontend/public/`，这样 README、站点图标和前端构建拿到的是同一套文件。

## 验证

```bash
make go-test
make frontend-lint
make frontend-build
make verify
```

## 贡献

欢迎提交 issue、设计建议和 PR。如果这个项目对你有帮助，欢迎点一个 star。

## License

本项目采用 [MIT License](./LICENSE)。
