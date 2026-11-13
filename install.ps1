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
# AB_MARKETPLACE, AB_NO_INIT=1, AB_API_URL and AB_RETRY_DELAY, as in install.sh.
# AB_EXIT_CODE=1 (set by `augmented-brain update`) ends with exit 1 when it fails.
#
# It runs inside `irm | iex`, in the person's own PowerShell window: it never calls exit,
# which would close that window (only with AB_EXIT_CODE=1, in a PowerShell of its own). The Spanish messages are written with \u escapes, so
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
  # Enter, yes or no: anything else asks again, so a stray word is never taken for a yes.
  while ($true) {
    $a = (Read-Host "$q $(Text '[Y/n]' '[S/n]')").Trim()
    if ($a -match '^(|y|yes|s|si|s\u00ed)$') { return $true }
    if ($a -match '^(n|no)$') { return $false }
    Say "Answer y (yes) or n (no)." "Contesta s (s\u00ed) o n (no)."
  }
}
# Quiet <scriptblock>: runs an installation without its noise; its output is shown only
# when it fails.
# Busy <program> <arguments> [-Hold]: a long step, shown as a dot every few seconds
# instead of silence. On failure it shows only the last lines of the output and keeps all
# of it. With -Hold it shows nothing yet: there is another way to get it, and Held shows
# the output only if that fails too, so a newcomer never reads an error that did not matter.
function Busy([string]$exe, [string[]]$arguments, [switch]$Hold) {
  $log = [IO.Path]::GetTempFileName()
  $p = Start-Process -FilePath $exe -ArgumentList $arguments -NoNewWindow -PassThru `
    -RedirectStandardOutput $log -RedirectStandardError "$log.err"
  $null = $p.Handle  # without this, ExitCode can come back empty
  while (-not $p.HasExited) { Write-Host -NoNewline '.'; Start-Sleep -Seconds 3 }
  Write-Host ''
  $p.WaitForExit()
  if ($p.ExitCode -eq 0) { Remove-Item -Force $log, "$log.err" -ErrorAction SilentlyContinue; return $true }
  $script:heldLog = $log
  if (-not $Hold) { Held }
  return $false
}
# Held: the last lines of the output Busy kept, and where the whole of it is.
function Held {
  if (-not $script:heldLog) { return }
  $log = $script:heldLog
  Get-Content $log, "$log.err" -ErrorAction SilentlyContinue | Select-Object -Last 12 | ForEach-Object { Write-Host "    | $_" }
  Say "(the whole output is in $log)" "(la salida completa est\u00e1 en $log)"
}

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
  if ($env:AB_LANGUAGE) { $lang = if ($env:AB_LANGUAGE -match '^es') { 'es' } else { 'en' } }
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
  # GitFromGitHub: Git for Windows straight from its GitHub release, for when winget is
  # missing or broken (an old App Installer whose source does not open). The download
  # must match the release's checksum and carry a valid signature before it runs.
  function GitFromGitHub {
    try {
      $rel = Invoke-RestMethod -Uri 'https://api.github.com/repos/git-for-windows/git/releases/latest' -TimeoutSec 30 -ErrorAction Stop
      $suffix = if ($arch -eq 'arm64') { '-arm64.exe' } else { '-64-bit.exe' }
      $asset = $rel.assets | Where-Object { $_.name -like 'Git-*' -and $_.name -like "*$suffix" -and $_.name -notlike '*rc*' } | Select-Object -First 1
      if (-not $asset) { return $false }
      $exe = Join-Path ([IO.Path]::GetTempPath()) $asset.name
      (New-Object Net.WebClient).DownloadFile($asset.browser_download_url, $exe)
      # Its checksum from the release, and the signature of Git for Windows itself: an
      # installer that cannot show both is not run.
      if (-not ($asset.digest -like 'sha256:*') -or (Get-FileHash -Algorithm SHA256 $exe).Hash -ne $asset.digest.Substring(7)) { Remove-Item $exe; return $false }
      $sig = Get-AuthenticodeSignature $exe
      if ($sig.Status -ne 'Valid' -or $sig.SignerCertificate.Subject -notlike '*Johannes Schindelin*') { Remove-Item $exe; return $false }
      $p = Start-Process -FilePath $exe -ArgumentList '/VERYSILENT', '/NORESTART', '/NOCANCEL', '/SP-', '/SUPPRESSMSGBOXES' -Wait -PassThru
      Remove-Item $exe -ErrorAction SilentlyContinue
      return $p.ExitCode -eq 0
    } catch { return $false }
  }
  function DesktopApp { Get-StartApps -ErrorAction SilentlyContinue | Where-Object { $_.Name -eq 'Claude' } | Select-Object -First 1 }

  # --- 1. What it needs ---------------------------------------------------------------
  if ($env:AB_NO_INIT -ne '1') {
    if (-not (HasGit)) {
      Say "augmented brain needs Git for Windows: for the history and backup of your notes, and for Git Bash, where Claude runs commands safely." `
          "augmented brain necesita Git for Windows: para el historial y la copia de tus notas, y por Git Bash, donde Claude ejecuta los comandos con seguridad."
      $cmd = if ($winget) { 'winget install --id Git.Git -e --source winget' } else { 'https://git-scm.com/download/win' }
      if (-not (Ask (Text "Install it now? Windows may ask for permission." "\u00bfLo instalo ahora? Windows puede pedirte permiso."))) {
        Say "Install it with this, then run this installer again:" "Inst\u00e1lalo con esto y vuelve a lanzar este instalador:"
        Write-Host "  $cmd"; return
      }
      Say "Installing Git for Windows..." "Instalando Git for Windows..."
      $script:heldLog = $null
      $ok = $winget -and (Busy 'winget' @('install', '--id', 'Git.Git', '-e', '--source', 'winget', '--silent', '--accept-source-agreements', '--accept-package-agreements', '--disable-interactivity') -Hold)
      if (-not $ok) {
        # No winget, or one that cannot reach its source (a Windows just installed often
        # has not set its sources up yet): Git's own release, checked. winget's output is
        # shown only if this fails too.
        if ($winget) { Say "Downloading it from Git for Windows' own page instead..." "Lo descargo de la p\u00e1gina de Git for Windows..." }
        if (-not (GitFromGitHub)) { Held; Fail (Text 'Git was not installed. Install it from https://git-scm.com/download/win, then run this installer again.' 'Git no se ha instalado. Inst\u00e1lalo desde https://git-scm.com/download/win y vuelve a lanzar este instalador.') }
      }
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
      if (-not (Busy 'powershell' @('-NoProfile', '-ExecutionPolicy', 'Bypass', '-Command', "`"$cmd`""))) { Fail (Text 'Claude Code was not installed.' 'Claude Code no se ha instalado.') }
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
        Say "Installing the desktop app (it can take a couple of minutes). If Windows asks for permission in a window, accept it." `
            "Instalando la app de escritorio (puede tardar un par de minutos). Si Windows te pide permiso en una ventana, ac\u00e9ptalo."
        $script:heldLog = $null
        $null = Busy 'winget' @('install', '--id', 'Anthropic.Claude', '-e', '--source', 'winget', '--silent', '--accept-source-agreements', '--accept-package-agreements', '--disable-interactivity') -Hold
        # What counts is whether the app is there, not winget's exit code: its output is
        # shown only when the app is not.
        if (DesktopApp) {
          Say "+ Claude desktop app installed" "+ app de escritorio de Claude instalada"
        } else {
          Held
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
    # GetFile <url> <file> [accept]: one address to a file. file:// works too, for
    # tests: Invoke-WebRequest does not take it, WebClient does. GitHub's API wants a
    # User-Agent.
    function GetFile([string]$url, [string]$dst, [string]$accept) {
      $web = New-Object Net.WebClient
      $web.Headers.Add('User-Agent', 'augmented-brain-installer')
      if ($accept) { $web.Headers.Add('Accept', $accept) }
      $web.DownloadFile($url, $dst)
    }
    # Fetch <name> <file>: one file of the release. The public address has answered 504
    # for over an hour (8-nov-2026) while GitHub's API served the same files, so when it
    # fails the file comes through the API. Three tries, a few seconds apart.
    # AB_API_URL, for tests, stands in for the API.
    function Fetch([string]$name, [string]$dst) {
      $api = if ($env:AB_API_URL) { $env:AB_API_URL.TrimEnd('/') } else { "https://api.github.com/repos/$Repo" }
      $api = if ($Version -eq 'latest') { "$api/releases/latest" } else { "$api/releases/tags/v$($Version.TrimStart('v'))" }
      $delay = if ($env:AB_RETRY_DELAY) { [int]$env:AB_RETRY_DELAY } else { 5 }
      for ($try = 1; $try -le 3; $try++) {
        try { GetFile "$base/$name" $dst ''; return $true } catch {}
        if (-not $env:AB_RELEASE_URL -or $env:AB_API_URL) {
          try {
            GetFile $api "$tmp\release.json" 'application/vnd.github+json'
            $asset = (Get-Content -Raw "$tmp\release.json" | ConvertFrom-Json).assets | Where-Object { $_.name -eq $name } | Select-Object -First 1
            if ($asset) { GetFile $asset.url $dst 'application/octet-stream'; return $true }
          } catch {}
        }
        if ($try -lt 3) { Start-Sleep -Seconds $delay }
      }
      Remove-Item $dst -Force -ErrorAction SilentlyContinue
      return $false
    }
    function Cannot([string]$name) {
      Fail (Text "cannot download $name from GitHub (tried 3 times, also through its API). GitHub may be having trouble: nothing was installed; try again in a while." `
                 "no puedo descargar $name de GitHub (3 intentos, tambi\u00e9n por su API). Puede que GitHub tenga problemas: no se ha instalado nada; vuelve a probar dentro de un rato.")
    }
    if (-not (Fetch 'checksums.txt' "$tmp\checksums.txt")) { Cannot 'checksums.txt' }
    $line = Get-Content "$tmp\checksums.txt" | Where-Object { $_ -match "_windows_$arch\.zip$" } | Select-Object -First 1
    if (-not $line) { Fail (Text "the release has no archive for windows/$arch." "la versi\u00f3n no tiene archivo para windows/$arch.") }
    $want, $archive = $line -split '\s+', 2
    if (-not (Fetch $archive "$tmp\$archive")) { Cannot $archive }
    $got = (Get-FileHash -Algorithm SHA256 "$tmp\$archive").Hash
    if ($got -ine $want) { Fail (Text 'the download does not match its checksum: not installed.' 'la descarga no coincide con su suma de comprobaci\u00f3n: no se ha instalado.') }
    Expand-Archive -Path "$tmp\$archive" -DestinationPath $tmp -Force
    $src = Join-Path $tmp ($archive -replace '\.zip$', '') | Join-Path -ChildPath 'augmented-brain.exe'
    # The new program must start before it replaces the one that works (an antivirus,
    # a build for another system): with none at all, every tool of Claude fails until
    # someone installs it again.
    $v = $null
    try { $v = & $src version 2>$null } catch {}
    if ($LASTEXITCODE -ne 0 -or -not $v) {
      Fail (Text "the new program does not start on this computer: not installed; the one you had, if any, is still there." `
                 "el programa nuevo no arranca en este ordenador: no se ha instalado; el que ten\u00edas, si hab\u00eda uno, sigue ah\u00ed.")
    }
    New-Item -ItemType Directory -Path $Dir -Force | Out-Null
    $exe = Join-Path $Dir 'augmented-brain.exe'
    # A running program cannot be overwritten on Windows, but it can be renamed: the
    # update command runs this installer from the very program it replaces. The old
    # one comes back if the new one cannot be copied or does not answer in its place.
    Remove-Item "$exe.old" -Force -ErrorAction SilentlyContinue
    if (Test-Path $exe) { Move-Item $exe "$exe.old" -Force }
    $v = $null
    try { Copy-Item $src $exe -ErrorAction Stop; $v = & $exe version 2>$null } catch {}
    if ($LASTEXITCODE -ne 0 -or -not $v) {
      Remove-Item $exe -Force -ErrorAction SilentlyContinue
      if (Test-Path "$exe.old") { Move-Item "$exe.old" $exe -Force }
      Fail (Text "the new program does not start in its place: not installed; the one you had, if any, is back." `
                 "el programa nuevo no arranca en su sitio: no se ha instalado; el que ten\u00edas, si hab\u00eda uno, ha vuelto.")
    }
  } finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
  }
  Say "+ augmented brain $v installed in $(Tilde $Dir)" "+ augmented brain $v instalado en $(Tilde $Dir)"

  # --- 3. PATH ------------------------------------------------------------------------
  # The plugin runs `augmented-brain` by name, so Claude Code must find it in the PATH.
  # Claude Code's own installer puts claude in ~\.local\bin but leaves the PATH to you.
  AddToPath $Dir
  $changed = $script:newTerminal
  if ((HasClaude) -and $claudeDir -ine $Dir) { AddToPath $claudeDir; $changed = $changed -or $script:newTerminal }
  if (HasGit) { if (-not (Get-Command git -ErrorAction SilentlyContinue)) { $env:Path = "$gitCmd;$env:Path"; $changed = $true } }

  Write-Host ''
  Say "This is a beta version: it may have bugs, but what you have keeps working. Disclaimer:" "Es una versi\u00f3n beta: puede tener errores, pero lo que tienes sigue funcionando. Aviso legal:"
  Write-Host "  https://github.com/$Repo/blob/main/DISCLAIMER.md"
  if (Get-Command tailscale -ErrorAction SilentlyContinue) {
    Say "You have Tailscale: augmented-brain panel remote opens your panel from your other devices." "Tienes Tailscale: augmented-brain panel remote abre tu panel desde tus otros dispositivos."
  }

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
    # A Claude app already running keeps the window: say so instead of opening another.
    $running = Get-Process -Name claude -ErrorAction SilentlyContinue | Where-Object { $_.Path -and $_.Path -notlike '*\.local\bin\*' }
    if ($running) {
      Say "The Claude app is already open. Close it completely and open it again, so it loads augmented brain; then the Code tab and + New." `
          "La app de Claude ya est\u00e1 abierta. Ci\u00e9rrala del todo y vuelve a abrirla, para que cargue augmented brain; luego, pesta\u00f1a Code y + New."
    } elseif (Ask (Text 'Open the Claude app now?' '\u00bfAbro ahora la app de Claude?')) {
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
  # `augmented-brain update` runs this file in a PowerShell of its own, and needs to
  # know it failed. In a person's own window, exit would close it.
  if ($env:AB_EXIT_CODE -eq '1') { exit 1 }
}
}
