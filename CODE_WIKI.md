# 景区导览服务 Code Wiki

## 目录

1. [项目概述](#项目概述)
2. [项目架构](#项目架构)
3. [目录结构](#目录结构)
4. [核心模块详解](#核心模块详解)
   - [配置模块 (config)](#配置模块-config)
   - [数据模型层 (model)](#数据模型层-model)
   - [数据访问层 (repository)](#数据访问层-repository)
   - [业务逻辑层 (service)](#业务逻辑层-service)
   - [控制器层 (handler)](#控制器层-handler)
   - [公共工具包 (pkg)](#公共工具包-pkg)
5. [关键类与函数说明](#关键类与函数说明)
6. [API接口文档](#api接口文档)
7. [依赖关系](#依赖关系)
8. [项目运行方式](#项目运行方式)
9. [RAG知识库系统](#rag知识库系统)

---

## 项目概述

**景区导览服务 (Scenic Guide)** 是一个基于Go语言开发的智能景区导览系统，集成了AI数字人、RAG（检索增强生成）知识库、语音合成等功能。该系统为游客提供智能化的景区导览服务，包括景点介绍、路线规划、问题解答等核心功能。

### 核心特性

- **AI智能对话**: 基于大语言模型的智能问答系统
- **RAG知识库**: 检索增强生成技术，提供准确的景区知识回答
- **多模态交互**: 支持文字、语音等多种交互方式
- **用户权限管理**: 完整的用户认证与授权系统
- **景点管理**: 景点信息、导览内容、游览路线的完整管理

### 技术栈

| 技术 | 用途 |
|------|------|
| Go 1.22 | 主要开发语言 |
| Gin | Web框架 |
| GORM | ORM框架 |
| SQLite | 数据库 |
| Viper | 配置管理 |
| JWT | 身份认证 |
| bcrypt | 密码加密 |

---

## 项目架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                         客户端 (Web Browser)                      │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Handler Layer (控制器层)                      │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │
│  │ AIHandler    │ │ UserHandler  │ │ ScenicSpot   │ ...          │
│  └──────────────┘ └──────────────┘ └──────────────┘              │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Service Layer (业务逻辑层)                    │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │
│  │ RAGService   │ │ UserService  │ │ ScenicSpot   │ ...          │
│  └──────────────┘ └──────────────┘ └──────────────┘              │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Repository Layer (数据访问层)                   │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐              │
│  │ KnowledgeRepo│ │ UserRepo     │ │ ScenicSpot   │ ...          │
│  └──────────────┘ └──────────────┘ └──────────────┘              │
└─────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Database (SQLite)                           │
└─────────────────────────────────────────────────────────────────┘
```

### 分层架构说明

项目采用经典的**分层架构**设计，遵循**依赖倒置原则**：

1. **Handler层**: 处理HTTP请求，参数验证，调用Service层
2. **Service层**: 业务逻辑处理，事务管理
3. **Repository层**: 数据访问，数据库操作
4. **Model层**: 数据模型定义
5. **Pkg层**: 公共工具组件

---

## 目录结构

```
scenic-guide/
├── config/                    # 全局配置
│   └── keywords.go           # 灵山景区相关关键词配置
├── internal/                  # 内部模块
│   ├── config/               # 配置加载
│   │   └── config.go         # 配置结构定义与加载
│   ├── handler/              # 控制器层
│   │   ├── ai_handler.go     # AI对话处理器
│   │   ├── guide_content_handler.go  # 导览内容处理器
│   │   ├── routes.go         # 路由注册
│   │   ├── scenic_spot_handler.go    # 景点处理器
│   │   ├── tour_route_handler.go     # 游览路线处理器
│   │   ├── tts_handler.go    # 语音合成处理器
│   │   ├── user_handler.go   # 用户处理器
│   │   └── visitor_query_handler.go  # 游客问题处理器
│   ├── model/                # 数据模型层
│   │   ├── models.go         # 核心数据模型
│   │   └── rag_models.go     # RAG相关模型
│   ├── pkg/                  # 公共工具包
│   │   ├── database.go       # 数据库初始化
│   │   ├── jwt.go            # JWT工具
│   │   ├── logger.go         # 日志工具
│   │   ├── middleware.go     # 中间件
│   │   └── response.go       # 响应工具
│   ├── repository/           # 数据访问层
│   │   ├── guide_content.go  # 导览内容仓储
│   │   ├── knowledge.go      # 知识库仓储
│   │   ├── scenic_spot.go    # 景点仓储
│   │   ├── tour_route.go     # 游览路线仓储
│   │   ├── user.go           # 用户仓储
│   │   └── visitor_query.go  # 游客问题仓储
│   └── service/              # 业务逻辑层
│       ├── embedding_service.go     # 向量嵌入服务
│       ├── guide_content_service.go # 导览内容服务
│       ├── rag_service.go    # RAG服务
│       ├── scenic_spot_service.go   # 景点服务
│       ├── tour_route_service.go    # 游览路线服务
│       ├── user_service.go   # 用户服务
│       └── visitor_query_service.go # 游客问题服务
├── knowledge/                 # 知识库数据
│   ├── lingshan_chunks.jsonl # 灵山景区知识片段
│   ├── lingshan_corpus.md    # 灵山景区语料
│   ├── lingshan_eval_qa.json # 评估问答集
│   └── lingshan_rag_guide.md # RAG指南
├── static/                    # 静态资源
│   ├── css/style.css         # 样式文件
│   ├── js/app.js             # 前端脚本
│   └── index.html            # 主页面
├── go.mod                     # Go模块定义
├── main.go                    # 程序入口
└── .gitignore                 # Git忽略配置
```

---

## 核心模块详解

### 配置模块 (config)

#### 配置结构定义

**文件**: [internal/config/config.go](file:///d:/go%20web%2001/scenic-guide/internal/config/config.go)

```go
type Config struct {
    Server     ServerConfig     // 服务器配置
    Database   DatabaseConfig   // 数据库配置
    Logging    LoggingConfig    // 日志配置
    AI         AIConfig         // AI服务配置
    Embedding  EmbeddingConfig  // 向量嵌入配置
    Speech     SpeechConfig     // 语音服务配置
    Security   SecurityConfig   // 安全配置
}
```

#### 配置项说明

| 配置项 | 说明 | 示例值 |
|--------|------|--------|
| Server.Port | 服务端口 | "8080" |
| Server.Host | 服务主机 | "0.0.0.0" |
| Database.Driver | 数据库驱动 | "sqlite" |
| Database.Path | 数据库路径 | "./data/scenic.db" |
| AI.APIKey | AI服务API密钥 | "sk-xxx" |
| AI.Model | AI模型名称 | "qwen-turbo" |
| AI.BaseURL | AI服务地址 | "https://dashscope.aliyuncs.com/compatible-mode/v1" |
| Security.JWTSecret | JWT密钥 | "your-secret-key" |

#### 关键词配置

**文件**: [config/keywords.go](file:///d:/go%20web%2001/scenic-guide/config/keywords.go)

定义了灵山景区相关的关键词列表，用于RAG系统判断问题相关性：

```go
var LingshanRelatedKeywords = []string{
    "灵山", "大佛", "九龙灌", "佛教", "佛文化", "祥符寺",
    "灵山胜境", "拈花湾", "禅意小镇", "门票", "票价", ...
}
```

---

### 数据模型层 (model)

#### 核心数据模型

**文件**: [internal/model/models.go](file:///d:/go%20web%2001/scenic-guide/internal/model/models.go)

##### ScenicSpot (景点)

```go
type ScenicSpot struct {
    ID          uint      // 主键
    Name        string    // 景点名称
    Description string    // 景点描述
    Location    string    // 位置
    Category    string    // 分类
    Rating      float64   // 评分
    Price       float64   // 价格
    ImageURL    string    // 图片URL
    CreatedAt   time.Time // 创建时间
    UpdatedAt   time.Time // 更新时间
}
```

##### GuideContent (导览内容)

```go
type GuideContent struct {
    ID          uint      // 主键
    SpotID      uint      // 关联景点ID
    Title       string    // 标题
    Content     string    // 内容
    Type        string    // 类型
    AudioURL    string    // 音频URL
    Duration    int       // 时长
    CreatedAt   time.Time // 创建时间
    UpdatedAt   time.Time // 更新时间
}
```

##### TourRoute (游览路线)

```go
type TourRoute struct {
    ID          uint      // 主键
    Name        string    // 路线名称
    Description string    // 路线描述
    Spots       string    // 包含景点(JSON)
    Duration    int       // 预计时长
    Difficulty  string    // 难度等级
    Rating      float64   // 评分
    CreatedAt   time.Time // 创建时间
    UpdatedAt   time.Time // 更新时间
}
```

##### User (用户)

```go
type User struct {
    ID        uint      // 主键
    Username  string    // 用户名(唯一)
    Password  string    // 密码(加密)
    Email     string    // 邮箱
    Role      string    // 角色(visitor/admin)
    CreatedAt time.Time // 创建时间
    UpdatedAt time.Time // 更新时间
}
```

##### VisitorQuery (游客问题)

```go
type VisitorQuery struct {
    ID         uint      // 主键
    Query      string    // 问题内容
    Response   string    // 回答内容
    SpotID     uint      // 关联景点ID
    IsAnswered bool      // 是否已回答
    CreatedAt  time.Time // 创建时间
}
```

#### RAG数据模型

**文件**: [internal/model/rag_models.go](file:///d:/go%20web%2001/scenic-guide/internal/model/rag_models.go)

##### KnowledgeChunk (知识片段)

```go
type KnowledgeChunk struct {
    ID        string    // 主键(UUID)
    Content   string    // 知识内容
    Source    string    // 来源
    Title     string    // 标题
    Metadata  string    // 元数据(JSON)
    Vector    string    // 向量表示(JSON)
    CreatedAt time.Time // 创建时间
    UpdatedAt time.Time // 更新时间
}
```

---

### 数据访问层 (repository)

Repository层负责与数据库交互，采用**接口抽象**设计，便于测试和扩展。

#### 接口定义示例

**文件**: [internal/repository/scenic_spot.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/scenic_spot.go)

```go
type ScenicSpotRepository interface {
    Create(spot *model.ScenicSpot) error
    FindByID(id uint) (*model.ScenicSpot, error)
    FindAll() ([]model.ScenicSpot, error)
    FindByCategory(category string) ([]model.ScenicSpot, error)
    Update(spot *model.ScenicSpot) error
    Delete(id uint) error
}
```

#### Repository模块一览

| 文件 | 接口 | 说明 |
|------|------|------|
| [scenic_spot.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/scenic_spot.go) | ScenicSpotRepository | 景点数据访问 |
| [guide_content.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/guide_content.go) | GuideContentRepository | 导览内容数据访问 |
| [tour_route.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/tour_route.go) | TourRouteRepository | 游览路线数据访问 |
| [user.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/user.go) | UserRepository | 用户数据访问 |
| [visitor_query.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/visitor_query.go) | VisitorQueryRepository | 游客问题数据访问 |
| [knowledge.go](file:///d:/go%20web%2001/scenic-guide/internal/repository/knowledge.go) | KnowledgeRepository | 知识库数据访问 |

---

### 业务逻辑层 (service)

Service层实现核心业务逻辑，是系统的核心部分。

#### RAG服务 (核心)

**文件**: [internal/service/rag_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/rag_service.go)

RAGService是系统最核心的服务，实现了检索增强生成功能：

```go
type RAGService struct {
    repo        *repository.KnowledgeRepository  // 知识库仓储
    chatAPIKey  string                           // AI API密钥
    chatModel   string                           // AI模型
    chatBaseURL string                           // AI服务地址
    embedding   EmbeddingProvider                // 向量嵌入提供者
    bm25        *BM25FallbackProvider            // BM25备用检索
    useBM25     bool                             // 是否使用BM25
    uploadDir   string                           // 上传目录
    httpClient  *http.Client                     // HTTP客户端
}
```

**核心方法**:

| 方法 | 说明 |
|------|------|
| `QueryWithRAG(query string)` | 使用RAG回答问题 |
| `RetrieveRelevantKnowledge(query string, topK int)` | 检索相关知识 |
| `LoadKnowledgeFromFile(filePath string)` | 从文件加载知识 |
| `GenerateEmbedding(text string)` | 生成文本向量 |
| `BuildRAGPrompt(query string, chunks []KnowledgeChunk)` | 构建RAG提示词 |

#### 向量嵌入服务

**文件**: [internal/service/embedding_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/embedding_service.go)

```go
type EmbeddingProvider interface {
    GenerateEmbedding(text string) ([]float64, error)
    Name() string
    IsAvailable() bool
}
```

提供两种向量嵌入实现：
- **QwenEmbeddingProvider**: 阿里通义千问向量API
- **BM25FallbackProvider**: 本地BM25算法备用方案

#### Service模块一览

| 文件 | 接口 | 说明 |
|------|------|------|
| [rag_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/rag_service.go) | RAGService | RAG检索增强生成 |
| [embedding_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/embedding_service.go) | EmbeddingProvider | 向量嵌入服务 |
| [scenic_spot_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/scenic_spot_service.go) | ScenicSpotService | 景点业务逻辑 |
| [guide_content_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/guide_content_service.go) | GuideContentService | 导览内容业务逻辑 |
| [tour_route_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/tour_route_service.go) | TourRouteService | 游览路线业务逻辑 |
| [user_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/user_service.go) | UserService | 用户业务逻辑 |
| [visitor_query_service.go](file:///d:/go%20web%2001/scenic-guide/internal/service/visitor_query_service.go) | VisitorQueryService | 游客问题业务逻辑 |

---

### 控制器层 (handler)

Handler层处理HTTP请求，负责参数验证和响应封装。

#### 路由注册

**文件**: [internal/handler/routes.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/routes.go)

```go
func SetupRoutes(r *gin.Engine, handlers *Handlers) {
    r.Static("/static", "./static")
    r.GET("/", func(c *gin.Context) {
        c.File("./static/index.html")
    })

    api := r.Group("/api/v1")

    handlers.ScenicSpot.Routes(api)
    handlers.GuideContent.Routes(api)
    handlers.TourRoute.Routes(api)
    handlers.VisitorQuery.Routes(api)
    handlers.User.Routes(api)
    handlers.AI.Routes(api)
    handlers.TTS.Routes(api)

    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status":  "ok",
            "message": "景区导览服务运行正常",
        })
    })
}
```

#### Handlers结构

```go
type Handlers struct {
    ScenicSpot   *ScenicSpotHandler
    GuideContent *GuideContentHandler
    TourRoute    *TourRouteHandler
    VisitorQuery *VisitorQueryHandler
    User         *UserHandler
    AI           *AIHandler
    TTS          *TTSHandler
}
```

#### Handler模块一览

| 文件 | 说明 |
|------|------|
| [ai_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/ai_handler.go) | AI对话、知识库管理 |
| [user_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/user_handler.go) | 用户注册、登录、管理 |
| [scenic_spot_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/scenic_spot_handler.go) | 景点CRUD操作 |
| [guide_content_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/guide_content_handler.go) | 导览内容管理 |
| [tour_route_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/tour_route_handler.go) | 游览路线管理 |
| [visitor_query_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/visitor_query_handler.go) | 游客问题管理 |
| [tts_handler.go](file:///d:/go%20web%2001/scenic-guide/internal/handler/tts_handler.go) | 语音合成服务 |

---

### 公共工具包 (pkg)

#### 数据库工具

**文件**: [internal/pkg/database.go](file:///d:/go%20web%2001/scenic-guide/internal/pkg/database.go)

```go
var DB *gorm.DB

func InitDatabase(cfg *config.DatabaseConfig) error
```

#### JWT工具

**文件**: [internal/pkg/jwt.go](file:///d:/go%20web%2001/scenic-guide/internal/pkg/jwt.go)

```go
type Claims struct {
    UserID   uint
    Username string
    Role     string
    jwt.RegisteredClaims
}

func GenerateToken(userID uint, username, role string, expireHours int) (string, error)
func ParseToken(tokenString string) (*Claims, error)
```

#### 中间件

**文件**: [internal/pkg/middleware.go](file:///d:/go%20web%2001/scenic-guide/internal/pkg/middleware.go)

```go
func AuthMiddleware() gin.HandlerFunc    // JWT认证中间件
func AdminMiddleware() gin.HandlerFunc   // 管理员权限中间件
```

#### 响应工具

**文件**: [internal/pkg/response.go](file:///d:/go%20web%2001/scenic-guide/internal/pkg/response.go)

```go
type Response struct {
    Code    int
    Message string
    Data    interface{}
}

func Success(c *gin.Context, data interface{})
func BadRequest(c *gin.Context, message string)
func Unauthorized(c *gin.Context, message string)
func Forbidden(c *gin.Context, message string)
func NotFound(c *gin.Context, message string)
func InternalError(c *gin.Context, message string)
```

#### 日志工具

**文件**: [internal/pkg/logger.go](file:///d:/go%20web%2001/scenic-guide/internal/pkg/logger.go)

```go
var Logger *slog.Logger

func InitLogger(level string)
```

---

## 关键类与函数说明

### RAGService 核心函数

#### QueryWithRAG

```go
func (s *RAGService) QueryWithRAG(query string) (string, error)
```

**功能**: 使用RAG技术回答用户问题

**流程**:
1. 调用 `RetrieveRelevantKnowledge` 检索相关知识片段
2. 如果没有检索到相关知识，调用 `QueryGeneralChat` 使用通用AI回答
3. 构建RAG提示词，包含知识库上下文
4. 调用AI API生成回答

#### RetrieveRelevantKnowledge

```go
func (s *RAGService) RetrieveRelevantKnowledge(query string, topK int) ([]model.KnowledgeChunk, error)
```

**功能**: 检索与问题最相关的知识片段

**流程**:
1. 获取所有知识片段
2. 计算查询向量与知识片段的相似度
3. 按相似度排序，返回TopK结果

### UserService 核心函数

#### Register

```go
func (s *userService) Register(user *model.User) error
```

**功能**: 用户注册

**流程**:
1. 检查用户名是否已存在
2. 使用bcrypt加密密码
3. 保存用户到数据库

#### Login

```go
func (s *userService) Login(username, password string) (*model.User, error)
```

**功能**: 用户登录验证

**流程**:
1. 根据用户名查找用户
2. 验证密码
3. 返回用户信息

---

## API接口文档

### 基础路径

所有API接口的基础路径为: `/api/v1`

### 用户相关接口

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/register` | 用户注册 | 否 |
| POST | `/login` | 用户登录 | 否 |
| GET | `/user/me` | 获取当前用户 | 是 |
| GET | `/users/:id` | 获取用户信息 | 是 |
| PUT | `/users/:id` | 更新用户信息 | 是 |
| DELETE | `/users/:id` | 删除用户 | 是 |
| GET | `/admin/users` | 获取所有用户(管理员) | 是 |
| GET | `/admin/users/role` | 按角色获取用户(管理员) | 是 |

### 景点相关接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/spots` | 创建景点 |
| GET | `/spots` | 获取所有景点 |
| GET | `/spots/category` | 按分类获取景点 |
| GET | `/spots/:id` | 获取单个景点 |
| PUT | `/spots/:id` | 更新景点 |
| DELETE | `/spots/:id` | 删除景点 |

### 导览内容接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/contents` | 创建导览内容 |
| GET | `/contents/:id` | 获取导览内容 |
| GET | `/contents/spot/:spot_id` | 按景点获取内容 |
| GET | `/contents/spot/:spot_id/type` | 按景点和类型获取内容 |
| PUT | `/contents/:id` | 更新导览内容 |
| DELETE | `/contents/:id` | 删除导览内容 |

### 游览路线接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/routes` | 创建路线 |
| GET | `/routes` | 获取所有路线 |
| GET | `/routes/difficulty` | 按难度获取路线 |
| GET | `/routes/:id` | 获取单个路线 |
| PUT | `/routes/:id` | 更新路线 |
| DELETE | `/routes/:id` | 删除路线 |

### AI对话接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/ai/chat` | AI对话 |
| POST | `/knowledge/upload` | 上传知识文件 |
| GET | `/knowledge/list` | 获取知识列表 |
| GET | `/knowledge/:id` | 获取知识详情 |
| DELETE | `/knowledge/:id` | 删除知识 |
| DELETE | `/knowledge/all` | 清空知识库 |

### 语音合成接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/ai/tts` | 文字转语音 |

### 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 服务健康检查 |

---

## 依赖关系

### 主要依赖

```go
require (
    github.com/gin-gonic/gin v1.10.0           // Web框架
    github.com/spf13/viper v1.18.2             // 配置管理
    golang.org/x/crypto v0.23.0                // 密码加密
    gorm.io/driver/sqlite v1.5.5               // SQLite驱动
    gorm.io/gorm v1.25.7                       // ORM框架
)
```

### 依赖关系图

```
main.go
    ├── config (配置加载)
    ├── pkg (工具包)
    │   ├── database (数据库)
    │   ├── jwt (认证)
    │   └── logger (日志)
    ├── model (数据模型)
    ├── repository (数据访问)
    │   └── gorm.DB
    ├── service (业务逻辑)
    │   └── repository
    └── handler (控制器)
        └── service
```

### 模块依赖

```
Handler → Service → Repository → Model
                ↓
              Pkg (工具包)
```

---

## 项目运行方式

### 环境要求

- Go 1.22+
- SQLite3

### 配置文件

创建 `configs/config.yaml`:

```yaml
server:
  port: "8080"
  host: "0.0.0.0"

database:
  driver: "sqlite"
  path: "./data/scenic.db"

logging:
  level: "info"
  output: "stdout"

ai:
  api_key: "your-api-key"
  model: "qwen-turbo"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"

embedding:
  api_key: "your-embedding-api-key"
  model: "text-embedding-v3"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"

security:
  jwt_secret: "your-jwt-secret"
  token_expire_hours: 24
```

### 运行命令

```bash
# 进入项目目录
cd scenic-guide

# 下载依赖
go mod download

# 运行项目
go run main.go
```

### 构建命令

```bash
# 构建
go build -o scenic-guide

# 运行
./scenic-guide
```

### 启动流程

程序启动流程 (定义在 [main.go](file:///d:/go%20web%2001/scenic-guide/main.go)):

1. **加载配置**: 从 `./configs` 目录加载配置文件
2. **初始化日志**: 根据配置设置日志级别
3. **初始化JWT**: 设置JWT密钥
4. **初始化数据库**: 连接SQLite数据库
5. **数据库迁移**: 自动创建数据表
6. **初始化RAG知识库**: 加载知识库数据
7. **设置路由**: 注册HTTP路由
8. **启动服务器**: 监听指定端口

---

## RAG知识库系统

### 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      用户问题                                │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    问题向量化                                │
│         EmbeddingProvider / BM25FallbackProvider            │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                  知识库检索 (Top-K)                          │
│              Cosine Similarity / BM25 Score                  │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    构建RAG提示词                             │
│              知识片段 + 用户问题 + 回答要求                   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│                    AI生成回答                                │
│                  Qwen / DeepSeek API                         │
└─────────────────────────────────────────────────────────────┘
```

### 知识库数据格式

**JSONL格式** (`lingshan_chunks.jsonl`):

```json
{"id":"chunk-001","content":"灵山大佛高88米...","source":"lingshan.md","title":"灵山大佛介绍","metadata":{}}
{"id":"chunk-002","content":"九龙灌浴表演时间...","source":"lingshan.md","title":"九龙灌浴","metadata":{}}
```

### 向量检索算法

#### 余弦相似度

```go
func (s *RAGService) CosineSimilarity(vec1, vec2 []float64) float64 {
    // 计算点积和模长
    // 返回 cos(θ) = (A·B) / (|A| × |B|)
}
```

#### BM25备用检索

当向量嵌入服务不可用时，使用BM25算法进行文本检索：

```go
func (p *BM25FallbackProvider) CalculateSimilarity(queryTokens, docTokens []string) float64 {
    // 基于词频和文档长度的相似度计算
}
```

### 知识库管理

| 操作 | API | 说明 |
|------|-----|------|
| 上传知识 | POST `/api/v1/knowledge/upload` | 上传JSONL文件 |
| 查看知识 | GET `/api/v1/knowledge/list` | 分页获取知识列表 |
| 删除知识 | DELETE `/api/v1/knowledge/:id` | 删除单条知识 |
| 清空知识 | DELETE `/api/v1/knowledge/all` | 清空整个知识库 |

---

## 附录

### 默认管理员账号

- 用户名: `admin`
- 密码: `admin123`

### 前端页面

访问 `http://localhost:8080/` 可查看前端界面，包含：
- AI数字人对话界面
- 知识库管理面板
- 路线推荐展示
- 用户认证模块

### 健康检查

```bash
curl http://localhost:8080/health
# 响应: {"status":"ok","message":"景区导览服务运行正常"}
```

---

*文档生成时间: 2026-05-06*
