package docaudit

import (
	"bytes"
	"strings"
	"text/template"
)

// Config holds the project-specific settings baked into the generated skill.
type Config struct {
	TargetDir   string
	ProjectName string

	// DocRoots is the documentation corpus to audit. Callers may include
	// AI-assistance configuration paths (CLAUDE.md, AGENTS.md, .claude/,
	// .github/instructions/, etc.) here to have them treated as docs.
	DocRoots []string

	// AnalyzeExclude is forwarded to readers as informational context.
	AnalyzeExclude []string
}

const skillTemplate = `---
name: doc-audit
description: Surface contributor-enablement gaps in {{.ProjectName}} — missing or stale docs and undocumented conventions that would prevent a new joiner (human or AI) from producing convention-adherent contributions immediately.
allowed-tools:
  - Bash
  - Read
  - Agent
---

# Doc Audit: {{.ProjectName}}

Surface actionable items — new docs or doc updates — that would block a new contributor (human or AI) from producing convention-adherent contributions on day one. Treat any AI-assistance configuration listed in the doc corpus (e.g. ` + "`CLAUDE.md`, `AGENTS.md`, rules, skills, instructions" + `) as documentation for the purposes of this audit.

The goal is not technical completeness. It is identifying where **missing or misleading guidance would cause someone to produce wrong, broken, or off-pattern output** — and proposing the smallest concrete artifact (a new doc or a doc update) that closes the gap.

## Project Configuration

- **Target directory:** ` + "`{{.TargetDir}}`" + `
- **Doc corpus:** {{formatList .DocRoots}}
{{- if .AnalyzeExclude}}
- **Analyze excludes:** {{formatList .AnalyzeExclude}}
{{- end}}

## Process

Steps 1, 2, and 3 run in parallel. Wait for all three before starting Step 4.

---

### Step 1 — Run static analysis

Run both commands and read the full output of each:

` + "```bash" + `
aicfg analyze {{.TargetDir}}
aicfg analyze {{.TargetDir}} --kind=doc
` + "```" + `

The first reports hub files, hotspots, and clusters — your primary inputs for the cross-reference in Step 4. The second reports each doc file's last-updated date and how many days have passed; this is **input to Task A** (used to prioritize which docs get a deep read), not a standalone output.

---

### Step 2 — Read documentation and dependency manifests

Glob and read every file under the configured doc corpus. Then read the dependency manifest(s) for this project's ecosystem(s):

| Ecosystem | Files to read |
|-----------|---------------|
| JavaScript / TypeScript | ` + "`package.json`" + ` |
| Go | ` + "`go.mod`" + ` |
| Ruby | ` + "`Gemfile`" + ` |
| Elixir | ` + "`mix.exs`" + ` |
| Python | ` + "`requirements.txt`, `pyproject.toml`, `Pipfile`" + ` |
| Rust | ` + "`Cargo.toml`" + ` |
| Java / Kotlin | ` + "`pom.xml`, `build.gradle`" + ` |
| PHP | ` + "`composer.json`" + ` |
| .NET | ` + "`*.csproj`, `*.fsproj`" + ` |

For each dependency, ask:

- Does it impose conventions a contributor must follow (routing, state management, ORM style, test structure)?
- Are those conventions documented anywhere in the doc corpus?
- Does the project have a **wrapper or abstraction** around it? (See Task B.)

> **Wrapper-precedence rule:** An undocumented project wrapper around a popular library is **more critical** than an undocumented raw library. Contributors will reach for the well-known library directly and silently bypass the team convention.

---

### Step 3 — Spawn parallel research tasks

Spawn the three tasks below **concurrently** with Steps 1 and 2. Use whichever subagent the project provides for general-purpose codebase exploration (or the most specialized agent available for each task). Do **not** assume any specific agent name exists — pick the best available.

Each task is independent. Each must return concrete findings with ` + "`file:line`" + ` references — not summaries.

---

#### Task A — Documentation coverage map

**Input:** every file under the doc corpus, plus the staleness output from ` + "`aicfg analyze --kind=doc`" + `.

**Goal:** for each doc file, return:

1. **Claims:** which code-level entities the doc says it covers (libraries, components, hooks, patterns, workflows, services).
2. **Omissions:** what a reader following this doc would *not* know that they need to know.
3. **Freshness signal:** last-updated date and days-since-update from the staleness report.

**Prioritization:** docs that are **older than ~90 days AND cover hub/hotspot areas** from Step 1 get a deeper comparison against current code. For these, include specific divergences with ` + "`file:line`" + ` evidence.

**Return format (per doc):**
` + "```" + `
<path>: covers [<entities>]; omits [<entities>]; last-updated <date> (<n> days). Specific divergences: <file:line — what the doc says vs. what code does>.
` + "```" + `

Do not return raw doc content — only synthesized findings.

---

#### Task B — Project wrappers and convention discovery

**Goal:** find **project-level abstractions** wrapping third-party dependencies — components wrapping a UI library, hooks wrapping a data-fetching library, helpers wrapping date/number/string libraries, façades over HTTP clients, generated-code adapters, etc.

For each wrapper found, return:

- Wrapper name and file path
- The library it wraps
- Approximate fan-in (how many feature files import it directly)
- Whether contributors should always prefer the wrapper over the raw library (yes/no, with reason)
- A short usage snippet with ` + "`file:line`" + ` reference

Also surface **how key dependencies from the manifest are actually used** in the codebase — concrete code snippets with ` + "`file:line`" + ` references for the most-imported third-party libraries.

---

#### Task C — Pattern survey (medium thoroughness)

**Goal:** survey conventions that static analysis cannot see. Spend effort proportional to project size on the following categories — adapt to what the project actually has:

- **Test conventions:** fixtures, mocking strategy, test data factories, integration vs unit boundaries.
- **DI / wiring patterns:** how new handlers, services, routes, or commands are registered.
- **Configuration patterns:** how new env vars, feature flags, or external service configs are added.
- **Error handling:** project-level error types, translation layers (e.g. domain → transport), user-facing vs internal errors.
- **Code generation:** what is generated, from what source, and how to regenerate.
- **Authorization / security:** how permissions are checked, where the contract for new permissions lives.

Return concrete findings with ` + "`file:line`" + ` references. Skip categories that don't apply.

---

### Step 4 — Cross-reference

The guiding question for every check: *"If a new contributor starts working in this area today, would they have enough context to make a correct change without breaking something?"*

#### 4a — Clusters (Contributor Blockers)

For each cluster from Step 1: does it contain files contributors directly author or edit? If yes and there is no doc explaining the authoring conventions → **Contributor Blocker**. Skip clusters that are pure internal implementation.

#### 4b — Hub files (Undocumented Contracts)

For each hub file: does it define a contract, format, or interface contributors must follow? If yes and there is no accessible doc covering it → **Undocumented Contract**.

#### 4c — Hotspots (Complexity Traps)

For each hotspot: does its high churn and size suggest non-obvious complexity (ordering rules, edge cases, platform differences, pitfalls)? If yes and there is no guidance → **Complexity Trap**.

#### 4d — Existing doc completeness (Docs Needing Updates)

Using Task A's coverage map plus Task B's wrappers and Task C's patterns:

For each doc in the corpus, check:

1. Does the doc mention wrappers Task B identified for libraries it covers? If a wrapper exists and the doc covers the underlying library without naming the wrapper → **Doc Needs Update**.
2. Does the doc's described pattern match what Task C found in actual usage? If the codebase has evolved a more specific convention → **Doc Needs Update**.
3. Does the doc claim broad coverage in its title/intro but only address one narrow aspect? → **Doc Needs Update** (false-coverage docs are worse than missing docs).
4. Use freshness signal as a tiebreaker: stale + hot = highest-confidence update target.

#### 4e — Dependency conventions (Undocumented Dependency Conventions)

For each key dependency from Step 2 with no corresponding doc coverage — and especially those with project wrappers from Task B — produce an **Undocumented Dependency Convention** entry.

---

### Step 5 — Produce the gap report

Output a report using the format below. Skip sections that have no findings rather than emitting empty headers.

## Output Format

### Contributor Blockers
For each cluster where authoring conventions are undocumented:
- Cluster name and the directly-editable files it contains
- What a contributor would need to know to work there safely
- Suggested doc file to create
- One-line rationale

### Undocumented Contracts
For each hub file whose contract or format is not documented:
- File path and the contract it defines
- What a contributor must know in order to comply with it
- Suggested target doc
- One-line rationale

### Complexity Traps
For each hotspot with non-obvious behavior and no guidance:
- File path, churn count, and line count
- The specific behavior, edge case, or pitfall that needs to be documented
- Risk to a contributor making changes without that context

### Docs Needing Updates
For each existing doc that is incomplete or misleading relative to actual code patterns:
- Doc file path and the specific section that needs updating
- What exists in code that the doc does not cover, with ` + "`file:line`" + ` reference
- Freshness signal (last-updated date) when relevant
- One-line rationale (what a contributor gets wrong by following the doc as-is)

### Undocumented Dependency Conventions
For each key library or framework with no documentation coverage (especially those with project wrappers):
- Library name and the conventions it imposes on contributors
- Project wrapper path (if any) — and whether the wrapper itself is documented
- What a contributor would likely get wrong without guidance
- Suggested target doc

### Suggested Actions

A prioritized table of concrete actions. Each row must include the **type** and the **target surface**. Prioritize by contributor impact (which gaps are hit first and most often) then by blast radius (fan-in) and churn.

| # | Action | Type | Target surface | Rationale |
|---|--------|------|----------------|-----------|
| … | … | …  | … | … |

**Type values:** ` + "`new doc`, `update doc`" + `.

**Target surface values:** the path the change lands in (any path under the configured doc corpus, e.g. ` + "`docs/`, `README.md`, `.claude/rules/`, `.github/instructions/`, `CLAUDE.md`, `AGENTS.md`" + `).
`

var parsedTemplate = template.Must(template.New("skill").Funcs(template.FuncMap{
	"formatList": formatList,
}).Parse(skillTemplate))

// GenerateSkill returns the content of a project-specific doc-audit SKILL.md.
func GenerateSkill(cfg Config) string {
	var buf bytes.Buffer
	if err := parsedTemplate.Execute(&buf, cfg); err != nil {
		// template.Must guarantees Parse never fails; Execute only fails on
		// writer errors against a bytes.Buffer, which cannot happen.
		panic("docaudit: executing skill template: " + err.Error())
	}
	return buf.String()
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = "`" + item + "`"
	}
	return strings.Join(quoted, ", ")
}
