#!/bin/sh
# install.sh — installs augmented brain on Linux or macOS and sets it up. No root needed.
#
#   curl -fsSL https://github.com/icalvete/augmented-brain/releases/latest/download/install.sh | sh
#
# 1. Checks what it needs first (git and Claude Code) and offers to install what is
#    missing; without a terminal to ask, it says the exact command and stops. It also
#    offers the Claude desktop app, friendlier than the terminal.
# 2. Downloads the archive for this system from the latest release, checks its SHA-256
#    against the release's checksums.txt, and puts the program in ~/.local/bin (or
#    $AB_INSTALL_DIR).
# 3. Adds that folder to the PATH in the shell's startup file, if it is not there.
# 4. Runs `augmented-brain init` the first time, which asks nothing, and offers to
#    open Claude right away: the desktop app if it is there, otherwise in this terminal.
#
# Settings: AB_YES=1 answers yes to every question, AB_LANGUAGE=en|es sets the language,
# AB_NO_DESKTOP=1 does not offer the desktop app (servers, tests). For testing:
# AB_VERSION (a version, default: latest), AB_RELEASE_URL (where the release files are,
# default: the GitHub release), AB_MARKETPLACE (where the plugin comes from, default:
# this repository on GitHub) and AB_NO_INIT=1 (only install the program: no checks, no
# init).
set -eu

REPO="icalvete/augmented-brain"
VERSION="${AB_VERSION:-latest}"
DIR="${AB_INSTALL_DIR:-$HOME/.local/bin}"

# The language of the messages, from the locale, as the program reads it.
lang=en
for v in "${LC_ALL:-}" "${LC_MESSAGES:-}" "${LANG:-}"; do
  if [ -n "$v" ]; then
    case "$v" in es*) lang=es ;; esac
    break
  fi
done
# Our own messages stand out in colour from what other installers print (apt, Claude
# Code's), when the output is a terminal and NO_COLOR is not set.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  on="$(printf '\033[1;32m')"; off="$(printf '\033[0m')"
else
  on=""; off=""
fi
# say <english> <spanish>
say() { if [ "$lang" = es ]; then echo "$on$2$off"; else echo "$on$1$off"; fi; }
fail() { echo "augmented brain: $*" >&2; exit 1; }
# ask <question>: true when the answer is yes (the default). The script itself arrives
# on stdin (curl | sh), so the question goes to the terminal. With no terminal, it is a
# no, unless AB_YES=1.
ask() {
  [ "${AB_YES:-}" = 1 ] && return 0
  (exec </dev/tty) 2>/dev/null || return 1
  printf '%s ' "$1" >/dev/tty
  read -r answer </dev/tty || return 1
  case "$answer" in [nN]*) return 1 ;; *) return 0 ;; esac
}
yn() { say "[Y/n]" "[S/n]"; }

# quiet <command…>: runs an installation without its noise (apt, other installers),
# keeping only what asks for a password, which goes to the terminal. Its output is
# shown only when it fails.
quiet() {
  log="$(mktemp)"
  tty=/dev/null
  (exec </dev/tty) 2>/dev/null && tty=/dev/tty
  if "$@" >"$log" 2>&1 <"$tty"; then
    rm -f "$log"
  else
    cat "$log" >&2
    rm -f "$log"
    return 1
  fi
}

# The language, asked first: the system's is the default. AB_LANGUAGE answers it.
if [ -n "${AB_LANGUAGE:-}" ]; then
  lang="$AB_LANGUAGE"
elif [ "${AB_YES:-}" != 1 ] && (exec </dev/tty) 2>/dev/null; then
  printf '%s ' "${on}Language / Idioma: English or español? [en/es] ($lang)${off}" >/dev/tty
  read -r answer </dev/tty || answer=""
  case "$answer" in
    es* | ES* | Es* | sp* | ca*) lang=es ;;
    en* | EN* | En* | in*) lang=en ;;
  esac
fi
# tilde <path>: the path with ~ for the home folder, shorter to read.
tilde() { case "$1" in "$HOME"/*) echo "~${1#"$HOME"}" ;; *) echo "$1" ;; esac; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  # On Windows, Git Bash can run this script too, but Windows has its own installer.
  MINGW* | MSYS* | CYGWIN*) fail "$(say "this is the installer for Linux and macOS. On Windows, open PowerShell and run: irm https://github.com/$REPO/releases/latest/download/install.ps1 | iex" "este es el instalador de Linux y macOS. En Windows, abre PowerShell y lanza: irm https://github.com/$REPO/releases/latest/download/install.ps1 | iex")" ;;
  *) fail "$(say "this system ($(uname -s)) is not supported yet: Linux, macOS and Windows only." "este sistema ($(uname -s)) aún no está soportado: solo Linux, macOS y Windows.")" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "$(say "this processor ($(uname -m)) is not supported yet." "este procesador ($(uname -m)) aún no está soportado.")" ;;
esac
# Claude Code and its desktop app need macOS 13 (Ventura) or later: say so before
# installing anything that would not start.
if [ "$os" = darwin ]; then
  macos="$(sw_vers -productVersion 2>/dev/null)"
  if [ "${macos%%.*}" -lt 13 ] 2>/dev/null; then
    fail "$(say "this Mac has macOS $macos, and Claude Code needs macOS 13 (Ventura) or later. Update it in System Settings → General → Software Update, then run this installer again." \
                "este Mac tiene macOS $macos, y Claude Code necesita macOS 13 (Ventura) o posterior. Actualízalo en Ajustes del Sistema → General → Actualización de software y vuelve a lanzar este instalador.")"
  fi
fi

# --- 1. What it needs -------------------------------------------------------------

# has_git: git that really works. On a Mac, /usr/bin/git exists even without Apple's
# developer tools: then it is only a stub that offers to install them.
has_git() {
  g="$(command -v git 2>/dev/null)" || return 1
  [ "$os" = darwin ] && [ "$g" = /usr/bin/git ] && { xcode-select -p >/dev/null 2>&1 || return 1; }
  return 0
}

# wait_for_xcode: the developer tools install in their own window; wait for them here
# (up to 45 minutes) instead of asking for the installer to be run again.
wait_for_xcode() {
  i=0
  until xcode-select -p >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -gt 540 ] && return 1
    sleep 5
  done
}

git_install() {
  if [ "$os" = darwin ]; then echo "xcode-select --install"; return; fi
  for pm in "apt-get:sudo apt install -y git" "dnf:sudo dnf install -y git" "pacman:sudo pacman -S --noconfirm git" "zypper:sudo zypper install -y git"; do
    if command -v "${pm%%:*}" >/dev/null 2>&1; then echo "${pm#*:}"; return; fi
  done
  echo "your system's package manager"
}

# Claude Code's own installer also uses ~/.local/bin; it may be there but not yet in
# this terminal's PATH.
has_claude() { command -v claude >/dev/null 2>&1 || [ -x "$HOME/.local/bin/claude" ]; }

# The Claude desktop app: its Code tab is Claude Code with windows instead of a terminal,
# and it loads the same plugins.
has_desktop() {
  command -v claude-desktop >/dev/null 2>&1 || [ -d /Applications/Claude.app ] || [ -d "$HOME/Applications/Claude.app" ]
}
DESKTOP_KEY_URL="https://downloads.claude.ai/claude-desktop/key.asc"
DESKTOP_KEY="/usr/share/keyrings/claude-desktop-archive-keyring.asc"
DESKTOP_REPO="deb [arch=amd64,arm64 signed-by=$DESKTOP_KEY] https://downloads.claude.ai/claude-desktop/apt/stable stable main"
# desktop_install: true when the app got installed. On Linux, from Anthropic's apt
# repository (Debian and Ubuntu only); on macOS, with Homebrew when it is there, or
# else straight from Anthropic's update feed (the one Homebrew reads too).
desktop_install() {
  case "$os" in
    linux)
      command -v apt-get >/dev/null 2>&1 || return 1
      sudo sh -c "curl -fsSLo '$DESKTOP_KEY' '$DESKTOP_KEY_URL' && echo '$DESKTOP_REPO' > /etc/apt/sources.list.d/claude-desktop.list && apt-get update -q && apt-get install -y claude-desktop"
      ;;
    darwin)
      if command -v brew >/dev/null 2>&1; then
        brew install --cask claude
      else
        mac_desktop_install
      fi
      ;;
  esac
}

# mac_desktop_install: the latest Claude.app from Anthropic's update feed, into
# /Applications (or ~/Applications when that is not writable). ditto keeps the app's
# signature intact, and codesign checks it before the app is put in place.
DESKTOP_FEED="https://downloads.claude.ai/releases/darwin/universal/RELEASES.json"
mac_desktop_install() {
  feed="$(curl -fsSL "$DESKTOP_FEED")" || return 1
  # the version the feed calls current, and the zip of that version
  cur="$(printf '%s' "$feed" | sed -n 's/.*"currentRelease":"\([^"]*\)".*/\1/p')"
  url="$(printf '%s' "$feed" | tr ',' '\n' | sed -n 's/.*"url":"\(https:[^"]*\.zip\)".*/\1/p' | grep -F "/$cur/" | head -n 1)"
  [ -n "$cur" ] && [ -n "$url" ] || return 1
  dl="$(mktemp -d)"
  curl -fsSL "$url" -o "$dl/Claude.zip" &&
    ditto -x -k "$dl/Claude.zip" "$dl/x" &&
    codesign --verify --deep --strict "$dl/x/Claude.app" || { rm -rf "$dl"; return 1; }
  dest=/Applications
  [ -w "$dest" ] || { dest="$HOME/Applications"; mkdir -p "$dest"; }
  rm -rf "$dest/Claude.app" && ditto "$dl/x/Claude.app" "$dest/Claude.app"
  rc=$?  # not "status": zsh, which can be sh on a Mac, keeps that name read-only
  rm -rf "$dl"
  return $rc
}

if [ "${AB_NO_INIT:-}" != 1 ]; then
  if ! has_git; then
    cmd="$(git_install)"
    say "augmented brain needs git, to keep the history and the backup of your notes." \
        "augmented brain necesita git, para el historial y la copia de seguridad de tus notas."
    case "$cmd" in
      sudo*)
        if ask "$(say "Install it now with: $cmd ? It will ask for your password." "¿Lo instalo ahora con: $cmd ? Te pedirá tu contraseña.") $(yn)"; then
          say "Installing git…" "Instalando git…"
          quiet sh -c "$cmd" || fail "$(say "git was not installed." "git no se ha instalado.")"
          say "✓ git installed" "✓ git instalado"
        else
          say "Install it with this, then run this installer again:" "Instálalo con esto y vuelve a lanzar este instalador:"
          echo "  $cmd"; exit 1
        fi ;;
      xcode-select*)
        xcode-select --install 2>/dev/null || true
        if [ "${AB_YES:-}" != 1 ] && (exec </dev/tty) 2>/dev/null; then
          say "A window opens to install Apple's developer tools, which bring git. Click Install and accept; it takes a few minutes. I wait here and carry on by myself when it finishes (if you closed the window, press Ctrl+C and run this installer again)." \
              "Se abre una ventana para instalar las herramientas de desarrollo de Apple, que traen git. Pulsa Instalar y acepta; tarda unos minutos. Yo espero aquí y sigo solo cuando acabe (si cerraste la ventana, pulsa Ctrl+C y vuelve a lanzar este instalador)."
          wait_for_xcode || fail "$(say "Apple's developer tools did not finish installing. Run this installer again." "las herramientas de desarrollo de Apple no terminaron de instalarse. Vuelve a lanzar este instalador.")"
          say "✓ git installed" "✓ git instalado"
        else
          say "A window opens to install Apple's developer tools, which bring git. Accept it, wait until it finishes, and run this installer again." \
              "Se abre una ventana para instalar las herramientas de desarrollo de Apple, que traen git. Acéptala, espera a que termine y vuelve a lanzar este instalador."
          exit 1
        fi ;;
      *)
        say "Install it with $cmd, then run this installer again." "Instálalo con $cmd y vuelve a lanzar este instalador."
        exit 1 ;;
    esac
  fi
  if ! has_claude; then
    say "augmented brain works with Claude Code, Anthropic's assistant, and it is not installed." \
        "augmented brain trabaja con Claude Code, el asistente de Anthropic, y no está instalado."
    if ask "$(say "Install it now with its official installer?" "¿Lo instalo ahora con su instalador oficial?") $(yn)"; then
      say "Installing Claude Code…" "Instalando Claude Code…"
      quiet sh -c "curl -fsSL https://claude.ai/install.sh | bash" || fail "$(say "Claude Code was not installed." "Claude Code no se ha instalado.")"
      say "✓ Claude Code installed" "✓ Claude Code instalado"
      has_claude || fail "$(say "Claude Code was installed but cannot be found; open a new terminal and run this installer again." "Claude Code se instaló, pero no lo encuentro; abre una terminal nueva y vuelve a lanzar este instalador.")"
    else
      say "Install it with this, then run this installer again:" "Instálalo con esto y vuelve a lanzar este instalador:"
      echo "  curl -fsSL https://claude.ai/install.sh | bash"; exit 1
    fi
  fi
  if [ "${AB_NO_DESKTOP:-}" != 1 ] && ! has_desktop; then
    can=0
    { [ "$os" = linux ] && command -v apt-get >/dev/null 2>&1; } && can=1
    [ "$os" = darwin ] && can=1
    # without a terminal to ask (and no AB_YES), say nothing: it is only an offer
    { [ "${AB_YES:-}" = 1 ] || (exec </dev/tty) 2>/dev/null; } || can=0
    if [ "$can" = 1 ]; then
      echo
      say "Claude also has a desktop app: the same Claude Code, with windows instead of a terminal. It is the easiest way to work." \
          "Claude tiene también una app de escritorio: el mismo Claude Code, con ventanas en vez de terminal. Es la forma más cómoda de trabajar."
      if ask "$(say "Install it too? On Linux it will ask for your password." "¿La instalo también? En Linux te pedirá tu contraseña.") $(yn)"; then
        say "Installing the desktop app (it can take a couple of minutes)…" "Instalando la app de escritorio (puede tardar un par de minutos)…"
        if quiet desktop_install; then
          say "✓ Claude desktop app installed" "✓ app de escritorio de Claude instalada"
        else
          say "The desktop app was not installed; you can use Claude in the terminal." \
              "La app de escritorio no se ha instalado; puedes usar Claude en la terminal."
        fi
      fi
    fi
  fi
fi

# --- 2. Download, check and install -----------------------------------------------

if [ -n "${AB_RELEASE_URL:-}" ]; then
  BASE="$AB_RELEASE_URL"
elif [ "$VERSION" = latest ]; then
  BASE="https://github.com/$REPO/releases/latest/download"
else
  BASE="https://github.com/$REPO/releases/download/v${VERSION#v}"
fi

fetch() { # fetch <url> <file>
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$1" -o "$2"
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$2" "$1"
  else
    fail "needs curl or wget to download."
  fi
}

if command -v sha256sum >/dev/null 2>&1; then
  sha() { sha256sum "$1" | awk '{print $1}'; }
elif command -v shasum >/dev/null 2>&1; then
  sha() { shasum -a 256 "$1" | awk '{print $1}'; }
else
  fail "needs sha256sum or shasum to check the download."
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fetch "$BASE/checksums.txt" "$TMP/checksums.txt" || fail "cannot download $BASE/checksums.txt"
archive="$(grep "_${os}_${arch}\.tar\.gz\$" "$TMP/checksums.txt" | awk '{print $2}' | head -n 1)"
[ -n "$archive" ] || fail "the release has no archive for ${os}/${arch}."
fetch "$BASE/$archive" "$TMP/$archive" || fail "cannot download $archive"

want="$(grep " ${archive}\$" "$TMP/checksums.txt" | awk '{print $1}')"
got="$(sha "$TMP/$archive")"
[ -n "$want" ] && [ "$got" = "$want" ] || fail "$(say "the download does not match its checksum: not installed." "la descarga no coincide con su suma de comprobación: no se ha instalado.")"

tar -xzf "$TMP/$archive" -C "$TMP"
mkdir -p "$DIR"
cp "$TMP/${archive%.tar.gz}/augmented-brain" "$DIR/augmented-brain.new"
chmod 755 "$DIR/augmented-brain.new"
mv "$DIR/augmented-brain.new" "$DIR/augmented-brain"

say "✓ augmented brain $("$DIR/augmented-brain" version) installed in $(tilde "$DIR")" \
    "✓ augmented brain $("$DIR/augmented-brain" version) instalado en $(tilde "$DIR")"

# --- 3. PATH ----------------------------------------------------------------------

# The plugin runs `augmented-brain` by name, so Claude Code must find it in the PATH.
new_terminal=0
case ":$PATH:" in
  *":$DIR:"*) ;;
  *)
    case "$(basename "${SHELL:-sh}")" in
      zsh) rc="$HOME/.zshrc" ;;
      bash)
        rc="$HOME/.bashrc"
        if [ "$os" = darwin ]; then
          # The first of these that exists is the one bash reads; creating .bash_profile
          # when only .profile exists would stop bash from reading .profile.
          rc="$HOME/.bash_profile"
          for f in "$HOME/.bash_profile" "$HOME/.bash_login" "$HOME/.profile"; do
            [ -f "$f" ] && { rc="$f"; break; }
          done
        fi ;;
      *) rc="$HOME/.profile" ;;
    esac
    case "$DIR" in
      "$HOME"/*) line="export PATH=\"\$HOME${DIR#"$HOME"}:\$PATH\"  # added by augmented brain" ;;
      *) line="export PATH=\"$DIR:\$PATH\"  # added by augmented brain" ;;
    esac
    if [ -f "$rc" ] && grep -qF "$DIR" "$rc"; then
      : # already there (Claude Code's installer may have added it)
    elif [ "$DIR" = "$HOME/.local/bin" ] && [ -f "$rc" ] && grep -q '\.local/bin' "$rc"; then
      :
    else
      printf '\n%s\n' "$line" >>"$rc"
      say "✓ added $(tilde "$DIR") to your PATH in $(tilde "$rc")" "✓ añadido $(tilde "$DIR") a tu PATH en $(tilde "$rc")"
    fi
    new_terminal=1
    PATH="$DIR:$PATH"
    export PATH
    ;;
esac
# Claude Code just installed in ~/.local/bin is not in this terminal's PATH yet either.
if [ -x "$HOME/.local/bin/claude" ]; then
  case ":$PATH:" in
    *":$HOME/.local/bin:"*) ;;
    *) PATH="$HOME/.local/bin:$PATH"; export PATH; new_terminal=1 ;;
  esac
fi

echo
say "This is an alpha version: it may have bugs and may change. Disclaimer:" \
    "Es una versión alfa: puede tener errores y cambiar. Aviso legal:"
echo "  https://github.com/$REPO/blob/main/DISCLAIMER.md"

# --- 4. Set up --------------------------------------------------------------------

[ "${AB_NO_INIT:-}" = 1 ] && exit 0
cfg="${AB_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/augmented-brain}"
if [ -f "$cfg/config.toml" ]; then
  echo
  say "Already set up ($(tilde "$cfg")): the program is updated; now the Claude Code plugin." \
      "Ya estaba configurado ($(tilde "$cfg")): programa actualizado; ahora, el plugin de Claude Code."
  "$DIR/augmented-brain" update --plugin </dev/null || true
  exit 0
fi
# Offer to open Claude right here: this terminal cannot pick up the new PATH, so the
# installer opens it with the right one. init then talks about the next times.
launch=0
(exec </dev/tty) 2>/dev/null && launch=1
desktop=0
has_desktop && desktop=1
AB_NEW_TERMINAL="$new_terminal" AB_LAUNCH="$launch" AB_DESKTOP="$desktop" "$DIR/augmented-brain" init --language "$lang" </dev/null
[ "$launch" = 1 ] || exit 0
if [ "$lang" = es ]; then work="$HOME/augmented-brain/trabajo"; else work="$HOME/augmented-brain/work"; fi
echo
if [ "$desktop" = 1 ]; then
  if ask "$(say "Open the Claude app now?" "¿Abro ahora la app de Claude?") $(yn)"; then
    if [ "$os" = darwin ]; then
      open -a Claude
    else
      nohup claude-desktop >/dev/null 2>&1 &
    fi
    say "In the app, sign in, open the Code tab and choose the folder $(tilde "$work")." \
        "En la app, entra con tu cuenta, abre la pestaña Code y elige la carpeta $(tilde "$work")."
  fi
elif ask "$(say "Open Claude now in your work space?" "¿Abro Claude ahora en tu espacio de trabajo?") $(yn)"; then
  cd "$work" && exec claude </dev/tty >/dev/tty 2>&1
fi
