# ai-config

A transpiler for AI assistant configurations. Write agent definitions, rules, commands, and skills once — generate platform-specific configs for Claude Code and GitHub Copilot automatically.

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
src/            # Canonical source definitions
  agents/       # AI agent definitions
  commands/     # Custom commands
  rules/        # Repository rules and guidelines
  skills/       # Reusable skills
config/
  platforms.yaml  # Output paths, target roots, field mappings, and drops per platform
  tool-map.yaml   # Canonical tool names → platform-specific equivalents
schemas/
  *.schema.json   # JSON Schema validation for each source type and config file
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

Platform-specific behaviour is controlled through `overrides.<platform>.<field>`. Fields listed under `drop_fields` in `platforms.yaml` are omitted for that platform unless an override is present.
