function Assert-Live2DAssetFile {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Description
    )

    if (!(Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Live2D required asset is missing ($Description): $Path"
    }
    if ((Get-Item -LiteralPath $Path).Length -le 0) {
        throw "Live2D required asset is empty ($Description): $Path"
    }
}

function Sync-ScenicGuideLive2DAssets {
    param(
        [Parameter(Mandatory = $true)][string]$ProjectRoot,
        [Parameter(Mandatory = $true)][string]$WorkspaceRoot
    )

    $vtuberRoot = Join-Path $WorkspaceRoot "Open-LLM-VTuber"
    $coreSource = Join-Path $vtuberRoot "frontend\libs\live2dcubismcore.min.js"
    $modelsSource = Join-Path $vtuberRoot "live2d-models"
    $maoProSource = Join-Path $modelsSource "mao_pro"
    $modelConfigSource = Join-Path $maoProSource "runtime\mao_pro.model3.json"
    $modelBinarySource = Join-Path $maoProSource "runtime\mao_pro.moc3"

    Assert-Live2DAssetFile -Path $coreSource -Description "Cubism Core"
    Assert-Live2DAssetFile -Path $modelConfigSource -Description "mao_pro model configuration"
    Assert-Live2DAssetFile -Path $modelBinarySource -Description "mao_pro model binary"

    $coreDestination = Join-Path $ProjectRoot "static\digital-human\libs\live2dcubismcore.min.js"
    $modelsDestination = Join-Path $ProjectRoot "static\live2d-models"
    New-Item -ItemType Directory -Force -Path (Split-Path $coreDestination -Parent) | Out-Null
    New-Item -ItemType Directory -Force -Path $modelsDestination | Out-Null

    Copy-Item -LiteralPath $coreSource -Destination $coreDestination -Force
    Copy-Item -LiteralPath $maoProSource -Destination $modelsDestination -Recurse -Force

    Assert-Live2DAssetFile -Path $coreDestination -Description "deployed Cubism Core"
    Assert-Live2DAssetFile -Path (Join-Path $modelsDestination "mao_pro\runtime\mao_pro.model3.json") -Description "deployed mao_pro model configuration"
    Assert-Live2DAssetFile -Path (Join-Path $modelsDestination "mao_pro\runtime\mao_pro.moc3") -Description "deployed mao_pro model binary"
    Write-Host "[OK] Live2D Cubism Core and models deployed for local demo"
}
