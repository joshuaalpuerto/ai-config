# Static Codebase Analysis

`aicfg analyze` runs a fully deterministic, LLM-free analysis of a codebase and produces a structured report. It supports two analysis kinds selected via `--kind`:

- **`code`** (default) — detects the tech stack, maps the import graph, surfaces the most important files, identifies hotspots, and groups files into logical clusters.
- **`doc`** — scans documentation files and reports how recently each was updated, using git history to compute per-file staleness.

Both kinds operate without network requests or reading beyond the source files and git history.

---

## Usage

```bash
aicfg analyze <directory> [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--kind` | `code` | Analysis kind: `code` (default) or `doc` |
| `--format` | `md` | Output format: `md` (LLM-ready markdown, default) or `json` (machine-readable) |
| `--output` | stdout | Write the report to a file (without extension) — the extension is appended automatically based on `--format` |
| `--since` | `6 months ago` | Git history window used for churn analysis (`--kind=code` only) |
| `--hubs` | `10` | Number of hub files to include in the report (`--kind=code` only) |
| `--hotspots` | `20` | Number of hotspot files to include in the report (`--kind=code` only) |
| `--verbose` | `false` | Include full per-file metrics in the JSON output (`--kind=code` only) |
| `--cache` | `false` | Cache the result in `.aicfg-cache.json`; reuse it on unchanged codebases (`--kind=code` only) |

### Examples

```bash
# Analyze the current directory and print markdown to stdout (default)
aicfg analyze .

# Analyze a specific repo and save the report as reports.md
aicfg analyze ~/projects/myapp --output reports

# Save the report as JSON
aicfg analyze ~/projects/myapp --output reports --format json

# Expand the git history window and show more hotspots
aicfg analyze . --since "1 year ago" --hotspots 30

# Full per-file detail with caching enabled
aicfg analyze . --verbose --cache

# Analyze documentation freshness (uses doc_audit.paths from aicfg.yaml, or auto-detects docs/ + README.md)
aicfg analyze . --kind=doc

# Save doc freshness report as JSON
aicfg analyze ~/projects/myapp --kind=doc --format json --output doc-report
```

---

## What It Analyzes

### `--kind=code` (default)

The code analyzer runs five sequential phases against the target directory.

### Phase 1 — Tech Stack Scanning

Detects languages and frameworks by reading well-known marker files (not file extensions or source content). Skips irrelevant directories entirely (`node_modules`, `vendor`, `dist`, `build`, `.git`, etc.).

Additional paths can be excluded via `analyze_exclude` in `aicfg.yaml` — see [aicfg-config.md](aicfg-config.md#analyze_exclude) for pattern syntax and examples.

**Supported marker files:**

| File | Language | Framework signals parsed |
|---|---|---|
| `go.mod` | Go | Module path extracted |
| `package.json` | JS/TS | `next`, `express`, `react`, `vue`, `vite` in `dependencies`/`devDependencies` |
| `requirements.txt` / `pyproject.toml` | Python | `fastapi`, `django`, `flask` |

Source files collected for graph analysis: `.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`.

### Phase 2 — Import Parsing

Reads every source file and extracts its imports and exports. Path aliases defined in `tsconfig.json` (`compilerOptions.paths`) are resolved to real paths. External dependencies (e.g., `react`, `fmt`) are discarded; only intra-repo edges are kept.

### Phase 3 — Import Graph Construction

Builds an adjacency graph from the resolved imports. For each file, the graph tracks:

- **Fan-in** — how many other files import this file
- **Fan-out** — how many files this file imports
- **Export count** — number of exported symbols
- **Priority** — a weighted score used to rank files by structural importance (see [Priority Formula](#priority-formula))

The top-N files by priority are reported as **Hubs**.

### Phase 4 — Git Churn Analysis

Runs `git log` over the configured time window and counts how many commits touched each file. If the directory is not a git repository, this phase is skipped gracefully and `gitChurnAvailable` is set to `false`.

Files with `churn > 3` AND `lines > 50` are candidates for the **Hotspots** list, ranked by `churn × lines`.

### Phase 5 — Domain Clustering

Treats the import graph as undirected and finds all connected components via BFS. Each component becomes a **Cluster**. The cluster's label is derived from the most common top-level directory across its files (e.g., if most files in a component are under `src/billing/`, the cluster is labeled `billing`). Inter-cluster dependencies are also computed.

### `--kind=doc`

The doc freshness analyzer runs two phases:

**Phase 1 — File Collection**

Walks each path in `doc_audit.paths` (from `aicfg.yaml`) and collects all `.md` files. Paths may be files or directories; relative paths are resolved against the analyzed root. If `doc_audit.paths` is not set, auto-detects `docs/` and `README.md` at the root.

**Phase 2 — Last-Updated Lookup**

Runs a single `git log` invocation and records the most recent commit timestamp for each collected doc file. If the directory is not a git repository, this phase is skipped gracefully and `gitChurnAvailable` is set to `false`.

Results are sorted stalest-first (highest `daysSinceUpdate` first). Files with no git history entry have `daysSinceUpdate: -1` and sort to the top.

---

## Output Reference

### `--kind=code` JSON output (`--format json`)

```jsonc
{
  "root": "/absolute/path/to/repo",
  "analyzedAt": "2026-04-19T10:00:00Z",
  "gitChurnAvailable": true,
  "techStack": {
    "languages": ["go"],
    "frameworks": []
  },
  "topLevelDirs": ["cmd", "internal", "docs"],
  "sourceFiles": [...],
  "hubs": [...],
  "hotspots": [...],
  "clusters": [...],
  "files": {...}   // only present when --verbose is set
}
```

#### Top-level fields

| Field | Type | Description |
|---|---|---|
| `root` | `string` | Absolute path to the analyzed directory |
| `analyzedAt` | `string` (ISO 8601) | Timestamp when the analysis ran |
| `gitChurnAvailable` | `bool` | Whether git history was accessible; `false` means hotspot data is absent |
| `techStack` | `object` | Detected languages and frameworks |
| `topLevelDirs` | `string[]` | Top-level source directories found (used to orient reading the report) |
| `sourceFiles` | `string[]` | All analyzed source file paths, repo-relative |
| `hubs` | `Hub[]` | Most structurally important files, sorted by priority descending |
| `hotspots` | `Hotspot[]` | Most frequently changed large files, sorted by churn×lines descending |
| `clusters` | `Cluster[]` | Logical feature groups, sorted by size descending |
| `files` | `map[string]FileNode` | Per-file detail map — only emitted with `--verbose` |

---

### `Hub`

A hub is a file with high structural importance in the import graph. These are the files the rest of the codebase depends on.

```jsonc
{
  "path": "internal/lib/db.go",
  "fanIn": 14,
  "fanOut": 3,
  "priority": 48.2,
  "exportNames": ["Connect", "Query", "Close"]
}
```

| Field | Description |
|---|---|
| `path` | Repo-relative file path |
| `fanIn` | Number of other files that import this file. High fan-in means the file is a dependency of many others — changing it has wide blast radius. |
| `fanOut` | Number of files this file imports. High fan-out means the file pulls in many dependencies. |
| `priority` | Weighted importance score (see [Priority Formula](#priority-formula)). Higher = more important. |
| `exportNames` | Named exports/symbols found in the file (may be omitted if none detected) |

**How to read it:** Hub files are your core abstractions. They are the first place to look when understanding the codebase, and the highest-risk files to modify.

---

### `Hotspot`

A hotspot is a large file that changes often. These indicate complexity, instability, or active development.

```jsonc
{
  "path": "src/api/payment-webhook.ts",
  "churn": 18,
  "lines": 320,
  "score": 5760
}
```

| Field | Description |
|---|---|
| `path` | Repo-relative file path |
| `churn` | Number of commits that touched this file within the `--since` window |
| `lines` | Current line count of the file |
| `score` | `churn × lines` — used for ranking. Combines frequency of change with size to surface complex, unstable files. |

**How to read it:** Hotspots are your highest-risk files for bugs and tech debt. They are also the best candidates for refactoring, documentation, or additional test coverage.

---

### `Cluster`

A cluster is a group of files that are directly or transitively connected in the import graph, treated as a logical feature or domain.

```jsonc
{
  "label": "billing",
  "size": 12,
  "singleton": false,
  "dependsOn": ["auth", "db"],
  "files": [
    "src/billing/invoice.ts",
    "src/billing/subscription.ts",
    "src/utils/currency.ts"
  ]
}
```

| Field | Description |
|---|---|
| `label` | Human-readable name derived from the dominant folder in the cluster |
| `size` | Number of files in the cluster |
| `singleton` | `true` if the cluster contains only one file (no import connections to other files) |
| `dependsOn` | Labels of other clusters that this cluster imports from, indicating cross-domain dependencies |
| `files` | Sorted list of repo-relative file paths in the cluster |

**How to read it:** Clusters reveal the real feature boundaries of your codebase, which may not match the folder structure. `dependsOn` shows which domains are tightly coupled. Large clusters with many dependencies are the highest-complexity areas.

---

### `--kind=doc` JSON output (`--format json`)

```jsonc
{
  "root": "/absolute/path/to/repo",
  "analyzedAt": "2026-05-02T10:00:00Z",
  "gitChurnAvailable": true,
  "docRoots": ["docs/", "README.md"],
  "docFiles": [
    {
      "path": "docs/hooks.md",
      "lastUpdated": "2026-04-12T07:23:25Z",
      "daysSinceUpdate": 20
    },
    {
      "path": "README.md",
      "lastUpdated": "2026-04-26T06:38:32Z",
      "daysSinceUpdate": 6
    }
  ]
}
```

#### Top-level fields

| Field | Type | Description |
|---|---|---|
| `root` | `string` | Absolute path to the analyzed directory |
| `analyzedAt` | `string` (ISO 8601) | Timestamp when the analysis ran |
| `gitChurnAvailable` | `bool` | Whether git history was accessible; `false` means `lastUpdated` and `daysSinceUpdate` are absent |
| `docRoots` | `string[]` | Doc roots that were scanned, repo-relative |
| `docFiles` | `DocFile[]` | All doc files found, sorted stalest-first |

### `DocFile`

| Field | Type | Description |
|---|---|---|
| `path` | `string` | Repo-relative file path |
| `lastUpdated` | `string` (ISO 8601) | Timestamp of the most recent commit that touched this file. Omitted when git is unavailable. |
| `daysSinceUpdate` | `int` | Days elapsed since `lastUpdated`. `-1` when git is unavailable or the file has no commit history. |

---

### `FileNode` (verbose only)

Emitted per file when `--verbose` is set. Provides the raw graph metrics used to compute hubs and priorities.

```jsonc
{
  "imports": ["internal/lib/db.go", "internal/utils/hash.go"],
  "importedBy": ["cmd/api/main.go", "internal/auth/login.go"],
  "lines": 150,
  "exportCount": 5,
  "exportNames": ["HashPassword", "CompareHash"],
  "churn": 7,
  "fanIn": 2,
  "fanOut": 2,
  "isEntryPoint": false,
  "priority": 22.5,
  "folderDepth": 2
}
```

| Field | Description |
|---|---|
| `imports` | Resolved intra-repo paths this file imports |
| `importedBy` | Files that import this file (reverse edges) |
| `lines` | Total line count |
| `exportCount` | Number of exported symbols detected |
| `exportNames` | Names of exported symbols |
| `churn` | Commit count from git history (0 if git unavailable) |
| `fanIn` | `len(importedBy)` |
| `fanOut` | `len(imports)` |
| `isEntryPoint` | `true` for recognized entry files (`main.go`, `index.ts`, etc.) |
| `priority` | Final weighted score |
| `folderDepth` | Nesting depth of the file (number of `/` separators in path) |

---

### Priority Formula

Files are ranked using a weighted formula that rewards structural centrality and active modification:

$$
\text{priority} = (\text{fanIn} \times 3.0) + (\text{exportCount} \times 1.5) + (\text{entryBonus}) + (\text{churn} \times 2.0) + (\text{fanOut} \times 0.5) + \frac{1}{\text{folderDepth} + 1}
$$

Where `entryBonus` is `10.0` for recognized entry points and `0` otherwise.

**Rationale:**
- `fanIn × 3.0` — the largest weight: files depended on by many others are the most critical
- `exportCount × 1.5` — files with more public API surface are more architecturally significant
- `entryBonus` — entry points (e.g., `main.go`, `index.ts`) are always structurally important
- `churn × 2.0` — frequently changed files are likely complex or central to the domain
- `fanOut × 0.5` — small bonus for files that coordinate many dependencies
- `1 / (folderDepth + 1)` — files closer to the root get a slight boost

---

## Markdown Format (`--format md`, default)

When `--format md` is used (or by default), the analyzer emits a compact markdown summary instead of JSON. Both kinds are designed to be pasted directly into an LLM prompt as codebase context.

### `--kind=code` markdown

```markdown
## Codebase: myapp
**Analyzed:** 2026-04-19

**Stack:** go

**Source folders:** cmd, internal, src

## Structure
cmd/
  aicfg/
    main.go
internal/
  billing/
    invoice.go
    subscription.go
  db/
    db.go
src/
  api/
    payment-webhook.ts

## Hub Files (most depended-on)
1. `internal/lib/db.go` — imported by 14 files. Exports: Connect, Query, Close
2. `internal/middleware/auth.go` — imported by 9 files. Exports: Authenticate, RequireRole

## Hotspots (high churn + size)
- `src/api/payment-webhook.ts` — changed 18x, 320 lines
- `internal/billing/invoice.go` — changed 11x, 210 lines
```

### `--kind=doc` markdown

```markdown
## Doc Freshness: myapp
**Analyzed:** 2026-05-02

**Doc roots:** docs/, README.md

| File | Last Updated | Days Since Update |
|------|-------------|-------------------|
| `docs/hooks.md` | 2026-04-12 | 20 |
| `docs/aicfg-config.md` | 2026-04-26 | 6 |
| `README.md` | 2026-04-26 | 6 |
```

Rows are sorted stalest-first. When git history is unavailable, the table is replaced with a note and no dates are shown.

---

## Caching

With `--cache`, the analyzer computes a fingerprint (SHA-256 of all source file paths and sizes) and stores the result in `.aicfg-cache.json` at the root of the analyzed directory. On subsequent runs, if the fingerprint matches (no files added, removed, or resized), the cached result is returned immediately.

Add `.aicfg-cache.json` to `.gitignore` if you don't want it committed.

---

## Supported Languages

| Language | Import parsing | Export detection | Path alias resolution |
|---|---|---|---|
| Go | Yes | Yes (`func`, `type`, `var`, `const`) | Via `go.mod` module path |
| TypeScript / TSX | Yes | Yes (`export`, `export default`) | Via `tsconfig.json` `paths` |
| JavaScript / JSX | Yes | Yes | Via `tsconfig.json` `paths` |
| Python | Yes (regex) | No | No |
