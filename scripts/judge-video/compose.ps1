param(
  [string]$ManifestPath = "scripts/judge-video/manifest.json",
  [string]$WorkDir = "tmp/video-work",
  [string]$OutputDir = "output/video",
  [switch]$WithBgm,
  [double]$PageStartOffset = 0.8
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$projectRoot = (Get-Location).Path
$manifestFile = Join-Path $projectRoot $ManifestPath
$workPath = Join-Path $projectRoot $WorkDir
$outputPath = Join-Path $projectRoot $OutputDir
$cardPath = Join-Path $workPath "cards"
$scenePath = Join-Path $workPath "scenes"
$voicePath = Join-Path $workPath "voice"
$segmentPath = Join-Path $workPath "segments"

if (-not (Test-Path -LiteralPath $manifestFile)) { throw "Manifest not found: $manifestFile" }
New-Item -ItemType Directory -Force -Path $segmentPath, $outputPath | Out-Null

$manifest = Get-Content -Raw -LiteralPath $manifestFile -Encoding UTF8 | ConvertFrom-Json
$total = ($manifest.scenes | Measure-Object -Property durationSec -Sum).Sum
if ($total -lt 285 -or $total -gt 315) { throw "Manifest duration outside 285-315 seconds: $total" }

function Invoke-Ffmpeg([string[]]$Arguments) {
  & ffmpeg @Arguments
  if ($LASTEXITCODE -ne 0) { throw "FFmpeg failed with exit code $LASTEXITCODE" }
}

$timelineFile = Join-Path $workPath "timeline.txt"
$timelineLines = @()
$index = 0
foreach ($scene in $manifest.scenes) {
  $index++
  $duration = [double]$scene.durationSec
  $sceneId = [string]$scene.id
  $videoInput = if ($scene.kind -eq "card") {
    Join-Path $cardPath ([string]$scene.asset)
  } else {
    Join-Path $scenePath "$sceneId.webm"
  }
  $audioInput = Join-Path $voicePath "$($scene.voiceSegment).mp3"
  $subtitleInput = Join-Path $voicePath "$($scene.voiceSegment).srt"
  if (-not (Test-Path -LiteralPath $videoInput)) { throw "Video asset missing: $videoInput" }
  if (-not (Test-Path -LiteralPath $audioInput)) { throw "Voice asset missing: $audioInput" }
  if (-not (Test-Path -LiteralPath $subtitleInput)) { throw "Subtitle asset missing: $subtitleInput" }

  $segmentFile = Join-Path $segmentPath ("{0:D2}-{1}.mp4" -f $index, $sceneId)
  if (-not $subtitleInput.StartsWith($projectRoot, [StringComparison]::OrdinalIgnoreCase)) { throw "Subtitle must be inside project root: $subtitleInput" }
  $subtitleFilterPath = $subtitleInput.Substring($projectRoot.Length + 1).Replace('\', '/')
  $subtitleStyle = "FontName=Microsoft YaHei,FontSize=12,PrimaryColour=&H00FFFFFF,OutlineColour=&H00101820,BackColour=&H90170F04,BorderStyle=3,Outline=1,Shadow=0,Alignment=2,MarginL=72,MarginR=72,MarginV=82"
  $videoFilter = "scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2,setsar=1,fps=30,tpad=stop_mode=clone:stop_duration=2,subtitles='$subtitleFilterPath':force_style='$subtitleStyle'"
  $voiceDuration = [double]((& ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 $audioInput).Trim())
  $voiceDelayMs = [int][Math]::Round([Math]::Max(0, ($duration - $voiceDuration) / 2) * 1000)
  $audioFilter = "adelay=$voiceDelayMs|$voiceDelayMs,apad=pad_dur=$duration"
  $args = @("-y", "-loglevel", "error")
  if ($scene.kind -eq "card") {
    $args += @("-loop", "1", "-framerate", "30", "-i", $videoInput)
  } else {
    $args += @("-ss", ([string]::Format([Globalization.CultureInfo]::InvariantCulture, "{0:0.###}", $PageStartOffset)), "-i", $videoInput)
  }
  $args += @(
    "-i", $audioInput,
    "-map", "0:v:0", "-map", "1:a:0",
    "-t", ([string]::Format([Globalization.CultureInfo]::InvariantCulture, "{0:0.###}", $duration)),
    "-vf", $videoFilter,
    "-af", $audioFilter,
    "-r", "30", "-c:v", "libx264", "-preset", "medium", "-crf", "18", "-pix_fmt", "yuv420p",
    "-c:a", "aac", "-b:a", "160k", "-ar", "48000", "-ac", "2",
    "-movflags", "+faststart", $segmentFile
  )
  Invoke-Ffmpeg $args
  $timelineLines += "file '$($segmentFile.Replace('\', '/').Replace("'", "'\\''"))'"
}

$timelineLines | Set-Content -LiteralPath $timelineFile -Encoding ASCII
$joined = Join-Path $workPath "joined.mp4"
Invoke-Ffmpeg @("-y", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", $timelineFile, "-c", "copy", $joined)

$suffix = if ($WithBgm) { "with-bgm" } else { "no-bgm" }
$finalFile = Join-Path $outputPath "judge-demo-$suffix.mp4"
if (-not $WithBgm) {
  Invoke-Ffmpeg @("-y", "-loglevel", "error", "-i", $joined, "-af", "loudnorm=I=-16:TP=-1.5:LRA=11", "-c:v", "copy", "-c:a", "aac", "-b:a", "160k", "-ar", "48000", "-movflags", "+faststart", $finalFile)
} else {
  $bgmFile = Join-Path $workPath "ambient-bed.m4a"
  $fadeStart = [Math]::Max(0, $total - 5)
  $bgmFilter = "[0:a][1:a]amix=inputs=2:duration=longest,volume=0.035,afade=t=in:st=0:d=4,afade=t=out:st=${fadeStart}:d=5"
  Invoke-Ffmpeg @(
    "-y", "-loglevel", "error",
    "-f", "lavfi", "-i", "sine=frequency=196:sample_rate=48000:duration=$total",
    "-f", "lavfi", "-i", "sine=frequency=293.66:sample_rate=48000:duration=$total",
    "-filter_complex", $bgmFilter,
    "-c:a", "aac", "-b:a", "96k", $bgmFile
  )
  $mixFilter = "[0:a]loudnorm=I=-16:TP=-1.5:LRA=11[voice];[1:a]volume=0.12[bed];[voice][bed]amix=inputs=2:duration=first:dropout_transition=2,loudnorm=I=-16:TP=-1.5:LRA=11[mix]"
  Invoke-Ffmpeg @(
    "-y", "-loglevel", "error", "-i", $joined, "-i", $bgmFile,
    "-filter_complex", $mixFilter, "-map", "0:v:0", "-map", "[mix]", "-t", $total,
    "-c:v", "copy", "-c:a", "aac", "-b:a", "160k", "-ar", "48000", "-movflags", "+faststart", $finalFile
  )
}

Write-Output (ConvertTo-Json ([ordered]@{ output = $finalFile; durationSec = $total; withBgm = [bool]$WithBgm }) -Depth 4)
