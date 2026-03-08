#!/usr/bin/env bash
# Generic transpiler library. Requires yq v4+.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SRC_DIR="${SRC_DIR:-$ROOT_DIR/src}"
PLATFORMS_CFG="$ROOT_DIR/config/platforms.yaml"
TOOL_MAP_CFG="$ROOT_DIR/config/tool-map.yaml"
TARGETS_CFG="$ROOT_DIR/config/targets.yaml"

# --- Frontmatter helpers ---

# Extract body content (everything after the second ---)
extract_body() {
  local file="$1"
  awk 'BEGIN{n=0} /^---$/{n++; next} n>=2{print}' "$file"
}

# Read a scalar field from frontmatter, checking overrides first
# Usage: resolve_field src/agents/foo.md github description
resolve_field() {
  local file="$1" platform="$2" field="$3"
  local val

  # Check override first
  val=$(yq --front-matter=extract ".overrides.${platform}.${field} // \"\"" "$file")
  if [[ -n "$val" ]]; then
    echo "$val"
    return
  fi

  # Fall back to top-level
  yq --front-matter=extract ".${field} // \"\"" "$file"
}

# Resolve tool list for a platform
# If override exists: use verbatim. Otherwise: auto-map from canonical.
resolve_tools() {
  local file="$1" platform="$2"

  # Check for override
  local override_count
  override_count=$(yq --front-matter=extract \
    ".overrides.${platform}.tools | length" "$file" 2>/dev/null || echo "0")

  if [[ "$override_count" -gt 0 ]]; then
    yq --front-matter=extract ".overrides.${platform}.tools[]" "$file"
    return
  fi

  # No override — read canonical tools and map them
  local tools
  tools=$(yq --front-matter=extract '.tools[]' "$file" 2>/dev/null || true)
  [[ -z "$tools" ]] && return

  if [[ "$platform" == "claude" ]]; then
    echo "$tools"
  else
    local mapped_tools=""
    while IFS= read -r tool; do
      [[ -z "$tool" ]] && continue
      # Handle scoped tools: Bash(npm test) → execute(npm test)
      local base="${tool%%(*}"
      local scope="${tool#"$base"}"
      local mapped
      mapped=$(yq ".${platform}.\"${base}\" // \"\"" "$TOOL_MAP_CFG")
      # Skip tools with no mapping (platform doesn't support them)
      [[ -n "$mapped" ]] && mapped_tools+="${mapped}${scope}"$'\n'
    done <<< "$tools"
    # Deduplicate while preserving order
    echo -n "$mapped_tools" | awk '!seen[$0]++'
  fi
}

# --- Output helpers ---

# Build frontmatter YAML string for a platform from a source file
# Handles: field resolution, tool mapping, field dropping, extra field injection
build_frontmatter() {
  local file="$1" platform="$2" type="$3"

  # Get platform config
  local drop_fields extra_fields_json
  drop_fields=$(yq ".${platform}.drop_fields[]" "$PLATFORMS_CFG" 2>/dev/null || true)
  extra_fields_json=$(yq -o=json \
    ".${platform}.types.${type}.extra_fields // {}" "$PLATFORMS_CFG")

  # Get all top-level keys from source (excluding overrides)
  local fields
  fields=$(yq --front-matter=extract '. | keys | .[]' "$file" \
    | grep -v '^overrides$' | grep -v '^override$')

  # Also include override-only fields (fields in overrides.<platform> not at top level)
  local override_keys
  override_keys=$(yq --front-matter=extract \
    ".overrides.${platform} // {} | keys | .[]" "$file" 2>/dev/null || true)
  if [[ -n "$override_keys" ]]; then
    while IFS= read -r okey; do
      [[ -z "$okey" ]] && continue
      if ! echo "$fields" | grep -qx "$okey"; then
        fields+=$'\n'"$okey"
      fi
    done <<< "$override_keys"
  fi

  local output=""

  # Inject extra fields first
  if [[ "$extra_fields_json" != "{}" && "$extra_fields_json" != "null" ]]; then
    local extra_keys
    extra_keys=$(echo "$extra_fields_json" | yq '. | keys | .[]')
    while IFS= read -r ekey; do
      local eval_val
      eval_val=$(echo "$extra_fields_json" | yq ".${ekey}")
      output+="${ekey}: ${eval_val}"$'\n'
    done <<< "$extra_keys"
  fi

  # Process each field
  while IFS= read -r field; do
    [[ -z "$field" ]] && continue

    # Skip if already injected as extra field
    if [[ "$extra_fields_json" != "{}" && "$extra_fields_json" != "null" ]]; then
      echo "$extra_fields_json" | yq -e ".${field}" &>/dev/null && continue || true
    fi

    # Check if field should be dropped (unless overridden)
    if echo "$drop_fields" | grep -qx "$field"; then
      # Check if there's an override for this field
      local has_override
      has_override=$(yq --front-matter=extract \
        ".overrides.${platform}.${field} // \"\"" "$file")
      [[ -z "$has_override" ]] && continue
    fi

    # Resolve the field value
    if [[ "$field" == "tools" ]]; then
      local tools
      tools=$(resolve_tools "$file" "$platform")
      if [[ -n "$tools" ]]; then
        output+="tools:"$'\n'
        while IFS= read -r t; do
          [[ -n "$t" ]] && output+="  - ${t}"$'\n'
        done <<< "$tools"
      fi
    else
      local val
      val=$(resolve_field "$file" "$platform" "$field")
      if [[ -n "$val" ]]; then
        if [[ "$val" == -* ]]; then
          # Value is a YAML sequence — emit field name then indented items
          output+="${field}:"$'\n'
          while IFS= read -r item; do
            [[ -n "$item" ]] && output+="  ${item}"$'\n'
          done <<< "$val"
        else
          output+="${field}: ${val}"$'\n'
        fi
      fi
    fi
  done <<< "$fields"

  echo -n "$output"
}

# Write frontmatter + body to output file
write_output() {
  local output_file="$1" frontmatter="$2" body="$3"

  mkdir -p "$(dirname "$output_file")"

  {
    echo "---"
    printf '%s\n' "$frontmatter"
    echo "---"
    echo "$body"
  } > "$output_file"
}

# --- Main transpile function ---

# Transpile a single source type for a single platform
# Usage: transpile_type agents github
transpile_type() {
  local type="$1" platform="$2"
  local src_dir="$SRC_DIR/${type}"

  [[ ! -d "$src_dir" ]] && return

  # Read platform config for this type
  local out_path out_suffix target_root out_dir
  out_path=$(yq ".${platform}.types.${type}.path" "$PLATFORMS_CFG")
  out_suffix=$(yq ".${platform}.types.${type}.suffix" "$PLATFORMS_CFG")

  [[ "$out_path" == "null" ]] && return

  # Resolve output root from targets config (absolute or relative to ROOT_DIR)
  target_root=$(yq ".${platform} // \".\"" "$TARGETS_CFG")
  [[ "$target_root" == "null" || -z "$target_root" ]] && target_root="."
  if [[ "$target_root" != /* ]]; then
    target_root="$ROOT_DIR/$target_root"
  fi
  out_dir="$target_root/$out_path"

  while IFS= read -r src_file; do
    [[ ! -f "$src_file" ]] && continue

    # Compute relative path from src_dir (e.g. "linear/SKILL.md" or "reviewer.md")
    local rel_path filename base dest_file
    rel_path="${src_file#"$src_dir/"}"
    filename=$(basename "$src_file")
    # For flat files: apply suffix. For subdirectory files: preserve path as-is.
    if [[ "$rel_path" == */* ]]; then
      dest_file="${out_dir}/${rel_path}"
    else
      base="${filename%.md}"
      dest_file="${out_dir}/${base}${out_suffix}"
    fi

    # If no frontmatter, copy file as-is
    if ! yq --front-matter=extract '.' "$src_file" &>/dev/null || \
       [[ "$(yq --no-doc --front-matter=extract '. | length' "$src_file" 2>/dev/null)" == "0" ]]; then
      mkdir -p "$(dirname "$dest_file")"
      cp "$src_file" "$dest_file"
      continue
    fi

    local fm body
    fm=$(build_frontmatter "$src_file" "$platform" "$type")
    body=$(extract_body "$src_file")

    write_output "$dest_file" "$fm" "$body"
  done < <(find "$src_dir" -name '*.md' | sort)
}

# Transpile all types for all platforms
transpile_all() {
  local platforms types

  platforms=$(yq '. | keys | .[]' "$PLATFORMS_CFG")
  types="agents rules commands skills"

  while IFS= read -r platform; do
    [[ -z "$platform" ]] && continue
    echo "[${platform}]"
    for type in $types; do
      transpile_type "$type" "$platform"
      # Count files processed
      local count
      count=$(find "$SRC_DIR/${type}" -name '*.md' 2>/dev/null | wc -l | tr -d ' ')
      [[ "$count" -gt 0 ]] && echo "  ${type}: ${count} file(s)"
    done
  done <<< "$platforms"
}
