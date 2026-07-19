param(
  [string]$ScriptPath = "docs/video/judge-demo-script.md",
  [string]$OutputDir = "tmp/video-work/voice",
  [string]$Voice = "zh-CN-YunjianNeural",
  [string]$Rate = "-8%"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$projectRoot = (Get-Location).Path
$scriptFile = Join-Path $projectRoot $ScriptPath
$outputPath = Join-Path $projectRoot $OutputDir
$edgeTts = Join-Path $projectRoot "..\Open-LLM-VTuber\.venv\Scripts\edge-tts.exe"

if (-not (Test-Path -LiteralPath $scriptFile)) { throw "Narration script not found: $scriptFile" }
if (-not (Test-Path -LiteralPath $edgeTts)) { throw "Edge TTS executable not found: $edgeTts" }
New-Item -ItemType Directory -Force -Path $outputPath | Out-Null

$segments = @{}
$segmentIds = @("01-intro", "02-visitor", "03-ai", "04-digital-human", "05-operations", "06-close")
$matchIndex = 0
$lines = Get-Content -LiteralPath $scriptFile -Encoding UTF8
for ($lineIndex = 0; $lineIndex -lt $lines.Count; $lineIndex++) {
  if ($lines[$lineIndex] -notmatch '^###\s+') { continue }
  if ($matchIndex -ge $segmentIds.Count) { break }
  $heading = $lines[$lineIndex].Trim()
  $copyLines = @()
  for ($copyIndex = $lineIndex + 1; $copyIndex -lt $lines.Count; $copyIndex++) {
    if ($lines[$copyIndex] -match '^##\s+') { break }
    if ($lines[$copyIndex] -match '^###\s+') { break }
    if (-not [string]::IsNullOrWhiteSpace($lines[$copyIndex])) { $copyLines += $lines[$copyIndex] }
  }
  $text = ($copyLines -join ' ' -replace '\s+', ' ').Trim()
  if ([string]::IsNullOrWhiteSpace($text)) { throw "Empty narration for $heading" }
  $segments[$segmentIds[$matchIndex]] = $text
  $matchIndex++
}

if ($segments.Count -ne 6) { throw "Expected six narration segments, found $($segments.Count)" }

$result = @()
foreach ($id in @("01-intro", "02-visitor", "03-ai", "04-digital-human", "05-operations", "06-close")) {
  $rawPath = Join-Path $outputPath "$id.raw.mp3"
  $sapiPath = Join-Path $outputPath "$id.raw.wav"
  $finalPath = Join-Path $outputPath "$id.mp3"
  $text = $segments[$id]
  $generated = $false
  for ($attempt = 1; $attempt -le 3; $attempt++) {
    Remove-Item -LiteralPath $rawPath -Force -ErrorAction SilentlyContinue
    $previousErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & $edgeTts "--voice=$Voice" "--rate=$Rate" "--text=$text" "--write-media=$rawPath" 2>&1 | Out-Null
    $edgeExitCode = $LASTEXITCODE
    $ErrorActionPreference = $previousErrorAction
    if ($edgeExitCode -eq 0 -and (Test-Path -LiteralPath $rawPath)) {
      $generated = $true
      break
    }
    if ($attempt -lt 3) { Start-Sleep -Seconds 2 }
  }
  $sourcePath = $rawPath
  $voiceUsed = $Voice
  if (-not $generated) {
    Add-Type -AssemblyName System.Speech
    $synth = New-Object System.Speech.Synthesis.SpeechSynthesizer
    try {
      $synth.SelectVoice("Microsoft Kangkang")
      $synth.Rate = -1
      $synth.SetOutputToWaveFile($sapiPath)
      $synth.Speak($text)
    } finally {
      $synth.Dispose()
    }
    if (-not (Test-Path -LiteralPath $sapiPath)) { throw "Local male voice fallback failed for $id" }
    $sourcePath = $sapiPath
    $voiceUsed = "Microsoft Kangkang"
  }

  & ffmpeg -y -loglevel error -i $sourcePath -af "loudnorm=I=-16:TP=-1.5:LRA=11" -ar 48000 -ac 2 -codec:a libmp3lame -b:a 160k $finalPath
  if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $finalPath)) { throw "Audio normalization failed for $id" }
  Remove-Item -LiteralPath $rawPath, $sapiPath -Force -ErrorAction SilentlyContinue
  $probe = (& ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 $finalPath).Trim()
  $size = (Get-Item -LiteralPath $finalPath).Length
  $result += [ordered]@{ id = $id; voice = $voiceUsed; rate = $Rate; durationSec = [double]$probe; sizeBytes = $size }
}

$result | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $outputPath "voice-result.json") -Encoding UTF8
Write-Output (ConvertTo-Json -InputObject $result -Depth 4)
