# Hooks Reference (`hooks.yaml`)

Hook rules enforce policies on every AI action. Define them in `hooks.yaml` at the repo root and point `aicfg.yaml` to it with `src_hooks_file`.

## Top-level structure

```yaml
version: "1"

settings:           # optional
  fail_open: true   # default: true

PreToolUse:
  - # rules evaluated before each tool call

PostToolUse:
  - # rules evaluated after each tool call
```

| Field | Description |
|-------|-------------|
| `version` | Required. Must be `"1"`. |
| `settings` | Optional global settings (see [Settings](#settings)). |
| `PreToolUse` | Rules evaluated before a tool executes. |
| `PostToolUse` | Rules evaluated after a tool completes. |

## Policy modes

Each rule has an optional `mode` field (default: `enforce`):

| Mode | Behaviour |
|------|-----------|
| `enforce` | Blocking rules stop the action. Inject/run rules are applied normally. |
| `warn` | Same as enforce but surfaces a warning instead of hard-blocking. |
| `audit` | Rule is evaluated and logged only; never blocks or modifies the action. |

## Rule structure

```yaml
- mode: enforce           # optional, default: enforce
  match:                  # required
    tools: [...]
    command_match: "..."
    paths: [...]
  action:                 # required
    block: true/false
    message: "..."
    inject: "..."
    inject_inline: "..."
    run: "..."
    run_inline: "..."
```

## Matchers

All matcher fields are optional. When multiple fields are present they are ANDed — the rule fires only when **all** specified matchers match.

| Field | Type | Description |
|-------|------|-------------|
| `tools` | `string[]` | Canonical tool names: `Bash`, `Read`, `Write`, `Edit`, `WebFetch`. |
| `command_match` | `string` | Regex matched against `tool_input.command` (applies to `Bash` tool only). |
| `paths` | `string[]` | Glob patterns matched against the file path. See [Path patterns](#path-patterns) below. |

### Path patterns

`paths` supports three pattern forms:

| Form | Example | Matches |
|------|---------|---------|
| No path separator | `*.go`, `*-suffix.tsx` | Filename only, regardless of directory |
| Trailing `/` | `src/api/`, `src/**/nested/` | Any file inside the matched directory |
| Full path glob | `**/src/**/*.ts`, `**/tasks/**/research.md` | Full file path; `**` matches zero or more path segments |

> **Important:** Full-path glob patterns are matched against the **absolute file path** on disk (e.g. `/home/user/project/src/api/routes.ts`). Always prefix them with `**/` so the pattern matches regardless of where the project is located (e.g. `**/src/api/**` not `src/api/**`).

## Actions

At least one action field should be set. Multiple action fields can be combined in a single rule.

| Field | Type | Description |
|-------|------|-------------|
| `block` | `boolean` | When `true`, prevents the tool from executing. |
| `message` | `string` | Human-readable reason shown to the developer when a rule blocks. |
| `inject` | `string` | Path to a Markdown file whose contents are injected into context. |
| `inject_inline` | `string` | Inline text injected directly into context (no external file needed). |
| `run` | `string` | Path to a validator script. The event JSON is passed on stdin. |
| `run_inline` | `string` | Inline shell command executed via `sh -c`. The event JSON is passed on stdin. Exit 0 injects stdout as context; non-zero exit blocks (or warns). |

### Inject path resolution

The `inject` field accepts a workspace-relative path. The file is read and its contents injected into context at evaluation time. The path is resolved relative to the repo root.

```yaml
# Workspace-relative path — resolved from the repo root
inject: "./src/context/python-standards.md"
```

## Settings

```yaml
settings:
  fail_open: true  # default
```

| Field | Default | Description |
|-------|---------|-------------|
| `fail_open` | `true` | When `true`, evaluation errors (e.g. missing inject file) are non-fatal — the hook is skipped rather than blocking the action. |

## Full example

```yaml
version: "1"

PreToolUse:
  # Block destructive shell commands
  - mode: enforce
    match:
      tools: ["Bash"]
      command_match: "rm\\s+-rf"
    action:
      block: true
      message: "Destructive rm -rf commands are not allowed."

  # Inject Python standards when editing Python files
  - match:
      tools: ["Edit", "Write"]
      paths: ["*.py", "*.pyi"]
    action:
      inject: "./src/context/python-standards.md"

  # Inject REST API guidelines for API routes
  - match:
      tools: ["Edit", "Write"]
      paths: ["**/src/api/**", "**/routes/**"]
    action:
      inject: "./src/context/api-design-guidelines.md"

  # Paths are workspace-relative — resolved from the repo root.
  - match:
      tools: ["Write"]
      paths: ["docs/**"]
    action:
      inject: "./docs/DOCUMENTATION_GUIDELINES.md"

  # Inline reminder when reading any file
  - match:
      tools: ["Read"]
    action:
      inject_inline: "Remember to check for sensitive credentials before sharing file contents."

  # Warn on web fetches (audit trail)
  - mode: warn
    match:
      tools: ["WebFetch"]
    action:
      message: "External web fetches are logged for review."

PostToolUse:
  # Run linter after editing JS/TS files
  - match:
      tools: ["Write", "Edit"]
      paths: ["*.ts", "*.tsx", "*.js", "*.jsx"]
    action:
      run: "./scripts/lint-check.sh"

  # Format Go files inline after any Write/Edit (no script file needed)
  - match:
      tools: ["Write", "Edit"]
      paths: ["*.go"]
    action:
      run_inline: "cd backend && go fmt ./..."

  # Run a custom validator after any Bash command
  - mode: audit
    match:
      tools: ["Bash"]
    action:
      run: "./scripts/audit-bash.sh"
```

## Rule precedence

- Rules are evaluated **top to bottom** for each event list (`PreToolUse`, `PostToolUse`).
- A matching blocking rule (`action.block: true` in `enforce` mode) **short-circuits** evaluation — later rules are skipped.
- `action.block: false` does not explicitly allow or short-circuit; it is effectively pass-through.

**Ordering guidance:**

1. Put highest-priority and most specific blocking rules first.
2. Put broader or catch-all rules later.
3. If two rules can both block, the first matching one wins.

This avoids surprising behaviour when regex patterns overlap, such as a broad `git commit` matcher and a narrower `git commit.*WIP` block rule.

## Performance

Path matching uses [`doublestar`](https://github.com/bmatcuk/doublestar) for `**` glob support. Benchmarks on a typical CI machine:

| Scenario | Time | Allocations |
|----------|------|-------------|
| Single pattern (`*.py`) × 4 files | ~1.1 µs | 0 |
| Directory glob (`src/api/`) × 4 files | ~246 ns | 0 |
| Double-star (`src/api/**/*.py`) × 4 files | ~202 ns | 0 |
| 4 mixed patterns × 4 files | ~1.3 µs | 0 |
| Full `Evaluate` — 1 rule, 2 patterns | ~1.3 µs | 608 B (10 allocs) |
| Full `Evaluate` — 5 rules, mixed patterns | ~6.1 µs | 3 KB (50 allocs) |

Allocations come from JSON unmarshalling of `tool_input`, not from glob matching itself. A realistic 5-rule evaluation adds ~6 µs of overhead per tool call — negligible compared to LLM latency (100 ms–10 s).
