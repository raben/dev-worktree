#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="${1:-$HOME/.local/bin}"

echo "Installing dev to $INSTALL_DIR ..."
mkdir -p "$INSTALL_DIR"

target="$INSTALL_DIR/dev"
[ -L "$target" ] || [ -e "$target" ] && rm "$target"
ln -s "$SCRIPT_DIR/bin/dev" "$target"
chmod +x "$SCRIPT_DIR/bin/dev"
echo "  dev -> $SCRIPT_DIR/bin/dev"

if ! echo "$PATH" | tr ':' '\n' | grep -q "^${INSTALL_DIR}$"; then
  echo ""
  echo "WARNING: $INSTALL_DIR is not in PATH."
  echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
fi

echo ""
echo "Checking dependencies..."
for dep in jq docker git; do
  command -v "$dep" &>/dev/null && echo "  $dep: OK" || echo "  $dep: MISSING"
done
command -v devcontainer &>/dev/null && echo "  devcontainer: OK" || echo "  devcontainer: MISSING (npm install -g @devcontainers/cli)"

echo ""
echo "Done. Run: dev --help"
