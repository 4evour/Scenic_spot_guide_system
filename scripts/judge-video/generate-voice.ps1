param(
  [string]$ManifestPath = "scripts/judge-video/manifest.json",
  [string]$OutputDir = "tmp/video-work/voice",
  [string]$Voice = "zh-CN-YunjianNeural",
  [string]$Rate = "-8%",
  [string]$SceneId = "",
  [switch]$AllowLocalFallback,
  [switch]$SubtitlesOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$projectRoot = (Get-Location).Path
$manifestFile = Join-Path $projectRoot $ManifestPath
$outputPath = Join-Path $projectRoot $OutputDir
$edgeTts = Join-Path $projectRoot "..\Open-LLM-VTuber\.venv\Scripts\edge-tts.exe"

if (-not (Test-Path -LiteralPath $manifestFile)) { throw "Manifest not found: $manifestFile" }
if (-not (Test-Path -LiteralPath $edgeTts)) { throw "Edge TTS executable not found: $edgeTts" }
New-Item -ItemType Directory -Force -Path $outputPath | Out-Null

function Format-SrtTime([double]$Seconds) {
  $span = [TimeSpan]::FromMilliseconds([Math]::Round([Math]::Max(0, $Seconds) * 1000))
  $hours = [Math]::Floor($span.TotalHours)
  return ("{0:00}:{1:00}:{2:00},{3:000}" -f $hours, $span.Minutes, $span.Seconds, $span.Milliseconds)
}

function Get-SubtitleChunks([string]$Text, [int]$MaxChars = 22) {
  $clauses = [regex]::Matches($Text, '[^\u3002\uff01\uff1f\uff1b\uff0c]+[\u3002\uff01\uff1f\uff1b\uff0c]?')
  $chunks = New-Object System.Collections.Generic.List[string]
  $current = ""
  foreach ($clauseMatch in $clauses) {
    $clause = $clauseMatch.Value.Trim()
    if ([string]::IsNullOrWhiteSpace($clause)) { continue }
    if ($clause.Length -gt $MaxChars) {
      if ($current) { $chunks.Add($current); $current = "" }
      for ($offset = 0; $offset -lt $clause.Length; $offset += $MaxChars) {
        $chunks.Add($clause.Substring($offset, [Math]::Min($MaxChars, $clause.Length - $offset)))
      }
    } elseif (($current + $clause).Length -le $MaxChars) {
      $current += $clause
    } else {
      if ($current) { $chunks.Add($current) }
      $current = $clause
    }
  }
  if ($current) { $chunks.Add($current) }
  return $chunks.ToArray()
}

function Write-SceneSubtitles([string]$Text, [double]$AudioDurationSec, [double]$SceneDurationSec, [string]$OutputFile) {
  $chunks = @(Get-SubtitleChunks $Text)
  if ($chunks.Count -eq 0) { throw "No subtitle chunks generated for $OutputFile" }
  $totalChars = ($chunks | ForEach-Object { $_.Length } | Measure-Object -Sum).Sum
  $sceneOffset = [Math]::Max(0, ($SceneDurationSec - $AudioDurationSec) / 2)
  $cursor = $sceneOffset + 0.15
  $usableDuration = [Math]::Max(0.5, $AudioDurationSec - 0.3)
  $lines = @()
  for ($index = 0; $index -lt $chunks.Count; $index++) {
    $chunkDuration = $usableDuration * ($chunks[$index].Length / $totalChars)
    $end = if ($index -eq $chunks.Count - 1) { $sceneOffset + $AudioDurationSec - 0.15 } else { $cursor + $chunkDuration }
    $lines += [string]($index + 1)
    $lines += "$(Format-SrtTime $cursor) --> $(Format-SrtTime $end)"
    $lines += $chunks[$index]
    $lines += ""
    $cursor = $end
  }
  $lines | Set-Content -LiteralPath $OutputFile -Encoding UTF8
}

$manifest = Get-Content -Raw -LiteralPath $manifestFile -Encoding UTF8 | ConvertFrom-Json
$scenes = if ($SceneId) { @($manifest.scenes | Where-Object { $_.id -eq $SceneId }) } else { @($manifest.scenes) }
if ($scenes.Count -eq 0) { throw "Scene not found: $SceneId" }
$result = @()
foreach ($scene in $scenes) {
  $id = [string]$scene.voiceSegment
  $text = ([string]$scene.narration).Trim()
  $sceneDuration = [double]$scene.durationSec
  if ($id -notmatch '^[a-z0-9-]+$') { throw "Invalid voice segment id: $id" }
  if ([string]::IsNullOrWhiteSpace($text)) { throw "Empty narration for $id" }

  $rawPath = Join-Path $outputPath "$id.raw.mp3"
  $sapiPath = Join-Path $outputPath "$id.raw.wav"
  $finalPath = Join-Path $outputPath "$id.mp3"
  $subtitlePath = Join-Path $outputPath "$id.srt"
  if ($SubtitlesOnly) {
    if (-not (Test-Path -LiteralPath $finalPath)) { throw "Voice asset missing for subtitle generation: $finalPath" }
    $probe = (& ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 $finalPath).Trim()
    $audioDuration = [double]$probe
    $coverage = $audioDuration / $sceneDuration
    Write-SceneSubtitles -Text $text -AudioDurationSec $audioDuration -SceneDurationSec $sceneDuration -OutputFile $subtitlePath
    $result += [ordered]@{
      id = $id
      voice = $Voice
      rate = $Rate
      durationSec = $audioDuration
      sceneDurationSec = $sceneDuration
      coveragePercent = [Math]::Round($coverage * 100, 1)
      subtitle = [IO.Path]::GetFileName($subtitlePath)
      sizeBytes = (Get-Item -LiteralPath $finalPath).Length
    }
    continue
  }
  $generated = $false
  for ($attempt = 1; $attempt -le 5; $attempt++) {
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
    if ($attempt -lt 5) { Start-Sleep -Seconds 2 }
  }

  $sourcePath = $rawPath
  $voiceUsed = $Voice
  if (-not $generated) {
    if (-not $AllowLocalFallback) { throw "Edge TTS failed for $id after five attempts" }
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
  $audioDuration = [double]$probe
  $coverage = $audioDuration / $sceneDuration
  $minimumCoverage = if ($scene.kind -eq "page") { 0.72 } else { 0.62 }
  if ($audioDuration -gt $sceneDuration) { throw "Narration exceeds scene duration for ${id}: $audioDuration > $sceneDuration" }
  if ($coverage -lt $minimumCoverage) { throw "Narration coverage too low for ${id}: $([Math]::Round($coverage * 100, 1))%" }

  Write-SceneSubtitles -Text $text -AudioDurationSec $audioDuration -SceneDurationSec $sceneDuration -OutputFile $subtitlePath
  $size = (Get-Item -LiteralPath $finalPath).Length
  $result += [ordered]@{
    id = $id
    voice = $voiceUsed
    rate = $Rate
    durationSec = $audioDuration
    sceneDurationSec = $sceneDuration
    coveragePercent = [Math]::Round($coverage * 100, 1)
    subtitle = [IO.Path]::GetFileName($subtitlePath)
    sizeBytes = $size
  }
}

$result | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $outputPath "voice-result.json") -Encoding UTF8
Write-Output (ConvertTo-Json -InputObject $result -Depth 4)
