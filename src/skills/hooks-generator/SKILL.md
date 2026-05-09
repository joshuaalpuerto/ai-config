---
name: hooks-generator
description: Generate a hooks.yaml policy file for ai-config — contextual reminders injected when contributors (human or AI) work in areas where missing context would cause incorrect, non-compiling, or insecure output.
allowed-tools:
  - Bash
  - Read
  - Write
  - Edit
  - Agent
---

# Hooks Generator: ai-config

Generate a `hooks.yaml` that injects contextual guidance precisely when needed — so contributors (human or AI) working in an area get exactly the conventions, contracts, and constraints required to produce correct output on the first attempt.

## Selection Principle

A hook rule is justified **only** when working in an area without the injected context would likely cause a contributor to produce incorrect, non-compiling, or insecure output. Do **not** create rules for:
- Style preferences
- Nice-to-know information
- Areas where the code itself is self-documenting

## Project Configuration

- **Target directory:** `/workspaces/ai-config`
- **Doc corpus:** `docs/`, `README.md`
- **Analyze excludes:** `.claude`, `.github`
- **Existing hooks file:** `hooks.yaml`

## Process

---

### Step 1 — Obtain doc-audit findings

A doc-audit report is a **required input** to this skill. It surfaces contributor blockers, undocumented contracts, complexity traps, and dependency conventions that directly determine which hook rules to generate.

- **If a doc-audit report was provided as input:** read it and proceed to Step 2.
- **If no doc-audit report was provided:** invoke the **doc-audit** skill now before continuing. Do not skip this step or substitute a lightweight analysis. Once the doc-audit completes, use its full report and proceed to Step 2.

---

### Step 2 — Gather additional context

Run these in parallel:

1. **Read existing hooks.yaml** (if `hooks.yaml` exists) — preserve all existing rules.
2. **Read dependency manifests** for the project's ecosystem(s):

| Ecosystem | Files |
|-----------|-------|
| JavaScript / TypeScript | `package.json` |
| Go | `go.mod`, `Makefile` |
| Python | `pyproject.toml`, `requirements.txt` |
| Ruby | `Gemfile` |
| Rust | `Cargo.toml` |

3. **Run `aicfg analyze /workspaces/ai-config`** (if not already run in Step 1) for hubs, hotspots, clusters.

---

### Step 3 — Select findings that warrant hook rules

From the doc-audit report, select findings using this priority hierarchy:

| Priority | Category | Criteria |
|----------|----------|----------|
| **Highest** | Contributor Blockers | Misuse causes runtime failures or security gaps |
| **High** | Undocumented Contracts | Compiler does not enforce; silent failure if missed |
| **Medium** | Complexity Traps | High churn + non-obvious ordering/edge cases |
| **Lower** | Undocumented Dependency Conventions | Consolidate into one rule per path pattern |
| **Skip** | Superseded Records | No hook needed — these are doc annotations |
| **Skip** | Trivial doc updates | Not worth a runtime rule |

**Consolidation rule:** multiple findings that share the same `paths` pattern (e.g. all frontend wrapper conventions applying to `*.ts`/`*.tsx`) become **one rule** with a consolidated `inject_inline` block. Do not create separate rules for each finding.

---

### Step 4 — Decide inject strategy per rule

For each selected finding:

```
Does an existing doc adequately cover this convention?
├── YES → Use action: inject: "path/to/doc.md"
└── NO → Does a doc exist but with gaps?
      ├── YES → Update the doc with missing content, then use inject: "path/to/doc.md"
      └── NO (no doc exists) → Use action: inject_inline with full convention block
```

**inject_inline format guidelines:**
- Keep under 10 lines
- Use a bulleted checklist of specific do/don't items
- Name specific wrapper files, functions, components
- State what the contributor must use AND what they must avoid

Example:
```yaml
inject_inline: |
  ## Required project wrappers (frontend/src/libs/)
  - API calls: use Orval-generated hooks, never import axios directly
  - Dates: use libs/formatters/date.ts, not date-fns
  - Logging: use logger from libs/errorTracking.ts, not datadogLogs
  - Validation: use parseResponse() from libs/validators/zod.ts
  - Forms: use Form* components, not raw MUI TextField/Select
```

---

### Step 5 — Detect PostToolUse lint commands

Read dependency manifests and project tooling to detect available lint commands:

| Signal | Lint command |
|--------|-------------|
| `package.json` has `scripts.lint` | `npm run lint -- --no-error-on-unmatched` or detected tool (biome, eslint) |
| `Makefile` has `lint` target | `make lint` |
| `pyproject.toml` has ruff/flake8 | `ruff check` |
| `.golangci.yml` exists | `golangci-lint run` |

Add a `PostToolUse` rule with `run_inline` for each detected linter, scoped to the appropriate file extensions.

---

### Step 6 — Merge and produce output

1. Start with all **existing rules** from `hooks.yaml` (preserved exactly).
2. Append new rules with a YAML comment annotation per rule:
   ```yaml
   # Source: Contributor Blocker #1 — frontend/src/libs/
   ```
3. If you updated existing docs in Step 4, list the changes made.

---

### Step 7 — Present the result

Output the complete `hooks.yaml` in a fenced YAML code block. Include:

- `version: "1"` header
- All PreToolUse rules (existing + new, annotated)
- All PostToolUse rules (existing + new, annotated)
- YAML comments explaining each new rule's origin and what it prevents

After the code block, include a brief summary table:

```
| # | Rule | Event | Action | Source |
|---|------|-------|--------|--------|
| 1 | ... | PreToolUse | inject_inline | Contributor Blocker #1 |
| 2 | ... | PostToolUse | run_inline | Lint detection |
```

End with:
> **To apply:** copy the YAML above into your `hooks.yaml` at the project root, then run `aicfg build` to deploy.
>
> **Not satisfied with doc edits?** Run the doc-audit skill to see the full list of documentation improvement opportunities.

## hooks.yaml Reference

Rules follow this structure:

```yaml
version: "1"

PreToolUse:
  - match:
      tools: ["Edit", "Write"]       # Canonical: Bash, Read, Write, Edit, WebFetch
      paths: ["*.py", "**/*.py"]     # Glob patterns (filename-only or full path)
    action:
      inject: "./docs/PYTHON.md"     # Path to doc file (relative to repo root)

  - match:
      tools: ["Edit", "Write"]
      paths: ["**/src/api/**"]
    action:
      inject_inline: |               # Inline text injected into context
        Use the generated API client, never write fetch calls manually.

PostToolUse:
  - match:
      tools: ["Write", "Edit"]
      paths: ["*.ts", "*.tsx"]
    action:
      run_inline: "npx biome check"  # Exit 0 = inject stdout; non-zero = block/warn
```

**Path pattern forms:**
- `*.ext` — matches filename only (any directory)
- `dir/` — matches any file inside that directory
- `**/dir/**/*.ext` — full path glob (`**` = zero or more segments)

**Action types:**
- `inject`: read file at path, inject contents into context
- `inject_inline`: inject literal text into context
- `run_inline`: execute shell command (event JSON on stdin)
- `block: true` + `message`: prevent tool execution with reason
