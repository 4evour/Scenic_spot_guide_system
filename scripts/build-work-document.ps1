param(
    [string]$OutputDirectory = "D:\go web 01\delivery",
    [string]$ReleaseLabel = "20260720"
)

$ErrorActionPreference = "Stop"
$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$OutputDirectory = [System.IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$File = Join-Path $OutputDirectory ("景区导览服务AI数字人-作品说明书-{0}.docx" -f $ReleaseLabel)
$Pdf = [System.IO.Path]::ChangeExtension($File, ".pdf")
$ImageRoot = Join-Path $ProjectRoot "output\playwright"
$DiagramRoot = Join-Path $ProjectRoot "docs\assets\judge-doc"
$EvalRoot = Join-Path $ProjectRoot "docs\eval-results"
$ComparisonReportPath = Join-Path $EvalRoot "judge-current-rag-comparison.json"
$LocalGenerationReportPath = Join-Path $EvalRoot "judge-current-rag-local-generation-50.json"
$RetrievalReportPath = Join-Path $EvalRoot "judge-current-rag-retrieval-50.json"

foreach ($requiredReport in @($ComparisonReportPath, $LocalGenerationReportPath, $RetrievalReportPath)) {
    if (!(Test-Path -LiteralPath $requiredReport -PathType Leaf)) {
        throw "缺少文档评测证据：$requiredReport。请先运行对应的 cmd/rag-eval 命令。"
    }
}

$Utf8 = [System.Text.UTF8Encoding]::new($false)
$ComparisonReport = [System.IO.File]::ReadAllText($ComparisonReportPath, $Utf8) | ConvertFrom-Json
$LocalGenerationReport = [System.IO.File]::ReadAllText($LocalGenerationReportPath, $Utf8) | ConvertFrom-Json
$RetrievalReport = [System.IO.File]::ReadAllText($RetrievalReportPath, $Utf8) | ConvertFrom-Json
$BM25Report = ($ComparisonReport.modes | Where-Object { $_.name -eq "bm25-local" }).report
$LightRerankReport = ($ComparisonReport.modes | Where-Object { $_.name -eq "light-rerank" }).report

function Format-Percent([double]$Value) {
    return ("{0:0.0}%" -f ($Value * 100))
}

if (Test-Path -LiteralPath $File) { Remove-Item -LiteralPath $File -Force }
if (Test-Path -LiteralPath $Pdf) { Remove-Item -LiteralPath $Pdf -Force }

function Add-Paragraph {
    param(
        [string]$Text,
        [string]$Style = "Normal",
        [int]$Size = 11,
        [bool]$Bold = $false,
        [bool]$Italic = $false,
        [string]$Align = "left",
        [string]$Before = "0pt",
        [string]$After = "6pt",
        [bool]$PageBreakBefore = $false,
        [bool]$KeepNext = $false
    )
    $props = [ordered]@{
        text = $Text
        style = $Style
        font = "Microsoft YaHei"
        size = "${Size}pt"
        bold = $Bold.ToString().ToLowerInvariant()
        italic = $Italic.ToString().ToLowerInvariant()
        align = $Align
        spaceBefore = $Before
        spaceAfter = $After
    }
    if ($PageBreakBefore) { $props.pageBreakBefore = "true" }
    if ($KeepNext) { $props.keepNext = "true" }
    $props.firstLineIndent = if ($Style -eq "Normal" -and $Align -eq "left") { "22pt" } else { "0pt" }
    if ($Style -eq "Heading1") { $props.outlineLvl = 0 }
    elseif ($Style -eq "Heading2") { $props.outlineLvl = 1 }
    elseif ($Style -eq "Heading3") { $props.outlineLvl = 2 }
    $script:Commands += [pscustomobject]@{ command = "add"; path = "/body"; type = "paragraph"; props = $props }
    $script:ParagraphIndex++
}

function Add-Image {
    param(
        [string]$Path,
        [string]$Alt,
        [string]$Width = "16cm",
        [string]$Height = "auto",
        [string]$Caption = "",
        [bool]$PageBreakBefore = $false
    )
    $resolved = [System.IO.Path]::GetFullPath($Path)
    if (!(Test-Path -LiteralPath $resolved -PathType Leaf)) { throw "图片不存在：$resolved" }
    $imageAfter = if ($Caption) { "0pt" } else { "8pt" }
    Add-Paragraph -Text "" -Align "center" -Before "0pt" -After $imageAfter -PageBreakBefore:$PageBreakBefore
    $imageProps = [ordered]@{ src = $resolved; width = $Width; alt = $Alt }
    if ($Height -ne "auto") { $imageProps.height = $Height }
    $script:Commands += [pscustomobject]@{ command = "add"; path = "/body/p[$script:ParagraphIndex]"; type = "picture"; props = $imageProps }
    if ($Caption) {
        Add-Paragraph -Text $Caption -Size 9 -Align "center" -Before "2pt" -After "8pt"
    }
}

function Add-Heading {
    param([string]$Text, [int]$Level = 1, [bool]$PageBreakBefore = $false)
    $style = "Heading$Level"
    $size = if ($Level -eq 1) { 20 } elseif ($Level -eq 2) { 14 } else { 12 }
    Add-Paragraph -Text $Text -Style $style -Size $size -Bold $true -Before ($(if ($Level -eq 1) { "18pt" } else { "12pt" })) -After "6pt" -PageBreakBefore $PageBreakBefore -KeepNext $true
}

function Add-Bullet {
    param([string]$Text)
    $props = [ordered]@{
        text = $Text
        style = "Normal"
        font = "Microsoft YaHei"
        size = "11pt"
        listStyle = "bullet"
        spaceAfter = "3pt"
    }
    $script:Commands += [pscustomobject]@{ command = "add"; path = "/body"; type = "paragraph"; props = $props }
    $script:ParagraphIndex++
}

function Add-SourceRef {
    param([string]$Text)
    $props = [ordered]@{
        text = "源码定位：$Text"
        style = "Normal"
        font = "Consolas"
        size = "9pt"
        italic = "true"
        color = "536B78"
        spaceBefore = "2pt"
        spaceAfter = "7pt"
        firstLineIndent = "0pt"
    }
    $script:Commands += [pscustomobject]@{ command = "add"; path = "/body"; type = "paragraph"; props = $props }
    $script:ParagraphIndex++
}

function Add-Table {
    param([string[][]]$Rows, [string[]]$Widths)
    $rowCount = $Rows.Count
    $colCount = $Rows[0].Count
    $script:Commands += [pscustomobject]@{ command = "add"; path = "/body"; type = "table"; props = [ordered]@{ rows = $rowCount; cols = $colCount; width = "100%"; "border.all" = "single;0.5pt;C7D6E0" } }
    $tableIndex = $script:TableIndex
    $script:TableIndex++
    for ($r = 0; $r -lt $rowCount; $r++) {
        $props = [ordered]@{ header = ($r -eq 0).ToString().ToLowerInvariant() }
        for ($c = 0; $c -lt $colCount; $c++) { $props["c$($c + 1)"] = $Rows[$r][$c] }
        $script:Commands += [pscustomobject]@{ command = "set"; path = "/body/tbl[$tableIndex]/tr[$($r + 1)]"; props = $props }
        for ($c = 0; $c -lt $colCount; $c++) {
            $cellPath = "/body/tbl[$tableIndex]/tr[$($r + 1)]/tc[$($c + 1)]"
            $fill = if ($r -eq 0) { "E6EFF5" } elseif ($r % 2 -eq 0) { "F7FAFC" } else { "FFFFFF" }
            if ($c -eq 0 -and $r -gt 0) { $fill = if ($r % 2 -eq 0) { "F0F6F8" } else { "F7FAFC" } }
            $script:Commands += [pscustomobject]@{ command = "set"; path = $cellPath; props = [ordered]@{ fill = $fill; "border.all" = "single;0.5pt;C7D6E0" } }
            $runProps = [ordered]@{ font = "Microsoft YaHei"; size = "10pt"; bold = ($r -eq 0 -or $c -eq 0).ToString().ToLowerInvariant(); color = "173B55" }
            $script:Commands += [pscustomobject]@{ command = "set"; path = "$cellPath/p[1]/r[1]"; props = $runProps }
        }
    }
}

$Commands = @()
$TableIndex = 1
$ParagraphIndex = 0

Add-Paragraph -Text "景区导览服务AI数字人" -Style "Title" -Size 30 -Bold $true -Align "center" -Before "52pt" -After "12pt"
Add-Paragraph -Text "Scenic Guide Service AI Digital Human" -Size 14 -Italic $true -Align "center" -After "18pt"
Add-Paragraph -Text "软件设计与实现说明书" -Size 20 -Bold $true -Align "center" -After "10pt"
Add-Paragraph -Text "面向专业技术评委 · 源码级评审版" -Size 14 -Align "center" -After "28pt"
Add-Paragraph -Text "文档版本：2026.07.20" -Size 11 -Align "center" -After "5pt"
Add-Paragraph -Text "系统版本：参赛交付版" -Size 11 -Align "center" -After "5pt"
Add-Paragraph -Text "统一入口：http://127.0.0.1:8080/digital-human#/login" -Size 10 -Align "center" -After "24pt"
Add-Paragraph -Text "本文以源码、配置、测试和本地评测报告为依据，说明系统架构、核心链路、关键算法、工程优化、部署边界与可复现实验。" -Size 11 -Align "center" -After "16pt"
Add-Paragraph -Text "交付文件不包含作者的外部服务凭据；在线增强由评审环境按需配置。" -Size 10 -Italic $true -Align "center" -After "50pt"

Add-Paragraph -Text "摘要" -Style "Normal" -Size 20 -Bold $true -Align "center" -After "12pt" -PageBreakBefore $true -KeepNext $true
Add-Paragraph -Text "景区导览服务AI数字人是一套面向游客导览与景区运营的双服务软件系统。主业务系统采用 Go、Gin、GORM 和 Vue 3，承担身份认证、景点路线、地图定位、知识管理、RAG 问答、会话持久化与运营分析；数字人服务采用 Python、FastAPI 和 WebSocket，以 ServiceContext 组织 ASR、Agent、TTS、VAD 与 Live2D。两套服务通过主系统的受鉴权 WebSocket 代理形成单一浏览器入口。"
Add-Paragraph -Text "系统的核心设计不是简单调用外部大模型，而是将景区知识作为事实边界：查询先经过多轮语境改写、景区配置化扩展、BM25 或混合检索、可解释重排和证据置信度判断，再由本地规则生成或 OpenAI 兼容模型完成表达。没有外部 Key 时，检索、边界判断和本地证据生成仍可工作；外部服务异常时自动回退，不允许无证据模型继续编造景区事实。"
Add-Paragraph -Text "关键词：AI 数字人；检索增强生成；BM25；可解释重排；Live2D；WebSocket；景区导览；本地优先" -Size 10 -Italic $true -After "16pt"
Add-Heading -Text "文档阅读说明" -Level 2
Add-Bullet "第 1—5 章说明需求边界、总体架构、源码组织、数据模型与请求安全链路。"
Add-Bullet "第 6 章从函数和数据结构层面解释 RAG，是本项目的核心算法章节。"
Add-Bullet "第 7 章解释数字人的真实渲染、消息时序、语音路径和故障回退。"
Add-Bullet "第 10—12 章说明工程优化、部署配置和可复现评测，不把实验指标外推为线上准确率。"

Add-Paragraph -Text "目录" -Style "Normal" -Size 20 -Bold $true -Align "center" -Before "0pt" -After "12pt" -PageBreakBefore $true -KeepNext $true
$Commands += [pscustomobject]@{ command = "add"; path = "/body"; type = "toc"; props = [ordered]@{ levels = "1-3"; hyperlinks = "true"; pageNumbers = "true" } }
$ParagraphIndex++

Add-Heading -Text "1. 项目目标与需求边界" -Level 1 -PageBreakBefore $true
Add-Heading -Text "1.1 角色与核心任务" -Level 2
Add-Table -Rows @(
    @("角色", "主要任务", "关键质量要求"),
    @("游客", "知识问答、景点浏览、路线规划、地图定位、二维码导览、数字人交互", "低门槛、事实可靠、可中断、移动端可用"),
    @("景区运营人员", "维护景点、路线、导览内容和知识库，查看查询、反馈与趋势", "数据可追溯、操作受权限控制"),
    @("系统管理员", "用户、角色、系统参数、数字人配置和运行状态管理", "最小权限、旧令牌可撤销、敏感配置不外泄"),
    @("技术评委 / 部署人员", "一键启动、验证核心链路、复现实验和检查源码", "无需作者本机环境，能力边界清晰")
) -Widths @("18%", "50%", "32%")
Add-Heading -Text "1.2 功能边界" -Level 2
Add-Paragraph -Text "系统覆盖游客端、管理端、RAG 和数字人四条业务线。游客端负责地图、路线、定位、二维码与对话；管理端负责内容和运营闭环；RAG 提供可追溯景区事实；数字人将回答转换为可视、可听、可交互的 Live2D 会话。地图实时底图、Edge TTS、浏览器语音服务和可选外部大模型属于外部依赖，文档中单独标注其网络要求。"
Add-Heading -Text "1.3 非目标与事实边界" -Level 2
Add-Bullet "系统不把 SQLite 描述为生产高可用数据库；演示包使用 SQLite，生产配置以 PostgreSQL 为主。"
Add-Bullet "系统不承诺票价、排队、停车余位、临时检修等实时运营信息，知识不足时必须拒绝直接确认。"
Add-Bullet "RAG 固定评测结果只代表指定语料与题集，不代表开放域任意问题准确率，也不等同于线上 SLA。"
Add-Bullet "外部大模型只增强表达与归纳，不取代知识库、证据检索、权限校验和实时信息边界。"
Add-Image -Path (Join-Path $ImageRoot "login-page.png") -Alt "统一登录页" -Caption "图 1 统一登录入口与游客、用户、管理员三类入口（真实本地截图）"

Add-Heading -Text "2. 架构目标与关键决策" -Level 1 -PageBreakBefore $true
Add-Heading -Text "2.1 架构质量目标" -Level 2
Add-Table -Rows @(
    @("目标", "设计响应", "验证方式"),
    @("可部署", "Go 主程序与 Python 数字人解耦，浏览器只使用统一 8080 入口", "双端口健康检查与全新目录启动验收"),
    @("本地优先", "BM25、本地证据生成、知识库、Live2D 资源随包", "无外部模型配置运行 RAG 回归"),
    @("可解释", "RAGTrace 记录来源、模式、耗时、置信度和拒答状态", "接口响应与评测报告"),
    @("安全", "HttpOnly Cookie、CSRF、CORS、CSP、角色校验、WS 令牌校验", "中间件与定向测试"),
    @("可退化", "Embedding、LLM、Redis、WS、TTS 均有明确降级路径", "故障分支测试与启动检查")
) -Widths @("18%", "54%", "28%")
Add-Heading -Text "2.2 系统上下文与容器" -Level 2
Add-Paragraph -Text "系统采用双容器边界。scenic-guide 是业务事实与权限的所有者：路由、用户、景点、路线、知识、会话和 RAG 均在该边界内；Open-LLM-VTuber 是交互编排与媒体能力的所有者：按 WebSocket 连接创建服务上下文，并以 Factory 选择 ASR、TTS、VAD 和 Agent。数字人服务通过主系统 OpenAI 兼容端点使用同一套景区 RAG，避免维护第二份事实逻辑。"
Add-Image -Path (Join-Path $DiagramRoot "architecture.svg") -Alt "总体技术架构" -Caption "图 2 系统上下文、容器与外部依赖边界"
Add-Heading -Text "2.3 技术选型与理由" -Level 2
Add-Table -Rows @(
    @("层次", "技术", "选择理由"),
    @("Web 前端", "Vue 3 + TypeScript + Vite + Pinia + Vue Router", "组合式 API 适合复杂交互；类型约束接口；路由和页面可懒加载"),
    @("UI / 可视化", "Naive UI + ECharts + Pixi / Live2D", "后台控件一致；运营图表成熟；Live2D 使用 GPU Canvas 渲染"),
    @("主后端", "Go + Gin + GORM", "静态编译、并发友好；中间件链清晰；支持 SQLite / PostgreSQL"),
    @("本地检索", "BM25 + 倒排候选 + 规则重排", "无 Key 可复现、中文短问法可解释、适合受控景区语料"),
    @("数字人", "FastAPI + WebSocket + Factory Engines", "长连接事件模型适合语音流；ASR / TTS / Agent 可替换"),
    @("部署", "PowerShell + 便携 uv + 双健康检查", "面向 Windows 评审环境，首次依赖准备可自动化")
) -Widths @("18%", "32%", "50%")
Add-SourceRef "scenic-guide/main.go；internal/handler/routes.go；Open-LLM-VTuber/src/open_llm_vtuber/service_context.py"

Add-Heading -Text "3. 源码组织与启动装配" -Level 1 -PageBreakBefore $true
Add-Heading -Text "3.1 双项目源码边界" -Level 2
Add-Image -Path (Join-Path $DiagramRoot "source-tree.svg") -Alt "源码目录结构" -Caption "图 3 双项目源码结构与职责边界"
Add-Table -Rows @(
    @("目录", "职责", "关键入口"),
    @("scenic-guide/internal/config", "主配置、景区 Profile、环境变量覆盖", "LoadConfig / LoadScenicProfile"),
    @("scenic-guide/internal/handler", "HTTP 输入、权限边界、响应契约", "SetupRoutes / *Handler.Routes"),
    @("scenic-guide/internal/service", "RAG、推荐、统计、会话、TTS 与业务规则", "RAGService / StatsService"),
    @("scenic-guide/internal/repository", "GORM 查询与持久化", "New*Repository"),
    @("scenic-guide/web-vue/src", "游客端、管理端、状态与 API 客户端", "main.ts / router/index.ts"),
    @("Open-LLM-VTuber/src/open_llm_vtuber", "数字人连接、引擎、会话与媒体流", "server.py / websocket_handler.py")
) -Widths @("32%", "44%", "24%")
Add-Heading -Text "3.2 主系统启动时序" -Level 2
Add-Paragraph -Text "main.run 先加载配置并初始化日志；随后执行 dev 构建发布护栏、JWT、Redis、数据库与自动迁移。Redis 连接失败不会阻断启动，而是降级为内存限流。数据库就绪后装载 ScenicProfile，构建 RAGService 并按稳定 ID 补齐 JSONL 知识，最后由 setupDI 依次创建 Repository、Service、Handler。SetupRoutes 在 Handler 已完成注入后注册中间件和 API，避免 Handler 自行访问全局数据库。"
Add-Image -Path (Join-Path $DiagramRoot "startup-di-flow.svg") -Alt "主系统启动和依赖注入" -Caption "图 4 main.run、setupDI 与 SetupRoutes 的启动装配顺序"
Add-Heading -Text "3.3 HTTP Server 生命周期" -Level 2
Add-Paragraph -Text "主系统没有直接使用 gin.Engine.Run 的默认参数，而是显式构造 http.Server：ReadHeaderTimeout 为 5 秒、ReadTimeout 为 30 秒、WriteTimeout 为 60 秒、IdleTimeout 为 120 秒、请求头上限为 1 MiB。SIGINT 或 SIGTERM 到达后进入 10 秒 Shutdown 上下文，完成连接优雅关闭并停止限流器后台清理协程。"
Add-SourceRef "main.go::run；main.go::initRAG；main.go::setupDI；internal/handler/routes.go::SetupRoutes"

Add-Heading -Text "4. 领域模型与数据设计" -Level 1 -PageBreakBefore $true
Add-Heading -Text "4.1 核心实体与关系" -Level 2
Add-Image -Path (Join-Path $DiagramRoot "data-model.svg") -Alt "领域数据模型" -Caption "图 5 核心 GORM 模型与游客体验数据闭环"
Add-Paragraph -Text "ScenicSpot 同时承载地图坐标、排序、二维码和地理围栏字段；TourRoute 保存景点序列、时长、难度和评分；KnowledgeChunk 将内容、来源、标题、知识分类、景点关联与可选向量统一为检索单元。User 通过 role 和 TokenVersion 参与授权，ChatSession / ChatMessage 保留多轮语境，InteractionLog、UserFeedback、VisitorSpotRating 和 RouteRecommendationLog 形成运营闭环。"
Add-Heading -Text "4.2 数据库与迁移" -Level 2
Add-Paragraph -Text "model.AutoMigrate 在启动时集中迁移模型。演示包用 SQLite 降低评审环境门槛；生产环境保留 PostgreSQL 配置。Repository 层隔离 GORM 查询，Service 层不拼接 SQL；对外 JSON 字段统一 snake_case。知识库向量字段不返回前端，避免无意义的数据暴露。"
Add-Heading -Text "4.3 演示数据设计" -Level 2
Add-Paragraph -Text "demo-seed 不只创建账号，还装载知识 JSONL、已校准景点坐标、路线、会话、交互趋势、评分与推荐记录。运营样本使用固定生成规则，使同一交付包在不同电脑上得到可重复的看板。脚本重建自身标记的演示记录，但不删除评委后续创建的非演示数据。"
Add-Image -Path (Join-Path $ImageRoot "dashboard-admin-after-semantic.png") -Alt "管理端数据看板" -Caption "图 6 由本地演示数据驱动的管理员运营看板（真实页面截图）"
Add-SourceRef "internal/model/models.go::AutoMigrate；internal/model/rag_models.go；cmd/demo-seed/main.go；cmd/demo-seed/operational_seed.go"

Add-Heading -Text "5. 请求、认证与接口链路" -Level 1 -PageBreakBefore $true
Add-Heading -Text "5.1 请求分层" -Level 2
Add-Paragraph -Text "前端统一通过 apiFetch 访问 /api/v1。apiFetch 自动携带 Cookie、合并取消信号与 15 秒超时、注入 CSRF Header，并把非 2xx、业务 code 非 0 和非 JSON 响应统一转换为错误。后端请求依次经过 BodySize、CORS、SecurityHeaders、Metrics 和 API 组内的 Language、CSRF；Handler 只处理传输层，Service 执行业务规则，Repository 持久化。"
Add-Heading -Text "5.2 Cookie、CSRF 与令牌撤销" -Level 2
Add-Paragraph -Text "登录成功后服务端写入 SameSite=Strict、HttpOnly 的 auth_token；GET /user/me 同时刷新可读 csrf_token。写请求采用 double-submit：X-CSRF-Token 必须与 Cookie 完全一致。JWT Claims 包含 TokenVersion，ClaimsValidator 每次认证查询用户当前角色与版本；管理员修改角色或用户修改密码后版本递增，旧令牌立即失效，不必等待过期。"
Add-Heading -Text "5.3 WebSocket 与管理员授权" -Level 2
Add-Paragraph -Text "WSTokenAuth 从 Sec-WebSocket-Protocol 的 auth.token.* 或 HttpOnly Cookie 读取令牌，明确不接受 URL query token，从源头避免凭据进入代理日志和浏览器历史。管理员 API 在后端使用 AuthMiddleware + AdminMiddleware 校验角色，前端 requiresAdmin 只负责导航体验，不构成安全边界。"
Add-Image -Path (Join-Path $DiagramRoot "security-request-flow.svg") -Alt "请求和认证安全链路" -Caption "图 7 REST、CSRF、管理员权限与 WebSocket 鉴权链路"
Add-Heading -Text "5.4 路由契约" -Level 2
Add-Table -Rows @(
    @("入口", "鉴权 / 限流", "用途"),
    @("POST /api/v1/ai/chat", "可选认证→自动游客；30 次/分钟", "主 RAG 对话与来源追踪"),
    @("POST /api/v1/ai/multimodal/chat", "可选认证→自动游客；10 次/分钟", "图片与文本联合问答"),
    @("/api/v1/admin/*", "JWT + AdminMiddleware", "内容、用户、运营与评测管理"),
    @("/vtuber-ws/*", "WSTokenAuth", "代理数字人 WebSocket"),
    @("POST /v1/chat/completions", "服务间 API Key；30 次/分钟", "供数字人 Agent 调用景区 RAG"),
    @("GET /metrics", "JWT + AdminMiddleware", "Prometheus 指标")
) -Widths @("34%", "30%", "36%")
Add-SourceRef "web-vue/src/services/api.ts::apiFetch；internal/pkg/middleware.go；internal/handler/routes.go"

Add-Heading -Text "6. RAG 设计与实现" -Level 1 -PageBreakBefore $true
Add-Heading -Text "6.1 知识摄取与存储" -Level 2
Add-Paragraph -Text "启动阶段 initRAG 按 ScenicProfile 的 knowledge 路径加载 lingshan_chunks.jsonl 与 real/lingshan_real_chunks.jsonl。LoadKnowledgeFromFile 逐行反序列化 ChunkData，经 normalizeKnowledgeChunk 统一 ID、标题、来源、分类和 Metadata，再由 upsertChunkData 写入 KnowledgeRepository。稳定 ID 使重复启动变成幂等 Upsert；写操作结束后 invalidateKnowledgeCaches 同时清空知识缓存、查询缓存、Token 缓存、倒排索引与 ID 映射，防止旧证据继续被召回。"
Add-Heading -Text "6.2 索引与候选集" -Level 2
Add-Paragraph -Text "getCachedKnowledge 使用 5 分钟知识缓存。知识块不超过 2000 时一次加载，超过 2000 时按 1000 条分页。rebuildBM25IndexLocked 为每个块缓存分词结果，并构造 tokenIndex（词项→块 ID）和 chunkByID（块 ID→块）；检索时 getBM25CandidateChunks 先取查询词对应候选，候选为空才回退全量，从而避免每次对所有块重复分词和打分。"
Add-Heading -Text "6.3 查询理解与多轮改写" -Level 2
Add-Paragraph -Text "QueryWithRAGTraceInSession 首先规范化 session_id；内存中无历史时由 ChatSessionService 从数据库恢复。RewriteFollowUpQuery 使用最近一轮的 Topic、Intent 和 Boundary，把省略问句（例如：还有别的吗、下雨怎么办）改写为带主题、意图或实时边界的可检索表达。改写只作用于 retrievalQuery，用户原始问题仍进入生成 Prompt，避免扩展词污染最终语义。ScenicProfile 还定义景点别名、query_expansion、conditional_boosts 与 intent_boosts，使同一检索引擎可迁移到不同景区。"
Add-Heading -Text "6.4 五种检索模式" -Level 2
Add-Table -Rows @(
    @("模式", "打分方法", "适用边界"),
    @("bm25-local", "BM25 + 标题词面 + 条件/意图 Boost", "默认本地模式；无外部依赖"),
    @("embedding", "Query / Chunk 向量余弦相似度", "需要可用 Embedding Provider"),
    @("hybrid-weighted", "0.6×Embedding + 0.4×BM25（默认权重）", "两类分数尺度稳定时"),
    @("rrf-fusion", "按两套排名计算 Reciprocal Rank Fusion，默认 K=60", "不希望直接比较原始分数尺度时"),
    @("light-rerank", "BM25 候选上叠加标题命中、词覆盖、实体、来源与规则 Boost", "完全本地、可解释的二阶段排序")
) -Widths @("20%", "52%", "28%")
Add-Paragraph -Text "RetrieveRelevantKnowledgeWithOptions 将 TopK 缺省为 8。BM25、Embedding 和混合模式都先形成 retrievalScoredChunk，再过滤低于 MinSimilarityThreshold 的候选并排序；light-rerank 不引入重型 Cross-Encoder，代价低且每个加分因素可定位到配置或规则。"
Add-Heading -Text "6.5 证据置信度、拒答与生成" -Level 2
Add-Paragraph -Text "检索返回后 calculateChunkEvidence 计算 Confidence 和 ShouldAbstain，并把前三个来源写入 RAGTrace。无块命中或低置信度且非边界问题时，系统直接 buildNoEvidenceAnswer，不把问题交给通用模型。涉及票价、停车余位、排队、演出取消等实时信息时，isBoundaryIntent 走专门的边界答案，可附带知识库中的限制说明，但不能给出未经实时接口确认的承诺。"
Add-Paragraph -Text "无外部 Key 时，generateAnswerFromChunksWithContext 对召回块二次拆句，用 BM25、标题、条件 Boost 和事实维度选择相关证据；路线问题使用编号建议，聚焦事实尽量只保留覆盖完整的句子。配置外部 Key 后，BuildRAGPromptWithContext 将证据和会话上下文交给 OpenAI 兼容模型；调用超时、429、5xx、熔断或无有效响应时，立即回退同一套本地生成。"
Add-Heading -Text "6.6 缓存、追踪与慢请求" -Level 2
Add-Paragraph -Text "RAGService 使用容量 1000 的 LRU 保存 5 分钟查询结果和 10 分钟 Embedding，避免旧的容量满时全清空策略造成周期性缓存抖动。singleflight.Group 以 cacheKey 合并相同问题的并发模型请求，防止缓存未命中时重复调用外部服务。RAGTrace 暴露 trace_id、provider、cache_hit、chunk_count、sources、retrieval_ms、embedding_ms、generation_ms、total_ms、retrieval_mode、rewritten_query、confidence 与 should_abstain；总耗时超过 5 秒标记 slow_request。"
Add-Image -Path (Join-Path $DiagramRoot "rag-flow.svg") -Alt "RAG 源码级流水线" -Caption "图 8 从多轮改写、五种检索模式到证据拒答和双生成分支的完整链路" -PageBreakBefore $true
Add-SourceRef "main.go::initRAG；knowledge_manager.go；session_manager.go；retrieval_engine.go；generation_service.go；rag_service.go::RAGTrace"

Add-Heading -Text "7. 数字人设计与交互时序" -Level 1 -PageBreakBefore $true
Add-Heading -Text "7.1 真实 Live2D 资产链" -Level 2
Add-Paragraph -Text "Live2DStage 动态加载随包的 live2dcubismcore.min.js、mao_pro.model3.json、.moc3、纹理、动作与表情文件，并使用 Pixi / Cubism 在 Canvas 中渲染。DigitalHumanView 将对话状态映射为 neutral、happy、thinking、interrupted 等表达；收到服务端 actions.expressions 或根据回答文本推断后切换表情，音频 volumes 驱动口型。启动打包检查把 Core、model3.json、纹理和 .moc3 设为硬性资产，缺失时打包或启动失败，而不是静默交付替代头像。"
Add-Image -Path (Join-Path $ImageRoot "03-digital-human.png") -Alt "真实数字人页面" -Caption "图 9 真实 Live2D 数字人、状态栏与对话面板（本地运行截图）"
Add-Heading -Text "7.2 连接与 ServiceContext" -Level 2
Add-Paragraph -Text "浏览器通过 VtuberSocketClient 连接 /vtuber-ws/client-ws，Go 主系统完成令牌校验并代理至 12393。WebSocketHandler.handle_new_connection 为客户端登记 UID、发送初始配置并复制默认 ServiceContext。ServiceContext 持有 Live2DModel、ASRInterface、TTSInterface、VADInterface、AgentInterface 和可选 Translator / MCP 组件；init_asr、init_tts、init_vad、init_agent 均通过 Factory 创建具体引擎，角色切换无需修改通信层。"
Add-Heading -Text "7.3 文字和原始音频会话" -Level 2
Add-Paragraph -Text "process_single_conversation 先发送 conversation-chain-start 与 Thinking 状态。文本直接进入 Agent；原始音频则由 process_user_input 调用 ASRInterface.async_transcribe_np 并回发 user-input-transcription。Agent.chat 以异步流输出 SentenceOutput 或 AudioOutput，process_agent_output 解析文字、Live2D 动作与 TTS；TTSTaskManager 并发合成音频并按序发送，前端播放完成后回发 frontend-playback-complete，服务端再发送 force-new-message 和 conversation-chain-end。"
Add-Heading -Text "7.4 当前浏览器语音路径的准确边界" -Level 2
Add-Paragraph -Text "当前 Vue 页面快捷语音按钮使用浏览器 SpeechRecognition 获取文本，并通过 VoiceEmotionCapture 提取声学特征；有声学特征时直接调用主系统 /api/v1/ai/chat，使情绪信息与 RAG 同步处理。Python 数字人仍提供 SenseVoice 原始音频识别能力，并在交付包首次启动时准备对应模型，供发送 audio-data 的客户端使用。两条路径不能混为一谈：浏览器语音服务是否离线取决于浏览器实现，SenseVoice 路径在模型下载后可本地推理。"
Add-Heading -Text "7.5 前端故障回退与中断" -Level 2
Add-Paragraph -Text "DigitalHumanView 为每轮维护 conversationTurn、pendingSocketQuestion、fallbackTimer 与 AbortController。WebSocket 超时、关闭或收到生成错误时，fallbackToBackend 断开数字人对话流并调用 Go /api/v1/ai/chat；答案继续驱动 Live2D 与 TTS。新输入会中断当前音频并向服务端发送 interrupt，旧轮次的音频和文本通过 turn 校验丢弃，避免快速追问时串话。"
Add-Image -Path (Join-Path $DiagramRoot "digital-human-flow.svg") -Alt "数字人WebSocket时序" -Caption "图 10 数字人连接、ASR、Agent、TTS、Live2D 与播放确认时序" -PageBreakBefore $true
Add-SourceRef "web-vue/src/views/DigitalHumanView.vue；components/Live2DStage.vue；services/vtuberSocket.ts；websocket_handler.py；service_context.py；conversations/single_conversation.py"

Add-Heading -Text "8. 地图、路线与位置服务" -Level 1 -PageBreakBefore $true
Add-Heading -Text "8.1 坐标与地图加载" -Level 2
Add-Paragraph -Text "ScenicSpot 保存经纬度、排序、地理围栏开关、半径、提示词和冷却时间。演示景点坐标来自校准文件并标注坐标系，demo-seed 的测试拒绝缺失坐标系或无效坐标。MapView 根据 ScenicProfile 的 map.provider 与 Key 加载高德地图；Key 缺失时地图能力降级，但不会阻断登录、后台、RAG 和数字人。"
Add-Heading -Text "8.2 路线与个性化推荐" -Level 2
Add-Paragraph -Text "TourRoute 维护景点序列、时长、难度和评分。VisitorExperienceService 根据 profile_type、interest_tags 和 difficulty 为路线计算匹配分、理由与 matched_tags，并记录 RouteRecommendationLog。RAGService 还可根据问题中的历史、亲子、轻松、观光车等词匹配 ScenicProfile.routes，使文字问答与地图路线共享同一配置来源。"
Add-Heading -Text "8.3 地理围栏与二维码" -Level 2
Add-Paragraph -Text "useGeolocation 与 useProximityGuide 负责浏览器定位、距离判断、冷却和自动讲解。二维码落地页获取景点导览词后直接播放，不重复触发 RAG；只有明确追问才进入问答链路，减少无意义模型调用。位置数据只用于当前导览逻辑，不作为用户身份凭据。"
Add-Image -Path (Join-Path $ImageRoot "map-routes-desktop-final.png") -Alt "地图与路线页面" -Caption "图 11 地图点位、路线轨迹和路线选择侧栏（真实本地截图）"
Add-SourceRef "internal/model/models.go::ScenicSpot / TourRoute；internal/geolocation/amap.go；web-vue/src/views/MapView.vue；composables/useGeolocation.ts；useProximityGuide.ts"

Add-Heading -Text "9. 前端工程设计" -Level 1 -PageBreakBefore $true
Add-Heading -Text "9.1 路由与界面边界" -Level 2
Add-Paragraph -Text "Vue Router 使用 Hash History，便于单一 index.html 在本地交付。管理端页面挂载 BasicLayout，地图、数字人和扫码使用全屏路由。组件全部采用动态 import，实现按路由拆包。beforeEach 先 fetchUser；未登录时创建游客会话；非管理员访问管理路由会被导向地图。该守卫优化用户体验，但最终授权仍由后端中间件完成。"
Add-Heading -Text "9.2 状态、服务与错误处理" -Level 2
Add-Paragraph -Text "Pinia auth store 缓存认证结果 5 分钟，session store 管理会话与历史，app store 管理主题和全局状态。页面不直接重复拼接 API：services/api.ts 负责业务请求，multimodalApi、ttsApi、vtuberSocket 和 audioPlayback 分别隔离多模态、语音、WebSocket 和播放控制。apiFetch 收到 401 会失效认证缓存并跳转登录；页面历史获取失败时可读取 localStorage 快照。"
Add-Heading -Text "9.3 国际化与响应式" -Level 2
Add-Paragraph -Text "可见文案使用 Vue I18n，中文和英文资源保持同键。DigitalHumanView 在移动端通过 avatar / chat 标签切换主区域，桌面端支持 320—620 px 可调对话栏；固定尺寸和最小宽度避免 Live2D 舞台、按钮和状态文本相互挤压。"
Add-Image -Path "D:\go web 01\tmp\admin-knowledge-final.png" -Alt "知识库管理界面" -Caption "图 12 知识库管理界面与卡片信息结构（真实本地截图）"
Add-SourceRef "web-vue/src/router/index.ts；stores/auth.ts；stores/session.ts；services/api.ts；views/DigitalHumanView.vue"

Add-Heading -Text "10. 性能、安全与可靠性优化" -Level 1 -PageBreakBefore $true
Add-Heading -Text "10.1 检索性能优化" -Level 2
Add-Table -Rows @(
    @("优化点", "原问题", "实现"),
    @("LRU 缓存", "容量满时全清空会产生周期性抖动", "查询与 Embedding 均使用容量 1000 的 LRU + TTL"),
    @("倒排候选", "每次全量分词和 BM25 计算随知识规模线性增长", "tokenCache、tokenIndex、chunkByID 缩小候选集"),
    @("分页装载", "大知识集一次查询可能受 ORM 页大小限制", "超过 2000 条后按 1000 条分页"),
    @("请求合并", "相同问题并发击穿外部模型", "singleflight 以 cacheKey 合并请求"),
    @("连接复用", "频繁新建模型 HTTP 连接增加延迟", "Transport 配置连接池并启用 HTTP/2")
) -Widths @("20%", "40%", "40%")
Add-Heading -Text "10.2 外部模型可靠性" -Level 2
Add-Paragraph -Text "modelGuard 对每次尝试施加 Timeout，只对网络错误、429 和 5xx 重试；连续失败达到阈值后打开熔断器，恢复窗口只允许一个半开探测。流式请求只有在尚未输出首 Token 时可重试，避免重复内容进入 UI。模型健康状态通过 ModelHealth 和 Prometheus 指标暴露。"
Add-Heading -Text "10.3 安全加固" -Level 2
Add-Bullet "默认不信任 X-Forwarded-For；只有 SCENIC_GUIDE_TRUSTED_PROXIES 显式配置后才接受代理链。"
Add-Bullet "发布模式拒绝包含 dev 管理员旁路的构建；旁路还要求双环境变量和回环来源。"
Add-Bullet "CSP 只为数字人路径开放必要的 blob / worker；全局设置 nosniff、SAMEORIGIN、Referrer-Policy 和 Permissions-Policy。"
Add-Bullet "普通请求体上限 12 MiB，多模态上限 64 MiB；Chat 与多模态采用不同限流额度。"
Add-Bullet "服务间 OpenAI 兼容端点使用独立 APIKeyMiddleware；用户 JWT 不复用为服务间密钥。"
Add-Heading -Text "10.4 分层降级" -Level 2
Add-Image -Path (Join-Path $DiagramRoot "resilience-flow.svg") -Alt "性能可靠性和降级设计" -Caption "图 13 检索、外部模型、实时交互与分层降级策略"
Add-SourceRef "rag_service.go；retrieval_engine.go；model_resilience.go；middleware.go；routes.go；DigitalHumanView.vue"

Add-Heading -Text "11. 配置、部署与一键启动" -Level 1 -PageBreakBefore $true
Add-Heading -Text "11.1 可执行包启动链" -Level 2
Add-Paragraph -Text "START-DEMO.bat 调用 Start-Demo.ps1。脚本先检查 scenic-guide.exe、demo-seed.exe、uv.exe、run_server.py、uv.lock、Live2D Core 与模型文件；每次启动生成随机 JWT Secret，仅注入当前进程；随后初始化 SQLite 演示数据。数字人虚拟环境不存在时执行 uv sync --frozen --no-dev，SenseVoice 模型不存在时首次联网下载，最后分别启动 8080 和 12393 并轮询健康端点。-Restart 使用 netstat 定位两个监听进程并重启。"
Add-Heading -Text "11.2 源码运行" -Level 2
Add-Table -Rows @(
    @("步骤", "主系统 scenic-guide", "数字人 Open-LLM-VTuber"),
    @("依赖", "Go、Node.js、npm", "Python 3.11+、uv"),
    @("配置", "复制 config.example.yaml；敏感值用环境变量", "基于模板生成 conf.yaml；不得提交真实凭据"),
    @("构建 / 安装", "web-vue: npm ci && npm run build", "uv sync"),
    @("启动", "go run . 或 scripts/start-local.ps1 -Restart", "uv run run_server.py"),
    @("验证", "GET :8080/health", "GET :12393/ 与 WebSocket")
) -Widths @("18%", "42%", "40%")
Add-Heading -Text "11.3 外部大模型配置" -Level 2
Add-Paragraph -Text "外部 Key 不应写入源码包或可执行包。可执行包通过 Configure-Online-LLM.ps1 把使用者自己的 OpenAI 兼容配置写入当前解压目录的 .env.local，启动器只注入进程且不打印 Key。主系统读取 AI BaseURL、Model 与 APIKey；数字人 Agent 使用主系统本地 /v1/chat/completions，因此景区知识链路保持单一。删除 .env.local 或执行脚本清理后恢复本地模式。"
Add-Heading -Text "11.4 演示账号" -Level 2
Add-Paragraph -Text "本地演示模式在回环地址的登录页显示游客与管理员测试凭据，便于评委直接使用；文档不复制口令，避免口令脱离本地演示边界。生产启动、非回环 Host 和源码默认配置不会显示这些提示。"
Add-SourceRef "scripts/delivery/START-DEMO.bat；scripts/delivery/Start-Demo.ps1；scripts/build-delivery.ps1；internal/handler/demo_info.go"

Add-Heading -Text "12. 测试体系与 RAG 评测" -Level 1 -PageBreakBefore $true
Add-Heading -Text "12.1 分层验证" -Level 2
Add-Table -Rows @(
    @("层次", "命令 / 方式", "覆盖重点"),
    @("Go 单元与集成", "go test ./...", "Handler、Service、Repository、中间件、RAG、迁移与演示数据"),
    @("静态检查", "go vet ./...；go build ./...", "类型、并发误用、构建完整性"),
    @("前端", "npm run check；npm run lint；npm run build", "TypeScript、i18n、组件和生产构建"),
    @("定位", "npm run test:geolocation", "坐标、距离、围栏与冷却逻辑"),
    @("RAG", "go run ./cmd/rag-eval ...", "关键词覆盖、Recall@K、MRR、分组与失败原因"),
    @("交付", "全新目录解压 + START-DEMO.bat", "双服务、账号接口、Live2D 资源和入口")
) -Widths @("18%", "38%", "44%")
Add-Heading -Text "12.2 评测定义" -Level 2
Add-Paragraph -Text "每个 RAGEvaluationCase 包含 question、expected_keywords、expected_chunk_ids、category、difficulty 和 source_type。检索阶段记录 TopK 中的目标块召回、首个相关块排名和检索延迟；生成阶段计算期望关键词覆盖。通过条件是关键词全部覆盖，且存在 expected_chunk_ids 时 Recall@K 大于 0。报告同时按类别、难度和来源分组，并把失败归类为 retrieval_miss、open_question_retrieval_miss、realtime_boundary_miss 等。"
Add-Heading -Text "12.3 当前本地可复现结果" -Level 2
Add-Table -Rows @(
    @("数据集 / 模式", "通过", "关键词覆盖", "Recall@8", "MRR@8", "p50 / p95"),
    @("162 块 / 50 题，本地生成", "$($LocalGenerationReport.passed)/$($LocalGenerationReport.total)", (Format-Percent $LocalGenerationReport.average_keyword_coverage), (Format-Percent $LocalGenerationReport.average_recall_at_k), ("{0:0.000}" -f $LocalGenerationReport.mrr_at_k), "$($LocalGenerationReport.retrieval_p50_ms) / $($LocalGenerationReport.retrieval_p95_ms) ms"),
    @("162 块 / 50 题，纯检索", "$($RetrievalReport.passed)/$($RetrievalReport.total)", (Format-Percent $RetrievalReport.average_keyword_coverage), (Format-Percent $RetrievalReport.average_recall_at_k), ("{0:0.000}" -f $RetrievalReport.mrr_at_k), "$($RetrievalReport.retrieval_p50_ms) / $($RetrievalReport.retrieval_p95_ms) ms"),
    @("162 块 / 210 题，bm25-local", "$($BM25Report.passed)/$($BM25Report.total)", (Format-Percent $BM25Report.average_keyword_coverage), (Format-Percent $BM25Report.average_recall_at_k), ("{0:0.000}" -f $BM25Report.mrr_at_k), "$($BM25Report.retrieval_p50_ms) / $($BM25Report.retrieval_p95_ms) ms"),
    @("162 块 / 210 题，light-rerank", "$($LightRerankReport.passed)/$($LightRerankReport.total)", (Format-Percent $LightRerankReport.average_keyword_coverage), (Format-Percent $LightRerankReport.average_recall_at_k), ("{0:0.000}" -f $LightRerankReport.mrr_at_k), "$($LightRerankReport.retrieval_p50_ms) / $($LightRerankReport.retrieval_p95_ms) ms")
) -Widths @("28%", "12%", "16%", "14%", "12%", "18%")
Add-Paragraph -Text "以上报告均由当前交付源码在 Windows 本机重新运行，top_k=8。50 题本地生成报告的 generation_provider=local，210 题对比报告的 generation_provider=disabled；两者均为 embedding_provider=bm25-local，没有调用外部大模型或外部 Embedding。50 题属于针对核心功能和边界的回归集，因此结果较高；210 题包含更多开放问法，更接近检索泛化检查，应作为主要风险参考。"
Add-Paragraph -Text "本次 210 题结果显示，light-rerank 相比 bm25-local 提高通过数和 MRR，但 p50 / p95 也增加；这反映了规则重排的收益与计算代价。剩余失败集中在开放文化推荐、语义对比、亲子解释和拍照边界问法，不能以精选 50 题的 100% 掩盖。"
Add-Heading -Text "12.4 评测口径限制" -Level 2
Add-Bullet "关键词覆盖只能验证关键事实是否出现，不能完整衡量语言自然度、逻辑质量或事实之间的因果关系。"
Add-Bullet "Recall@8 和 MRR@8 评价目标知识块是否进入前八及排名，不等同于最终回答正确率。"
Add-Bullet "本地生成结果不能归因于外部模型；外部模型质量需另做盲测、事实一致性和成本延迟评估。"
Add-Bullet "3000 块 / 300 题合成闭集只用于规模回归，不作为真实游客问答准确率。"
Add-SourceRef "internal/service/rag_evaluation.go；cmd/rag-eval/main.go；docs/eval-results/judge-current-rag-*.json"

Add-Heading -Text "13. 交付范围、运行矩阵与许可证" -Level 1 -PageBreakBefore $true
Add-Heading -Text "13.1 交付包" -Level 2
Add-Image -Path (Join-Path $DiagramRoot "delivery-packages.svg") -Alt "交付包组成" -Caption "图 14 可执行包、源码包、文档与完整性校验"
Add-Paragraph -Text "49014999作品.zip 保留双服务运行所需源码或二进制、前端构建产物、知识库、演示数据初始化、真实 Live2D、便携 uv 和启动脚本；不重复携带可再生的 Python 虚拟环境与 1.12GB ASR 模型，因此压缩体积约 90 MB。49014999源码.zip 排除依赖目录、缓存、日志、数据库、密钥、模型缓存与开发过程记录，保留可复现构建所需源码、锁文件、配置模板、知识与测试。"
Add-Heading -Text "13.2 功能降级矩阵" -Level 2
Add-Table -Rows @(
    @("能力", "完全断网且依赖已准备", "联网、无外部 LLM Key", "联网并配置外部 LLM"),
    @("登录 / 后台 / 看板 / 景点 / 路线", "可用", "可用", "可用"),
    @("本地 RAG 事实问答", "可用", "可用", "可用"),
    @("回答表达", "本地证据组织", "本地证据组织", "在线模型增强，失败回退本地"),
    @("真实 Live2D", "可用", "可用", "可用"),
    @("Python SenseVoice 原始音频 ASR", "模型已下载则可用", "可用", "可用"),
    @("浏览器快捷语音识别", "取决于浏览器实现", "通常可用", "通常可用"),
    @("Edge TTS", "不可用，回退文字/浏览器能力", "可用", "可用"),
    @("高德在线底图", "不可用", "配置地图 Key 后可用", "相同")
) -Widths @("30%", "24%", "23%", "23%")
Add-Heading -Text "13.3 Live2D 与第三方边界" -Level 2
Add-Paragraph -Text "Live2D Cubism Core、模型、纹理和动作属于第三方资产，随包用于本项目评审演示。评委可以运行和展示，但不得把资产从项目中拆出单独销售或再分发。最终约束以 LICENSE-Live2D.md 和原始资产许可为准。外部地图、TTS 和模型服务受各自服务条款约束。"

Add-Heading -Text "14. 附录：核心源码索引" -Level 1 -PageBreakBefore $true
Add-Heading -Text "14.1 关键链路索引" -Level 2
Add-Table -Rows @(
    @("链路", "文件 / 符号", "评审关注点"),
    @("启动与 DI", "main.go::run / initRAG / setupDI", "依赖顺序、降级、Handler 注入"),
    @("路由与安全", "internal/handler/routes.go::SetupRoutes", "中间件顺序、REST / WS / 服务间端点"),
    @("认证", "internal/pkg/middleware.go", "Cookie、CSRF、角色、WS token、游客会话"),
    @("数据模型", "internal/model/models.go / rag_models.go", "实体边界、索引、JSON 契约"),
    @("知识摄取", "internal/service/knowledge_manager.go", "JSONL、Upsert、缓存失效"),
    @("检索", "internal/service/retrieval_engine.go", "五种模式、倒排候选、重排"),
    @("生成与拒答", "internal/service/generation_service.go", "证据生成、Prompt、模型失败回退"),
    @("多轮会话", "internal/service/session_manager.go", "历史恢复、追问改写、边界继承"),
    @("模型保护", "internal/service/model_resilience.go", "超时、重试、熔断、流式语义"),
    @("数字人前端", "DigitalHumanView.vue / Live2DStage.vue", "轮次、中断、回退、渲染和口型"),
    @("数字人服务", "service_context.py / websocket_handler.py", "连接级上下文、消息路由、Factory"),
    @("会话编排", "conversations/single_conversation.py", "ASR→Agent→TTS→播放确认"),
    @("评测", "rag_evaluation.go / cmd/rag-eval", "指标定义、分组、失败归因")
) -Widths @("20%", "43%", "37%")
Add-Heading -Text "14.2 关键配置索引" -Level 2
Add-Table -Rows @(
    @("配置", "位置", "作用"),
    @("主系统", "scenic-guide/configs/config.example.yaml", "Server、Database、Security、AI、Embedding、Multimodal"),
    @("景区 Profile", "configs/scenic_profiles/lingshan.yaml", "景点词表、扩展、Boost、Prompt、路线、地图"),
    @("数字人", "Open-LLM-VTuber/config_templates/conf.ZH.default.yaml", "端口、ASR、TTS、Agent、VAD、模型"),
    @("角色", "Open-LLM-VTuber/characters/*.yaml", "人设、模型、声音和表达"),
    @("交付启动", "scripts/delivery/Start-Demo.ps1", "依赖、模型、数据库、双服务、健康检查")
) -Widths @("20%", "46%", "34%")
Add-Heading -Text "14.3 结论" -Level 2
Add-Paragraph -Text "本项目的技术重点是把景区事实系统、可解释 RAG 与实时数字人交互组合为一条可部署、可降级、可评测的工程链路。其核心价值不在于单一模型或单一页面，而在于：数据有来源、检索有指标、生成有边界、身份有撤销、数字人有真实资产和消息时序、外部依赖失败时仍保留核心导览能力。"

$batchPath = Join-Path $env:TEMP ("scenic-guide-doc-{0}.json" -f [guid]::NewGuid().ToString("N"))
$Commands += [pscustomobject]@{ command = "add"; path = "/"; type = "header"; props = [ordered]@{ type = "default"; text = "景区导览服务AI数字人 · 作品说明书"; align = "right"; size = "9pt"; font = "Microsoft YaHei" } }
$Commands += [pscustomobject]@{ command = "add"; path = "/"; type = "footer"; props = [ordered]@{ type = "default"; text = "第 "; align = "center"; size = "9pt"; font = "Microsoft YaHei" } }
$Commands += [pscustomobject]@{ command = "add"; path = "/footer[1]/p[1]"; type = "field"; props = [ordered]@{ fieldType = "page" } }
$Commands += [pscustomobject]@{ command = "add"; path = "/footer[1]/p[1]"; type = "run"; props = [ordered]@{ text = " 页 / 共 "; font = "Microsoft YaHei"; size = "9pt" } }
$Commands += [pscustomobject]@{ command = "add"; path = "/footer[1]/p[1]"; type = "field"; props = [ordered]@{ fieldType = "numpages" } }
$Commands += [pscustomobject]@{ command = "set"; path = "/settings"; props = [ordered]@{ updateFields = "true" } }
$Commands | ConvertTo-Json -Depth 12 | Set-Content -LiteralPath $batchPath -Encoding UTF8

officecli create $File
if ($LASTEXITCODE -ne 0) { throw "OfficeCLI 创建 DOCX 失败。" }
officecli open $File
if ($LASTEXITCODE -ne 0) { throw "OfficeCLI 打开 DOCX 失败。" }
officecli batch $File --input $batchPath --stop-on-error
if ($LASTEXITCODE -ne 0) { throw "OfficeCLI 批处理失败，文档未完成。" }
officecli set $File / --prop docDefaults.font="Microsoft YaHei" --prop docDefaults.fontSize=11pt
if ($LASTEXITCODE -ne 0) { throw "OfficeCLI 设置默认字体失败。" }
officecli close $File
if ($LASTEXITCODE -ne 0) { throw "OfficeCLI 保存 DOCX 失败。" }
Remove-Item -LiteralPath $batchPath -Force

if (!(Test-Path -LiteralPath $File -PathType Leaf)) { throw "DOCX 未生成。" }

try {
    $word = New-Object -ComObject Word.Application
    $word.Visible = $false
    $word.DisplayAlerts = 0
    try {
        $document = $word.Documents.Open($File, $false, $false)
        $document.Repaginate()
        if ($document.TablesOfContents.Count -eq 0) {
            throw "文档中未找到目录字段。"
        }

        $toc = $document.TablesOfContents.Item(1)
        $toc.IncludePageNumbers = $true
        $toc.RightAlignPageNumbers = $true
        $toc.UseHeadingStyles = $true
        $toc.UpperHeadingLevel = 1
        $toc.LowerHeadingLevel = 3
        $toc.UseHyperlinks = $true
        $toc.TabLeader = 1
        [void]$toc.Update()

        $tocStyles = @(
            @{ Id = -20; Size = 11; LeftIndent = 0; Bold = $true },
            @{ Id = -21; Size = 10; LeftIndent = 18; Bold = $false },
            @{ Id = -22; Size = 10; LeftIndent = 36; Bold = $false }
        )
        foreach ($tocStyle in $tocStyles) {
            $style = $document.Styles.Item($tocStyle.Id)
            $style.Font.Name = "Microsoft YaHei"
            $style.Font.NameFarEast = "Microsoft YaHei"
            $style.Font.Size = $tocStyle.Size
            $style.Font.Bold = [int]$tocStyle.Bold
            $style.ParagraphFormat.LeftIndent = $tocStyle.LeftIndent
            $style.ParagraphFormat.FirstLineIndent = 0
            $style.ParagraphFormat.SpaceAfter = 1
        }

        $heading1 = $document.Styles.Item(-2)
        $heading1.Font.Name = "Microsoft YaHei"
        $heading1.Font.NameFarEast = "Microsoft YaHei"
        $heading1.Font.Color = 5585687
        $heading1.ParagraphFormat.KeepWithNext = -1
        $heading1.ParagraphFormat.KeepTogether = -1

        $heading2 = $document.Styles.Item(-3)
        $heading2.Font.Name = "Microsoft YaHei"
        $heading2.Font.NameFarEast = "Microsoft YaHei"
        $heading2.Font.Color = 6515503
        $heading2.ParagraphFormat.KeepWithNext = -1
        $heading2.ParagraphFormat.KeepTogether = -1

        $document.Repaginate()
        [void]$toc.UpdatePageNumbers()
        $document.Save()
        $document.ExportAsFixedFormat($Pdf, 17)
        $document.Close([ref]$false)
    } finally {
        $word.Quit()
    }
} catch {
    throw "目录更新或 PDF 导出失败：$($_.Exception.Message)"
}
if (!(Test-Path -LiteralPath $Pdf -PathType Leaf)) {
    throw "PDF 导出失败：文件未生成。"
}
Write-Host "DOCX: $File"
Write-Host "PDF: $Pdf"
