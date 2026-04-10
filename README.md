# ai-config

A transpiler for AI assistant configurations. Write agent definitions, rules, commands, and skills once — generate platform-specific configs for Claude Code and GitHub Copilot automatically.

## Installation

Install the `aicfg` binary globally with Go (no cloning required):

```bash
go install github.com/joshuaalpuerto/ai-config/cmd/aicfg@latest
```

This installs `aicfg` to `~/go/bin`, making it available as a command anywhere on your machine. Ensure `~/go/bin` is on your `$PATH`:

```bash
# add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

To run without installing:

```bash
go run github.com/joshuaalpuerto/ai-config/cmd/aicfg@latest build
```

### Using in your project

Copy `aicfg.yaml` from this repo into your project root, create a `src/` directory with your definitions, then run `aicfg build` from your project root.

```
your-project/
  aicfg.yaml
  src/
    agents/
    commands/
    rules/
    skills/
```

## How it works

Source files live in `src/` as Markdown with YAML frontmatter. The build system reads each file, maps tool names, resolves platform overrides, drops unsupported fields, and writes the result to the appropriate output directory.

```
src/ (canonical source)
  └── agents, commands, rules, skills

→ transpile →

.claude/   (Claude Code output)
.github/   (GitHub Copilot output)
```

## Project structure

```
aicfg.yaml      # Unified config: platform output paths and tool name mappings
src/            # Canonical source definitions
  agents/       # AI agent definitions
  commands/     # Custom commands
  rules/        # Repository rules and guidelines
  skills/       # Reusable skills
schemas/
  *.schema.json # JSON Schema validation for source types (embedded in binary)
```

## Commands

```bash
make install    # Build and install the aicfg binary
make build      # Transpile all src/ files to platform outputs
make clean      # Remove all generated files
make validate   # Validate source file frontmatter
make watch      # Rebuild on changes (requires fswatch)
```

## Custom source directory

By default the build reads from `src/` in the repo root. Override with `SRC_DIR`:

```bash
SRC_DIR=/path/to/custom/src make build
```

## Source file format

Each source file is a Markdown file with YAML frontmatter:

```markdown
---
name: my-agent
description: Does something useful
tools:
  - Bash
  - Read
overrides:
  github:
    description: Slightly different description for Copilot
---

Agent instructions go here.
```

Platform-specific behaviour is controlled through `overrides.<platform>.<field>`. Fields listed under `drop_fields` in `aicfg.yaml` are omitted for that platform unless an override is present.

## Hooks (`hooks.yaml`)

Hook rules let you enforce policies on every AI action. Define them in `hooks.yaml` at the repo root.

### Top-level structure

| Mode      | Behaviour                                                                 |
|-----------|---------------------------------------------------------------------------|
| `enforce` | Blocking rules stop the action. Inject/run rules are applied normally.    |
| `warn`    | Same as enforce but surfaces a warning instead of hard-blocking.          |
| `audit`   | Rule is evaluated and logged only; never blocks or modifies the action.   |

### Matchers

All matcher fields are optional. When multiple fields are present they are ANDed — the rule fires only when **all** specified matchers match.

| Field           | Type       | Description                                                                 |
|-----------------|------------|-----------------------------------------------------------------------------|
| `tools`         | `string[]` | Canonical tool names: `Bash`, `Read`, `Write`, `Edit`, `WebFetch`.          |
| `command_match` | `string`   | Regex matched against `tool_input.command` (applies to `Bash` tool only).   |
| `paths`         | `string[]` | Glob patterns matched against the file path. See [Path patterns](#path-patterns) below. |

### Path patterns

`paths` supports three pattern forms:

| Form | Example | Matches |
|------|---------|----------|
| No path separator | `*.go`, `*-suffix.tsx` | Filename only, regardless of directory |
| Trailing `/` | `src/api/`, `src/**/nested/` | Any file inside the matched directory |
| Full path glob | `src/**/*.ts`, `src/docs/**/component.tsx` | Full file path; `**` matches zero or more path segments |

### Actions

At least one action field should be set. Multiple action fields can be combined in a single rule.

| Field           | Type      | Description                                                                                  |
|-----------------|-----------|----------------------------------------------------------------------------------------------|
| `block`         | `boolean` | When `true`, prevents the tool from executing.                                               |
| `message`       | `string`  | Human-readable reason shown to the developer when a rule blocks.                             |
| `inject`        | `string`  | Path (relative to repo root) to a Markdown file whose contents are injected into context.    |
| `inject_inline` | `string`  | Inline text injected directly into context (no external file needed).                        |
| `run`           | `string`  | Path to a validator script. The event JSON is passed on stdin.                                |

### Full example

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
      inject: "context/python-standards.md"

  # Inject REST API guidelines for API routes
  - match:
      tools: ["Edit", "Write"]
      paths: ["src/api/**", "routes/**"]
    action:
      inject: "context/api-design-guidelines.md"

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

  # Run a custom validator after any Bash command
  - mode: audit
    match:
      tools: ["Bash"]
    action:
      run: "./scripts/audit-bash.sh"
```

### Rule precedence

- Rules are evaluated **top to bottom** for each event list (`PreToolUse`, `PostToolUse`).
- A matching blocking rule (`action.block: true` in `enforce` mode) **short-circuits** evaluation — later rules are skipped.
- `action.block: false` does not explicitly allow or short-circuit; it is effectively pass-through.

**Ordering guidance:**

1. Put highest-priority and most specific blocking rules first.
2. Put broader or catch-all rules later.
3. If two rules can both block, the first matching one wins.

This avoids surprising behaviour when regex patterns overlap, such as a broad `git commit` matcher and a narrower `git commit.*WIP` block rule.

### Performance

Path matching uses [`doublestar`](https://github.com/bmatcuk/doublestar) for `**` glob support. Benchmarks on a typical CI machine (Xeon Platinum 8370C):

| Scenario | Time | Allocations |
|----------|------|-------------|
| Single pattern (`*.py`) × 4 files | ~1.1 µs | 0 |
| Directory glob (`src/api/`) × 4 files | ~246 ns | 0 |
| Double-star (`src/api/**/*.py`) × 4 files | ~202 ns | 0 |
| 4 mixed patterns × 4 files | ~1.3 µs | 0 |
| Full `Evaluate` — 1 rule, 2 patterns | ~1.3 µs | 608 B (10 allocs) |
| Full `Evaluate` — 5 rules, mixed patterns | ~6.1 µs | 3 KB (50 allocs) |

Allocations come from JSON unmarshalling of `tool_input`, not from glob matching itself. A realistic 5-rule evaluation adds ~6 µs of overhead per tool call — negligible compared to LLM latency (100 ms–10 s).
