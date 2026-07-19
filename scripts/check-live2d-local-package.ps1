$ErrorActionPreference = "Stop"

$ProjectRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$WorkspaceRoot = Split-Path $ProjectRoot -Parent
. (Join-Path $ProjectRoot "scripts\live2d-assets.ps1")

function Assert-Condition {
    param(
        [bool]$Condition,
        [string]$Message
    )
    if (!$Condition) {
        throw $Message
    }
}

function Write-TestAsset {
    param([string]$Path)
    New-Item -ItemType Directory -Force -Path (Split-Path $Path -Parent) | Out-Null
    [System.IO.File]::WriteAllText($Path, "test asset")
}

$actualVTuberRoot = Join-Path $WorkspaceRoot "Open-LLM-VTuber"
Assert-Live2DAssetFile -Path (Join-Path $actualVTuberRoot "frontend\libs\live2dcubismcore.min.js") -Description "packaged Cubism Core"
Assert-Live2DAssetFile -Path (Join-Path $actualVTuberRoot "live2d-models\mao_pro\runtime\mao_pro.model3.json") -Description "packaged mao_pro model configuration"
Assert-Live2DAssetFile -Path (Join-Path $actualVTuberRoot "live2d-models\mao_pro\runtime\mao_pro.moc3") -Description "packaged mao_pro model binary"

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("scenic-guide-live2d-assets-" + [guid]::NewGuid().ToString("N"))
$tempBase = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath())
$resolvedTempRoot = [System.IO.Path]::GetFullPath($tempRoot)
if (!$resolvedTempRoot.StartsWith($tempBase, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to use an unsafe temporary test path: $resolvedTempRoot"
}

try {
    $workspaceRoot = Join-Path $tempRoot "workspace"
    $fixtureProjectRoot = Join-Path $workspaceRoot "scenic-guide"
    $vtuberRoot = Join-Path $workspaceRoot "Open-LLM-VTuber"
    Write-TestAsset (Join-Path $vtuberRoot "frontend\libs\live2dcubismcore.min.js")
    Write-TestAsset (Join-Path $vtuberRoot "live2d-models\mao_pro\runtime\mao_pro.model3.json")
    Write-TestAsset (Join-Path $vtuberRoot "live2d-models\mao_pro\runtime\mao_pro.moc3")

    Sync-ScenicGuideLive2DAssets -ProjectRoot $fixtureProjectRoot -WorkspaceRoot $workspaceRoot
    Sync-ScenicGuideLive2DAssets -ProjectRoot $fixtureProjectRoot -WorkspaceRoot $workspaceRoot
    Assert-Condition (Test-Path -LiteralPath (Join-Path $fixtureProjectRoot "static\digital-human\libs\live2dcubismcore.min.js") -PathType Leaf) "Cubism Core was not deployed"
    Assert-Condition (Test-Path -LiteralPath (Join-Path $fixtureProjectRoot "static\live2d-models\mao_pro\runtime\mao_pro.moc3") -PathType Leaf) "Live2D model binary was not deployed"
    Assert-Condition (!(Test-Path -LiteralPath (Join-Path $fixtureProjectRoot "static\live2d-models\mao_pro\mao_pro"))) "Repeated deployment created a nested mao_pro directory"

    $missingWorkspaceRoot = Join-Path $tempRoot "missing-workspace"
    $missingProjectRoot = Join-Path $missingWorkspaceRoot "scenic-guide"
    $missingFailed = $false
    try {
        Sync-ScenicGuideLive2DAssets -ProjectRoot $missingProjectRoot -WorkspaceRoot $missingWorkspaceRoot
    } catch {
        $missingFailed = $_.Exception.Message -like "Live2D required asset is missing*"
    }
    Assert-Condition $missingFailed "Missing Cubism Core did not fail the deployment"

    $startScript = Get-Content (Join-Path $ProjectRoot "scripts\start-local.ps1") -Raw
    $syncCall = "Sync-ScenicGuideLive2DAssets -ProjectRoot `$ProjectRoot -WorkspaceRoot `$WorkspaceRoot"
    Assert-Condition ($startScript.Contains($syncCall)) "start-local.ps1 does not deploy Live2D assets"
    $initializeCall = "    Initialize-ScenicGuide"
    Assert-Condition ($startScript.IndexOf($syncCall) -lt $startScript.LastIndexOf($initializeCall)) "Live2D assets are not deployed before initialization"
    Write-Host "Live2D local package check passed."
} finally {
    if (Test-Path -LiteralPath $resolvedTempRoot) {
        Remove-Item -LiteralPath $resolvedTempRoot -Recurse -Force
    }
}
