# ai-config

A transpiler for AI assistant configurations. Write agent definitions, rules, commands, and skills once — generate platform-specific configs for Claude Code and GitHub Copilot automatically.

## Installation

Install the `aicfg` binary globally with Go (no cloning required):

```bash
go install github.com/joshuacalpuerto/ai-config/cmd/aicfg@latest
```

This installs `aicfg` to `~/go/bin`, making it available as a command anywhere on your machine. Ensure `~/go/bin` is on your `$PATH`:

```bash
# add to ~/.zshrc or ~/.bashrc
export PATH="$HOME/go/bin:$PATH"
```

To run without installing:

```bash
go run github.com/joshuacalpuerto/ai-config/cmd/aicfg@latest build
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

## Hook rule precedence

If you define hook policies in `hooks.yaml`, rule order matters.

- Rules are evaluated from top to bottom for each event list (`PreToolUse`, `PostToolUse`).
- A matching blocking rule (`action.block: true` in enforce mode) short-circuits evaluation.
- `action.block: false` does not explicitly allow or short-circuit; it is effectively pass-through.

Practical ordering guidance:

- Put highest-priority and most specific blocking rules first.
- Put broader or catch-all rules later.
- If two rules can both block, the first matching one wins.

This avoids surprising behavior when regex patterns overlap, such as a broad `git commit` matcher and a narrower `git commit.*WIP` block rule.
