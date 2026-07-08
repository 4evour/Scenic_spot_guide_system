# 灵山胜境景区智能导览系统

> 最后更新: 2026-06-18
> 本文档描述当前代码实现。接口清单以 `docs/api.md` 为准，路线图以 `docs/ROADMAP.md` 为准。

## 项目概述

本项目是面向景区游客和运营人员的智能导览系统。后端使用 Go/Gin/GORM 提供 API，主数据库配置为 PostgreSQL，SQLite 仅保留为本地开发和轻量测试配置；前端使用 Vue 3 + Vite + TypeScript 构建游客地图、数字人导览、管理后台和数据大屏。

核心能力：

- 智能问答：基于本地 JSONL 景区知识库的 RAG 问答，支持 BM25、本地重排、Embedding、混合召回和 SSE 流式响应。
- 数字人导览：Vue Live2D 舞台、Open-LLM-VTuber WebSocket 代理、情绪标签、口型驱动、流式 TTS、打断和文字降级。
- 会话持久化：游客/用户会话通过 Cookie 鉴权，聊天消息同时写入 localStorage、Pinia 状态和后端会话表。
- 景区管理：景点、路线、导览内容、知识库、二维码、游客问题处理、数字人配置和系统设置管理。
- 运营分析：交互日志、游客反馈、游客问题跟进、感受度报告、AI 会话分析和知识候选审核入库。

## 系统架构

```text
Vue 3 前端
  |-- 游客地图 / 数字人 / 扫码页
  |-- 管理后台 / 数据大屏 / 报告页
        |
        v
Go Gin API
  |-- Cookie/JWT 鉴权、CSRF、限流、安全响应头
  |-- RAG 问答、TTS、OpenAI-compatible /v1/chat/completions
  |-- /vtuber-ws/* 代理 Open-LLM-VTuber
        |
        v
GORM 数据层
  |-- PostgreSQL 主配置
  |-- SQLite 本地开发/测试配置
        |
        v
知识库与外部服务
  |-- knowledge/*.jsonl 和 knowledge/real/*.jsonl
  |-- OpenAI-compatible LLM / DashScope Embedding
  |-- Edge TTS / Open-LLM-VTuber / Live2D 资产
```

## 技术栈

| 层级 | 当前实现 |
|---|---|
| 后端 | Go 1.25、Gin、GORM、slog |
| 数据库 | PostgreSQL 主配置；SQLite 本地开发/测试配置 |
| 前端 | Vue 3、Vite、TypeScript、Pinia、Vue Router、Naive UI、PixiJS、Live2D |
| AI/RAG | OpenAI-compatible LLM、DashScope Embedding、本地 BM25/重排兜底 |
| 数字人 | Open-LLM-VTuber、`/vtuber-ws/*` 代理、Live2D `mao_pro` / `shizuku` |
| 语音 | `/api/v1/ai/tts` MP3 响应、`/api/v1/ai/tts/stream` 流式响应 |
| 部署 | Docker Compose 已包含 PostgreSQL 16 和 Go 服务 |

## 目录结构

```text
scenic-guide/
├── main.go                         # 服务启动、依赖组装、迁移、静态资源挂载
├── configs/                        # 配置模板和景区 profile
├── internal/
│   ├── config/                     # 配置结构、环境变量加载、景区 profile
│   ├── handler/                    # HTTP 路由和接口处理器
│   ├── model/                      # GORM 模型和 AutoMigrate
│   ├── pkg/                        # 数据库、JWT、中间件、响应、日志等公共包
│   ├── repository/                 # 数据访问层
│   └── service/                    # RAG、会话、统计、游客洞察、TTS 等业务逻辑
├── knowledge/                      # 景区知识库、真实资料评估集和合成评估集
├── web-vue/                        # Vue 前端源码
├── static/                         # Go 服务托管的静态资源和 Vue 构建产物
├── docs/                           # API、路线图、数字人运行和评估文档
├── scripts/                        # 检查、启动、录屏等辅助脚本
└── cmd/                            # rag-eval、demo-seed 等命令
```

## 启动流程

1. 读取 `configs/config.yaml`，并叠加 `SCENIC_GUIDE_` 环境变量。
2. 校验敏感配置：`SCENIC_GUIDE_AI_API_KEY` 和 `SCENIC_GUIDE_SECURITY_JWT_SECRET` 必须存在。
3. 初始化日志、JWT、数据库连接、GORM 自动迁移。
4. 初始化知识库、RAG 检索、会话持久化、游客洞察和统计服务。
5. 挂载 `/api/v1`、`/v1/chat/completions`、`/vtuber-ws/*`、Vue 应用入口和静态资源。
6. 接收 `SIGINT` / `SIGTERM` 后优雅关闭。

## 认证与安全

- 浏览器主路径使用 `auth_token` HttpOnly Cookie，登录接口 `POST /api/v1/login` 不在响应体返回 token。
- 非浏览器客户端仍可使用 `Authorization: Bearer <token>` 兼容路径。
- 修改类请求走 CSRF 双提交 Cookie，前端通过 `X-CSRF-Token` 发送 token。
- `/api/v1/admin/*`、知识库管理、会话管理和数字人用户接口按权限走 `AuthMiddleware` / `AdminMiddleware`。
- `/v1/chat/completions` 使用内部 API Key 中间件，兼容 `X-API-Key` 和 `Authorization: Bearer ...`。
- 日志不应输出 API Key、完整请求体、完整用户问题或完整回答。

## 核心接口概览

详细接口见 `docs/api.md`。当前主要路径包括：

- 认证：`/api/v1/register`、`/api/v1/login`、`/api/v1/logout`、`/api/v1/user/me`、`/api/v1/auth/guest-login`、`/api/v1/auth/upgrade-guest`
- AI：`/api/v1/ai/chat`、`/api/v1/ai/feedback`、`/api/v1/ai/tts`、`/api/v1/ai/tts/stream`
- 会话：`/api/v1/sessions`、`/api/v1/sessions/:session_id/messages`、`/api/v1/sessions/search`
- 数字人：`/api/v1/digital-human/avatar-options`、`/api/v1/dh/session/create`、`/api/v1/dh/chat/text`、`/api/v1/dh/chat/voice-transcript`、`/api/v1/dh/feedback`
- 管理：`/api/v1/admin/dashboard/*`、`/api/v1/admin/reports/visitor?period=7d|30d`、`/api/v1/admin/digital-human/config`、`/api/v1/admin/settings`、`/api/v1/admin/knowledge/*`、`/api/v1/admin/qr/*`、`/api/v1/admin/insights/*`
- 游客问题：`/api/v1/queries`、`/api/v1/queries/unanswered`、`/api/v1/queries/:id`
- 兼容：`/v1/chat/completions`、`/vtuber-ws/*`、`/health`

## 数据模型

`internal/model` 当前自动迁移以下主要模型：

- 景区业务：`ScenicSpot`、`GuideContent`、`TourRoute`、`VisitorQuery`
- 用户与访问：`User`、`VisitRecord`、`SystemLog`
- RAG：`KnowledgeChunk`
- 运营日志：`InteractionLog`
- 会话：`ChatSession`、`ChatMessage`
- 反馈闭环：`UserFeedback`、`VisitorInsightAnalysis`、`KnowledgeCandidate`
- 配置：`SystemSetting`、`DigitalHumanConfig`

## RAG 与数字人链路

- `/api/v1/ai/chat` 是游客端文本问答入口，支持 `session_id` 短期上下文。
- `/v1/chat/completions` 是 OpenAI-compatible 入口，供 Open-LLM-VTuber 调用，支持 `stream=true` SSE。
- `/vtuber-ws/*` 同源代理到本机 Open-LLM-VTuber 默认端口 `127.0.0.1:12393`。
- `DigitalHumanView.vue` 在数字人语音服务不可用时仍走 Go 后端文字问答，TTS 或自动播放失败不阻塞文字输出。
- 当前支持 `mao_pro` 和 `shizuku` 两个真实 Live2D 模型；管理员可限制游客只能使用默认模型。

## 配置要点

配置模板见 `configs/config.example.yaml`。

```yaml
database:
  driver: "postgres" # 生产/容器主路径；sqlite 仅用于本地开发和轻量测试

ai:
  api_key: ""        # SCENIC_GUIDE_AI_API_KEY
  model: "qwen-vl-max"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"

embedding:
  api_key: ""        # SCENIC_GUIDE_EMBEDDING_API_KEY
  model: "text-embedding-v2"

security:
  jwt_secret: ""     # SCENIC_GUIDE_SECURITY_JWT_SECRET，至少 32 字符随机值
  token_expire_hours: 24

redis:
  addr: "localhost:6379" # 可选；为空时使用内存限流
```

## 运行与验证

```bash
go mod download
go test ./...

cd web-vue
npm install
npm run check
npm run build
```

容器启动：

```bash
docker compose up --build
```

本地联调数字人时，可使用 `scripts/start-local.ps1` 启动 Go 服务、演示数据和 Open-LLM-VTuber。

## 不做微信小程序

当前产品主路径是 Web/Vue。微信小程序环境无法完整承载当前 Live2D 渲染、麦克风交互、WebSocket 代理、流式 TTS、口型驱动和打断链路，因此不把小程序作为交付目标。预约订阅可以后续作为独立票务模块评估，但不等同于小程序端重写。

## 已知边界

- SQLite 不是生产数据库能力，也不是 PostgreSQL 故障接管方案。
- Docker Compose 已存在，但生产化还需要版本化 migration、备份恢复、监控告警和发布流程。
- 真实景区落地需要持续补充真实资料、人工标注评估、来源引用和运营审核。
- 旧静态数字人页面仍保留部分资源，删除前需要确认没有外部入口依赖。
