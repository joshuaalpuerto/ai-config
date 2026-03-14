#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLATFORMS_CFG="$ROOT_DIR/config/platforms.yaml"
TOOL_MAP_CFG="$ROOT_DIR/config/tool-map.yaml"
errors=0
warnings=0

# Derive types from platforms.yaml (single source of truth)
types=$(yq '.[].types | keys | .[]' "$PLATFORMS_CFG" | sort -u)

# Collect valid platform names from platforms.yaml
valid_platforms=$(yq '. | keys | .[]' "$PLATFORMS_CFG")

# Collect valid canonical tool names from tool-map.yaml
valid_tools=$(yq '.claude | keys | .[]' "$TOOL_MAP_CFG" 2>/dev/null || true)

while IFS= read -r type; do
  [[ -z "$type" ]] && continue
  src_dir="$ROOT_DIR/src/${type}"
  [[ ! -d "$src_dir" ]] && continue

  # Use find to recurse into subdirectories
  while IFS= read -r src_file; do
    [[ ! -f "$src_file" ]] && continue
    rel_path="${src_file#"$ROOT_DIR/"}"

    # Skip files with no frontmatter (passthrough files)
    if ! head -1 "$src_file" | grep -q '^---$'; then
      continue
    fi

    # Validate frontmatter is parseable by yq
    if ! yq --front-matter=extract '.' "$src_file" &>/dev/null; then
      echo "ERROR: ${rel_path} — invalid frontmatter YAML"
      ((errors++))
      continue
    fi

    # Skip files with empty frontmatter
    fm_len=$(yq --no-doc --front-matter=extract '. | length' "$src_file" 2>/dev/null || echo "0")
    [[ "$fm_len" == "0" ]] && continue

    # Check required 'description' field
    desc=$(yq --front-matter=extract '.description // ""' "$src_file")
    if [[ -z "$desc" ]]; then
      echo "WARN: ${rel_path} — missing 'description' field"
      ((warnings++))
    fi

    # Validate override platform names match platforms.yaml
    override_platforms=$(yq --front-matter=extract \
      '.overrides // {} | keys | .[]' "$src_file" 2>/dev/null || true)
    if [[ -n "$override_platforms" ]]; then
      while IFS= read -r op; do
        [[ -z "$op" ]] && continue
        if ! echo "$valid_platforms" | grep -qx "$op"; then
          echo "ERROR: ${rel_path} — override targets unknown platform '${op}' (valid: $(echo $valid_platforms | tr '\n' ', ' | sed 's/, $//'))"
          ((errors++))
        fi
      done <<< "$override_platforms"
    fi

    # Validate tool names exist in tool-map.yaml
    tools=$(yq --front-matter=extract '.tools[]' "$src_file" 2>/dev/null || true)
    if [[ -n "$tools" && -n "$valid_tools" ]]; then
      while IFS= read -r tool; do
        [[ -z "$tool" ]] && continue
        # Strip scope: Bash(npm test) → Bash
        base="${tool%%(*}"
        if ! echo "$valid_tools" | grep -qx "$base"; then
          echo "WARN: ${rel_path} — tool '${base}' has no mapping in tool-map.yaml (will be dropped for non-Claude platforms)"
          ((warnings++))
        fi
      done <<< "$tools"
    fi

  done < <(find "$src_dir" -name '*.md' | sort)
done <<< "$types"

if [[ $errors -gt 0 ]]; then
  echo ""
  echo "$errors error(s), $warnings warning(s) found."
  exit 1
else
  echo "All source files valid. ($warnings warning(s))"
fi
