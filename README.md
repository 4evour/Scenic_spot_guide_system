# Scenic Spot Guide System

景区智能导览系统，面向景区游客问答、导览内容管理、路线推荐、运营数据看板和数字人导览场景。项目采用 Go/Gin 提供后端 API，PostgreSQL/GORM 作为主数据库配置，SQLite 仅作为本地开发和轻量测试配置，Vue 3 + Vite 构建前端页面，并集成本地知识库 RAG 与 Open-LLM-VTuber 数字人联调能力。

## 功能概览

- 游客问答：基于景区知识库进行检索增强问答；无 Embedding Key 时使用本地 BM25/词面检索，保证无 Key 可复现。
- 景点与导览管理：维护景点、导览内容、游览路线等基础数据。
- 管理后台：提供内容、路线、数字人配置和系统设置管理入口。
- 数据看板：展示访问量、问答趋势、热门问题、满意度和数字人交互数据。
- 数字人导览：通过 OpenAI 兼容接口、SSE 流式响应和 `/vtuber-ws/*` 代理对接 Open-LLM-VTuber，并提供景区定制前端二开层。
- OpenAI 兼容接口：提供 `/v1/chat/completions`，便于外部数字人服务调用。

## 技术栈

- 后端：Go 1.25.0、Gin、GORM、PostgreSQL、SQLite local/dev profile
- 前端：Vue 3、Vite、TypeScript、PixiJS、Live2D
- AI/RAG：DeepSeek 兼容接口、DashScope `text-embedding-v2` 示例配置、本地 JSONL 知识库、BM25/词面本地检索
- 静态资源：Go 服务托管 `static` 目录，Vue 构建产物输出到 `static/vue-app`

## 目录结构

```text
.
├── main.go                      # 服务启动和依赖装配
├── configs/                     # 本地配置目录，config.yaml 不提交
├── internal/                    # 后端配置、模型、仓储、服务和处理器
├── knowledge/                   # 景区知识库语料、基础样例和 3000/300 合成规模验证集
├── web-vue/                     # Vue 前端源码
├── static/                      # 静态页面、数字人资源和 Vue 构建产物
├── docs/                        # API、数字人联调和面试说明
└── PROJECT_OVERVIEW.md          # 项目长期说明文档
```

## 环境要求

- Go 1.25.0 或与 `go.mod` 匹配的版本
- Node.js 20+ 与 npm
- Docker Desktop 或本地 PostgreSQL 16+
- 可选：DeepSeek API Key、DashScope Embedding API Key、语音服务配置；无 Key 时仍可启动页面并运行本地检索评估

## 快速启动

1. 准备配置文件：

```powershell
Copy-Item configs/config.example.yaml configs/config.yaml
```

在 `configs/config.yaml` 中按需填写 `ai.api_key`、`embedding.api_key`、`speech.api_key`，并将 `security.jwt_secret` 改为自己的随机密钥。真实配置文件已被 `.gitignore` 忽略，不应提交。

2. 使用 Docker Compose 启动 PostgreSQL 和应用：

```powershell
$env:SCENIC_GUIDE_SECURITY_JWT_SECRET="至少32位随机字符串"
docker compose up --build
```

Compose 默认启动 `postgres:16-alpine`，应用通过 `SCENIC_GUIDE_DATABASE_DRIVER=postgres` 连接数据库。需要本地轻量运行时，也可以显式切换到 `database.driver: sqlite`；这只是开发配置，不是 PostgreSQL 故障后的自动接管或高可用方案。

3. 安装后端依赖：

```powershell
go mod download
```

4. 构建前端：

```powershell
Set-Location web-vue
npm install
npm run build
Set-Location ..
```

公开仓库或提交前建议额外运行：

```powershell
node scripts/check-secrets.mjs
```

5. 可选：初始化演示账号与演示数据：

```powershell
go run ./cmd/demo-seed
```

默认演示账号为 `admin / DemoAdmin123456`。该命令会写入管理员、游客、景点、路线、交互日志，并在知识库为空时导入默认知识片段。

6. 本地直启服务：

```powershell
go run .
```

默认监听 `0.0.0.0:8080`。启动后会自动迁移 PostgreSQL 或显式配置的 SQLite 本地数据库，并在知识库为空时导入 `knowledge/lingshan_chunks.jsonl`。

## 一键复现

本项目不提供公网 Demo 链接，仓库负责可复现，博客负责展示说明。默认复现路径不需要 DeepSeek、DashScope 或语音服务 Key：

```powershell
$env:SCENIC_GUIDE_SECURITY_JWT_SECRET="至少32位随机字符串"
docker compose up --build
```

启动后可访问 `http://127.0.0.1:8080/`、`/app`、`/admin`、`/dashboard` 等页面。无 Key 情况下，RAG 评估和基础问答使用本地 BM25/词面检索；DeepSeek、DashScope Embedding 和语音服务只作为可选增强。

复现 RAG smoke test：

```powershell
go run ./cmd/rag-eval -k 8 -fail-on-miss
```

复现 3000/300 合成闭集实验：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -fail-on-miss
```

## 访问入口

- 首页：`http://127.0.0.1:8080/`
- Vue 应用：`http://127.0.0.1:8080/app`
- 数据看板：`http://127.0.0.1:8080/dashboard`
- 管理后台：`http://127.0.0.1:8080/admin`
- 数字人导览：`http://127.0.0.1:8080/digital-human`
- 健康检查：`http://127.0.0.1:8080/health`

## 数字人服务

项目保留 Vue Live2D 视图，同时主联调路径定位为 Open-LLM-VTuber 协议适配和前端二开。后端提供 `/v1/chat/completions` OpenAI 兼容接口、`stream=true` SSE 流式响应，并将 `/vtuber-ws/*` 代理到本机 `127.0.0.1:12393`。

`Open-LLM-VTuber/frontend/assets/scenic-tech-demo.js` 和 `.css` 是景区定制注入层，包含品牌导览面板、连接状态、麦克风权限状态、回答流式状态、打断/重试按钮和当前会话 `trace_id` 展示。联调清单见 `docs/digital-human-production-check.md`。

如不启动外部数字人服务，普通后台、看板和基础问答接口仍可运行；涉及实时语音或 WebSocket 驱动的能力会受限。

## 常用命令

```powershell
make check
go test ./...
go vet ./...
go run ./cmd/rag-eval -k 8 -format text
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -fail-on-miss

Set-Location web-vue
npm run check
npm run check:encoding
npm run build
Set-Location ..
```

## 配置与环境变量

运行时配置默认读取 `configs/config.yaml`，也可以用 `SCENIC_GUIDE_` 前缀的环境变量覆盖嵌套配置，例如：

```powershell
$env:SCENIC_GUIDE_SECURITY_JWT_SECRET="至少32位随机字符串"
$env:SCENIC_GUIDE_DATABASE_DRIVER="postgres"
$env:SCENIC_GUIDE_DATABASE_HOST="127.0.0.1"
$env:SCENIC_GUIDE_DATABASE_PORT="5432"
$env:SCENIC_GUIDE_DATABASE_NAME="scenic_guide"
$env:SCENIC_GUIDE_DATABASE_USER="scenic"
$env:SCENIC_GUIDE_DATABASE_PASSWORD="scenic_password"
$env:SCENIC_GUIDE_SECURITY_TOKEN_EXPIRE_HOURS="4"
$env:SCENIC_GUIDE_AI_API_KEY="你的服务端密钥"
```

## 编码约定

- 源码和文档统一使用 UTF-8，规则见 `.editorconfig`。
- Windows PowerShell 若直接输出中文出现乱码，通常是控制台代码页/宿主解码问题，不代表文件已损坏。排查时优先使用 UTF-8 明确输出或运行 `npm run check:encoding`。
- `npm run check:encoding` 会扫描源码和文档中的替换字符及常见 mojibake 模式；构建产物和第三方资源不纳入该检查。

## RAG 评估

项目保留 `knowledge/lingshan_chunks.jsonl` 与 `knowledge/lingshan_eval_qa.json` 作为 32 个知识切片、5 条评测问答的快速 smoke test；同时提供 `knowledge/lingshan_scale_3000.jsonl` 与 `knowledge/lingshan_eval_300.json` 作为“合成规模验证集”，用于复现 3000 切片、300 问答的闭集检索实验。该数据集不是独立真实景区生产数据，也不等同完整景区知识库。

```powershell
go run ./cmd/rag-eval -k 8 -format text
go run ./cmd/rag-eval -k 8 -format json
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -fail-on-miss
```

评估数据格式包含 `question`、`expected_keywords`、`expected_chunk_ids`、`category`、`difficulty`。评估报告包含用例总数、通过率、Recall@K、MRR@K、关键词平均覆盖率、失败样例和检索耗时 p50/p95；如需在 CI 或脚本中失败退出，可追加 `-fail-on-miss`。

3000/300 合成闭集实验仅作为内部回归数据集，不能作为简历主卖点，也不能外推为开放域真实问答召回率。简历主口径使用 `knowledge/real/` 真实资料评估集：122 个真实资料切片、203 条独立评测问答，并发 16、repeat 3 的 retrieval-only bench 结果为 Recall@8 85.5%、MRR@8 0.749、关键词覆盖率 94.3%、纯检索 p50/p95 约 7ms/10ms；该结果不包含外部 Embedding、大模型生成、ASR 或 TTS。

数据边界和评估口径见 `knowledge/DATASET.md` 与 `docs/rag-eval-report.md`。

## 演示数据初始化

`cmd/demo-seed` 会写入当前配置指向的数据库，适合本地演示或答辩录制前准备数据，不属于只读检查命令：

```powershell
go run ./cmd/demo-seed
go run ./cmd/demo-seed -admin-password "替换成本地演示密码"
```

默认账号 `admin / DemoAdmin123456` 仅用于本地演示，公开部署或生产环境不要使用默认演示密码。

## 清理与提交约定

- 不提交 `configs/config.yaml`、`.env`、数据库文件、日志、可执行文件和本地缓存。
- 前端生产构建已关闭 sourcemap，避免提交大体积 `.map` 调试文件。
- 如需更新前端静态资源，请在 `web-vue` 中运行 `npm run build`，并提交刷新后的 `static/vue-app` 产物。
