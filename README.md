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

Hook rules enforce policies on every AI action — blocking dangerous commands, injecting context, or running validators. Define them in `hooks.yaml` at the repo root:

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
