param(
  [string]$ConfigPath = "config.yaml",
  [string]$BinaryPath = "cli-proxy-api.exe",
  [int]$PollIntervalMs = 800
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

$logDir = Join-Path $repoRoot "logs"
New-Item -ItemType Directory -Force -Path $logDir | Out-Null
$serverOutLog = Join-Path $logDir "dev-hot-reload-server.out.log"
$serverErrLog = Join-Path $logDir "dev-hot-reload-server.err.log"

$script:serverProcess = $null

function Write-Log {
  param([string]$Message)
  $line = "[{0}] {1}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $Message
  Write-Host $line
}

function Get-WatchedSignature {
  $items = Get-ChildItem -Path $repoRoot -Recurse -File |
    Where-Object {
      $_.Extension -in @(".go", ".yaml", ".yml", ".html", ".js", ".css", ".mod", ".sum") -and
      $_.FullName -notmatch "\\\.git\\" -and
      $_.FullName -notmatch "\\logs\\" -and
      $_.FullName -notmatch "\\auths\\" -and
      $_.FullName -notmatch "cli-proxy-api\.exe$" -and
      $_.FullName -notmatch "test-output(\.exe)?$"
    } |
    Sort-Object FullName

  $builder = New-Object System.Text.StringBuilder
  foreach ($item in $items) {
    [void]$builder.Append($item.FullName)
    [void]$builder.Append("|")
    [void]$builder.Append($item.Length)
    [void]$builder.Append("|")
    [void]$builder.Append($item.LastWriteTimeUtc.Ticks)
    [void]$builder.AppendLine()
  }

  return $builder.ToString()
}

function Stop-RunningServer {
  if ($null -ne $script:serverProcess) {
    try {
      if (-not $script:serverProcess.HasExited) {
        Write-Log "Stopping server PID $($script:serverProcess.Id)"
        Stop-Process -Id $script:serverProcess.Id -Force -ErrorAction SilentlyContinue
      }
    } catch {
      Write-Log "Failed to stop running server cleanly: $($_.Exception.Message)"
    }
  }

  $binaryFullPath = Join-Path $repoRoot $BinaryPath
  $stale = Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq $binaryFullPath }
  foreach ($proc in $stale) {
    Write-Log "Stopping stale server PID $($proc.ProcessId)"
    Stop-Process -Id $proc.ProcessId -Force -ErrorAction SilentlyContinue
  }
}

function Build-Server {
  Write-Log "Building server"
  & go build -o $BinaryPath ./cmd/server
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
  }
}

function Start-Server {
  if (Test-Path $serverOutLog) {
    Clear-Content -Path $serverOutLog
  }
  if (Test-Path $serverErrLog) {
    Clear-Content -Path $serverErrLog
  }

  $binaryFullPath = Join-Path $repoRoot $BinaryPath
  $proc = Start-Process `
    -FilePath $binaryFullPath `
    -ArgumentList @("--config", $ConfigPath) `
    -WorkingDirectory $repoRoot `
    -WindowStyle Hidden `
    -RedirectStandardOutput $serverOutLog `
    -RedirectStandardError $serverErrLog `
    -PassThru

  Start-Sleep -Milliseconds 800
  if ($proc.HasExited) {
    $outText = if (Test-Path $serverOutLog) { Get-Content -Path $serverOutLog -Raw } else { "" }
    $errText = if (Test-Path $serverErrLog) { Get-Content -Path $serverErrLog -Raw } else { "" }
    throw "server exited immediately with code $($proc.ExitCode)`nSTDOUT:`n$outText`nSTDERR:`n$errText"
  }

  $script:serverProcess = $proc
  Write-Log "Server started on http://127.0.0.1:8317/codex-extract.html (PID $($proc.Id))"
}

function Restart-Server {
  Stop-RunningServer
  Build-Server
  Start-Server
}

Write-Log "Starting hot reload watcher in $repoRoot"
Stop-RunningServer
Build-Server
Start-Server

$lastSignature = Get-WatchedSignature
Write-Log "Watching source files for changes"

while ($true) {
  Start-Sleep -Milliseconds $PollIntervalMs

  $currentSignature = Get-WatchedSignature
  if ($currentSignature -ne $lastSignature) {
    Write-Log "Source change detected, rebuilding"
    try {
      Restart-Server
      $lastSignature = $currentSignature
      Write-Log "Reload complete"
    } catch {
      Write-Log "Reload failed: $($_.Exception.Message)"
      $lastSignature = $currentSignature
      if (Test-Path $serverErrLog) {
        Write-Log "Server stderr snapshot:"
        Get-Content -Path $serverErrLog | ForEach-Object { Write-Host $_ }
      }
    }
  }
}
