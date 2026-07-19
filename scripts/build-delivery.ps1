param(
    [string]$OutputDirectory = "",
    [string]$ReleaseLabel = ""
)

$ErrorActionPreference = "Stop"

$ProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$WorkspaceRoot = Split-Path $ProjectRoot -Parent
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $WorkspaceRoot "delivery"
}
$OutputDirectory = [System.IO.Path]::GetFullPath($OutputDirectory)
if ([string]::IsNullOrWhiteSpace($ReleaseLabel)) {
    $ReleaseLabel = Get-Date -Format "yyyyMMdd-HHmm"
}

$stagingRoot = Join-Path $OutputDirectory (".staging-" + $ReleaseLabel)
$executableName = "ScenicGuide-Executable-" + $ReleaseLabel
$sourceName = "ScenicGuide-Source-" + $ReleaseLabel
$executableRoot = Join-Path $stagingRoot $executableName
$sourceRoot = Join-Path $stagingRoot $sourceName
$executableZip = Join-Path $OutputDirectory ($executableName + ".zip")
$sourceZip = Join-Path $OutputDirectory ($sourceName + ".zip")

function Assert-SafeStagingPath {
    $outputFull = [System.IO.Path]::GetFullPath($OutputDirectory).TrimEnd('\') + '\'
    $stagingFull = [System.IO.Path]::GetFullPath($stagingRoot)
    if (!$stagingFull.StartsWith($outputFull, [System.StringComparison]::OrdinalIgnoreCase) -or
        !(Split-Path $stagingFull -Leaf).StartsWith(".staging-")) {
        throw "拒绝使用不安全的打包临时目录：$stagingFull"
    }
}

function Invoke-CheckedCommand {
    param([string]$FilePath, [string[]]$Arguments, [string]$WorkingDirectory)
    Push-Location $WorkingDirectory
    try {
        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$FilePath 执行失败，退出码：$LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }
}

function Copy-DirectoryFiltered {
    param(
        [string]$Source,
        [string]$Destination,
        [string[]]$ExcludeDirectories = @(),
        [string[]]$ExcludeFiles = @()
    )
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    $arguments = @($Source, $Destination, "/E", "/R:1", "/W:1", "/NFL", "/NDL", "/NJH", "/NJS", "/NP")
    if ($ExcludeDirectories.Count -gt 0) {
        $arguments += "/XD"
        $arguments += $ExcludeDirectories
    }
    if ($ExcludeFiles.Count -gt 0) {
        $arguments += "/XF"
        $arguments += $ExcludeFiles
    }
    & robocopy @arguments | Out-Null
    $code = $LASTEXITCODE
    if ($code -ge 8) {
        throw "复制目录失败（robocopy 退出码 $code）：$Source"
    }
}

function Copy-WindowsBatchFile {
    param([string]$Source, [string]$Destination)
    $lines = [System.IO.File]::ReadAllLines($Source)
    $content = ($lines -join "`r`n") + "`r`n"
    [System.IO.File]::WriteAllText($Destination, $content, [System.Text.ASCIIEncoding]::new())
}

function Assert-WindowsBatchFile {
    param([string]$Path)
    $bytes = [System.IO.File]::ReadAllBytes($Path)
    for ($index = 0; $index -lt $bytes.Length; $index++) {
        if ($bytes[$index] -eq 10 -and ($index -eq 0 -or $bytes[$index - 1] -ne 13)) {
            throw "批处理文件包含非 Windows 换行：$Path"
        }
    }
}

function Copy-WindowsPowerShellFile {
    param([string]$Source, [string]$Destination)
    $lines = [System.IO.File]::ReadAllLines($Source, [System.Text.UTF8Encoding]::new($false, $true))
    $content = ($lines -join "`r`n") + "`r`n"
    [System.IO.File]::WriteAllText($Destination, $content, [System.Text.UTF8Encoding]::new($true))
}

function Assert-WindowsPowerShellFile {
    param([string]$Path)
    $bytes = [System.IO.File]::ReadAllBytes($Path)
    if ($bytes.Length -lt 3 -or $bytes[0] -ne 0xEF -or $bytes[1] -ne 0xBB -or $bytes[2] -ne 0xBF) {
        throw "Windows PowerShell 脚本缺少 UTF-8 BOM：$Path"
    }
    for ($index = 3; $index -lt $bytes.Length; $index++) {
        if ($bytes[$index] -eq 10 -and ($index -eq 0 -or $bytes[$index - 1] -ne 13)) {
            throw "Windows PowerShell 脚本包含非 Windows 换行：$Path"
        }
    }
}

function New-SanitizedVTuberConfig {
    param([string]$Source, [string]$Destination)
    $section = ""
    $sourceLines = [System.IO.File]::ReadAllLines($Source, [System.Text.UTF8Encoding]::new($false, $true))
    $result = foreach ($line in $sourceLines) {
        if ($line -match '^\s{6}([A-Za-z0-9_]+):\s*(?:#.*)?$') {
            $section = $matches[1]
        }
        if ($line -match '^(\s*)(api_key|llm_api_key|secret_id|secret_key):(?:.*)$') {
            $indent = $matches[1]
            $key = $matches[2]
            $value = "''"
            if ($section -eq "openai_compatible_llm" -and $key -eq "llm_api_key") {
                $value = "'not-needed'"
            }
            "${indent}${key}: $value"
        } else {
            $line
        }
    }
    [System.IO.File]::WriteAllLines($Destination, $result, [System.Text.UTF8Encoding]::new($false))
}

function Assert-YamlConfigParses {
    param([string]$UvPath, [string]$WorkingDirectory, [string]$Path)
    $code = "from pathlib import Path; import sys; from ruamel.yaml import YAML; YAML(typ='safe').load(Path(sys.argv[1]).read_text(encoding='utf-8'))"
    Invoke-CheckedCommand -FilePath $UvPath -Arguments @("run", "--no-sync", "python", "-c", $code, $Path) -WorkingDirectory $WorkingDirectory
}

function Assert-SanitizedVTuberConfig {
    param([string]$Path)
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -match '^\s*(api_key|llm_api_key|secret_id|secret_key):\s*["'']?([^#"'']*)') {
            $value = $matches[2].Trim()
            if (![string]::IsNullOrWhiteSpace($value) -and $value -ne "not-needed") {
                throw "清理后的数字人配置仍包含非空凭据字段：$($matches[1])"
            }
        }
    }
}

function Copy-VTuberSource {
    param([string]$Destination)
    $source = Join-Path $WorkspaceRoot "Open-LLM-VTuber"
    Copy-DirectoryFiltered -Source $source -Destination $Destination `
        -ExcludeDirectories @(".git", ".venv", "models", "logs", "cache", "chat_history", "__pycache__", "superpowers", ".ruff_cache", ".pytest_cache", ".mypy_cache", ".cursor", ".gemini") `
        -ExcludeFiles @(".env", ".env.*", "conf.yaml", "conf.yaml.backup", "*.log", "server_output*.txt", "server_error*.txt", "CHANGELOG.md", "CLAUDE.md", "PROJECT_OVERVIEW.md", "cleanup-git-history.sh", "数字人表情和语音问题修复说明.md")
    New-SanitizedVTuberConfig -Source (Join-Path $source "conf.yaml") -Destination (Join-Path $Destination "conf.yaml")
    Assert-SanitizedVTuberConfig -Path (Join-Path $Destination "conf.yaml")
}

function Copy-ScenicSource {
    param([string]$Destination)
    Copy-DirectoryFiltered -Source $ProjectRoot -Destination $Destination `
        -ExcludeDirectories @(".git", ".gocache", ".playwright-cli", ".prismo", ".ruff_cache", "node_modules", "data", "logs", "output", "tmp", "coverage", "bin", "dist", "__pycache__", "superpowers") `
        -ExcludeFiles @(".env", ".env.local", ".env.*.local", "config.yaml", "*.db", "*.log", "*.exe", "*.pem", "*.key", "*.crt", "CHANGELOG.md", "CLAUDE.md", "PROJECT_OVERVIEW.md", "cleanup-git-history.sh")
}

function Remove-DeliveryDevelopmentFiles {
    param([string]$Root)
    $relativeFiles = @(
        "scenic-guide\CHANGELOG.md",
        "Open-LLM-VTuber\CHANGELOG.md",
        "Open-LLM-VTuber\CLAUDE.md",
        "Open-LLM-VTuber\PROJECT_OVERVIEW.md",
        "Open-LLM-VTuber\cleanup-git-history.sh",
        "Open-LLM-VTuber\数字人表情和语音问题修复说明.md"
    )
    foreach ($relativeFile in $relativeFiles) {
        $path = Join-Path $Root $relativeFile
        if (Test-Path -LiteralPath $path -PathType Leaf) {
            [System.IO.File]::Delete($path)
        }
    }
    $prismo = Join-Path $Root "scenic-guide\web-vue\.prismo"
    if (Test-Path -LiteralPath $prismo -PathType Container) {
        [System.IO.Directory]::Delete($prismo, $true)
    }

    $vtuberRoot = Join-Path $Root "Open-LLM-VTuber"
    $vtuberMarkdownKeep = @(
        "README.md", "README.CN.md", "README.JP.md", "README.KR.md",
        "LICENSE-Live2D.md", "CONTRIBUTING.md"
    )
    if (Test-Path -LiteralPath $vtuberRoot -PathType Container) {
        Get-ChildItem -LiteralPath $vtuberRoot -File -Filter "*.md" |
            Where-Object { $vtuberMarkdownKeep -notcontains $_.Name } |
            ForEach-Object { [System.IO.File]::Delete($_.FullName) }
    }

    $scenicRoot = Join-Path $Root "scenic-guide"
    $scenicMarkdownKeep = @("README.md", "LICENSE-Live2D.md")
    if (Test-Path -LiteralPath $scenicRoot -PathType Container) {
        Get-ChildItem -LiteralPath $scenicRoot -File -Filter "*.md" |
            Where-Object { $scenicMarkdownKeep -notcontains $_.Name } |
            ForEach-Object { [System.IO.File]::Delete($_.FullName) }
    }
}

function New-ExecutableScenicConfig {
    param([string]$Destination)
    $content = @'
server:
  host: "0.0.0.0"
  port: "8080"

scenic_profile: "lingshan"

database:
  driver: "sqlite"
  path: "./data/scenic_guide.db"

logging:
  level: "info"
  output: "console"

ai:
  api_key: ""
  model: "qwen-vl-max"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"

embedding:
  api_key: ""
  model: "text-embedding-v2"
  base_url: "https://dashscope.aliyuncs.com/api/v1"

multimodal:
  enabled: false
  provider: "qwen"
  model: "qwen3.5-omni-plus"
  base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
  api_key: ""
  timeout_seconds: 60

speech:
  api_key: ""
  region: ""

security:
  jwt_secret: ""
  token_expire_hours: 24

redis:
  addr: ""
  password: ""
  db: 0
'@
    [System.IO.File]::WriteAllText($Destination, $content.TrimStart() + "`r`n", [System.Text.UTF8Encoding]::new($false))
}

function Assert-ExecutableScenicConfig {
    param([string]$Path)
    foreach ($line in Get-Content -LiteralPath $Path) {
        if ($line -match '^\s*(api_key|jwt_secret|password):\s*["'']?([^#"'']*)') {
            if (![string]::IsNullOrWhiteSpace($matches[2])) {
                throw "可执行包演示配置包含非空凭据字段：$($matches[1])"
            }
        }
    }
}

function Copy-ExecutableScenicFiles {
    param([string]$Destination)
    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    foreach ($directory in @("static", "knowledge")) {
        Copy-DirectoryFiltered -Source (Join-Path $ProjectRoot $directory) -Destination (Join-Path $Destination $directory)
    }
    Copy-DirectoryFiltered -Source (Join-Path $ProjectRoot "configs") -Destination (Join-Path $Destination "configs") -ExcludeFiles @("config.yaml")
    $demoConfig = Join-Path $Destination "configs\config.yaml"
    New-ExecutableScenicConfig -Destination $demoConfig
    Assert-ExecutableScenicConfig -Path $demoConfig
    foreach ($file in @("README.md", "LICENSE-Live2D.md")) {
        Copy-Item -LiteralPath (Join-Path $ProjectRoot $file) -Destination (Join-Path $Destination $file) -Force
    }
    New-Item -ItemType Directory -Force -Path (Join-Path $Destination "data") | Out-Null
}

function Assert-ZipSafe {
    param([string]$Path)
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [System.IO.Compression.ZipFile]::OpenRead($Path)
    try {
        if ($archive.Entries.Count -eq 0) {
            throw "压缩包为空：$Path"
        }
        foreach ($entry in $archive.Entries) {
            $name = $entry.FullName.Replace('\', '/')
            if ($name.StartsWith('/') -or $name -match '(^|/)\.\.(/|$)' -or $name -match '^[A-Za-z]:') {
                throw "压缩包包含不安全路径：$name"
            }
        }
    } finally {
        $archive.Dispose()
    }
}

function Write-Manifest {
    param([string]$Root, [string]$PackageType)
    $files = Get-ChildItem -LiteralPath $Root -Recurse -Force -File
    $size = ($files | Measure-Object Length -Sum).Sum
    $lines = @(
        "Package: $PackageType",
        "Release: $ReleaseLabel",
        "Generated: $((Get-Date).ToString('yyyy-MM-dd HH:mm:ss zzz'))",
        "Files: $($files.Count)",
        "UncompressedBytes: $size",
        "ContainsRealApiKeys: false",
        "ContainsASRModel: false",
        "ASRModelBehavior: downloaded on first launch"
    )
    [System.IO.File]::WriteAllLines((Join-Path $Root "PACKAGE-MANIFEST.txt"), $lines, [System.Text.UTF8Encoding]::new($false))
}

function Write-HashFile {
    param([string]$Path)
    $hash = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    $line = "$hash  $(Split-Path $Path -Leaf)"
    [System.IO.File]::WriteAllText(($Path + ".sha256.txt"), $line + [Environment]::NewLine, [System.Text.UTF8Encoding]::new($false))
}

Assert-SafeStagingPath
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
if (Test-Path -LiteralPath $stagingRoot) {
    Remove-Item -LiteralPath $stagingRoot -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $executableRoot, $sourceRoot | Out-Null

try {
    Write-Host "[1/7] 构建 Vue 前端..."
    Invoke-CheckedCommand -FilePath "npm" -Arguments @("run", "build") -WorkingDirectory (Join-Path $ProjectRoot "web-vue")

    . (Join-Path $ProjectRoot "scripts\live2d-assets.ps1")
    Sync-ScenicGuideLive2DAssets -ProjectRoot $ProjectRoot -WorkspaceRoot $WorkspaceRoot

    Write-Host "[2/7] 构建 Windows 可执行程序..."
    $executableScenic = Join-Path $executableRoot "scenic-guide"
    Copy-ExecutableScenicFiles -Destination $executableScenic
    Invoke-CheckedCommand -FilePath "go" -Arguments @("build", "-trimpath", "-ldflags", "-s -w", "-o", (Join-Path $executableScenic "scenic-guide.exe"), ".") -WorkingDirectory $ProjectRoot
    Invoke-CheckedCommand -FilePath "go" -Arguments @("build", "-trimpath", "-ldflags", "-s -w", "-o", (Join-Path $executableScenic "demo-seed.exe"), "./cmd/demo-seed") -WorkingDirectory $ProjectRoot

    Write-Host "[3/7] 组装数字人运行目录..."
    Copy-VTuberSource -Destination (Join-Path $executableRoot "Open-LLM-VTuber")
    $uvPath = (Get-Command uv -ErrorAction Stop).Source
    New-Item -ItemType Directory -Force -Path (Join-Path $executableRoot "tools") | Out-Null
    Copy-Item -LiteralPath $uvPath -Destination (Join-Path $executableRoot "tools\uv.exe") -Force
    $launcherPowerShell = Join-Path $executableRoot "Start-Demo.ps1"
    Copy-WindowsPowerShellFile -Source (Join-Path $ProjectRoot "scripts\delivery\Start-Demo.ps1") -Destination $launcherPowerShell
    Assert-WindowsPowerShellFile -Path $launcherPowerShell
    $launcherBatch = Join-Path $executableRoot "START-DEMO.bat"
    Copy-WindowsBatchFile -Source (Join-Path $ProjectRoot "scripts\delivery\START-DEMO.bat") -Destination $launcherBatch
    Assert-WindowsBatchFile -Path $launcherBatch
    $onlineLlmPowerShell = Join-Path $executableRoot "Configure-Online-LLM.ps1"
    Copy-WindowsPowerShellFile -Source (Join-Path $ProjectRoot "scripts\delivery\Configure-Online-LLM.ps1") -Destination $onlineLlmPowerShell
    Assert-WindowsPowerShellFile -Path $onlineLlmPowerShell
    Copy-Item -LiteralPath (Join-Path $ProjectRoot "scripts\delivery\README-START-HERE.txt") -Destination $executableRoot -Force

    Write-Host "[4/7] 组装双项目源码目录..."
    Copy-ScenicSource -Destination (Join-Path $sourceRoot "scenic-guide")
    Copy-VTuberSource -Destination (Join-Path $sourceRoot "Open-LLM-VTuber")
    Remove-DeliveryDevelopmentFiles -Root $executableRoot
    Remove-DeliveryDevelopmentFiles -Root $sourceRoot
    Write-Manifest -Root $executableRoot -PackageType "Windows executable demo"
    Write-Manifest -Root $sourceRoot -PackageType "Source code"

    Write-Host "[5/7] 验证配置并运行源码秘密扫描..."
    $vtuberWorkingDirectory = Join-Path $WorkspaceRoot "Open-LLM-VTuber"
    Assert-YamlConfigParses -UvPath $uvPath -WorkingDirectory $vtuberWorkingDirectory -Path (Join-Path $executableRoot "Open-LLM-VTuber\conf.yaml")
    Assert-YamlConfigParses -UvPath $uvPath -WorkingDirectory $vtuberWorkingDirectory -Path (Join-Path $sourceRoot "Open-LLM-VTuber\conf.yaml")
    Invoke-CheckedCommand -FilePath "node" -Arguments @("scripts/check-secrets.mjs") -WorkingDirectory (Join-Path $sourceRoot "scenic-guide")

    Write-Host "[6/7] 创建 ZIP 压缩包..."
    foreach ($archive in @($executableZip, $sourceZip)) {
        if (Test-Path -LiteralPath $archive) {
            Remove-Item -LiteralPath $archive -Force
        }
    }
    Compress-Archive -LiteralPath $executableRoot -DestinationPath $executableZip -CompressionLevel Optimal
    Compress-Archive -LiteralPath $sourceRoot -DestinationPath $sourceZip -CompressionLevel Optimal

    Write-Host "[7/7] 校验大小、路径与哈希..."
    foreach ($archive in @($executableZip, $sourceZip)) {
        Assert-ZipSafe -Path $archive
        $size = (Get-Item -LiteralPath $archive).Length
        if ($size -ge 1GB) {
            throw "压缩包超过 1GB：$archive"
        }
        Write-HashFile -Path $archive
        Write-Host ("[OK] {0}: {1:N2} MB" -f (Split-Path $archive -Leaf), ($size / 1MB))
    }
} finally {
    if (Test-Path -LiteralPath $stagingRoot) {
        Remove-Item -LiteralPath $stagingRoot -Recurse -Force
    }
}

Write-Host "交付压缩包已生成：$OutputDirectory"
