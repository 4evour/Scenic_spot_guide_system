param(
    [switch]$Restart,
    [switch]$NoBrowser
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$WorkspaceRoot = Split-Path $ProjectRoot -Parent
$LogRoot = Join-Path $WorkspaceRoot "tmp\scenic-guide-start"
$AdminPassword = "ScenicDemo123456"

New-Item -ItemType Directory -Force -Path $LogRoot | Out-Null

function Import-DotEnv {
    param([string]$Path)
    if (!(Test-Path $Path)) {
        return
    }

    Get-Content $Path | ForEach-Object {
        $line = $_.Trim().TrimStart([char]0xFEFF)
        if (!$line -or $line.StartsWith("#") -or !$line.Contains("=")) {
            return
        }

        $parts = $line.Split("=", 2)
        $name = $parts[0].Trim()
        $value = $parts[1].Trim().Trim('"').Trim("'")
        if ($name -and [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($name, "Process"))) {
            Set-Item -Path "Env:$name" -Value $value | Out-Null
        }
    }
}

function Get-ListenerProcessId {
    param([int]$Port)
    $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($conn) {
        return [int]$conn.OwningProcess
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
    param(
        [string]$Url,
        [int]$TimeoutSeconds = 45
    )
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 2 | Out-Null
            return $true
        } catch {
            Start-Sleep -Milliseconds 800
        }
    }
    return $false
}

function Set-EnvDefault {
    param(
        [string]$Name,
        [string]$Value
    )
    $current = [Environment]::GetEnvironmentVariable($Name, "Process")
    if ([string]::IsNullOrWhiteSpace($current)) {
        Set-Item -Path "Env:$Name" -Value $Value | Out-Null
        return $true
    }
    return $false
}

function Get-EnvValue {
    param([string]$Name)
    foreach ($scope in @("Process", "User", "Machine")) {
        $value = [Environment]::GetEnvironmentVariable($Name, $scope)
        if (![string]::IsNullOrWhiteSpace($value)) {
            return $value
        }
    }
    return $null
}

function Set-EnvDefaultFromName {
    param(
        [string]$TargetName,
        [string]$SourceName
    )
    if (![string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($TargetName, "Process"))) {
        return $false
    }

    $value = Get-EnvValue -Name $SourceName
    if ([string]::IsNullOrWhiteSpace($value)) {
        return $false
    }

    Set-Item -Path "Env:$TargetName" -Value $value | Out-Null
    return $true
}

function Resolve-ScenicGuideAIEnv {
    if (![string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable("SCENIC_GUIDE_AI_API_KEY", "Process"))) {
        return "scenic"
    }

    if (Set-EnvDefaultFromName -TargetName "SCENIC_GUIDE_AI_API_KEY" -SourceName "DEEPSEEK_API_KEY") {
        Set-EnvDefault -Name "SCENIC_GUIDE_AI_BASE_URL" -Value "https://api.deepseek.com/v1" | Out-Null
        Set-EnvDefault -Name "SCENIC_GUIDE_AI_MODEL" -Value "deepseek-chat" | Out-Null
        return "deepseek"
    }

    if (Set-EnvDefaultFromName -TargetName "SCENIC_GUIDE_AI_API_KEY" -SourceName "QWEN_API_KEY") {
        Set-EnvDefault -Name "SCENIC_GUIDE_AI_BASE_URL" -Value "https://dashscope.aliyuncs.com/compatible-mode/v1" | Out-Null
        Set-EnvDefault -Name "SCENIC_GUIDE_AI_MODEL" -Value "qwen-turbo" | Out-Null
        Set-EnvDefault -Name "SCENIC_GUIDE_EMBEDDING_API_KEY" -Value $env:SCENIC_GUIDE_AI_API_KEY | Out-Null
        return "qwen"
    }

    if (Set-EnvDefaultFromName -TargetName "SCENIC_GUIDE_AI_API_KEY" -SourceName "DASHSCOPE_API_KEY") {
        Set-EnvDefault -Name "SCENIC_GUIDE_AI_BASE_URL" -Value "https://dashscope.aliyuncs.com/compatible-mode/v1" | Out-Null
        Set-EnvDefault -Name "SCENIC_GUIDE_AI_MODEL" -Value "qwen-turbo" | Out-Null
        Set-EnvDefault -Name "SCENIC_GUIDE_EMBEDDING_API_KEY" -Value $env:SCENIC_GUIDE_AI_API_KEY | Out-Null
        return "dashscope"
    }

    Set-EnvDefault -Name "SCENIC_GUIDE_AI_API_KEY" -Value "local-dev-placeholder" | Out-Null
    return "placeholder"
}

function Set-ScenicGuideEnv {
    Import-DotEnv -Path (Join-Path $ProjectRoot ".env")
    Import-DotEnv -Path (Join-Path $ProjectRoot ".env.local")

    $env:SCENIC_GUIDE_SECURITY_JWT_SECRET = "local-dev-jwt-secret-20260615-32chars"
    $aiSource = Resolve-ScenicGuideAIEnv
    $env:SCENIC_GUIDE_API_KEY = "not-needed"
    $env:SCENIC_GUIDE_DATABASE_DRIVER = "sqlite"
    $env:SCENIC_GUIDE_DATABASE_PATH = "./data/scenic_guide.db"

    if ($aiSource -eq "placeholder") {
        Write-Warning "SCENIC_GUIDE_AI_API_KEY is not set. Using local placeholder; RAG retrieval works, LLM generation falls back locally."
    } else {
        Write-Host "[OK] Real LLM key detected from $aiSource; start script will not print or overwrite it"
    }
}

function Initialize-ScenicGuide {
    Push-Location $ProjectRoot
    try {
        Set-ScenicGuideEnv
        go run ./cmd/demo-seed --admin-password $AdminPassword
    } finally {
        Pop-Location
    }
}

function Start-ScenicGuide {
    if (Get-ListenerProcessId -Port 8080) {
        Write-Host "[OK] scenic-guide is already running on port 8080"
        return
    }

    Push-Location $ProjectRoot
    try {
        Set-ScenicGuideEnv
        $env:SCENIC_GUIDE_ADMIN_PASSWORD = $AdminPassword

        Write-Host "[START] scenic-guide: go run ."
        Start-Process -FilePath "go" `
            -ArgumentList @("run", ".") `
            -WorkingDirectory $ProjectRoot `
            -WindowStyle Hidden `
            -RedirectStandardOutput (Join-Path $LogRoot "scenic-guide.out.log") `
            -RedirectStandardError (Join-Path $LogRoot "scenic-guide.err.log") | Out-Null
    } finally {
        Pop-Location
    }

    if (Wait-HttpOk -Url "http://127.0.0.1:8080/health" -TimeoutSeconds 60) {
        Write-Host "[OK] scenic-guide started: http://127.0.0.1:8080/"
    } else {
        Write-Warning "scenic-guide health check did not pass within 60 seconds. Logs: $LogRoot"
    }
}

function Start-OpenLLMVTuber {
    $vtuberRoot = Join-Path $WorkspaceRoot "Open-LLM-VTuber"
    if (!(Test-Path $vtuberRoot)) {
        Write-Warning "Open-LLM-VTuber directory not found: $vtuberRoot"
        return
    }

    if (Get-ListenerProcessId -Port 12393) {
        Write-Host "[OK] Open-LLM-VTuber is already running on port 12393"
        return
    }

    $venvPython = Join-Path $vtuberRoot ".venv\Scripts\python.exe"
    $pythonPath = if (Test-Path $venvPython) { $venvPython } else { "python" }

    Write-Host "[START] Open-LLM-VTuber: python run_server.py"
    Start-Process -FilePath $pythonPath `
        -ArgumentList @("run_server.py") `
        -WorkingDirectory $vtuberRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput (Join-Path $LogRoot "open-llm-vtuber.out.log") `
        -RedirectStandardError (Join-Path $LogRoot "open-llm-vtuber.err.log") | Out-Null

    if (Wait-HttpOk -Url "http://127.0.0.1:12393/" -TimeoutSeconds 90) {
        Write-Host "[OK] Open-LLM-VTuber started: http://127.0.0.1:12393/"
    } else {
        Write-Warning "Open-LLM-VTuber health check did not pass within 90 seconds. Logs: $LogRoot"
    }
}

if ($Restart) {
    Write-Host "[RESTART] Stopping existing listener on port 8080"
    Stop-PortListener -Port 8080
    Write-Host "[RESTART] Stopping existing listener on port 12393"
    Stop-PortListener -Port 12393
}

if (-not (Get-ListenerProcessId -Port 8080)) {
    Initialize-ScenicGuide
}

Start-ScenicGuide
Start-OpenLLMVTuber

Write-Host ""
Write-Host "URLs:"
Write-Host "  Login: http://127.0.0.1:8080/digital-human#/login"
Write-Host "  Digital human: http://127.0.0.1:8080/digital-human#/digital-human"
Write-Host "  Open-LLM-VTuber: http://127.0.0.1:12393/"
Write-Host "  Admin knowledge: http://127.0.0.1:8080/digital-human#/admin/knowledge"
Write-Host "  Health: http://127.0.0.1:8080/health"
Write-Host ""
Write-Host "Login accounts:"
Write-Host "  Admin: admin / $AdminPassword"
Write-Host "  Visitor: visitor / $AdminPassword"
Write-Host ""
Write-Host "Log directory: $LogRoot"

if (-not $NoBrowser) {
    Start-Process "http://127.0.0.1:8080/digital-human#/login"
}
