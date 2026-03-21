#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$SCRIPT_DIR"
PLATFORMS_CFG="$ROOT_DIR/config/platforms.yaml"
TARGETS_CFG="$ROOT_DIR/config/targets.yaml"

platforms=$(yq '. | keys | .[]' "$PLATFORMS_CFG")

while IFS= read -r platform; do
  [[ -z "$platform" ]] && continue

  target_root=$(yq ".${platform} // \".\"" "$TARGETS_CFG")
  [[ "$target_root" == "null" || -z "$target_root" ]] && target_root="."
  if [[ "$target_root" != /* ]]; then
    target_root="$ROOT_DIR/$target_root"
  fi

  paths=$(yq ".${platform}.types | to_entries | .[].value.path" "$PLATFORMS_CFG" | sort -u)

  while IFS= read -r path; do
    [[ -z "$path" || "$path" == "null" ]] && continue
    full_path="$target_root/$path"
    if [[ -d "$full_path" ]]; then
      rm -rf "$full_path"
      echo "  removed $full_path"
    fi
  done <<< "$paths"
done <<< "$platforms"

echo "Clean complete."
