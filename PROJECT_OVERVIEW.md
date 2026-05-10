# 项目总览

## 项目目标

Scenic Spot Guide System 是一个景区智能导览系统，提供游客问答、景点/导览内容管理、路线管理、运营数据看板和数字人导览能力。

## 技术栈

- 后端：Go 1.25.0、Gin、GORM、SQLite。
- 前端：Vue 3、Vite、TypeScript、PixiJS、Live2D。
- 知识库：本地 JSONL 分块语料，启动时导入到数据库；未配置 Embedding API 时回退到 BM25 检索。
- AI：DeepSeek 兼容接口；Embedding 示例配置使用 DashScope。

## 目录结构

- `main.go`：服务启动、依赖组装、RAG 初始化和优雅关闭。
- `internal/config`：配置结构和加载逻辑，默认读取 `configs/config.yaml`，支持 `SCENIC_GUIDE_` 环境变量覆盖。
- `internal/handler`：HTTP 路由与接口处理。
- `internal/service`：业务逻辑、RAG、统计和 AI 相关服务。
- `internal/repository`：数据库访问层。
- `internal/model`：GORM 模型与自动迁移。
- `knowledge`：景区知识库语料、分块文件和评测样例。
- `web-vue`：Vue 管理端、数据看板和数字人前端源码。
- `static`：Go 服务直接托管的静态资源和 Vue 构建产物。
- `docs`：数字人集成、运行说明和 API 文档。
- `scripts`：项目辅助脚本，例如编码检查。

## 核心流程

1. 启动时加载 `configs/config.yaml`，再叠加 `SCENIC_GUIDE_` 环境变量。
2. 初始化日志、JWT、SQLite 数据库和模型迁移。
3. 初始化 RAG 服务：优先使用 Embedding 检索，缺少配置时使用 BM25。
4. 若知识库表为空，从 `knowledge/lingshan_chunks.jsonl` 导入数据。
5. Gin 挂载 API、静态页面、Vue 应用入口和 Open-LLM-VTuber 代理。
6. 服务收到 `SIGINT` 或 `SIGTERM` 后执行优雅关闭。

## 运行与测试

- 复制 `configs/config.example.yaml` 为 `configs/config.yaml`，再填写本地密钥。
- `security.jwt_secret` 必须替换为至少 32 位的随机字符串，不能使用示例占位值。
- 后端依赖：`go mod download`。
- 前端构建：在 `web-vue` 目录运行 `npm install` 和 `npm run build`。
- 启动服务：仓库根目录运行 `go run .`。
- 验证命令：`go test ./...`、`go vet ./...`、`npm run check`、`npm run check:encoding`、`npm run build`。

## 关键约定

- 不提交 `configs/config.yaml`、数据库、日志、可执行文件、缓存目录和本地环境文件。
- `static/vue-app` 是 Vue 构建输出，会随 `web-vue` 构建刷新。
- 前端生产构建关闭 sourcemap，避免提交大体积调试映射文件。
- `static/digital-human/libs/live2dcubismcore.min.js` 被 Vue 入口引用，不能随意删除。
- 当前 Live2D 主模型使用 `static/live2d-models/mao_pro`。
- 未被业务入口引用的旧 `static/live2d-models/shizuku` 模型已清理。
- 源码和文档统一使用 UTF-8；提交前应运行编码检查，防止中文内容 mojibake。

## 安全与鉴权补充

- `/api/v1/admin/*` 管理接口统一要求 `Authorization: Bearer <token>`，并必须通过 `AuthMiddleware` 与 `AdminMiddleware`。
- `/api/v1/knowledge/*` 知识库管理接口按管理员接口处理，读取、创建、上传、更新和删除都需要管理员权限。
- `/api/v1/register` 不接受客户端传入角色，后端会强制把新注册用户角色设为 `visitor`。
- 静态管理页和 Vue 管理/大屏页面从 `localStorage.authToken` 读取 JWT 并附加到管理请求。
- 旧静态游客页和大屏中由用户或接口返回的文本插入 HTML 前必须经过 `escapeHtml` 转义。
- RAG 通用问答日志不得打印 API Key 或其前缀；调试日志应避免输出完整请求体和敏感响应体。

## 已知风险

- `configs/config.yaml` 本地可能含真实 API Key，必须保持忽略状态。
- `static/digital-human` 保留了旧数字人静态前端及运行库，删除前需确认没有外部入口依赖。
- 服务启动日志在部分 Windows 终端可能出现编码显示问题，但 Go 编译和测试不受影响。
