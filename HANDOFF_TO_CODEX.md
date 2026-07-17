# 景区导览系统 — 交接文档(2026-07-17)

> 前一开发者(ZCode,基于 DeepSeek-V4)完成两轮代码审查与修复后交接。请 Codex 在上手前通读此文档。

## 项目概况

- **仓库**:`github.com/4evour/Scenic_spot_guide_system`
- **技术栈**:Go 1.25 + Gin + GORM(Postgres/SQLite) + JWT(HS256) + Vue3(NaiveUI, Pinia, ECharts) + RAG(检索增强生成) + Live2D 数字人
- **部署**:Docker Compose(postgres:16 + Go 后端,多阶段 Dockerfile,非 root 用户)
- **最晚 commit**:`f569c31` — 安全加固与 bug 修复(两轮审查修复)
- **验证结果**:`go build ./...` / `go build -tags dev ./...` / `go vet ./...` / `go test ./...` 全部通过

## 已完成修复总览(两轮,共 22 项)

### Critical(5 项)

| 编号 | 问题 | 修复方式 |
|---|---|---|
| C2 | 会话消息双写 | `appendSessionTurn` 只更新内存缓存,写库统一由 handler 的 `AppendSessionTurnWithUser`(带 userID)触发 |
| C3 | LLM/Embedding HTTP 调用未绑 request context | 11 个方法加 `ctx context.Context` 参数;4 处 `http.NewRequest` → `NewRequestWithContext`;客户端断开后可取消 LLM 调用 |
| C4 | 异步 goroutine 无 panic recover | 新增 `internal/pkg/safe_go.go`(`pkg.SafeGo`),替换 session_manager 和 ai_handler 的 3 处裸 goroutine |
| A1 | CSRF 防护可被 `X-Requested-With` 绕过 | 移除单头豁免;`csrfExemptPaths` 改为完整路径 `/api/v1/...` + 精确匹配(原 `HasSuffix` 会误匹配同后缀路径) |
| A3 | `.env.example` 占位密钥可被直接复制 | `InitJWT` 新增 `looksLikePlaceholderSecret` 关键词检测;`.env.example` 占位值加入黑名单 |

### High(5 项)

| 编号 | 问题 | 修复方式 |
|---|---|---|
| H1 | 扫码 recordScan 未填 SessionID | `SessionID = "qr:"+code`,修复大屏 KPI `Distinct("session_id")` 失真 |
| H9 | parseUintParam 校验失效(error 恒 nil) | 改用 `strconv.ParseUint`,返回真实 error |
| A4 | dev 旁路可被误部署到生产 | 双开关(`SCENIC_GUIDE_DEV_ADMIN_BYPASS` + `SCENIC_GUIDE_DEV_ALLOW_BYPASS`);`GIN_MODE=release` 下 dev 构建拒绝启动 |
| A5 | 限流按 c.ClientIP,未设 SetTrustedProxies | `main.go` 默认 `SetTrustedProxies(nil)`(关闭 XFF 信任);支持 `SCENIC_GUIDE_TRUSTED_PROXIES` 白名单 |
| H11 | 构建产物/日志/重复资源入库(~60MB) | `git rm --cached` 取消跟踪 128+ 文件;补齐 `.gitignore` 规则;磁盘文件保留 |

### Medium/Low(12 项)

| 编号 | 问题 | 修复方式 |
|---|---|---|
| M1 | CosineSimilarity 维度不匹配时按 min 截断 | `len !=` 返回 0 |
| M3 | BulkGenerateQR 对 range 副本取地址 | 改用索引 `spots[i]` |
| L12 | deriveKeywordsFromAnswer 每次 MustCompile + 孤儿注释 | `numberPattern` 提包级;删除 `generation_service.go` 末尾孤儿注释 |
| M19 | rag-eval `CPU=GOARCH`(架构误填) | `runtime.NumCPU()` |
| A7 | TTS voice 透传 Neural + rate 未校验 | `ResolveVoice` 严格白名单;`validateRate` 正则 `^[+-]\d{1,3}%$` |
| A8 | WS 代理透传 Authorization/Cookie 到下游;query token 回退 | 转发前剔除 4 个敏感头;移除 `c.Query("token")` 回退 |
| B1 | digital_human 反馈误写 VisitorQuery 表 | 删除写 VisitorQuery 逻辑;清理未使用字段和构造参数 |
| B2 | chat_message_repo LIKE 关键字 `%`/`_` 未转义 | `escapeLikePattern` + `ESCAPE '\'` |
| B3 | RAG 缓存满时整表清空(非 LRU) | 引入 `hashicorp/golang-lru/v2`,embeddingCache + queryCache 改为 LRU |
| B4 | GetDashboardOverview 串行 ~10 次 DB 查询 | 加 30s TTL 缓存(`dashCacheData`) |
| C2 | docker-compose PG 端口暴露到宿主 | 删除 `ports: "5432:5432"`(compose 内部网络访问) |
| C3 | CI 硬编码测试密码 | 动态生成 `ci-test-${{github.run_id}}-${{github.run_attempt}}` |

## 仍待处理(按优先级)

### 安全策略(需决策)
- **C1 token 撤销机制**:用户表中加 `TokenVersion` 字段,claims 与 `AuthMiddleware` 校验;改密/降级时递增。**涉及 DB 迁移 + 认证流程改动**。
- **C2 占位密钥更激进防控**:考虑用正则 `^[0-9a-f]{64}$` 强制 JWT 密钥为纯 hex 生成(而非黑名单)。
- **H2 SetTrustedProxies 白名单**:反代部署时需运维设 `SCENIC_GUIDE_TRUSTED_PROXIES`;当前默认关闭 XFF 信任。
- **CSP 收紧**:`routes.go:190` 的 `'unsafe-inline'`(style-src)需前端配合改造为 nonce/hash;Live2D 的 `'unsafe-eval'` 需评估能否隔离到子资源。

### 测试覆盖(需补)
- `internal/handler/ai_handler.go`(646 行)完全无测试(`/ai/chat` 主入口)
- `internal/service/stats_service.go`(851 行)仅有 `formatNumber` 单元测试,Dashboard 聚合逻辑零覆盖
- `internal/service/rag_llm_enhance.go`(287 行)、`knowledge_manager.go`(303 行)无测试

### 代码质量(建议重构)
- `internal/handler/digital_human_handler.go` 的 `ChatText`/`ChatVoiceTranscript` 95% 重复(可抽私有方法)
- CRUD handler(scenic_spot/guide_content/tour_route/visitor_query)大量重复模板(可抽泛型)
- `scripts/check-*-i18n.mjs` 35 个高度重复的 i18n 检查脚本(应收敛为参数化单脚本)
- 根目录 6 份文档(README/CODE_WIKI/PROJECT_OVERVIEW/PROJECT_DOCUMENTATION/CHANGELOG/SECURITY_PUBLIC_RELEASE_REVIEW)章节严重重叠

## 关键技术决策与约定

### Context 传递链
所有从 HTTP handler → service → LLM 调用的路径现在都显式传递 `context.Context`:
```
handler(取 c.Request.Context)→ QueryWithRAGAndRouteTraceInSession(ctx, ...)
  → QueryWithRAGTraceInSession(ctx, ...) → queryWithRAGTraceInternal(ctx, ...)
    → http.NewRequestWithContext(ctx, ...)
```
新增方法**必须**在首参接受 ctx 并向下传递,否则 LLM 调用无法感知客户端断开。

### 缓存层
- **RAG 缓存**:两个 LRU 缓存(`embeddingCache` / `queryCache`,容量 1000),用 `hashicorp/golang-lru/v2`,带 TTL(入仓检查 `ExpireTime`)
- **Dashboard 缓存**:`StatsService.dashCacheData`(30s TTL),`sync.Mutex` 保护
- **会话历史缓存**:`sessionHistory`(每会话最多 10 轮内存缓存,30min TTL),`cacheMutex`(RWMutex)保护

### goroutine 安全
所有后台异步任务(会话持久化、统计记录等)必须通过 `pkg.SafeGo(name, fn)` 启动:
```go
pkg.SafeGo("AppendSessionTurnWithUser", func() {
    // fn body
})
```
裸 `go func(){...}` 应被 linter 阻止。

### dev 旁路
- prod 构建(`go build .`,不带 `-tags dev`):`IsDevBuild=false`,`applyDevAdminBypass()` 恒返回 false
- dev 构建(`go build -tags dev`):旁路需同时设 `SCENIC_GUIDE_DEV_ADMIN_BYPASS=1` + `SCENIC_GUIDE_DEV_ALLOW_BYPASS=true`,且仅限 loopback(127.0.0.1)请求;`GIN_MODE=release` 下 dev 构建直接拒绝启动

### CSRF 抵御
- **策略**:SameSite=Strict cookie + 严格双重提交(强制 X-CSRF-Token header == csrf_token cookie)
- **豁免**:只对 `/api/v1/login`、`/api/v1/register`、`/api/v1/auth/guest-login` 精确豁免
- **前端约束**:所有写请求(POST/PUT/DELETE)必须在 `web-vue/src/services/api.ts` 层面自动注入 X-CSRF-Token(已实现)

## 开发与部署注意事项

### 本地开发
- `static/vue-app/` 已从版本控制移除,运行为空白目录 → **必须先 `cd web-vue && npm run build`**
- `static/digital-human/` 已移除跟踪 → 是第三方 Open-LLM-Vtuber 产物,按独立流程获取
- `static/live2d-models/mao_pro/mao_pro/` 是无引用的重复副本(磁盘上仍存在,可手动删除释放约 9.3MB)
- 用工具直连 PG 需 `docker-compose.override.yml` 临时加 `ports: "5432:5432"`

### Docker 部署
- Dockerfile 多阶段构建:**前端阶段 `npm run build`**→ **后端阶段 `go build .`**(非 dev 标签)→ **运行阶段 Alpine 非 root**
- 运行阶段 `COPY --from=frontend /src/static/vue-app ./static/vue-app`(重新生成的前端产物,不依赖仓库中的旧构建产物)
- docker-compose 生产部署时需设 `SCENIC_GUIDE_TRUSTED_PROXIES=10.0.0.1,...`(反代 IP)

### 验证命令
```bash
go build ./... && go vet ./... && go test ./...             # default build
go build -tags dev ./... && go test -tags dev ./...          # dev build
cd web-vue && npm run build                                  # 前端构建
```

## 文件导航

| 模块 | 关键文件 |
|---|---|
| 入口 | `main.go`(DI 组装、HTTP 超时、优雅关闭) |
| 路由 | `internal/handler/routes.go`(中间件链、CORS、CSP、track) |
| 认证 | `internal/pkg/{jwt,middleware,middleware_dev,middleware_prod}.go` |
| WebSocket | `internal/pkg/wsproxy.go`(代理头过滤)、`middleware.go:WSTokenAuth` |
| AI/RAG | `internal/service/{rag_service,generation_service,retrieval_engine,session_manager}.go` |
| 统计 | `internal/service/stats_service.go`(Dashboard、记录) |
| TTS | `internal/handler/tts_handler.go` + `internal/service/edge_tts.go` |
| 数字人 | `internal/handler/digital_human_handler.go` |
| 前端(Vue) | `web-vue/src/`(NaiveUI,路由,国际化) |
| 遗留前端 | `static/index.html` + `static/js/app.js`(已标注 @deprecated,将在 v3.0 移除) |
| 配置 | `configs/config.example.yaml`、`.env.example` |
| 部署 | `Dockerfile`、`docker-compose.yml`、`.github/workflows/ci.yml` |

---

**提交 hash**:`f569c31` — 安全加固与 bug 修复(两轮审查修复)
**验证状态**:✅ `go build ./...` / `go build -tags dev ./...` / `go vet ./...` / `go test ./...` 全部通过
