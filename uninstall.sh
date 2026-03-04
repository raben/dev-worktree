#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${1:-$HOME/.local/bin}"
target="$INSTALL_DIR/dev"

if [ -L "$target" ] || [ -e "$target" ]; then
  rm "$target"
  echo "Removed $target"
else
  echo "Not installed at $target"
fi

echo "State directory (~/.wt-dev/) was kept."
