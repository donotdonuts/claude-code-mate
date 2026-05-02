# ccmate installer (Windows / PowerShell) — builds the binary, copies tips
# into place, and merges the statusLine + Stop-hook into ~/.claude/settings.json.
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
    Copy-Item $tip.FullName -Destination $dest -Force
}

# ---------------------------------------------------------------------------
# Merge statusLine + Stop hook into settings.json
# ---------------------------------------------------------------------------
$SettingsPath = Join-Path $ClaudeDir 'settings.json'
Write-Output ''
Write-Output "==> Updating $SettingsPath"

if ($PSVersionTable.PSVersion.Major -lt 6) {
    Write-Output "    ! Detected Windows PowerShell $($PSVersionTable.PSVersion). Auto-merge requires PowerShell 7+."
    Write-Output "    Install pwsh (winget install Microsoft.PowerShell) and re-run, or merge the snippet below manually:"
    Write-Output ''
    Get-Content (Join-Path $Here 'settings.example.json') | Write-Output
} else {
    if (-not (Test-Path $ClaudeDir)) { New-Item -ItemType Directory -Path $ClaudeDir -Force | Out-Null }

    if (Test-Path $SettingsPath) {
        $raw = Get-Content $SettingsPath -Raw
        if ([string]::IsNullOrWhiteSpace($raw)) {
            $settings = [ordered]@{}
        } else {
            $backup = "$SettingsPath.bak"
            Copy-Item $SettingsPath $backup -Force
            Write-Output "    backed up existing file to $backup"
            $settings = $raw | ConvertFrom-Json -AsHashtable
        }
    } else {
        $settings = [ordered]@{}
    }

    # statusLine: only set if missing or already pointed at ccmate.
    # Don't clobber a user-customized statusLine.
    if (-not $settings.Contains('statusLine')) {
        $settings['statusLine'] = [ordered]@{
            type    = 'command'
            command = 'ccmate'
            padding = 0
        }
        Write-Output "    + added statusLine -> ccmate"
    } else {
        $existingCmd = $settings['statusLine']['command']
        if ($existingCmd -eq 'ccmate') {
            Write-Output "    = statusLine already set to ccmate"
        } else {
            Write-Output "    ! statusLine already configured (command='$existingCmd'); leaving as-is."
            Write-Output "      Edit $SettingsPath manually if you want to switch to ccmate."
        }
    }

    # hooks.Stop: append `ccmate record` only if it's not already present
    if (-not $settings.Contains('hooks')) { $settings['hooks'] = [ordered]@{} }
    if (-not $settings['hooks'].Contains('Stop')) { $settings['hooks']['Stop'] = @() }

    $alreadyHas = $false
    foreach ($group in @($settings['hooks']['Stop'])) {
        if ($null -ne $group -and $group.Contains('hooks')) {
            foreach ($h in @($group['hooks'])) {
                if ($null -ne $h -and $h['command'] -eq 'ccmate record') { $alreadyHas = $true }
            }
        }
    }
    if (-not $alreadyHas) {
        $newGroup = [ordered]@{
            matcher = ''
            hooks   = @(
                [ordered]@{
                    type    = 'command'
                    command = 'ccmate record'
                }
            )
        }
        $settings['hooks']['Stop'] = @($settings['hooks']['Stop']) + @($newGroup)
        Write-Output "    + added Stop hook -> ccmate record"
    } else {
        Write-Output "    = Stop hook 'ccmate record' already present"
    }

    $json = $settings | ConvertTo-Json -Depth 20
    Set-Content -Path $SettingsPath -Value $json -Encoding UTF8
}

Write-Output ''
Write-Output "==> Make sure $BinDir is on your PATH."
$pathParts = ($env:Path -split ';')
if ($pathParts -notcontains $BinDir) {
    Write-Output "    (currently it is NOT on PATH for this session — add it via System Properties > Environment Variables, or:"
    Write-Output "     [Environment]::SetEnvironmentVariable('Path', `"`$env:Path;$BinDir`", 'User')  )"
}
Write-Output '    Then start a new Claude Code session — the statusline should appear.'
