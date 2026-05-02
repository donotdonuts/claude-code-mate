# ccmate uninstaller (Windows / PowerShell) — removes the binary, optionally
# purges the data dir, and prints the settings.json entries to remove by hand.
#
# Usage:  pwsh -File .\uninstall.ps1 [-Purge] [-Yes]
#
#   -Purge   also delete %USERPROFILE%\.claude\ccmate\ (tips, sessions.csv, state)
#   -Yes     skip the confirmation prompt for -Purge

[CmdletBinding()]
param(
    [switch]$Purge,
    [switch]$Yes
)

$ErrorActionPreference = 'Stop'

$ClaudeDir  = if ($env:CLAUDE_DIR) { $env:CLAUDE_DIR } else { Join-Path $env:USERPROFILE '.claude' }
$CcstatsDir = Join-Path $ClaudeDir 'ccmate'
$BinDir     = if ($env:BIN_DIR) { $env:BIN_DIR } else { Join-Path $env:USERPROFILE '.local\bin' }
$BinPath    = Join-Path $BinDir 'ccmate.exe'

Write-Output '==> Removing binary'
if (Test-Path $BinPath) {
    Remove-Item -Path $BinPath -Force
    Write-Output "    removed $BinPath"
} else {
    Write-Output "    not found at $BinPath (skipping)"
}

if ($Purge) {
    if (Test-Path $CcstatsDir) {
        $proceed = $true
        if (-not $Yes) {
            Write-Output ''
            Write-Output "==> -Purge will delete $CcstatsDir"
            Write-Output '    This includes sessions.csv (your full session history),'
            Write-Output '    any tips you may have edited, and per-session state.'
            $reply = Read-Host '    Continue? [y/N]'
            if ($reply -notmatch '^(y|Y|yes|YES)$') {
                Write-Output '    aborted - data dir kept'
                $proceed = $false
            }
        }
        if ($proceed) {
            Remove-Item -Path $CcstatsDir -Recurse -Force
            Write-Output "    removed $CcstatsDir"
        }
    } else {
        Write-Output "==> -Purge: $CcstatsDir not found (skipping)"
    }
} elseif (Test-Path $CcstatsDir) {
    Write-Output ''
    Write-Output "==> Keeping $CcstatsDir (tips, sessions.csv, state)"
    Write-Output '    Pass -Purge to delete it.'
}

Write-Output ''
Write-Output "==> Remove these entries from $ClaudeDir\settings.json (or settings.local.json):"
Write-Output '    - the entire "statusLine" block, if its "command" is "ccmate"'
Write-Output '    - the Stop hook entry whose "command" is "ccmate record"'
Write-Output '      (leave the rest of "hooks.Stop" intact)'
Write-Output ''
Write-Output '==> Done. Restart Claude Code to drop the statusline.'
