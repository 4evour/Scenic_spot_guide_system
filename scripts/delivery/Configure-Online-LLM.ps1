param([switch]$Clear)

$ErrorActionPreference = "Stop"
$envFile = Join-Path $PSScriptRoot "scenic-guide\.env.local"

if ($Clear) {
    if (Test-Path -LiteralPath $envFile -PathType Leaf) {
        Remove-Item -LiteralPath $envFile -Force
    }
    Write-Host "已清除本机在线大模型配置。"
    exit 0
}

Write-Host "该配置完全可选；不配置时系统仍可使用本地知识库回答。"
Write-Host "Key 只保存在当前解压目录的 scenic-guide\.env.local，不会上传或写入压缩包。"
$provider = (Read-Host "请选择服务商：deepseek 或 qwen（直接回车使用 deepseek）").Trim().ToLowerInvariant()
if ([string]::IsNullOrWhiteSpace($provider)) {
    $provider = "deepseek"
}
if ($provider -notin @("deepseek", "qwen")) {
    throw "只支持 deepseek 或 qwen。"
}

$secureKey = Read-Host "请输入你自己的 API Key" -AsSecureString
$bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureKey)
try {
    $apiKey = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr)
} finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr)
}
if ([string]::IsNullOrWhiteSpace($apiKey)) {
    throw "API Key 不能为空。"
}

if ($provider -eq "qwen") {
    $baseUrl = "https://dashscope.aliyuncs.com/compatible-mode/v1"
    $model = "qwen-turbo"
} else {
    $baseUrl = "https://api.deepseek.com/v1"
    $model = "deepseek-chat"
}

$lines = @(
    "SCENIC_GUIDE_AI_API_KEY=$apiKey",
    "SCENIC_GUIDE_AI_BASE_URL=$baseUrl",
    "SCENIC_GUIDE_AI_MODEL=$model"
)
[System.IO.File]::WriteAllLines($envFile, $lines, [System.Text.UTF8Encoding]::new($false))
$apiKey = $null
Write-Host "[OK] 在线大模型配置已保存。重新运行 START-DEMO.bat 后生效。"
