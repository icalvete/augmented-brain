# install.ps1 - installs augmented brain on Windows and sets it up. No administrator needed.
#
#   irm https://github.com/icalvete/augmented-brain/releases/latest/download/install.ps1 | iex
#
# The same steps as install.sh on Linux and macOS:
# 1. Checks what it needs first (Git for Windows and Claude Code) and offers to install
#    what is missing, with winget and Claude Code's own installer; without a terminal to
#    ask, it says the exact command and stops. It also offers the Claude desktop app.
# 2. Downloads the archive for this system from the latest release, checks its SHA-256
#    against the release's checksums.txt, and puts the program in
#    %USERPROFILE%\.local\bin (or $env:AB_INSTALL_DIR).
# 3. Adds that folder to the user's PATH, if it is not there.
# 4. Runs `augmented-brain init` the first time, which asks nothing, and offers to open
#    Claude right away: the desktop app if it is there, otherwise in this terminal.
#
# Settings: AB_YES=1 answers yes to every question, AB_LANGUAGE=en|es sets the language,
# AB_NO_DESKTOP=1 does not offer the desktop app. For testing: AB_VERSION, AB_RELEASE_URL,
# AB_MARKETPLACE and AB_NO_INIT=1, as in install.sh.
#
# It runs inside `irm | iex`, in the person's own PowerShell window: it never calls exit,
# which would close that window. The Spanish messages are written with \u escapes, so
# they survive however the download is decoded. It works on Windows PowerShell 5.1, the
# one that comes with Windows.

& {
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
try { [Console]::OutputEncoding = [Text.Encoding]::UTF8 } catch {}

$Repo = 'icalvete/augmented-brain'
$Version = if ($env:AB_VERSION) { $env:AB_VERSION } else { 'latest' }
$Dir = if ($env:AB_INSTALL_DIR) { $env:AB_INSTALL_DIR } else { Join-Path $env:USERPROFILE '.local\bin' }
$Pricing = 'https://claude.com/pricing'

# say <english> <spanish>: our own messages, in colour, so they stand out from what
# other installers print.
$lang = if ((Get-UICulture).Name -like 'es*') { 'es' } else { 'en' }
# Un turns the \uXXXX escapes into their letters, and leaves every other backslash
# alone: Windows paths are full of them.
function Un([string]$t) {
  [regex]::Replace($t, '\\u([0-9a-fA-F]{4})', { param($m) [string][char][Convert]::ToInt32($m.Groups[1].Value, 16) })
}
function Text([string]$en, [string]$es) {
  if ($lang -eq 'es') { Un $es } else { Un $en }
}
function Say([string]$en, [string]$es) { Write-Host (Text $en $es) -ForegroundColor Green }
function Fail([string]$msg) { throw "augmented brain: $msg" }

# A person at a terminal can be asked; a script, CI or a remote session cannot.
# CI runners look interactive to Windows, so CI and a redirected input count as nobody.
$interactive = [Environment]::UserInteractive -and $Host.Name -eq 'ConsoleHost' -and
  -not $env:CI -and -not [Console]::IsInputRedirected -and
  -not ([Environment]::GetCommandLineArgs() | Where-Object { $_ -like '-NonI*' })
# Ask <question>: true when the answer is yes (the default). With nobody to ask, it is
# a no, unless AB_YES=1.
function Ask([string]$q) {
  if ($env:AB_YES -eq '1') { return $true }
  if (-not $interactive) { return $false }
  $a = Read-Host "$q $(Text '[Y/n]' '[S/n]')"
  return -not ($a -match '^\s*[nN]')
}
# Quiet <scriptblock>: runs an installation without its noise; its output is shown only
# when it fails.
function Quiet([scriptblock]$b) {
  # Windows PowerShell 5.1 turns a program's stderr into errors, which 'Stop' would
  # treat as a failure even when the program succeeds: only the exit code counts.
  $ErrorActionPreference = 'Continue'
  $out = & $b 2>&1 | Out-String
  if ($LASTEXITCODE -ne 0 -and $null -ne $LASTEXITCODE) { Write-Host $out; return $false }
  return $true
}
function Tilde([string]$p) {
  if ($p.StartsWith($env:USERPROFILE, [StringComparison]::OrdinalIgnoreCase)) { return '~' + $p.Substring($env:USERPROFILE.Length) }
  return $p
}
# AddToPath <folder>: for this window now, and for the user from the next one.
function AddToPath([string]$d) {
  $script:newTerminal = $false
  $user = [Environment]::GetEnvironmentVariable('Path', 'User')
  $parts = @($user -split ';' | Where-Object { $_ })
  if (-not ($parts | Where-Object { $_.TrimEnd('\') -ieq $d.TrimEnd('\') })) {
    [Environment]::SetEnvironmentVariable('Path', (@($d) + $parts) -join ';', 'User')
    Say "+ added $(Tilde $d) to your PATH" "+ a\u00f1adido $(Tilde $d) a tu PATH"
    $script:newTerminal = $true
  }
  if (-not (($env:Path -split ';') | Where-Object { $_.TrimEnd('\') -ieq $d.TrimEnd('\') })) { $env:Path = "$d;$env:Path" }
}

try {
  # The language, asked first: the system's is the default. AB_LANGUAGE answers it.
  if ($env:AB_LANGUAGE) { $lang = $env:AB_LANGUAGE }
  elseif ($interactive -and $env:AB_YES -ne '1') {
    $a = Read-Host "Language / Idioma: English or espa$([char]0xF1)ol? [en/es] ($lang)"
    if ($a -match '^\s*(es|sp|ca)') { $lang = 'es' } elseif ($a -match '^\s*(en|in)') { $lang = 'en' }
  }

  $cpu = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
  $arch = switch ($cpu) { 'AMD64' { 'amd64' } 'ARM64' { 'arm64' } default { Fail (Text "this processor ($cpu) is not supported yet." "este procesador ($cpu) a\u00fan no est\u00e1 soportado.") } }
  $winget = [bool](Get-Command winget -ErrorAction SilentlyContinue)
  $gitCmd = Join-Path $env:ProgramFiles 'Git\cmd'
  $claudeDir = Join-Path $env:USERPROFILE '.local\bin'
  function HasGit { [bool](Get-Command git -ErrorAction SilentlyContinue) -or (Test-Path (Join-Path $gitCmd 'git.exe')) }
  function HasClaude { [bool](Get-Command claude -ErrorAction SilentlyContinue) -or (Test-Path (Join-Path $claudeDir 'claude.exe')) }
  function ClaudeExe {
    $c = Get-Command claude -ErrorAction SilentlyContinue
    if ($c) { $c.Source } else { Join-Path $claudeDir 'claude.exe' }
  }
  # ClaudeUpdate: an old Claude Code can lack what the plugin uses (the end-of-session
  # hook that starts the backup, for one). When it is behind Anthropic's stable release,
  # offer to update it with its own updater. It never stops the installation.
  function ClaudeUpdate {
    $url = if ($env:AB_CLAUDE_STABLE_URL) { $env:AB_CLAUDE_STABLE_URL } else { 'https://downloads.claude.ai/claude-code-releases/stable' }
    try {
      $have = (((& (ClaudeExe) --version) 2>$null | Out-String).Trim() -split '\s+')[0]
      $want = ([string](Invoke-RestMethod -Uri $url -TimeoutSec 10 -ErrorAction Stop)).Trim()
      if ([version]($have -replace '[^0-9.].*$', '') -ge [version]($want -replace '[^0-9.].*$', '')) { return }
    } catch { return }  # offline, or not a version
    Say "Your Claude Code is version $have and the current one is ${want}: an old one can lack what augmented brain uses." `
        "Tu Claude Code est\u00e1 en la versi\u00f3n $have y la actual es la ${want}: una antigua puede no tener lo que usa augmented brain."
    if (Ask (Text 'Update it now?' '\u00bfLo actualizo ahora?')) {
      Say "Updating Claude Code..." "Actualizando Claude Code..."
      if (Quiet { & (ClaudeExe) update }) {
        Say "+ Claude Code updated" "+ Claude Code actualizado"
      } else {
        Say "Claude Code was not updated. Do it later with: claude update" "Claude Code no se ha actualizado. Hazlo m\u00e1s tarde con: claude update"
      }
    } else {
      Say "Update it later with: claude update" "Actual\u00edzalo m\u00e1s tarde con: claude update"
    }
  }
  function DesktopApp { Get-StartApps -ErrorAction SilentlyContinue | Where-Object { $_.Name -eq 'Claude' } | Select-Object -First 1 }

  # --- 1. What it needs ---------------------------------------------------------------
  if ($env:AB_NO_INIT -ne '1') {
    if (-not (HasGit)) {
      Say "augmented brain needs Git for Windows: for the history and backup of your notes, and for Git Bash, where Claude runs commands safely." `
          "augmented brain necesita Git for Windows: para el historial y la copia de tus notas, y por Git Bash, donde Claude ejecuta los comandos con seguridad."
      $cmd = 'winget install --id Git.Git -e --source winget'
      if (-not $winget) {
        Say "Install it from https://git-scm.com/download/win, then run this installer again." "Inst\u00e1lalo desde https://git-scm.com/download/win y vuelve a lanzar este instalador."
        return
      }
      if (-not (Ask (Text "Install it now with: $cmd ? Windows may ask for permission." "\u00bfLo instalo ahora con: $cmd ? Windows puede pedirte permiso."))) {
        Say "Install it with this, then run this installer again:" "Inst\u00e1lalo con esto y vuelve a lanzar este instalador:"
        Write-Host "  $cmd"; return
      }
      Say "Installing Git for Windows..." "Instalando Git for Windows..."
      if (-not (Quiet { winget install --id Git.Git -e --source winget --silent --accept-source-agreements --accept-package-agreements --disable-interactivity })) { Fail (Text 'Git was not installed.' 'Git no se ha instalado.') }
      $env:Path = "$gitCmd;$env:Path"
      if (-not (HasGit)) { Fail (Text 'Git was installed but cannot be found; open a new terminal and run this installer again.' 'Git se instal\u00f3, pero no lo encuentro; abre una terminal nueva y vuelve a lanzar este instalador.') }
      Say "+ Git for Windows installed" "+ Git for Windows instalado"
    }
    if (-not (HasClaude)) {
      Say "augmented brain works with Claude Code, Anthropic's assistant, and it is not installed." "augmented brain trabaja con Claude Code, el asistente de Anthropic, y no est\u00e1 instalado."
      # Claude Code is not in Claude's free plan: said before installing it.
      Say "Claude Code needs a paid Claude plan (Pro, Max, Team or Enterprise) or an Anthropic API key: the free plan does not include it. Plans and prices: $Pricing" `
          "Claude Code necesita un plan de pago de Claude (Pro, Max, Team o Enterprise) o una clave de la API de Anthropic: el plan gratuito no lo incluye. Planes y precios: $Pricing"
      if ($interactive -and $env:AB_YES -ne '1' -and -not (Ask (Text 'Do you have one already?' '\u00bfYa tienes uno?'))) {
        Say "No problem: the installation goes on. Get a plan at $Pricing whenever you like; Claude asks you to sign in the first time you open it." `
            "No pasa nada: la instalaci\u00f3n sigue. Consigue un plan en $Pricing cuando quieras; Claude te pedir\u00e1 entrar la primera vez que lo abras."
      }
      $cmd = 'irm https://claude.ai/install.ps1 | iex'
      if (-not (Ask (Text 'Install it now with its official installer?' '\u00bfLo instalo ahora con su instalador oficial?'))) {
        Say "Install it with this, then run this installer again:" "Inst\u00e1lalo con esto y vuelve a lanzar este instalador:"
        Write-Host "  $cmd"; return
      }
      Say "Installing Claude Code..." "Instalando Claude Code..."
      if (-not (Quiet { powershell -NoProfile -ExecutionPolicy Bypass -Command $cmd })) { Fail (Text 'Claude Code was not installed.' 'Claude Code no se ha instalado.') }
      if (-not (HasClaude)) { Fail (Text 'Claude Code was installed but cannot be found; open a new terminal and run this installer again.' 'Claude Code se instal\u00f3, pero no lo encuentro; abre una terminal nueva y vuelve a lanzar este instalador.') }
      Say "+ Claude Code installed" "+ Claude Code instalado"
    } else {
      ClaudeUpdate
    }
    if ($env:AB_NO_DESKTOP -ne '1' -and $winget -and -not (DesktopApp) -and ($interactive -or $env:AB_YES -eq '1')) {
      Write-Host ''
      Say "Claude also has a desktop app: the same Claude Code, with windows instead of a terminal. It is the easiest way to work." `
          "Claude tiene tambi\u00e9n una app de escritorio: el mismo Claude Code, con ventanas en vez de terminal. Es la forma m\u00e1s c\u00f3moda de trabajar."
      if (Ask (Text 'Install it too?' '\u00bfLa instalo tambi\u00e9n?')) {
        Say "Installing the desktop app (it can take a couple of minutes)..." "Instalando la app de escritorio (puede tardar un par de minutos)..."
        if (Quiet { winget install --id Anthropic.Claude -e --source winget --silent --accept-source-agreements --accept-package-agreements --disable-interactivity }) {
          Say "+ Claude desktop app installed" "+ app de escritorio de Claude instalada"
        } else {
          Say "The desktop app was not installed; you can use Claude in the terminal." "La app de escritorio no se ha instalado; puedes usar Claude en la terminal."
        }
      }
    }
  }

  # --- 2. Download, check and install -------------------------------------------------
  $base = if ($env:AB_RELEASE_URL) { $env:AB_RELEASE_URL.TrimEnd('/') }
          elseif ($Version -eq 'latest') { "https://github.com/$Repo/releases/latest/download" }
          else { "https://github.com/$Repo/releases/download/v$($Version.TrimStart('v'))" }
  $tmp = Join-Path ([IO.Path]::GetTempPath()) ("ab-" + [Guid]::NewGuid().ToString('N'))
  New-Item -ItemType Directory -Path $tmp | Out-Null
  try {
    # file:// works too, for tests: Invoke-WebRequest does not take it, WebClient does.
    $web = New-Object Net.WebClient
    try { $web.DownloadFile("$base/checksums.txt", "$tmp\checksums.txt") } catch { Fail "cannot download $base/checksums.txt" }
    $line = Get-Content "$tmp\checksums.txt" | Where-Object { $_ -match "_windows_$arch\.zip$" } | Select-Object -First 1
    if (-not $line) { Fail "the release has no archive for windows/$arch." }
    $want, $archive = $line -split '\s+', 2
    try { $web.DownloadFile("$base/$archive", "$tmp\$archive") } catch { Fail "cannot download $archive" }
    $got = (Get-FileHash -Algorithm SHA256 "$tmp\$archive").Hash
    if ($got -ine $want) { Fail (Text 'the download does not match its checksum: not installed.' 'la descarga no coincide con su suma de comprobaci\u00f3n: no se ha instalado.') }
    Expand-Archive -Path "$tmp\$archive" -DestinationPath $tmp -Force
    $src = Join-Path $tmp ($archive -replace '\.zip$', '') | Join-Path -ChildPath 'augmented-brain.exe'
    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    $exe = Join-Path $Dir 'augmented-brain.exe'
    # A running program cannot be overwritten on Windows, but it can be renamed: the
    # update command runs this installer from the very program it replaces.
    Remove-Item "$exe.old" -Force -ErrorAction SilentlyContinue
    if (Test-Path $exe) { Move-Item $exe "$exe.old" -Force }
    Copy-Item $src $exe
  } finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
  }
  $v = & $exe version
  Say "+ augmented brain $v installed in $(Tilde $Dir)" "+ augmented brain $v instalado en $(Tilde $Dir)"

  # --- 3. PATH ------------------------------------------------------------------------
  # The plugin runs `augmented-brain` by name, so Claude Code must find it in the PATH.
  # Claude Code's own installer puts claude in ~\.local\bin but leaves the PATH to you.
  AddToPath $Dir
  $changed = $script:newTerminal
  if ((HasClaude) -and $claudeDir -ine $Dir) { AddToPath $claudeDir; $changed = $changed -or $script:newTerminal }
  if (HasGit) { if (-not (Get-Command git -ErrorAction SilentlyContinue)) { $env:Path = "$gitCmd;$env:Path"; $changed = $true } }

  Write-Host ''
  Say "This is an alpha version: it may have bugs and may change. Disclaimer:" "Es una versi\u00f3n alfa: puede tener errores y cambiar. Aviso legal:"
  Write-Host "  https://github.com/$Repo/blob/main/DISCLAIMER.md"

  # --- 4. Set up ----------------------------------------------------------------------
  if ($env:AB_NO_INIT -eq '1') { return }
  $cfg = if ($env:AB_CONFIG_DIR) { $env:AB_CONFIG_DIR } else { Join-Path $env:APPDATA 'augmented-brain' }
  if (Test-Path (Join-Path $cfg 'config.toml')) {
    Write-Host ''
    Say "Already set up ($(Tilde $cfg)): the program is updated; now the Claude Code plugin." "Ya estaba configurado ($(Tilde $cfg)): programa actualizado; ahora, el plugin de Claude Code."
    & $exe update --plugin
    return
  }
  $app = DesktopApp
  $env:AB_NEW_TERMINAL = if ($changed) { '1' } else { '' }
  $env:AB_LAUNCH = if ($interactive) { '1' } else { '' }
  $env:AB_DESKTOP = if ($app) { '1' } else { '' }
  & $exe init --language $lang
  if (-not $interactive) { return }
  $work = Join-Path $env:USERPROFILE ('augmented-brain\' + $(if ($lang -eq 'es') { 'trabajo' } else { 'work' }))
  Write-Host ''
  if ($app) {
    if (Ask (Text 'Open the Claude app now?' '\u00bfAbro ahora la app de Claude?')) {
      Start-Process "shell:AppsFolder\$($app.AppID)"
      Say "In the app, sign in, open the Code tab and click + New: without choosing a folder you work in your personal space; for work, pick $(Tilde $work) with Choose\u2026" `
          "En la app, entra con tu cuenta, abre la pesta\u00f1a Code y pulsa + New: sin elegir carpeta trabajas en tu espacio personal; para el de trabajo, elige $(Tilde $work) con Choose\u2026"
    }
  } elseif (Ask (Text 'Open Claude now in your work space?' '\u00bfAbro Claude ahora en tu espacio de trabajo?')) {
    Set-Location $work
    & claude
  }
} catch {
  $m = $_.Exception.Message
  if ($m -notlike 'augmented brain:*') { $m = "augmented brain: $m" }
  Write-Host $m -ForegroundColor Red
}
}
