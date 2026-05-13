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
---

# Doc Audit: {{.ProjectName}}

Surface actionable gaps — new docs or doc updates — that would cause a new contributor (human or AI) to produce wrong, broken, or off-pattern output on day one. AI-assistance configuration in the doc corpus (` + "`CLAUDE.md`, `AGENTS.md`, rules, skills, instructions" + `) counts as documentation.

## Core Principles

These three principles govern the entire audit. Every finding, task, and output must satisfy all three. They are referenced by name throughout.

**Code-first principle:** Document what is hard to discover from code. Never duplicate what is obvious from reading 2–3 files in the relevant area. High-leverage: architecture decisions (why), cross-cutting constraints, failure modes, contracts between systems. Low-leverage (avoid): implementation walkthroughs, pattern inventories, rapidly-changing details. Test: *"Will this doc still be correct in 6 months without maintenance?"*

**Severity gate:** Before including ANY finding, answer all three — drop if any triggers:

1. *"What goes wrong in week 1 without this?"* — Drop if hypothetical, prevented by tooling (types, lint, CI), or already documented.
2. *"Discoverable from 2–3 existing files?"* — Drop if the convention is visible in code patterns, tests, or module structure.
3. *"Will it drift within 3 months?"* — Drop if it tracks volatile implementation details. Prefer stable constraints.

**Concrete-breakage rule:** Only flag wrappers, conventions, and doc gaps where bypassing/ignoring them causes build failures, security holes, data corruption, or review rejection — not style inconsistency.

## Constraints

| Constraint | Value |
|-----------|-------|
| Max findings per category | 3 (state total, note omissions) |
| Category precedence | Contributor Blocker > Complexity Trap > Undocumented Contract > Docs Needing Updates > Undocumented Dep Convention > Superseded Records |
| AGENTS.md max lines | 15 (pointers only — no examples, conventions, or explanations) |
| Reference doc max lines | 150 |
| New focused guide max lines | 100 |
| Section in existing doc max lines | 30 |
| Code examples max lines | 15 (cut comments that restate the code) |
| Zero findings | Report explicitly and end. Do not invent findings. |

## Project Configuration

- **Target directory:** ` + "`{{.TargetDir}}`" + `
- **Doc corpus:** {{formatList .DocRoots}} (supports glob patterns, e.g. ` + "`**/AGENTS.md`" + `)
{{- if .AnalyzeExclude}}
- **Analyze excludes:** {{formatList .AnalyzeExclude}}
{{- end}}

## Audience and style

**Audience:** senior engineers. Lead with tables, code blocks, or checklists. Prose is the fallback.

- Max 2 sentences of context per section before the artifact.
- One "must / always / never" per doc maximum.
- No tutorial framing, no "See also" footers, no "Use when" columns that restate the name.
- File:line references over descriptions of file structure.

## Process

Steps 1, 2, and 3 run **in parallel**. Wait for all three before Step 4.

---

### Step 1 — Run static analysis

` + "```bash" + `
aicfg analyze {{.TargetDir}}
aicfg analyze {{.TargetDir}} --kind=doc
` + "```" + `

First command → hub files, hotspots, clusters (input to Step 4). Second command → doc staleness dates (input to Task A).

---

### Step 2 — Read documentation and dependency manifests

Read every file under the doc corpus. Then read dependency manifests:

| Ecosystem | Manifest |
|-----------|----------|
| JS / TS | ` + "`package.json`" + ` |
| Go | ` + "`go.mod`" + ` |
| Ruby | ` + "`Gemfile`" + ` |
| Elixir | ` + "`mix.exs`" + ` |
| Python | ` + "`requirements.txt`" + `, ` + "`pyproject.toml`" + `, ` + "`Pipfile`" + ` |
| Rust | ` + "`Cargo.toml`" + ` |
| Java / Kotlin | ` + "`pom.xml`" + `, ` + "`build.gradle`" + ` |
| PHP | ` + "`composer.json`" + ` |
| .NET | ` + "`*.csproj`" + `, ` + "`*.fsproj`" + ` |

For each major dependency (skip stdlib/utility), check: non-obvious conventions not discoverable from existing usage? Wrapper where bypassing causes concrete breakage? (→ Task B)

---

### Step 3 — Spawn parallel research tasks

Spawn Tasks A, B, C concurrently using the best available subagent. Each is independent and must return concrete findings with ` + "`file:line`" + ` references.

#### Task A — Documentation coverage map

**Goal:** For each doc file, return claims (entities covered), omissions (hard-to-discover knowledge missing — apply code-first principle), and freshness signal from ` + "`aicfg analyze --kind=doc`" + `.

**Priority:** Docs older than ~90 days covering hub/hotspot areas get deeper code comparison with ` + "`file:line`" + ` divergence evidence.

**Return format:**
` + "```" + `
<path>: covers [<entities>]; omits [<entities>]; last-updated <date> (<n> days). Specific divergences: <file:line — doc vs. code>.
` + "```" + `

#### Task B — Critical wrapper discovery

**Goal:** Find project wrappers where bypassing causes concrete breakage (apply concrete-breakage rule). For each, return: wrapper name/path/library, what breaks, ` + "`file:line`" + ` bypass evidence.

#### Task C — Hard-to-discover convention survey

**Goal:** Survey conventions invisible to static analysis AND not discoverable from 2–3 files (apply code-first principle + severity gate). Focus areas:

- **Code generation** — what is generated, from what, how to regenerate
- **Authorization / security** — permission checks, contract for new permissions
- **Error handling contracts** — domain-to-transport translation, user-facing vs internal boundaries
- **Operational procedures** — deploy, migrate, rollback steps not in CI/scripts

Skip categories where code reveals the pattern. Return ` + "`file:line`" + ` references. Skip non-applicable categories.

---

### Step 4 — Cross-reference

Guiding question: *"Would a new contributor have enough context to make a correct change without breaking something?"*

Apply the **severity gate** to every candidate finding.

#### 4a — Clusters → Contributor Blockers

For each cluster with contributor-authored files: is there undocumented authoring knowledge that requires non-obvious context (ordering constraints, compatibility rules, cross-system side effects)? If self-documenting from existing files → drop.

#### 4b — Hub files → Undocumented Contracts

Hub file defines a contributor-facing contract/format/interface with no doc? → **Undocumented Contract**. Skip pure wiring nexuses.

#### 4c — Hotspots → Complexity Traps

High-churn + large file with non-obvious complexity (ordering rules, edge cases, pitfalls) and no guidance? → **Complexity Trap**.

#### 4d — Build the Single Source of Truth (SoT) map

Assign one authoritative doc per cross-cutting topic from Tasks B and C. Cap at 8 rows — only topics with multiple competing docs.

| Topic | SoT doc | Allowed references |
|---|---|---|

Actions writing into non-SoT docs → reject or rewrite as "link to SoT". Two docs claiming same topic → pick one, add ` + "`consolidate`" + ` action.

#### 4e — Doc completeness → Docs Needing Updates

Using Tasks A/B/C and the SoT map:

**Immutable-document rule:** ADRs, RFCs, design docs, postmortems are point-in-time records. Never flag as "Docs Needing Updates." If superseded → report under **Superseded Records** with a status annotation action. Detect by path pattern (` + "`ADR/`, `RFC/`, `adr-`, `rfc-`" + `) or heading conventions.

For each mutable doc, flag as "Doc Needs Update" only if following it produces **incorrect output**:

1. Doc covers a library with an undocumented wrapper (Task B) + ` + "`file:line`" + ` offenders exist
2. Doc's pattern contradicts actual usage (Task C) causing build/test/review failure
3. Doc claims broad coverage but covers one narrow aspect, leading to wrong assumptions
4. Doc example contradicts SoT pattern producing silently incorrect behavior (not cosmetic mismatches caught by compiler)

Cosmetic-accuracy items (approximate counts, version badges, line-count claims) are not doc problems. Use staleness as a tiebreaker only.

#### 4f — Dependency conventions

Report only when: (1) wrapper exists (Task B) AND (2) ` + "`file:line`" + ` offenders import raw library in feature code.

---

### Step 5 — Produce the gap report

Skip sections with no findings.

## Output Format

Each action is self-contained — the reader can accept, reject, or defer every row independently.

### Summary

` + "```" + `
## Summary

**Headline:** <one sentence: single most impactful gap>

| Category | Count |
|----------|-------|
| Contributor Blocker | N |
| Undocumented Contract | N |
| Complexity Trap | N |
| Doc Needs Update | N |
| Superseded Record | N |
| Undocumented Dep Convention | N |

Total actions: N (capped at 3 per category; M lower-severity items omitted)
` + "```" + `

### Actions

One finding per row. Order by severity (highest first), then effort (cheapest first).

` + "```" + `
#### <N>. <short description>

| Field | Value |
|-------|-------|
| Category | <Contributor Blocker / Undocumented Contract / Complexity Trap / Doc Needs Update / Superseded Record / Undocumented Dep Convention> |
| Target | <file path to create or update> |
| Type | <new doc / update doc / annotate status / consolidate / reduce> |
| Effort | <trivial (<5 min) / small (<30 min) / medium (1–2 h) / large (half day+)> |
| Evidence | <bullet list of file:line references> |
| What breaks | <one concrete sentence> |
| Drift risk | <low / medium / high> — reject high unless breakage is critical |
` + "```" + `

**Type definitions:** ` + "`consolidate`" + ` — move content to SoT doc, replace original with link. ` + "`reduce`" + ` — cut verbosity, add nothing.

---

## Phase 2 — Apply Fixes

### Step 6 — Offer to apply

> Would you like me to apply these fixes now?
> 1. **Apply all** — every action in the table.
> 2. **Apply selected** — pick by number.
> 3. **Skip** — end here, no files changed.

Wait for response. Proceed to Step 7 on option 1 or 2.

### Step 7 — Write (subagent-delegated)

Spawn a fresh writing subagent with: (a) Suggested Actions table, (b) SoT map from 4d, (c) Audience and style defaults.

**Pre-write validation per action:**

1. **SoT check** — belongs in SoT doc per map? If this isn't the SoT doc → replace with 1-line link.
2. **Example validation** — verify against SoT doc. Do not copy existing examples (they may violate SoT).
3. **Length budget** — enforce limits from Constraints table.

**Action types:** ` + "`new doc`" + ` → create at suggested path. ` + "`update doc`" + ` → edit, preserve structure/voice. ` + "`annotate status`" + ` → prepend status line below frontmatter. ` + "`consolidate`" + ` → move to SoT, replace with link. ` + "`reduce`" + ` → cut verbosity, add nothing.

Return summary of files created/modified.

### Step 8 — Review (subagent-delegated)

Spawn a separate review subagent (not the writer). Pass: (a) files created/modified, (b) SoT map, (c) this test:

> *"If I deleted this finding or doc section, would a senior engineer still ship incorrect code on day one?"*

Additionally verify:

- No duplicate content across docs (→ ` + "`consolidate`" + `)
- AGENTS.md ≤ 15 lines
- Examples match SoT
- No content obvious from reading the code it describes (→ delete)
- No volatile implementation details that drift within months (→ remove unless critical)

Report findings and apply additional cuts before reporting done.
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
