---
name: mermaid-diagram
description: Generate Mermaid diagrams for GitHub Markdown that render correctly. Use when user mentions 'diagram', 'mermaid', 'flowchart', 'sequence diagram', 'class diagram', or asks to visualize a workflow, architecture, or data relationship. Do NOT use for non-diagram documentation tasks.
argument-hint: description of what to diagram
---

# Mermaid Diagram Generation

Generate GitHub-compatible Mermaid diagrams to visualize workflows, architectures, data relationships, and processes.

## Supported Diagram Types

| User Request | Diagram Type |
|---|---|
| Flow, process, workflow, steps | `graph TD` (flowchart) |
| API calls, interactions, messages | `sequenceDiagram` |
| Data models, classes, entities | `classDiagram` or `erDiagram` |
| State machine, status transitions | `stateDiagram-v2` |
| Proportions, breakdown | `pie` |
| Timeline, schedule, milestones | `gantt` |
| Concepts, topics, brainstorm | `mindmap` |

## Instructions

### Step 1: Choose the Right Diagram Type

Select based on the user's intent using the table above. When in doubt, prefer `graph TD` — it renders well and is the most versatile.

### Step 2: Apply GitHub-Compatible Syntax Rules

**Always quote labels** containing spaces, punctuation, or Mermaid keywords:

```
A["User submits form"] --> B["Validate input"]
```

**Use simple alphanumeric IDs** — no spaces or special characters in node IDs:

```
userAuth --> validateToken --> returnClaims
```

**Use standard arrow types:**
- `-->` directed arrow
- `---` undirected line
- `==>` thick arrow
- `->>` sequence diagram arrow

**Add comments with `%%`** when clarifying complex logic.

### Step 3: Follow Type-Specific Rules

#### Flowcharts (`graph TD`)
- Use `TD` (top-down) for readability in GitHub PRs
- Use `{...}` for decisions, `[...]` for actions, `(...)` for rounded nodes
- Keep to ~15 nodes max per diagram; split if larger

#### Sequence Diagrams (`sequenceDiagram`)
- Declare `participant` aliases at the top
- Use `->>` for async, `-->>` for responses
- Group related calls with `activate`/`deactivate`

#### State Diagrams (`stateDiagram-v2`)
- Always use `stateDiagram-v2` (not `stateDiagram`)
- Use `[*]` for start and end states

#### Mindmaps (`mindmap`)
- Use indentation only to define hierarchy
- Plain text nodes only — no `::icon()` syntax (breaks GitHub rendering)

### Step 4: Avoid Common Mistakes

**DO NOT:**
- Add theme/init directives (`%%{init: {'theme': 'dark'}}%%`) — GitHub ignores these
- Apply `classDef` or inline `style` rules — GitHub overrides them
- Use icons in mindmaps (`::icon(fa fa-x)`) — causes rendering errors
- Leave labels unquoted when they contain spaces or special characters
- Use `graph LR` — prefer `graph TD` for vertical readability

**DO:**
- Quote all labels with spaces or special characters: `A["My Label"]`
- Break complex diagrams into multiple focused ones (~15 nodes max each)
- Use GitHub's preview tab to validate rendering

### Step 5: Wrap in Fenced Code Block

Always output the diagram inside a fenced code block with the `mermaid` language tag.

## Error Recovery

| Symptom | Likely Cause | Fix |
|---|---|---|
| Diagram shows raw text | Syntax error in labels | Add quotes around all labels |
| Icons not rendering | Used `::icon()` in mindmap | Remove icon syntax |
| Theme not applying | Used `%%{init}%%` | Remove — GitHub controls theming |
| Diagram too wide | Used `graph LR` | Switch to `graph TD` |
| Node ID error | Special chars in ID | Use alphanumeric IDs only |
