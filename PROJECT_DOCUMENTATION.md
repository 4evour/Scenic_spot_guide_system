# 灵山胜境景区智能导览系统

## 📋 项目概述

本项目是一个基于Go语言开发的**智能景区导览系统**，集成了**RAG（检索增强生成）技术**、**向量嵌入服务**和**智能问答功能**，为游客提供关于灵山胜境景区的智能化导览服务。

### 核心功能

- 🎯 **智能问答**：基于知识库的AI导览助手，回答游客关于景区的各类问题
- 📚 **知识库管理**：支持知识的上传、查询、删除等管理功能
- 👤 **用户管理**：完整的用户注册、登录、权限管理功能
- 🗺️ **景点管理**：景区景点信息的CRUD操作
- 🛤️ **游览路线管理**：推荐游览路线的管理
- 🔊 **语音合成(TTS)**：文字转语音功能（预留）

---

## 🏗️ 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                        客户端层                              │
│   (Web浏览器 / 移动端 / 第三方应用)                           │
└────────────────────────┬──────────────────────────────────┘
                         │ HTTP/REST API
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                      API网关层 (Gin)                         │
│   • 路由管理    • 认证授权    • 请求限流    • 日志记录      │
└────────────────────────┬──────────────────────────────────┘
                         │
┌────────────────────────▼──────────────────────────────────┐
│                      业务逻辑层 (Service)                    │
│  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐   │
│  │ RAG服务      │ │ 用户服务      │ │ 景点/路线服务     │   │
│  │ - 向量检索   │ │ - 注册/登录   │ │ - CRUD操作       │   │
│  │ - AI问答    │ │ - JWT认证    │ │ - 数据验证       │   │
│  │ - BM25备选  │ │ - 权限管理    │ │                  │   │
│  └─────────────┘ └──────────────┘ └──────────────────┘   │
└────────────────────────┬──────────────────────────────────┘
                         │
┌────────────────────────▼──────────────────────────────────┐
│                      数据访问层 (Repository)                │
│  ┌─────────────┐ ┌──────────────┐ ┌──────────────────┐   │
│  │ 知识库仓库   │ │ 用户仓库      │ │ 业务数据仓库      │   │
│  │ - 知识存储   │ │ - 用户CRUD   │ │ - 景点/路线      │   │
│  │ - 向量存储   │ │ - 密码加密   │ │ - 游览记录       │   │
│  └─────────────┘ └──────────────┘ └──────────────────┘   │
└────────────────────────┬──────────────────────────────────┘
                         │
┌────────────────────────▼──────────────────────────────────┐
│                       数据存储层                            │
│  ┌─────────────────────┐  ┌─────────────────────────────┐ │
│  │   SQLite 数据库      │  │   知识库文件 (JSONL)        │ │
│  │   • 用户表          │  │   • lingshan_chunks.jsonl  │ │
│  │   • 景点表          │  │   • 预加载的景区知识        │ │
│  │   • 知识向量表       │  │   • 支持增量更新            │ │
│  └─────────────────────┘  └─────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                         │
┌────────────────────────▼──────────────────────────────────┐
│                      外部AI服务                             │
│  ┌──────────────────────┐  ┌──────────────────────────┐   │
│  │  DeepSeek Chat API   │  │  千问 Embedding API     │   │
│  │  (AI对话服务)         │  │  (向量嵌入服务)          │   │
│  │  <DEEPSEEK_API_KEY> │  │  <DASHSCOPE_API_KEY>    │   │
│  └──────────────────────┘  └──────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

### 技术栈

| 层级 | 技术选型 | 说明 |
|------|---------|------|
| **后端框架** | Gin (v1.10.0) | 轻量级Web框架，高性能路由 |
| **数据库** | SQLite + GORM | 嵌入式数据库，免安装维护 |
| **认证** | JWT | 无状态身份认证 |
| **配置管理** | Viper (v1.18.2) | 支持YAML配置和环境变量 |
| **AI服务** | DeepSeek API | AI对话能力 |
| **向量服务** | 千问 Embedding API | 文本向量化，支持RAG |
| **编程语言** | Go 1.22 | 高并发、编译型语言 |

---

## 📁 项目结构

```
scenic-guide/
├── main.go                    # 程序入口，初始化所有组件
├── configs/
│   └── config.yaml           # 应用配置文件
├── config/
│   └── keywords.go           # 灵山相关关键词配置
├── internal/                  # 内部包（核心业务逻辑）
│   ├── config/               # 配置加载
│   │   └── config.go
│   ├── handler/              # HTTP处理器（路由+业务逻辑）
│   │   ├── routes.go        # 路由配置
│   │   ├── ai_handler.go    # AI问答处理器
│   │   ├── user_handler.go  # 用户管理处理器
│   │   ├── scenic_spot_handler.go
│   │   ├── guide_content_handler.go
│   │   ├── tour_route_handler.go
│   │   ├── visitor_query_handler.go
│   │   └── tts_handler.go
│   ├── model/               # 数据模型
│   │   ├── models.go        # 核心业务模型
│   │   └── rag_models.go    # RAG专用模型
│   ├── pkg/                 # 公共工具包
│   │   ├── database.go      # 数据库连接
│   │   ├── jwt.go          # JWT认证
│   │   ├── middleware.go    # 中间件
│   │   ├── response.go     # 统一响应格式
│   │   └── logger.go
│   ├── repository/          # 数据访问层
│   │   ├──     # 知识库数据访问
│   │   ├── user.go
│   │   ├── scenic_spot.go
│   │   └── ...
│   └── service/             # 业务逻辑层
│       ├── rag_service.go   # RAG核心逻辑
│       ├── embedding_service.go  # 向量嵌入服务
│       ├── user_service.go
│       └── ...
├── knowledge/               # 知识库文件
│   ├── lingshan_chunks.jsonl   # 灵山景区知识（JSONL格式）
│   ├── lingshan_corpus.md      # 原始语料
│   ├── lingshan_eval_qa.json   # 评估问答对
│   └── lingshan_rag_guide.md   # RAG使用指南
├── static/                 # 静态资源
│   ├── index.html
│   ├── css/style.css
│   └── js/app.js
└── data/                   # 数据目录（运行时创建）
    └── example_db.db        # SQLite数据库文件
```

---

## 🔧 核心模块详解

### 1. RAG服务 (Retrieval-Augmented Generation)

RAG是本系统的核心，实现流程如下：

```
用户问题
    │
    ▼
┌─────────────────────────────────────┐
│  1. 知识检索                          │
│  ┌─────────────────────────────────┐ │
│  │ a. 向量相似度检索（优先）          │ │
│  │    • 使用千问API生成查询向量       │ │
│  │    • 计算余弦相似度               │ │
│  │    • 返回Top-K相关知识片段        │ │
│  ├─────────────────────────────────┤ │
│  │ b. BM25备选检索（当向量服务不可用） │ │
│  │    • 中文n-gram分词              │ │
│  │    • 关键词权重计算              │ │
│  └─────────────────────────────────┘ │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  2. 提示词构建                        │
│  • 系统角色设定（景区导览助手）         │
│  • 注入检索到的知识片段                │
│  • 用户问题                           │
│  • 回答约束（不编造、不提到技术细节）   │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  3. AI生成                           │
│  • 调用DeepSeek Chat API             │
│  • 60秒超时保护                      │
│  • 异常时使用本地知识片段回答          │
└─────────────────┬───────────────────┘
                  │
                  ▼
              AI回答
```

#### 关键代码位置

- **RAG服务入口**：[rag_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/rag_service.go#L438-L548)
- **知识检索**：[rag_service.go#L310-L385](file:///d:/go%20web%2001/scenic-guide/internal/service/rag_service.go#L310-L385)
- **提示词构建**：[rag_service.go#L402-L436](file:///d:/go%20web%2001/scenic-guide/internal/service/rag_service.go#L402-L436)
- **向量嵌入服务**：[embedding_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/embedding_service.go)

#### 向量检索 vs BM25备选

| 特性 | 向量检索 (千问) | BM25备选 |
|------|----------------|----------|
| **精度** | 高，适合语义相似 | 中，依赖关键词匹配 |
| **速度** | 依赖网络API | 本地元计算，快 |
| **成本** | API调用费用 | 免费 |
| **可用性** | 需要API Key | 始终可用 |
| **中文支持** | ✅ 优秀 | ✅ 良好（n-gram） |

### 2. 向量嵌入服务

```go
// EmbeddingProvider 接口定义
type EmbeddingProvider interface {
    GenerateEmbedding(text string) ([]float64, error)
    Name() string
    IsAvailable() bool
}
```

当前实现：
- **QwenEmbeddingProvider**：调用千问text-embedding-v4 API
- **BM25FallbackProvider**：本地BM25算法，当向量API不可用时使用

### 3. 用户认证系统

#### JWT认证流程

```
用户登录
    │
    ▼
┌─────────────────────────────────────┐
│  POST /api/v1/user/login            │
│  Body: {username, password}          │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  验证用户名密码                       │
│  • 数据库查询                        │
│  • 密码bcrypt比对                   │
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│  生成JWT Token                       │
│  • Payload: user_id, username, role │
│  • Expire: 24小时                   │
│  • Algorithm: HS256                  │
└─────────────────┬───────────────────┘
                  │
                  ▼
              返回Token给客户端
```

#### 权限层级

| 角色 | 权限说明 |
|------|---------|
| `visitor` | 普通游客，可查看公开信息、提交咨询 |
| `admin` | 管理员，可管理所有数据、用户 |

### 4. 数据库模型

#### 核心业务模型

```go
// 景区景点
type ScenicSpot struct {
    ID          uint     // 主键
    Name        string   // 景点名称
    Description string   // 景点描述
    Location    string   // 位置
    Category    string   // 分类
    Rating      float64  // 评分
    Price       float64  // 票价
    ImageURL    string   // 图片URL
}

// 导览内容
type GuideContent struct {
    ID         uint    // 主键
    SpotID     uint    // 关联景点ID
    Title      string  // 标题
    Content    string  // 内容（富文本）
    Type       string  // 内容类型（文本/音频/视频）
    AudioURL   string  // 音频URL（TTS生成）
    Duration   int     // 时长（秒）
}

// 游览路线
type TourRoute struct {
    ID          uint    // 主键
    Name        string  // 路线名称
    Description string  // 路线描述
    Spots       string  // 途经景点（JSON数组）
    Duration    int     // 预计时长（分钟）
    Difficulty  string  // 难度等级
    Rating      float64 // 评分
}

// 游客咨询
type VisitorQuery struct {
    ID          uint   // 主键
    Query       string // 游客问题
    Response    string // AI回答
    SpotID      uint   // 关联景点
    IsAnswered  bool   // 是否已回答
}

// 用户
type User struct {
    ID       uint   // 主键
    Username string // 用户名（唯一）
    Password string // 密码（bcrypt加密）
    Email    string // 邮箱
    Role     string // 角色（visitor/admin）
}
```

#### RAG专用模型

```go
// 知识片段
type KnowledgeChunk struct {
    ID        string    // 知识ID（唯一）
    Content   string    // 知识内容
    Source    string    // 来源
    Title     string    // 标题
    Metadata  string    // 元数据（JSON格式）
    Vector    string    // 向量（JSON格式的float64数组）
    CreatedAt time.Time // 创建时间
}
```

---

## 🌐 API接口文档

### 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: Bearer Token (JWT)
- **通用Headers**:
  ```
  Content-Type: application/json
  Authorization: Bearer <token>  # 仅需要认证的接口
  ```

### 通用响应格式

```json
{
    "code": 0,           // 状态码，0=成功，非0=失败
    "message": "success", // 消息描述
    "data": {}           // 业务数据
}
```

### 状态码说明

| HTTP状态码 | code值 | 说明 |
|-----------|--------|------|
| 200 | 0 | 成功 |
| 200 | 400 | 请求参数错误 |
| 401 | 401 | 未登录/Token无效 |
| 403 | 403 | 权限不足 |
| 404 | 404 | 资源不存在 |
| 500 | 500 | 服务器内部错误 |

---

### 用户相关接口

#### 1. 用户注册
```
POST /user/register
Body: {
    "username": "testuser",
    "password": "password123",
    "email": "test@example.com",
    "role": "visitor"  // 可选，默认visitor
}
Response: {
    "code": 0,
    "message": "注册成功"
}
```

#### 2. 用户登录
```
POST /user/login
Body: {
    "username": "testuser",
    "password": "password123"
}
Response: {
    "code": 0,
    "data": {
        "id": 1,
        "username": "testuser",
        "email": "test@example.com",
        "role": "visitor",
        "token": "eyJhbGciOiJIUzI1NiIs..."
    }
}
```

#### 3. 获取当前用户信息（需认证）
```
GET /user/me
Headers: Authorization: Bearer <token>
Response: {
    "code": 0,
    "data": {
        "id": 1,
        "username": "testuser",
        "role": "visitor"
    }
}
```

#### 4. 获取所有用户（需管理员权限）
```
GET /admin/users
Headers: Authorization: Bearer <token>
Response: {
    "code": 0,
    "data": [
        {"id": 1, "username": "admin", "role": "admin"},
        {"id": 2, "username": "visitor1", "role": "visitor"}
    ]
}
```

---

### AI问答接口

#### 1. 智能问答（核心接口）
```
POST /ai/chat
Body: {
    "message": "灵山大佛有多高？"
}
Response: {
    "code": 0,
    "data": {
        "response": "灵山大佛高88米，主体高79米，莲花瓣高9米..."
    }
}
```

#### 2. 上传知识文件
```
POST /knowledge/upload
Headers: Authorization: Bearer <token> (需管理员权限)
Content-Type: multipart/form-data
Body: file: <lingshan_chunks.jsonl>
Response: {
    "code": 0,
    "data": {
        "file_path": "./knowledge/lingshan_chunks.jsonl",
        "loaded_count": 50,
        "message": "知识上传并加载成功"
    }
}
```

#### 3. 获取知识列表
```
GET /knowledge/list?page=1&page_size=20
Headers: Authorization: Bearer <token> (需管理员权限)
Response: {
    "code": 0,
    "data": {
        "list": [
            {
                "id": "chunk_001",
                "title": "灵山大佛介绍",
                "content": "灵山大佛高88米...",
                "source": "景区官网"
            }
        ],
        "total": 50,
        "page": 1,
        "page_size": 20
    }
}
```

#### 4. 获取单条知识
```
GET /knowledge/:id
Headers: Authorization: Bearer <token> (需管理员权限)
Response: {
    "code": 0,
    "data": {
        "knowledge": {
            "id": "chunk_001",
            "title": "灵山大佛介绍",
            "content": "灵山大佛高88米..."
        }
    }
}
```

#### 5. 删除单条知识
```
DELETE /knowledge/:id
Headers: Authorization: Bearer <token> (需管理员权限)
Response: {
    "code": 0,
    "message": "知识删除成功"
}
```

#### 6. 清空所有知识
```
DELETE /knowledge/all
Headers: Authorization: Bearer <token> (需管理员权限)
Response: {
    "code": 0,
    "message": "知识清空成功"
}
```

---

### 其他业务接口

#### 景点管理
```
GET    /spots              # 获取所有景点
GET    /spots/:id          # 获取单个景点
POST   /spots             # 创建景点（需管理员）
PUT    /spots/:id         # 更新景点（需管理员）
DELETE /spots/:id         # 删除景点（需管理员）
```

#### 导览内容
```
GET    /guide-contents           # 获取导览内容列表
GET    /guide-contents/:id       # 获取单条导览内容
POST   /guide-contents           # 创建导览内容（需管理员）
PUT    /guide-contents/:id       # 更新导览内容（需管理员）
DELETE /guide-contents/:id       # 删除导览内容（需管理员）
```

#### 游览路线
```
GET    /tour-routes             # 获取所有路线
GET    /tour-routes/:id         # 获取单条路线
POST   /tour-routes             # 创建路线（需管理员）
PUT    /tour-routes/:id         # 更新路线（需管理员）
DELETE /tour-routes/:id         # 删除路线（需管理员）
```

---

### 系统接口

#### 健康检查
```
GET /health
Response: {
    "code": 0,
    "message": "景区导览服务运行正常",
    "status": "ok"
}
```

---

## ⚙️ 配置说明

### 配置文件位置
```
configs/config.yaml
```

### 配置项详解

```yaml
server:
  host: "0.0.0.0"    # 监听地址
  port: "8080"       # 监听端口

database:
  driver: "sqlite"   # 数据库驱动（仅支持sqlite）
  path: "./data/example_db.db"  # 数据库文件路径

logging:
  level: "debug"     # 日志级别：debug/info/warn/error
  output: "console"  # 输出目标：console/file

ai:
  api_key: "<DEEPSEEK_API_KEY>"  # DeepSeek API Key
  model: "deepseek-chat"    # AI模型
  base_url: "https://api.deepseek.com/v1"  # API地址

embedding:
  api_key: "<DASHSCOPE_API_KEY>"  # 千问API Key
  model: "text-embedding-v4"   # 嵌入模型
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"  # API地址

speech:
  api_key: ""     # 语音合成API Key（预留）
  region: ""      # 区域（预留）

security:
  jwt_secret: "your-secret-key-here"  # JWT密钥
  token_expire_hours: 24  # Token有效期（小时）
```

---

## 🚀 部署指南

### 环境要求

- **Go版本**: >= 1.22
- **操作系统**: Windows / Linux / macOS
- **网络**: 能访问DeepSeek和千问API

### 快速启动

#### 1. 克隆项目
```bash
git clone <repository-url>
cd scenic-guide
```

#### 2. 安装依赖
```bash
go mod download
```

#### 3. 修改配置
编辑 `configs/config.yaml`，填入你的API Key：
```yaml
ai:
  api_key: "你的DeepSeek API Key"
  model: "deepseek-chat"
  base_url: "https://api.deepseek.com/v1"

embedding:
  api_key: "你的千问API Key"
  model: "text-embedding-v4"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
```

#### 4. 启动服务
```bash
go run main.go
```

#### 5. 验证服务
```bash
# 健康检查
curl http://localhost:8080/health

# 测试AI问答
curl -X POST http://localhost:8080/api/v1/ai/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"灵山大佛有多高"}'
```

### 端口占用问题

如果启动时提示端口被占用：
```bash
# Windows 查看端口占用
netstat -ano | findstr ":8080"

# 终止占用进程
taskkill /F /PID <进程ID>
```

---

## 📊 数据库设计

### E-R图

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│   User      │       │  ScenicSpot │       │  TourRoute  │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ id (PK)     │       │ id (PK)     │       │ id (PK)     │
│ username    │       │ name        │       │ name        │
│ password    │       │ description │       │ description │
│ email       │       │ location    │       │ spots       │
│ role        │       │ category    │       │ duration    │
└─────────────┘       │ rating      │       │ difficulty  │
                      │ price       │       │ rating      │
                      └─────────────┘       └─────────────┘
                             │
                             │ 1:N
                             ▼
                      ┌─────────────┐
                      │GuideContent │
                      ├─────────────┤
                      │ id (PK)     │
                      │ spot_id (FK)│
                      │ title       │
                      │ content     │
                      │ type        │
                      │ audio_url   │
                      └─────────────┘

┌─────────────┐       ┌─────────────────┐
│VisitorQuery │       │  KnowledgeChunk │
├─────────────┤       ├─────────────────┤
│ id (PK)     │       │ id (PK)         │
│ query       │       │ content         │
│ response    │       │ source          │
│ spot_id     │       │ title           │
│ is_answered │       │ metadata        │
└─────────────┘       │ vector          │
                      └─────────────────┘
```

### 数据库表清单

| 表名 | 说明 | 记录类型 |
|------|------|---------|
| scenic_spots | 景区景点表 | 景点基本信息 |
| guide_contents | 导览内容表 | 景点介绍内容 |
| tour_routes | 游览路线表 | 推荐游览路线 |
| visitor_queries | 游客咨询表 | 问答记录 |
| users | 用户表 | 系统用户 |
| visit_records | 访问记录表 | 游客访问记录 |
| system_logs | 系统日志表 | 操作日志 |
| knowledge_chunks | 知识向量表 | RAG知识库 |

---

## 🔍 RAG系统工作流程详解

### 完整查询流程

```
1. 用户发送问题
   "灵山大佛有多高？"

2. 意图识别（可选）
   检查是否与灵山相关（通过关键词匹配）
   LingshanRelatedKeywords = ["灵山", "大佛", "九龙灌", ...]

3. 知识检索
   ├─ 如果Embedding可用：
   │   a. 调用千问API生成查询向量
   │   b. 遍历所有知识片段
   │   c. 计算余弦相似度
   │   d. 返回Top-5最相关片段
   │
   └─ 否则使用BM25：
       a. 对查询和文档进行n-gram分词
       b. 计算TF-IDF权重
       c. 返回Top-5最相关片段

4. 相似度阈值判断
   if 最高相似度 < 0.01:
       跳过知识库，使用通用AI模式

5. 提示词构建
   System: 你是灵山胜境景区智能导览助手...
   Context: [检索到的知识片段]
   Query: 灵山大佛有多高？

6. 调用DeepSeek API
   POST https://api.deepseek.com/v1/chat/completions
   Headers: Authorization: Bearer <API_KEY>
   Body: {
       "model": "deepseek-chat",
       "messages": [
           {"role": "system", "content": "..."},
           {"role": "user", "content": "..."}
       ]
   }

7. 返回AI回答
   "灵山大佛高88米，主体高79米，莲花瓣高9米..."
```

### 知识库格式

知识库使用JSONL格式，每行一条记录：

```jsonl
{"id": "chunk_001", "title": "灵山大佛", "content": "灵山大佛高88米，主体高79米...", "source": "景区官网", "metadata": {"category": "景点介绍"}}
{"id": "chunk_002", "title": "门票价格", "content": "灵山胜境门票价格为210元/人...", "source": "景区官网", "metadata": {"category": "票务信息"}}
```

---

## 🔒 安全说明

### 密码存储
- 使用bcrypt加密用户密码
- 永不明文存储或传输密码

### JWT安全
- Token有效期24小时
- 使用HS256签名算法
- Token中包含用户ID、用户名、角色

### API安全
- 敏感接口需要认证
- 管理接口需要admin权限
- 建议生产环境使用HTTPS

### 建议的生产环境配置

```yaml
security:
  jwt_secret: "<随机生成的32位密钥>"  # 使用随机密钥
  token_expire_hours: 2  # 缩短Token有效期

ai:
  base_url: "https://api.deepseek.com/v1"  # 使用HTTPS

logging:
  level: "info"  # 生产环境使用info级别
```

---

## 🛠️ 开发指南

### 添加新的处理器

1. 在 `internal/handler/` 创建 `xxx_handler.go`
2. 定义Handler结构体和方法
3. 在 `handler.Routes()` 注册路由

```go
// 示例：internal/handler/example_handler.go
package handler

type ExampleHandler struct {}

func NewExampleHandler() *ExampleHandler {
    return &ExampleHandler{}
}

func (h *ExampleHandler) Routes(r *gin.RouterGroup) {
    r.GET("/example", h.GetExample)
}

func (h *ExampleHandler) GetExample(c *gin.Context) {
    // 业务逻辑
}
```

### 添加新的数据模型

1. 在 `internal/model/models.go` 添加结构体
2. 在 `model.AutoMigrate()` 注册模型

```go
// 示例
type Example struct {
    ID      uint    `gorm:"primaryKey"`
    Name    string  `gorm:"size:255;not null"`
    Content string  `gorm:"text"`
}

// 在AutoMigrate中添加
func AutoMigrate(db *gorm.DB) error {
    return db.AutoMigrate(
        // ... 其他模型
        &Example{},
    )
}
```

---

## 📝 更新日志

### v1.0.0 (2024-当前版本)

- ✅ 实现完整的RAG知识问答系统
- ✅ 集成DeepSeek Chat API作为AI引擎
- ✅ 集成千问Embedding API作为向量服务
- ✅ 实现BM25备选检索（当向量服务不可用时）
- ✅ 完整的用户认证和权限管理系统
- ✅ 景区景点、导览内容、游览路线管理
- ✅ 知识库上传、查询、删除管理功能
- ✅ 统一JSON响应格式
- ✅ SQLite数据库集成

---

## 🤝 贡献指南

欢迎提交Issue和Pull Request！

### 提交Issue时请包含

- 问题描述
- 复现步骤
- 预期行为
- 实际行为
- 环境信息（操作系统、Go版本等）

---

## 📧 联系方式

如有问题，请通过以下方式联系：

- 项目Issues: GitHub Issues
- 邮箱: support@example.com

---

## 📄 许可证

本项目仅供学习交流使用。

---

*最后更新: 2024年5月*
