param(
    [switch]$Restart,
    [switch]$NoBrowser,
    [switch]$SkipVoiceModelDownload
)

$ErrorActionPreference = "Stop"

$PackageRoot = $PSScriptRoot
$AppRoot = Join-Path $PackageRoot "scenic-guide"
$VTuberRoot = Join-Path $PackageRoot "Open-LLM-VTuber"
$ToolsRoot = Join-Path $PackageRoot "tools"
$LogRoot = Join-Path $PackageRoot "logs"
$UvPath = Join-Path $ToolsRoot "uv.exe"
$AppExe = Join-Path $AppRoot "scenic-guide.exe"
$SeedExe = Join-Path $AppRoot "demo-seed.exe"
$DemoPassword = "ScenicDemo123456"

New-Item -ItemType Directory -Force -Path $LogRoot | Out-Null

function Assert-RequiredFile {
    param([string]$Path, [string]$Description)
    if (!(Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "缺少$Description：$Path。请重新解压完整的可执行包。"
    }
}

function Import-DotEnv {
    param([string]$Path)
    if (!(Test-Path -LiteralPath $Path -PathType Leaf)) {
        return
    }

    Get-Content -LiteralPath $Path | ForEach-Object {
        $line = $_.Trim().TrimStart([char]0xFEFF)
        if (!$line -or $line.StartsWith("#") -or !$line.Contains("=")) {
            return
        }
        $parts = $line.Split("=", 2)
        $name = $parts[0].Trim()
        $value = $parts[1].Trim().Trim('"').Trim("'")
        if ($name) {
            Set-Item -Path "Env:$name" -Value $value | Out-Null
        }
    }
}

function Get-ListenerProcessId {
    param([int]$Port)
    $netstat = Join-Path $env:SystemRoot "System32\netstat.exe"
    $pattern = "^\s*TCP\s+\S+:$Port\s+\S+\s+LISTENING\s+(\d+)\s*$"
    $lines = & $netstat -ano -p tcp
    if ($LASTEXITCODE -ne 0) {
        throw "端口检测失败，netstat 退出码：$LASTEXITCODE"
    }
    foreach ($line in $lines) {
        if ($line -match $pattern) {
            return [int]$matches[1]
        }
    }
    return $null
}

function Stop-PortListener {
    param([int]$Port)
    $listenerPid = Get-ListenerProcessId -Port $Port
    if ($listenerPid) {
        Stop-Process -Id $listenerPid -Force
        Start-Sleep -Seconds 1
    }
}

function Wait-HttpOk {
    param([string]$Url, [int]$TimeoutSeconds = 60)
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3 | Out-Null
            return $true
        } catch {
            Start-Sleep -Milliseconds 800
        }
    }
    return $false
}

function Set-ApplicationEnvironment {
    Import-DotEnv -Path (Join-Path $AppRoot ".env.local")

    $jwtBytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $rng.GetBytes($jwtBytes)
    } finally {
        $rng.Dispose()
    }

    $env:SCENIC_GUIDE_SECURITY_JWT_SECRET = ($jwtBytes | ForEach-Object { $_.ToString("x2") }) -join ""
    $env:SCENIC_GUIDE_API_KEY = "not-needed"
    $env:SCENIC_GUIDE_DATABASE_DRIVER = "sqlite"
    $env:SCENIC_GUIDE_DATABASE_PATH = "./data/scenic_guide.db"
    $env:SCENIC_GUIDE_DEMO_MODE = "true"
    $env:SCENIC_GUIDE_DEMO_PASSWORD = $DemoPassword
    $env:SCENIC_GUIDE_ADMIN_PASSWORD = $DemoPassword

    if ([string]::IsNullOrWhiteSpace($env:SCENIC_GUIDE_AI_API_KEY)) {
        $env:SCENIC_GUIDE_AI_API_KEY = "local-dev-placeholder"
        Write-Host "[INFO] 未配置外部大模型 Key，将使用本地知识库生成回答。"
    } else {
        Write-Host "[OK] 已检测到本机外部大模型配置，Key 不会被打印。"
    }
}

function Initialize-DemoDatabase {
    Push-Location $AppRoot
    try {
        & $SeedExe --admin-password $DemoPassword
        if ($LASTEXITCODE -ne 0) {
            throw "演示数据初始化失败，退出码：$LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

function Initialize-VTuberRuntime {
    Push-Location $VTuberRoot
    try {
        if (!(Test-Path -LiteralPath (Join-Path $VTuberRoot ".venv\Scripts\python.exe") -PathType Leaf)) {
            Write-Host "[FIRST RUN] 正在安装数字人 Python 运行环境，请保持联网..."
            & $UvPath sync --frozen --no-dev
            if ($LASTEXITCODE -ne 0) {
                throw "数字人 Python 依赖安装失败，退出码：$LASTEXITCODE"
            }
        }

        $voiceModel = Join-Path $VTuberRoot "models\sherpa-onnx-sense-voice-zh-en-ja-ko-yue-2024-07-17\model.int8.onnx"
        if (!(Test-Path -LiteralPath $voiceModel -PathType Leaf)) {
            if ($SkipVoiceModelDownload) {
                Write-Warning "已跳过语音识别模型下载，数字人服务可能无法启动。"
            } else {
                Write-Host "[FIRST RUN] 正在下载约 1.12GB 的 SenseVoice 语音识别模型..."
                Write-Host "[FIRST RUN] 下载时间取决于网络速度，请勿关闭窗口。"
                & $UvPath run --no-sync python -m src.open_llm_vtuber.asr.utils
                if ($LASTEXITCODE -ne 0 -or !(Test-Path -LiteralPath $voiceModel -PathType Leaf)) {
                    throw "语音识别模型下载或解压失败。请检查网络后重新运行 START-DEMO.bat。"
                }
            }
        }
    } finally {
        Pop-Location
    }
}

function Start-ScenicGuide {
    if (Get-ListenerProcessId -Port 8080) {
        Write-Host "[OK] 主系统已在 8080 端口运行。"
        return
    }

    Start-Process -FilePath $AppExe `
        -WorkingDirectory $AppRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput (Join-Path $LogRoot "scenic-guide.out.log") `
        -RedirectStandardError (Join-Path $LogRoot "scenic-guide.err.log") | Out-Null

    if (!(Wait-HttpOk -Url "http://127.0.0.1:8080/health" -TimeoutSeconds 60)) {
        throw "主系统启动失败。请查看日志：$LogRoot"
    }
    Write-Host "[OK] 主系统已启动：http://127.0.0.1:8080/"
}

function Start-VTuber {
    if (Get-ListenerProcessId -Port 12393) {
        Write-Host "[OK] 数字人服务已在 12393 端口运行。"
        return
    }

    Start-Process -FilePath $UvPath `
        -ArgumentList @("run", "--no-sync", "run_server.py") `
        -WorkingDirectory $VTuberRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput (Join-Path $LogRoot "open-llm-vtuber.out.log") `
        -RedirectStandardError (Join-Path $LogRoot "open-llm-vtuber.err.log") | Out-Null

    if (!(Wait-HttpOk -Url "http://127.0.0.1:12393/" -TimeoutSeconds 120)) {
        throw "数字人服务启动失败。请查看日志：$LogRoot"
    }
    Write-Host "[OK] 数字人服务已启动：http://127.0.0.1:12393/"
}

Assert-RequiredFile -Path $UvPath -Description "便携 uv"
Assert-RequiredFile -Path $AppExe -Description "主程序"
Assert-RequiredFile -Path $SeedExe -Description "演示数据初始化程序"
Assert-RequiredFile -Path (Join-Path $AppRoot "static\digital-human\libs\live2dcubismcore.min.js") -Description "Live2D Cubism Core"
Assert-RequiredFile -Path (Join-Path $AppRoot "static\live2d-models\mao_pro\runtime\mao_pro.moc3") -Description "Live2D 模型"
Assert-RequiredFile -Path (Join-Path $VTuberRoot "run_server.py") -Description "数字人服务入口"
Assert-RequiredFile -Path (Join-Path $VTuberRoot "uv.lock") -Description "数字人依赖锁文件"

if ($Restart) {
    Stop-PortListener -Port 8080
    Stop-PortListener -Port 12393
}

Set-ApplicationEnvironment
Initialize-VTuberRuntime
Initialize-DemoDatabase
Start-ScenicGuide
Start-VTuber

Write-Host ""
Write-Host "============================================"
Write-Host "景区智能导览系统已启动"
Write-Host "登录入口：http://127.0.0.1:8080/digital-human#/login"
Write-Host "评委账号与密码会显示在本地登录页，可点击自动填入。"
Write-Host "日志目录：$LogRoot"
Write-Host "============================================"

if (!$NoBrowser) {
    Start-Process "http://127.0.0.1:8080/digital-human#/login"
}
