#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
errors=0

for src_file in "$ROOT_DIR"/src/{agents,rules,commands,skills}/*.md; do
  [[ ! -f "$src_file" ]] && continue

  # Validate frontmatter is parseable by yq
  if ! yq --front-matter=extract '.' "$src_file" &>/dev/null; then
    echo "ERROR: $(basename "$src_file") — invalid frontmatter YAML"
    ((errors++))
    continue
  fi

  # Check required 'description' field
  desc=$(yq --front-matter=extract '.description // ""' "$src_file")
  if [[ -z "$desc" ]]; then
    echo "WARN: $(basename "$src_file") — missing 'description' field"
  fi
done

if [[ $errors -gt 0 ]]; then
  echo ""
  echo "$errors validation error(s) found."
  exit 1
else
  echo "All source files valid."
fi
