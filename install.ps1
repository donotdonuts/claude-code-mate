# ccmate installer (Windows / PowerShell) — builds the binary, copies tips
# into place, and prints the settings.json snippet to merge.
#
# Usage:  pwsh -File .\install.ps1     (or)     powershell -File .\install.ps1

$ErrorActionPreference = 'Stop'

$Here       = Split-Path -Parent $MyInvocation.MyCommand.Path
$ClaudeDir  = if ($env:CLAUDE_DIR) { $env:CLAUDE_DIR } else { Join-Path $env:USERPROFILE '.claude' }
$CcstatsDir = Join-Path $ClaudeDir 'ccmate'
$BinDir     = if ($env:BIN_DIR) { $env:BIN_DIR } else { Join-Path $env:USERPROFILE '.local\bin' }

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "Go toolchain not found on PATH. Install Go from https://go.dev/dl/ and re-run."
}

Write-Output '==> Building ccmate...'
Push-Location $Here
try {
    & go build -o ccmate.exe ./cmd/ccmate
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
} finally {
    Pop-Location
}

if (-not (Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir -Force | Out-Null }
Move-Item -Path (Join-Path $Here 'ccmate.exe') -Destination (Join-Path $BinDir 'ccmate.exe') -Force
Write-Output "    installed to $BinDir\ccmate.exe"

$TipsDir = Join-Path $CcstatsDir 'tips'
Write-Output "==> Installing default tips into $TipsDir"
if (-not (Test-Path $TipsDir)) { New-Item -ItemType Directory -Path $TipsDir -Force | Out-Null }
foreach ($tip in Get-ChildItem -Path (Join-Path $Here 'tips') -Filter '*.md') {
    $dest = Join-Path $TipsDir $tip.Name
    if (-not (Test-Path $dest)) { Copy-Item $tip.FullName -Destination $dest }
}

Write-Output ''
Write-Output "==> Merge this into $ClaudeDir\settings.json (or settings.local.json):"
Get-Content (Join-Path $Here 'settings.example.json') | Write-Output
Write-Output ''
Write-Output "==> Make sure $BinDir is on your PATH."
$pathParts = ($env:Path -split ';')
if ($pathParts -notcontains $BinDir) {
    Write-Output "    (currently it is NOT on PATH for this session — add it via System Properties > Environment Variables, or:"
    Write-Output "     [Environment]::SetEnvironmentVariable('Path', `"`$env:Path;$BinDir`", 'User')  )"
}
Write-Output '    Then start a new Claude Code session — the statusline should appear.'
