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
src/                      (canonical source)
  ├── agents/
  ├── commands/
  ├── rules/
  └── skills/

→ transpile →

.claude/                  (Claude Code)
  ├── agents/             *.md
  ├── commands/           *.md
  ├── rules/              *.md
  └── skills/             *.md

.github/                  (GitHub Copilot)
  ├── agents/             *.agent.md
  ├── prompts/            *.prompt.md        (source: commands/)
  ├── instructions/       *.instructions.md  (source: rules/)
  └── skills/             *.md
```

Output directory names and file suffixes are platform-specific and configured in `aicfg.yaml`.

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

See [docs/aicfg-config.md](docs/aicfg-config.md) for a full reference of all `aicfg.yaml` fields.

## Commands

```bash
make install    # Build and install the aicfg binary
make build      # Transpile all src/ files to platform outputs
make clean      # Remove all generated files
make validate   # Validate source file frontmatter
make watch      # Rebuild on changes (requires fswatch)
```

The `aicfg` binary also exposes a `hooks` subcommand used at runtime by Claude Code:

```bash
aicfg hooks     # Evaluate a PreToolUse hook event from stdin
```

## Custom source directory

By default the build reads from `src/` in the repo root. Change the `src_dir` field in `aicfg.yaml` to use a different path — absolute or relative to the repo root:

```yaml
src_dir: /path/to/custom/src
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

Hook rules enforce policies on every AI action — blocking dangerous commands, deterministically injecting context, or running side-effect scripts. Define them in `hooks.yaml` at the repo root:

```yaml
version: "1"

PreToolUse:
  - mode: enforce
    match:
      tools: ["Bash"]
      command_match: "rm\\s+-rf"
    action:
      block: true
      message: "Destructive rm -rf commands are not allowed."

PostToolUse:
  - match:
      tools: ["Write", "Edit"]
      paths: ["*.ts", "*.tsx"]
    action:
      run: "./scripts/lint-check.sh"
```

For the full reference — modes, matchers, path patterns, inject actions, rule precedence, and settings — see [docs/hooks.md](docs/hooks.md).

## Context injection via hooks

Context injection is the primary reason to use hooks. It delivers instructions to the AI **unconditionally** — no AI judgment, no prior file read required.

### Why rules alone are not enough

Claude rules (`.claude/rules/`) and `CLAUDE.md` files are loaded by the `Read` tool. This creates a critical gap:

- **New file creation** — when the AI writes a file that does not yet exist, no `Read` event fires first. The rule is never loaded, so coding standards and architectural constraints are silently skipped.
- **Subdirectory rules** — there are [known issues](https://github.com/anthropics/claude-code/issues/23478) where Claude does not reliably read rules in subdirectories, meaning instructions scoped to `src/` or deeper may never reach the model.
- **Skills are probabilistic** — a skill may not be loaded if the AI does not judge the task as matching its description, particularly for less common workflows.

Hook injection sidesteps all of this by firing at the tool-call level, not the file-read level.

### Primary use cases

**1. Inject standards before writing a new file**

The most impactful use. When the AI creates a file that does not exist yet, there is nothing to read — rules cannot load. A `PreToolUse` hook on the `Write` tool fires before the file is written, injecting any standards the AI needs:

```yaml
PreToolUse:
  - match:
      tools: ["Write"]
      paths: ["*.ts", "*.tsx"]
    action:
      inject: "./src/context/typescript-standards.md"

  - match:
      tools: ["Write"]
      paths: ["**/src/api/**"]
    action:
      inject: "./src/context/api-design-guidelines.md"
```

**2. Enforce non-negotiable policies unconditionally**

Policies that cannot rely on the AI remembering a rule it may or may not have read earlier in the session:

```yaml
PreToolUse:
  - match:
      tools: ["Write", "Edit"]
      paths: ["*.go"]
    action:
      inject_inline: |
        All Go code must handle every returned error explicitly.
        Never use _ to discard errors from external packages.
```

**3. Deterministic `PostToolUse` side-effects**

Run formatters, linters, or validators unconditionally after specific file types are written — regardless of what the AI did or did not read:

```yaml
PostToolUse:
  - match:
      tools: ["Write", "Edit"]
      paths: ["*.go"]
    action:
      run_inline: "cd backend && go fmt ./..."

  - match:
      tools: ["Write", "Edit"]
      paths: ["*.ts", "*.tsx", "*.js", "*.jsx"]
    action:
      run: "./scripts/lint-check.sh"
```

**4. Guard against creating files without a plan**

When the AI creates a new file opportunistically (no blueprint or prior context), inject a reminder to follow the established structure. This covers the gap where a skill would normally provide this guidance but was not loaded:

```yaml
PreToolUse:
  - match:
      tools: ["Write"]
      paths: ["**/components/**/*.tsx"]
    action:
      inject: "./src/context/component-authoring-guide.md"
```

## When to use rules vs skills vs hook context injection

| Mechanism | When loaded | Best for |
|-----------|-------------|---------|
| **Rules** | When AI reads a file whose path matches the rule's `applyTo` glob | Repo-wide standards and guidelines the AI will encounter while reading existing files |
| **Skills** | Description always in context; full body loaded when the AI judges the task matches | Step-by-step workflows and reusable domain procedures |
| **Hook context injection** | Every matching `tool × path` — unconditional, no AI judgment needed | Any standard that must apply when **writing** files, new files the AI hasn't read yet, non-negotiable constraints, and `PostToolUse` side-effects |

**The key difference:** Rules and skills depend on the AI deciding to load them. Hook injection fires unconditionally — use it when correct behaviour cannot rely on the AI having encountered the instruction earlier in the session.
