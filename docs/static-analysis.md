# Static Codebase Analysis

`aicfg analyze` runs a fully deterministic, LLM-free analysis of a codebase and produces a structured report. It detects the tech stack, maps the import graph, surfaces the most important files, identifies hotspots, and groups files into logical clusters — all without making any network requests or reading beyond the source files and git history.

---

## Usage

```bash
aicfg analyze <directory> [flags]
```

### Flags

| Flag | Default | Description |
|---|---|---|
| `--format` | `json` | Output format: `json` (machine-readable) or `context` (LLM-ready markdown) |
| `--output` | stdout | Write the report to a file instead of printing it |
| `--since` | `6 months ago` | Git history window used for churn analysis |
| `--hubs` | `10` | Number of hub files to include in the report |
| `--hotspots` | `20` | Number of hotspot files to include in the report |
| `--verbose` | `false` | Include full per-file metrics in the JSON output |
| `--cache` | `false` | Cache the result in `.aicfg-cache.json`; reuse it on unchanged codebases |

### Examples

```bash
# Analyze the current directory and print JSON to stdout
aicfg analyze .

# Analyze a specific repo and save the report
aicfg analyze ~/projects/myapp --output report.json

# Get LLM-ready markdown summary
aicfg analyze . --format context

# Expand the git history window and show more hotspots
aicfg analyze . --since "1 year ago" --hotspots 30

# Full per-file detail with caching enabled
aicfg analyze . --verbose --cache
```

---

## What It Analyzes

The analyzer runs five sequential phases against the target directory.

### Phase 1 — Tech Stack Scanning

Detects languages and frameworks by reading well-known marker files (not file extensions or source content). Skips irrelevant directories entirely (`node_modules`, `vendor`, `dist`, `build`, `.git`, etc.).

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

---

## Output Reference

### JSON output (default)

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

## Context Format (`--format context`)

When `--format context` is used, the analyzer emits a compact markdown summary instead of JSON. This is designed to be pasted directly into an LLM prompt as codebase context.

```markdown
## Codebase: myapp
**Analyzed:** 2026-04-19

**Stack:** go

**Source folders:** cmd, internal, src

## Hub Files (most depended-on)
1. `internal/lib/db.go` — imported by 14 files. Exports: Connect, Query, Close
2. `internal/middleware/auth.go` — imported by 9 files. Exports: Authenticate, RequireRole

## Feature Clusters
- **billing** (12 files) → depends on: auth, db
- **auth** (8 files) → depends on: db
- **db** (3 files)
- **misc** (5 isolated files)

## Hotspots (high churn + size)
- `src/api/payment-webhook.ts` — changed 18x, 320 lines
- `internal/billing/invoice.go` — changed 11x, 210 lines
```

Typical output is under 300 tokens, making it suitable as a system prompt prefix.

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
