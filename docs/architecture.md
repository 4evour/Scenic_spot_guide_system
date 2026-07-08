# 系统架构图

## 架构总览

```mermaid
flowchart LR
    visitor["游客浏览器"]
    admin["管理员浏览器"]
    vtuberPage["Open-LLM-VTuber 前端<br/>Live2D / 语音交互"]

    subgraph vue["Vue 3 前端 scenic-guide/web-vue"]
        map["地图导览 /map"]
        dashboard["数据大屏 /dashboard"]
        adminUi["管理后台 /admin"]
        apiClient["统一 API 服务<br/>src/services/api.ts"]
        vtuberClient["VTuber WS 客户端<br/>src/services/vtuberSocket.ts"]
    end

    subgraph go["Go 景区主服务 scenic-guide"]
        entry["main.go<br/>配置 / JWT / Redis / DB / RAG / DI"]
        gin["Gin Router<br/>internal/handler/routes.go"]
        middleware["中间件<br/>CORS / CSRF / Auth / RateLimit / Metrics"]
        handlers["Handler 层<br/>景点 / 路线 / 用户 / AI / 数字人 / 管理"]
        services["Service 层<br/>业务编排 / RAG / 统计 / 会话"]
        repos["Repository 层<br/>GORM 数据访问"]
        static["静态资源托管<br/>static/ 与 static/vue-app/"]
        openaiCompat["OpenAI 兼容接口<br/>POST /v1/chat/completions"]
        wsProxy["WebSocket 代理<br/>/vtuber-ws/*path"]
        metrics["Prometheus 指标<br/>/metrics"]
    end

    subgraph rag["RAG 与知识库"]
        ragService["RAGService"]
        retrieval["检索引擎<br/>Embedding / BM25 / Hybrid / RRF"]
        generation["生成服务<br/>Prompt / LLM 调用 / 流式输出"]
        knowledgeFiles["knowledge/*.jsonl<br/>默认知识切片"]
        scenicProfile["configs/scenic_profiles/*.yaml<br/>景区画像 / Prompt / 路线"]
    end

    subgraph storage["存储与基础设施"]
        db["PostgreSQL<br/>或 SQLite 本地配置"]
        redis["Redis 可选<br/>分布式限流"]
        logs["结构化日志"]
    end

    subgraph external["外部或旁路服务"]
        llm["DeepSeek/OpenAI 兼容 Chat API"]
        embedding["DashScope text-embedding-v2"]
        vtuberServer["Open-LLM-VTuber Python 服务<br/>FastAPI / WebSocket :12393"]
        live2dAssets["Live2D 模型与角色配置<br/>conf.yaml / model_dict.json"]
    end

    visitor --> map
    visitor --> vtuberPage
    admin --> adminUi
    admin --> dashboard
    map --> apiClient
    dashboard --> apiClient
    adminUi --> apiClient
    apiClient --> gin
    vtuberClient --> wsProxy
    vtuberPage --> vtuberServer

    entry --> gin
    gin --> middleware
    gin --> static
    gin --> handlers
    gin --> openaiCompat
    gin --> wsProxy
    gin --> metrics
    handlers --> services
    services --> repos
    repos --> db
    services --> ragService
    ragService --> retrieval
    ragService --> generation
    ragService --> knowledgeFiles
    ragService --> scenicProfile
    retrieval --> embedding
    generation --> llm
    openaiCompat --> ragService
    wsProxy --> vtuberServer
    vtuberServer --> live2dAssets
    vtuberServer --> openaiCompat
    middleware --> redis
    services --> logs
```

## 后端分层图

```mermaid
flowchart TB
    request["HTTP / WebSocket 请求"]
    route["internal/handler/routes.go<br/>统一注册路由"]
    auth["pkg 中间件<br/>Auth / Admin / CSRF / API Key / RateLimit"]
    handlerLayer["internal/handler<br/>参数绑定、鉴权语义、统一响应"]
    serviceLayer["internal/service<br/>业务逻辑、RAG、统计、会话、数字人编排"]
    repoLayer["internal/repository<br/>GORM CRUD 与查询聚合"]
    modelLayer["internal/model<br/>数据模型与 AutoMigrate"]
    database["PostgreSQL / SQLite"]

    request --> route --> auth --> handlerLayer --> serviceLayer --> repoLayer --> modelLayer --> database
```

## 启动链路

```mermaid
sequenceDiagram
    participant Main as scenic-guide/main.go
    participant Config as configs + env
    participant Pkg as internal/pkg
    participant DB as Database
    participant RAG as RAGService
    participant Router as Gin Router

    Main->>Config: LoadConfig("./configs")
    Main->>Pkg: InitLogger / InitJWT / InitRedis
    Main->>DB: InitDatabase + AutoMigrate
    Main->>Main: ensureAdminUser()
    Main->>Config: LoadScenicProfile(SCENIC_GUIDE_SCENIC_ID)
    Main->>RAG: NewRAGService + LoadKnowledgeFromFile()
    Main->>Router: setupDI() + handler.SetupRoutes()
    Router-->>Main: HTTP Server ListenAndServe()
```

## RAG 问答链路

```mermaid
sequenceDiagram
    participant Client as Vue/数字人客户端
    participant API as /api/v1/ai/chat 或 /v1/chat/completions
    participant Handler as AIHandler/OpenAIProxyHandler
    participant RAG as RAGService
    participant Repo as KnowledgeRepository
    participant Embed as Embedding Provider
    participant LLM as Chat API

    Client->>API: 提交游客问题
    API->>Handler: 认证、限流、CSRF/API Key 校验
    Handler->>RAG: QueryWithRAGTraceInSession()
    RAG->>Repo: 读取知识切片
    RAG->>Embed: 生成查询向量（可选）
    RAG->>RAG: BM25/Embedding/Hybrid 检索与 rerank
    alt 命中知识且配置了 LLM
        RAG->>LLM: 构造 RAG Prompt 调用 Chat API
        LLM-->>RAG: 返回回答
    else 未配置 LLM 或无命中
        RAG->>RAG: 本地片段答案或通用 Chat 兜底
    end
    RAG-->>Handler: answer + trace + route
    Handler-->>Client: 统一 JSON 响应
```

## 数字人交互链路

```mermaid
sequenceDiagram
    participant Browser as 浏览器 Live2D 页面
    participant VTuber as Open-LLM-VTuber :12393
    participant Go as scenic-guide :8080
    participant RAG as Go RAGService
    participant LLM as 外部 Chat API

    Browser->>VTuber: WebSocket /client-ws
    VTuber-->>Browser: set-model-and-conf / start-mic
    Browser->>VTuber: text-input 或 mic-audio-end
    VTuber->>Go: POST /v1/chat/completions
    Go->>RAG: OpenAIProxyHandler.ChatCompletions
    RAG->>LLM: 可选调用外部 LLM
    LLM-->>RAG: 回答文本
    RAG-->>Go: 兼容 OpenAI 响应
    Go-->>VTuber: assistant message
    VTuber-->>Browser: 文本、音频、表情控制消息
```

## 两个系统完全独立

### 景区系统（scenic-guide）
**职责**：景区数据、知识库、RAG 问答、管理后台
**配置文件**：`configs/scenic_profiles/lingshan.yaml`
**启动命令**：`cd scenic-guide && ./scenic-guide.exe`

**切换景区只需**：
1. 创建新配置 `configs/scenic_profiles/xihu.yaml`
2. 准备知识库 `knowledge/xihu_chunks.jsonl`
3. 设置环境变量 `SCENIC_GUIDE_SCENIC_ID=xihu`
4. 重启

### 数字人系统（Open-LLM-VTuber）
**职责**：Live2D 渲染、语音交互、表情控制
**配置文件**：`conf.yaml`
**启动命令**：`cd Open-LLM-VTuber && python run_server.py`

**切换数字人只需**：
1. 修改 `conf.yaml` 中的 `persona_prompt`
2. 修改 `model_dict.json` 中的表情映射
3. 重启

## API 接口（连接点）

两个系统通过以下 API 连接：

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/v1/scenic/profile` | GET | 景区完整信息，包含景区名称、数字人配置、快捷问题、路线和主题实体 |
| `/api/v1/scenic/persona` | GET | 数字人角色提示词（从景区配置动态生成） |
| `/api/v1/scenic/quick-asks` | GET | 快捷提问列表 |
| `/api/v1/spots` | GET | 公开景点列表 |
| `/api/v1/routes` | GET | 公开游览路线列表 |
| `/api/v1/ai/chat` | POST | RAG 问答（数字人发送问题，景区系统回答） |
| `/v1/chat/completions` | POST | OpenAI 兼容接口（数字人直接调用） |

## 解耦验证

**场景 1：只换景区，不换数字人**
```bash
# 1. 创建新景区配置
cp configs/scenic_profiles/lingshan.yaml configs/scenic_profiles/xihu.yaml
# 编辑 xihu.yaml

# 2. 设置环境变量
export SCENIC_GUIDE_SCENIC_ID=xihu

# 3. 重启景区系统
./scenic-guide.exe

# 数字人自动获取新景区的 persona_prompt、quick_asks、routes
```

**场景 2：只换数字人，不换景区**
```bash
# 1. 修改数字人配置
vim Open-LLM-VTuber/conf.yaml
# 修改 persona_prompt、TTS 音色、Live2D 模型

# 2. 重启数字人系统
python run_server.py

# 景区系统完全不受影响
```

**场景 3：两个系统分别部署**
```bash
# 机器 A：景区系统
cd scenic-guide && ./scenic-guide.exe  # 监听 :8080

# 机器 B：数字人系统
# conf.yaml 中 base_url 改为机器 A 的地址
python run_server.py  # 监听 :12393
```

## 独立迁移清单

### 迁移景区系统到新景区
- [ ] 创建 `configs/scenic_profiles/新景区.yaml`
- [ ] 准备知识库 JSONL 文件
- [ ] 更新数据库中的景点数据
- [ ] 设置 `SCENIC_GUIDE_SCENIC_ID` 环境变量
- [ ] 重启 Go 后端

### 迁移数字人系统到新形象
- [ ] 准备新的 Live2D 模型文件
- [ ] 更新 `model_dict.json` 表情映射
- [ ] 更新 `conf.yaml` 中的角色配置
- [ ] 重启 Python 后端

### 两个系统同时迁移
- [ ] 完成景区系统迁移
- [ ] 完成数字人系统迁移
- [ ] 验证 API 连通性
- [ ] 测试完整对话流程
