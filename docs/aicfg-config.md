# aicfg.yaml Reference

`aicfg.yaml` is the unified configuration file placed in your project root. It tells the transpiler where your source files live, how to map output for each platform, and which tools to remap or drop.

## Top-level fields

| Field | Required | Description |
|-------|----------|-------------|
| `src_dir` | ✓ | Source directory. Relative to repo root or absolute. |
| `src_hooks_file` | — | Path to the hooks definition file. Relative to repo root. |
| `analyze_exclude` | — | Glob patterns for paths to exclude from `aicfg analyze`. |
| `platforms` | ✓ | Output configuration keyed by platform name. |
| `tool_map` | ✓ | Canonical tool name → platform-specific name mappings. |

---

## `src_dir`

```yaml
src_dir: src
```

Points to the root of your source definitions. Relative paths are resolved from the repo root. To use a different source directory, change this value — the binary resolves it at startup.

---

## `src_hooks_file`

```yaml
src_hooks_file: hooks.yaml
```

Points to the hooks definition file. Relative paths are resolved from the repo root. When set, the hooks engine reads rules from this file at runtime. If omitted, hook evaluation is skipped.

---

## `analyze_exclude`

```yaml
analyze_exclude:
  - .claude
  - .github
  - docs/*.md
```

A list of gitignore-style glob patterns for paths to skip when running `aicfg analyze`. Patterns are matched against every file and directory encountered during the scan:

- **Patterns without `/`** are matched against the entry name only (e.g. `.github` skips any directory named `.github` anywhere in the tree).
- **Patterns with `/`** are matched against the path relative to the analyzed root (e.g. `docs/*.md` skips only markdown files directly inside `docs/`).

Directories that match are skipped entirely (their contents are not walked). Files that match are excluded from both the file tree and source graph analysis.

Always-excluded directories (hardcoded, not configurable): `.git`, `node_modules`, `vendor`, `dist`, `build`, `bin`, `__pycache__`, `.next`, `coverage`.

---

## `platforms`

Each key is a platform name (e.g. `claude`, `github`). Platform names must match `^[a-z][a-z0-9_-]*$`.

### Per-platform fields

| Field | Required | Description |
|-------|----------|--------------|
| `target` | ✓ | Output root directory. Relative to repo root or absolute. |
| `types` | ✓ | Per-type output config. Must define all four: `agents`, `commands`, `rules`, `skills`. |
| `drop_fields` | — | Frontmatter fields to omit for this platform. |

### `types`

Each of `agents`, `commands`, `rules`, and `skills` must be defined:

| Field | Required | Description |
|-------|----------|-------------|
| `path` | ✓ | Output directory path, relative to `target`. |
| `suffix` | ✓ | File suffix appended to output files (e.g. `.md`, `.agent.md`, `.prompt.md`). |
| `extra_fields` | — | Additional frontmatter key/value pairs injected verbatim for this type. |

### `drop_fields`

An array of canonical frontmatter field names to omit when writing output for this platform. A field can still appear in the output if a per-file `overrides.<platform>.<field>` is set.

Droppable field names: `name`, `description`, `model`, `context`, `agent`, `path`, `applyTo`, `argument-hint`, `disable-model-invocation`, `tools`, `allowed-tools`, `paths`.

---

## `tool_map`

Maps canonical tool names to platform-specific equivalents. If a canonical tool is absent from a platform's map entry, it is **dropped** from output for that platform.

Canonical tool names:

| Name | Description |
|------|-------------|
| `Bash` | Shell command execution |
| `Read` | File read |
| `Write` | File write |
| `Edit` | File edit/patch |
| `Glob` | File glob search |
| `Grep` | Content search |
| `Agent` | Sub-agent invocation |
| `WebSearch` | Web search |
| `WebFetch` | Web fetch/request |
| `TaskCreate` | Create a task/todo |
| `TaskGet` | Get a task/todo |
| `TaskList` | List tasks/todos |
| `TaskUpdate` | Update a task/todo |

---

## Complete example

The following is the `aicfg.yaml` used in this repository:

```yaml
src_dir: src
src_hooks_file: hooks.yaml
analyze_exclude:
  - .claude
  - .github

platforms:
  # Claude Code — canonical platform
  claude:
    target: .
    types:
      agents:
        path: .claude/agents
        suffix: .md
      rules:
        path: .claude/rules
        suffix: .md
      commands:
        path: .claude/commands
        suffix: .md
      skills:
        path: .claude/skills
        suffix: .md
    drop_fields:
      - applyTo

  # GitHub Copilot
  github:
    target: .
    types:
      agents:
        path: .github/agents
        suffix: .agent.md
      rules:
        path: .github/instructions       # GitHub uses "instructions", not "rules"
        suffix: .instructions.md
      commands:
        path: .github/prompts            # GitHub uses "prompts", not "commands"
        suffix: .prompt.md
        extra_fields:
          agent: agent
      skills:
        path: .github/skills
        suffix: .md
    drop_fields:
      - model
      - context
      - agent
      - disable-model-invocation
      - paths
      - path

tool_map:
  github:
    Bash: execute
    Read: read
    Write: edit
    Edit: edit
    Glob: search
    Grep: search
    Agent: agent
    WebSearch: web/fetch
    TaskCreate: todo
    TaskGet: todo
    TaskList: todo
    TaskUpdate: todo
    # Tools with no mapping are dropped (e.g. WebFetch has no GitHub equivalent)
```
