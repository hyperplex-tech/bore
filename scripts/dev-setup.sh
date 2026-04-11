#!/usr/bin/env bash
#
# Interactive developer setup for bore.
#
# Checks for required build dependencies and offers to install missing ones.
# Safe to re-run — skips anything already installed.
#
# Usage:
#   ./scripts/dev-setup.sh          # interactive (prompts before installing)
#   ./scripts/dev-setup.sh --auto   # non-interactive (installs everything)

set -euo pipefail

AUTO=false
if [ "${1:-}" = "--auto" ]; then
    AUTO=true
fi

# --- Colors (respect NO_COLOR) ---
if [ -z "${NO_COLOR:-}" ] && [ -t 1 ]; then
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[0;33m'
    BOLD='\033[1m'
    RESET='\033[0m'
else
    GREEN='' RED='' YELLOW='' BOLD='' RESET=''
fi

ok()      { printf "  %-35s ${GREEN}[ok]${RESET} %s\n" "$1" "${2:-}"; }
missing() { printf "  %-35s ${RED}[MISSING]${RESET}\n" "$1"; }
warn()    { printf "  ${YELLOW}%s${RESET}\n" "$1"; }
header()  { printf "\n${BOLD}%s${RESET}\n" "$1"; }

# --- Detect package manager ---
PKG_MANAGER=""
PKG_INSTALL=""

detect_pkg_manager() {
    if [ "$(uname)" = "Darwin" ]; then
        if command -v brew >/dev/null 2>&1; then
            PKG_MANAGER="brew"
            PKG_INSTALL="brew install"
        else
            echo "macOS detected but Homebrew not found."
            echo "Install it from https://brew.sh/ then re-run this script."
            exit 1
        fi
    elif command -v apt-get >/dev/null 2>&1; then
        PKG_MANAGER="apt"
        PKG_INSTALL="sudo apt-get install -y"
    elif command -v dnf >/dev/null 2>&1; then
        PKG_MANAGER="dnf"
        PKG_INSTALL="sudo dnf install -y"
    elif command -v pacman >/dev/null 2>&1; then
        PKG_MANAGER="pacman"
        PKG_INSTALL="sudo pacman -S --noconfirm"
    else
        PKG_MANAGER="unknown"
    fi
}

# --- Helpers ---

confirm() {
    if $AUTO; then return 0; fi
    local prompt="${1:-Install?} [y/N] "
    read -r -p "  $prompt" answer
    [[ "$answer" =~ ^[Yy]$ ]]
}

install_pkg() {
    local pkg_apt="${1:-}" pkg_dnf="${2:-}" pkg_pacman="${3:-}" pkg_brew="${4:-}"
    local pkg=""

    case "$PKG_MANAGER" in
        apt)    pkg="$pkg_apt" ;;
        dnf)    pkg="$pkg_dnf" ;;
        pacman) pkg="$pkg_pacman" ;;
        brew)   pkg="$pkg_brew" ;;
        *)
            warn "Could not detect package manager. Install manually."
            return 1
            ;;
    esac

    if [ -z "$pkg" ]; then
        warn "No package available for $PKG_MANAGER. Install manually."
        return 1
    fi

    echo "  Running: $PKG_INSTALL $pkg"
    $PKG_INSTALL $pkg
}

INSTALLED=0
SKIPPED=0
FAILED=0

# --- Core dependencies ---

header "Core dependencies (required)"

# Go
if command -v go >/dev/null 2>&1; then
    GO_VER=$(go version | sed -n 's/.*go1\.\([0-9]*\).*/\1/p')
    if [ -n "$GO_VER" ] && [ "$GO_VER" -ge 25 ] 2>/dev/null; then
        ok "go" "$(go version | sed 's/go version //')"
    else
        missing "go (need 1.25+, got $(go version))"
        warn "Install from https://go.dev/dl/"
        FAILED=$((FAILED + 1))
    fi
else
    missing "go (1.25+)"
    warn "Install from https://go.dev/dl/"
    warn "Go is not auto-installed — too many ways to manage versions."
    FAILED=$((FAILED + 1))
fi

# protoc
if command -v protoc >/dev/null 2>&1; then
    ok "protoc" "$(protoc --version 2>&1)"
else
    missing "protoc"
    if confirm "Install protoc?"; then
        install_pkg protobuf-compiler protobuf-compiler protobuf protobuf && ok "protoc" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
    else
        SKIPPED=$((SKIPPED + 1))
    fi
fi

# Ensure GOPATH/bin is in PATH for go install binaries
GOBIN="${GOBIN:-${GOPATH:-$HOME/go}/bin}"

# protoc-gen-go
if command -v protoc-gen-go >/dev/null 2>&1; then
    ok "protoc-gen-go"
else
    missing "protoc-gen-go"
    if confirm "Install via 'go install'?"; then
        echo "  Running: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && ok "protoc-gen-go" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
    else
        SKIPPED=$((SKIPPED + 1))
    fi
fi

# protoc-gen-go-grpc
if command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    ok "protoc-gen-go-grpc"
else
    missing "protoc-gen-go-grpc"
    if confirm "Install via 'go install'?"; then
        echo "  Running: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && ok "protoc-gen-go-grpc" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
    else
        SKIPPED=$((SKIPPED + 1))
    fi
fi

# Check GOBIN in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$GOBIN"; then
    echo ""
    warn "Note: $GOBIN is not in your PATH."
    warn "Add it:  export PATH=\"$GOBIN:\$PATH\""
fi

# --- Desktop dependencies ---

SETUP_DESKTOP=false
echo ""
if $AUTO; then
    SETUP_DESKTOP=true
else
    read -r -p "  Do you plan to work on the desktop app? [y/N] " answer
    [[ "$answer" =~ ^[Yy]$ ]] && SETUP_DESKTOP=true
fi

if $SETUP_DESKTOP; then
    header "Desktop dependencies"

    # Node.js
    if command -v node >/dev/null 2>&1; then
        ok "node" "$(node --version)"
    else
        missing "node"
        if confirm "Install Node.js?"; then
            install_pkg nodejs nodejs nodejs node && ok "node" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
        else
            SKIPPED=$((SKIPPED + 1))
        fi
    fi

    # npm
    if command -v npm >/dev/null 2>&1; then
        ok "npm" "$(npm --version)"
    else
        missing "npm"
        warn "npm is bundled with Node.js — install Node.js first."
        FAILED=$((FAILED + 1))
    fi

    # Linux-only: webkit/gtk headers
    if [ "$(uname)" = "Linux" ]; then
        # pkg-config
        if command -v pkg-config >/dev/null 2>&1; then
            ok "pkg-config"
        else
            missing "pkg-config"
            if confirm "Install pkg-config?"; then
                install_pkg pkg-config pkg-config pkg-config pkg-config && ok "pkg-config" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
            else
                SKIPPED=$((SKIPPED + 1))
            fi
        fi

        # webkit2gtk
        if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then
            ok "webkit2gtk-4.1"
        else
            missing "webkit2gtk-4.1"
            if confirm "Install webkit2gtk dev headers?"; then
                install_pkg libwebkit2gtk-4.1-dev webkit2gtk4.1-devel webkit2gtk-4.1 "" && ok "webkit2gtk-4.1" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
            else
                SKIPPED=$((SKIPPED + 1))
            fi
        fi

        # build-essential / gcc
        if command -v gcc >/dev/null 2>&1; then
            ok "gcc"
        else
            missing "gcc"
            if confirm "Install build tools (gcc)?"; then
                install_pkg build-essential gcc base-devel "" && ok "gcc" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
            else
                SKIPPED=$((SKIPPED + 1))
            fi
        fi
    fi
else
    header "Desktop dependencies"
    echo "  (skipped)"
fi

# --- Linting ---

header "Linting (optional)"

if command -v golangci-lint >/dev/null 2>&1; then
    ok "golangci-lint" "$(golangci-lint version 2>&1 | head -1)"
else
    missing "golangci-lint"
    if confirm "Install golangci-lint?"; then
        echo "  Running: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && ok "golangci-lint" "(just installed)" && INSTALLED=$((INSTALLED + 1)) || FAILED=$((FAILED + 1))
    else
        SKIPPED=$((SKIPPED + 1))
    fi
fi

# --- Summary ---

header "Summary"
echo "  Installed: $INSTALLED"
[ $SKIPPED -gt 0 ] && echo "  Skipped:   $SKIPPED"
[ $FAILED -gt 0 ]  && echo "  Failed:    $FAILED"
echo ""

if [ $FAILED -gt 0 ]; then
    echo "Some dependencies still need attention. Fix them and re-run this script."
    exit 1
else
    echo "You're ready to go! Next steps:"
    echo "  make check   — verify everything"
    echo "  make build   — build all binaries"
    echo "  make dev     — build + run daemon for development"
fi
