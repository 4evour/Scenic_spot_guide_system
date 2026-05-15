# Scenic Spot Guide System

景区智能导览系统，面向景区游客问答、导览内容管理、路线推荐、运营数据看板和数字人导览场景。项目采用 Go/Gin 提供后端 API，SQLite/GORM 保存业务数据，Vue 3 + Vite 构建前端页面，并集成本地知识库 RAG 与 Live2D 数字人展示。

## 功能概览

- 游客问答：基于景区知识库进行检索增强问答，未配置 Embedding 时可回退到 BM25 检索。
- 景点与导览管理：维护景点、导览内容、游览路线等基础数据。
- 管理后台：提供内容、路线、数字人配置和系统设置管理入口。
- 数据看板：展示访问量、问答趋势、热门问题、满意度和数字人交互数据。
- 数字人导览：Vue 前端渲染 Live2D 模型，并可通过 WebSocket 代理对接 Open-LLM-VTuber 服务。
- OpenAI 兼容接口：提供 `/v1/chat/completions`，便于外部数字人服务调用。

## 技术栈

- 后端：Go 1.25.0、Gin、GORM、SQLite
- 前端：Vue 3、Vite、TypeScript、PixiJS、Live2D
- AI/RAG：DeepSeek 兼容接口、DashScope Embedding 示例配置、本地 JSONL 知识库
- 静态资源：Go 服务托管 `static` 目录，Vue 构建产物输出到 `static/vue-app`

## 目录结构

```text
.
├── main.go                      # 服务启动和依赖装配
├── configs/                     # 本地配置目录，config.yaml 不提交
├── internal/                    # 后端配置、模型、仓储、服务和处理器
├── knowledge/                   # 景区知识库语料与分块数据
├── web-vue/                     # Vue 前端源码
├── static/                      # 静态页面、数字人资源和 Vue 构建产物
├── docs/                        # 数字人集成说明
└── PROJECT_OVERVIEW.md          # 项目长期说明文档
```

## 环境要求

- Go 1.25.0 或与 `go.mod` 匹配的版本
- Node.js 20+ 与 npm
- 可选：DeepSeek API Key、DashScope Embedding API Key、语音服务配置

## 快速启动

1. 准备配置文件：

```powershell
Copy-Item configs/config.example.yaml configs/config.yaml
```

在 `configs/config.yaml` 中按需填写 `ai.api_key`、`embedding.api_key`、`speech.api_key`，并将 `security.jwt_secret` 改为自己的随机密钥。真实配置文件已被 `.gitignore` 忽略，不应提交。

2. 安装后端依赖：

```powershell
go mod download
```

3. 构建前端：

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

4. 可选：初始化演示账号与演示数据：

```powershell
go run ./cmd/demo-seed
```

默认演示账号为 `admin / DemoAdmin123456`。该命令会写入管理员、游客、景点、路线、交互日志，并在知识库为空时导入默认知识片段。

5. 启动服务：

```powershell
go run .
```

默认监听 `0.0.0.0:8080`。启动后会自动迁移数据库，并在知识库为空时导入 `knowledge/lingshan_chunks.jsonl`。

## 访问入口

- 首页：`http://127.0.0.1:8080/`
- Vue 应用：`http://127.0.0.1:8080/app`
- 数据看板：`http://127.0.0.1:8080/dashboard`
- 管理后台：`http://127.0.0.1:8080/admin`
- 数字人导览：`http://127.0.0.1:8080/digital-human`
- 健康检查：`http://127.0.0.1:8080/health`

## 数字人服务

项目内置 Live2D 前端资源，当前 Vue 数字人入口使用 `static/live2d-models/mao_pro` 模型。后端将 `/vtuber-ws/*` 代理到本机 `127.0.0.1:12393`，用于对接 Open-LLM-VTuber 等外部数字人运行服务。

如不启动外部数字人服务，普通后台、看板和基础问答接口仍可运行；涉及实时语音或 WebSocket 驱动的能力会受限。

## 常用命令

```powershell
make check
go test ./...
go vet ./...
go run ./cmd/rag-eval

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
$env:SCENIC_GUIDE_SECURITY_TOKEN_EXPIRE_HOURS="4"
$env:SCENIC_GUIDE_AI_API_KEY="你的服务端密钥"
```

## 编码约定

- 源码和文档统一使用 UTF-8，规则见 `.editorconfig`。
- Windows PowerShell 若直接输出中文出现乱码，通常是控制台代码页/宿主解码问题，不代表文件已损坏。排查时优先使用 UTF-8 明确输出或运行 `npm run check:encoding`。
- `npm run check:encoding` 会扫描源码和文档中的替换字符及常见 mojibake 模式；构建产物和第三方资源不纳入该检查。

## RAG 评估

项目内置 `knowledge/lingshan_eval_qa.json` 作为基础评测集，可用本地 BM25 模式离线验证知识库检索与回答兜底效果：

```powershell
go run ./cmd/rag-eval -format text
go run ./cmd/rag-eval -format json
```

评估报告包含用例总数、通过率、关键词平均覆盖率、缺失关键词和回答预览；如需在 CI 或脚本中失败退出，可追加 `-fail-on-miss`。

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
