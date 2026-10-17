# install-check.ps1 - the Windows installer from nothing, hands-off, and a check of what
# it left. The Windows twin of install-check.sh.
#
#   powershell -ExecutionPolicy Bypass -File scripts\install-check.ps1
#
# Builds the Windows archive from this checkout, runs install.ps1 against it answering
# yes to everything (AB_YES=1), in Spanish, with the plugin from this checkout. Then it
# checks what a newcomer would get: Git, Claude Code, the program, the PATH, the two
# spaces, the default space, the plugin and its MCP server, the first-session interview,
# doctor, and that a second run only updates.
#
# It installs Git, Claude Code and the program for the user who runs it, so run it on a
# throwaway machine: GitHub Actions does (.github/workflows/windows.yml), and so does the
# Windows VM in test/vm/windows-11 from its "clean" snapshot. The desktop app is skipped
# unless AB_CHECK_DESKTOP=1.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$Root = Split-Path -Parent $PSScriptRoot
$Tmp = Join-Path ([IO.Path]::GetTempPath()) ("ab-check-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $Tmp | Out-Null
$script:fails = 0
function Ok([string]$what) { Write-Host "  OK    $what" }
function Bad([string]$what) { Write-Host "  FAIL  $what"; $script:fails++ }
function Check([string]$what, [scriptblock]$test) {
  $r = $false
  try { $r = [bool](& $test) } catch {}
  if ($r) { Ok $what } else { Bad $what }
}

try {
  Write-Host "1. build the Windows archive from this checkout"
  $version = '0.0.0-check'
  $arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
  $name = "augmented-brain_${version}_windows_$arch"
  $rel = Join-Path $Tmp 'release'
  New-Item -ItemType Directory -Path (Join-Path $rel $name) -Force | Out-Null
  Push-Location $Root
  try {
    $env:CGO_ENABLED = '0'
    go build -trimpath -ldflags "-s -w -X github.com/icalvete/augmented-brain/internal/version.Version=$version" -o (Join-Path $rel "$name\augmented-brain.exe") ./cmd/augmented-brain
    if ($LASTEXITCODE -ne 0) { throw "go build failed" }
  } finally { Pop-Location }
  Copy-Item (Join-Path $Root 'LICENSE'), (Join-Path $Root 'README.md') (Join-Path $rel $name)
  Compress-Archive -Path (Join-Path $rel $name) -DestinationPath (Join-Path $rel "$name.zip")
  $hash = (Get-FileHash -Algorithm SHA256 (Join-Path $rel "$name.zip")).Hash.ToLower()
  Set-Content -Path (Join-Path $rel 'checksums.txt') -Value "$hash  $name.zip" -Encoding Ascii

  # A newcomer's PATH does not have ~\.local\bin yet.
  $dir = Join-Path $env:USERPROFILE '.local\bin'
  $env:Path = ($env:Path -split ';' | Where-Object { $_ -and $_.TrimEnd('\') -ine $dir }) -join ';'

  Write-Host "2. run the installer, hands-off"
  $env:AB_YES = '1'; $env:AB_LANGUAGE = 'es'
  $env:AB_NO_DESKTOP = if ($env:AB_CHECK_DESKTOP -eq '1') { '' } else { '1' }
  $env:AB_RELEASE_URL = ([Uri]$rel).AbsoluteUri
  $env:AB_MARKETPLACE = $Root
  $log = & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Root 'install.ps1') 2>&1 | Out-String
  $log -split "`n" | ForEach-Object { "    | $_" }
  if ($log -match 'augmented brain: ') { Bad "install.ps1 stopped with an error" }

  Write-Host "3. what a newcomer gets"
  $env:Path = [Environment]::GetEnvironmentVariable('Path', 'User') + ';' + [Environment]::GetEnvironmentVariable('Path', 'Machine')
  $ab = Join-Path $dir 'augmented-brain.exe'
  Check "program in ~\.local\bin" { Test-Path $ab }
  Check "~\.local\bin in the user's PATH" { [Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_.TrimEnd('\') -ieq $dir } }
  Check "Git installed" { Get-Command git -ErrorAction SilentlyContinue }
  Check "Claude Code installed" { Test-Path (Join-Path $dir 'claude.exe') }
  $cfg = Join-Path $env:APPDATA 'augmented-brain\config.toml'
  Check "configuration in Spanish" { Select-String -Path $cfg -Pattern '^language = "es"' -Quiet }
  Check "default space is personal" { Select-String -Path $cfg -Pattern '^default_space = "personal"' -Quiet }
  Check "space trabajo, notes inside" { Test-Path "$env:USERPROFILE\augmented-brain\trabajo\notas\MEMORY.md" }
  Check "space personal, notes inside" { Test-Path "$env:USERPROFILE\augmented-brain\personal\notas\MEMORY.md" }
  Check "plugin installed" { (claude plugin list 2>&1 | Out-String) -match 'brain@augmented-brain' }
  Push-Location "$env:USERPROFILE\augmented-brain\trabajo"
  try { Check "its MCP server connects" { (claude mcp list 2>&1 | Out-String) -match 'plugin:brain:augmented-brain: .*Connected' } } finally { Pop-Location }
  $scratch = Join-Path $Tmp 'scratch'
  New-Item -ItemType Directory -Path $scratch | Out-Null
  Push-Location $scratch
  try { $start = (@{ cwd = $scratch } | ConvertTo-Json -Compress) | & $ab hook session-start | Out-String } finally { Pop-Location }
  Check "outside the folders, the personal space loads" { $start -match 'the default space, personal, is loaded' }
  Check "and the first-session interview starts" { $start -match 'First session: get to know the user' }
  Push-Location "$env:USERPROFILE\augmented-brain\trabajo"
  try { & $ab doctor | Out-Null; $code = $LASTEXITCODE } finally { Pop-Location }
  Check "doctor finds nothing missing" { $code -eq 0 }
  $again = & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $Root 'install.ps1') 2>&1 | Out-String
  Check "a second run only updates" { $again -match 'Ya estaba configurado' }
  if ($env:AB_CHECK_DESKTOP -eq '1') {
    Check "desktop app installed" { Get-StartApps | Where-Object { $_.Name -eq 'Claude' } }
  }
} finally {
  Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}

Write-Host ''
if ($script:fails -gt 0) { Write-Host "$($script:fails) FAIL"; exit 1 }
Write-Host 'all OK'
