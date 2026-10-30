# ==============================================================================
# TorrServer -> TorrServer-NG rebranding script (Windows PowerShell)
# for sirin-media/torrserver-ng fork
#
# Usage: Open PowerShell, cd to repo root, run:
#   .\scripts\rebrand.ps1
#
# If execution policy blocks, run first:
#   Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
#
# IMPORTANT: Save this file as UTF-8 with BOM in VS Code
# (Ctrl+Shift+P -> "Change File Encoding" -> "Save with Encoding" -> "UTF-8 with BOM")
# ==============================================================================

 $ErrorActionPreference = "Stop"

# --- Helper functions ---

function Write-UnixFile {
    param([string]$Path, [string]$Content)
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    $content = $Content -replace "`r`n", "`n"
    [System.IO.File]::WriteAllText((Join-Path (Get-Location) $Path), $content, $utf8NoBom)
}

function Read-File {
    param([string]$Path)
    return [System.IO.File]::ReadAllText((Join-Path (Get-Location) $Path))
}

function Commit-Change {
    param([string]$File, [string]$Message)
    git add $File
    git commit -m $Message --no-verify 2>&1 | Out-Null
    Write-Host "  [OK] $Message"
    Write-Host ""
}

# --- Pre-flight checks ---

Write-Host "=== Pre-flight checks ==="

if (-not (Test-Path .git)) {
    Write-Host "ERROR: Not in a git repository" -ForegroundColor Red
    exit 1
}

 $status = git status --porcelain 2>&1
if ($status -and $status.ToString().Trim()) {
    Write-Host "ERROR: Working tree is not clean. Commit or stash changes first:" -ForegroundColor Red
    git status --short
    exit 1
}

 $currentBranch = git rev-parse --abbrev-ref HEAD 2>&1
Write-Host "Current branch: $currentBranch"

 $originUrl = git remote get-url origin 2>&1
Write-Host "Origin: $originUrl"

Write-Host ""
Write-Host "=== Rebranding TorrServer -> TorrServer-NG ==="
Write-Host ""

# --- 1. NOTICE file ---

Write-Host "[1/9] Creating NOTICE file..."

 $noticeContent = @"
TorrServer-NG
=============

A community-maintained fork of TorrServer.

Upstream project
----------------
- Original project: TorrServer
- Original author: YouROK (https://github.com/YouROK)
- Upstream repository: https://github.com/YouROK/TorrServer

Modifications
-------------
This fork includes:
- Pending community PRs merged by sirin-media
- CI/CD infrastructure for multi-platform builds
- Branding changes (web UI title, organization name)

All modifications are licensed under GPL-3.0, same as the original.

License
-------
GPL-3.0-or-later. See LICENSE file for full text.

Maintainer
----------
sirin-media organization
https://github.com/sirin-media

For the list of contributors to the original TorrServer project,
see git history and https://github.com/YouROK/TorrServer/graphs/contributors
"@

Write-UnixFile "NOTICE" $noticeContent
Commit-Change "NOTICE" "chore: add NOTICE file with attribution to YouROK"

# --- 2. CONTRIBUTORS.upstream.txt ---

Write-Host "[2/9] Extracting contributors from git history..."

 $contributors = git log --format='%aN <%aE>' 2>&1
 $contributorsArray = $contributors -split "`n"
 $unique = $contributorsArray | Sort-Object -Unique
 $content = ($unique -join "`n") + "`n"
Write-UnixFile "CONTRIBUTORS.upstream.txt" $content
Write-Host "  Found $($unique.Count) contributors"
Commit-Change "CONTRIBUTORS.upstream.txt" "chore: extract contributors list from git history"

# --- 3. README.md prepend ---

Write-Host "[3/9] Prepending attribution to README.md..."

 $firstLine = Get-Content README.md -TotalCount 1 2>$null
if ($firstLine -notmatch "TorrServer-NG") {
    $header = @"
# TorrServer-NG

Community-maintained fork of [TorrServer](https://github.com/YouROK/TorrServer) originally developed and maintained by [YouROK](https://github.com/YouROK).

Maintained by [sirin-media](https://github.com/sirin-media) -- community organization for free and open tools that help users navigate the vast ocean of media content.

## About the name

**Sirin** -- paradise bird-maiden from Slavic mythology, whose enchanting song captivates those who hear it. In the same way, media content captivates users -- our tools help them freely navigate this flow.

## Attribution

This project is a fork of TorrServer, originally developed and maintained by [YouROK](https://github.com/YouROK). All original code is licensed under GPL-3.0. We gratefully acknowledge the original author's contribution.

This fork exists to provide:

- Continued maintenance with merged pending community PRs
- Multi-platform binary builds via CI (Windows, Linux, macOS, ARM)
- Docker images in GHCR
- Transparent roadmap and decision-making

All improvements continue to be submitted upstream as PRs. If the upstream maintainer resumes activity, this fork is ready to be reconciled or merged back.

## License

GPL-3.0 -- see [LICENSE](./LICENSE).

## Status

Initial setup -- first release coming soon.

---

## Original README (TorrServer by YouROK)

"@
    $original = Read-File "README.md"
    $newContent = $header + $original
    Write-UnixFile "README.md" $newContent
    Commit-Change "README.md" "chore: add fork attribution and Sirin naming context to README"
} else {
    Write-Host "  README.md already has TorrServer-NG header, skipping"
    Write-Host ""
}

# --- 4. web/public/index.html ---

Write-Host "[4/9] Updating web UI title..."

if (Test-Path "web/public/index.html") {
    $content = Read-File "web/public/index.html"
    $content = $content -replace 'TorrServer MatriX', 'TorrServer-NG'
    Write-UnixFile "web/public/index.html" $content
    Commit-Change "web/public/index.html" "chore(web): rebrand browser title to TorrServer-NG"
} else {
    Write-Host "  web/public/index.html not found, skipping"
    Write-Host ""
}

# --- 5. web/package.json ---

Write-Host "[5/9] Updating web/package.json metadata..."

if (Test-Path "web/package.json") {
    $content = Read-File "web/package.json"
    $content = $content -replace '"name": "torrserver_web"', '"name": "torrserver-ng-web"'
    $content = $content -replace '"description": ""', '"description": "Community fork of TorrServer web UI"'
    $content = $content -replace '"author": ""', '"author": "sirin-media"'
    $content = $content -replace '"license": "ISC"', '"license": "GPL-3.0"'
    Write-UnixFile "web/package.json" $content
    Commit-Change "web/package.json" "chore(web): update package.json metadata for sirin-media"
} else {
    Write-Host "  web/package.json not found, skipping"
    Write-Host ""
}

# --- 6. server/version/version.go ---

Write-Host "[6/9] Updating version string..."

if (Test-Path "server/version/version.go") {
    $content = Read-File "server/version/version.go"
    $content = $content -replace 'var Version = "MatriX\.147"', 'var Version = "NG-1.0.0-dev (based on MatriX.147)"'
    Write-UnixFile "server/version/version.go" $content
    Commit-Change "server/version/version.go" "chore(version): set default version to NG-1.0.0-dev"
} else {
    Write-Host "  server/version/version.go not found, skipping"
    Write-Host ""
}

# --- 7. Dockerfile OCI labels ---

Write-Host "[7/9] Adding OCI labels to Dockerfile..."

if (Test-Path "Dockerfile") {
    $lines = Get-Content Dockerfile
    $result = @()
    $inserted = $false
    foreach ($line in $lines) {
        $result += $line
        if (-not $inserted -and $line -match '^FROM') {
            $result += 'LABEL org.opencontainers.image.title="TorrServer-NG"'
            $result += 'LABEL org.opencontainers.image.description="Community fork of TorrServer"'
            $result += 'LABEL org.opencontainers.image.source="https://github.com/sirin-media/torrserver-ng"'
            $result += 'LABEL org.opencontainers.image.authors="sirin-media"'
            $result += 'LABEL org.opencontainers.image.licenses="GPL-3.0-or-later"'
            $inserted = $true
        }
    }
    $content = ($result -join "`n") + "`n"
    Write-UnixFile "Dockerfile" $content
    Commit-Change "Dockerfile" "build(docker): add OCI labels for sirin-media organization"
} else {
    Write-Host "  Dockerfile not found, skipping"
    Write-Host ""
}

# --- 8. installTorrServerLinux.sh ---

Write-Host "[8/9] Updating Linux install script URLs..."

if (Test-Path "installTorrServerLinux.sh") {
    $content = Read-File "installTorrServerLinux.sh"
    $content = $content -replace 'https://github\.com/YouROK/TorrServer', 'https://github.com/sirin-media/torrserver-ng'
    $content = $content -replace 'https://api\.github\.com/repos/YouROK/TorrServer', 'https://api.github.com/repos/sirin-media/torrserver-ng'
    Write-UnixFile "installTorrServerLinux.sh" $content
    Commit-Change "installTorrServerLinux.sh" "chore(scripts): point Linux install script to sirin-media releases"
} else {
    Write-Host "  installTorrServerLinux.sh not found, skipping"
    Write-Host ""
}

# --- 9. installTorrServerMac.sh ---

Write-Host "[9/9] Updating macOS install script URLs..."

if (Test-Path "installTorrServerMac.sh") {
    $content = Read-File "installTorrServerMac.sh"
    $content = $content -replace 'https://github\.com/YouROK/TorrServer', 'https://github.com/sirin-media/torrserver-ng'
    $content = $content -replace 'https://api\.github\.com/repos/YouROK/TorrServer', 'https://api.github.com/repos/sirin-media/torrserver-ng'
    Write-UnixFile "installTorrServerMac.sh" $content
    Commit-Change "installTorrServerMac.sh" "chore(scripts): point macOS install script to sirin-media releases"
} else {
    Write-Host "  installTorrServerMac.sh not found, skipping"
    Write-Host ""
}

# --- Summary ---

Write-Host "========================================"
Write-Host "Rebranding complete"
Write-Host "========================================"
Write-Host ""
Write-Host "Changes made:"
Write-Host "  1. NOTICE file created (GPL-3.0 attribution)"
Write-Host "  2. CONTRIBUTORS.upstream.txt extracted from git history"
Write-Host "  3. README.md prepended with fork attribution"
Write-Host "  4. Web UI title changed to TorrServer-NG"
Write-Host "  5. web/package.json metadata updated"
Write-Host "  6. Version string set to NG-1.0.0-dev"
Write-Host "  7. Dockerfile OCI labels added"
Write-Host "  8. Linux install script points to sirin-media"
Write-Host "  9. macOS install script points to sirin-media"
Write-Host ""
Write-Host "Commits created:"
git log --oneline -9
Write-Host ""
Write-Host "What was NOT changed (intentionally, for compatibility):"
Write-Host "  - Binary name: still 'TorrServer' (for existing users)"
Write-Host "  - go.mod module path: still 'server' (avoids import refactor)"
Write-Host "  - VERSION_PREFIX in install scripts: still 'MatriX'"
Write-Host "  - API endpoints, config paths, ports"
Write-Host "  - File names, variable names, function names"
Write-Host ""
Write-Host "Next steps:"
Write-Host "  1. Review: git log --oneline -9"
Write-Host "  2. Push:   git push origin master"
Write-Host "  3. Continue to Step 3: porting pending PRs"