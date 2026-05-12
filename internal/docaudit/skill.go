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

Surface actionable items — new docs or doc updates — that would block a new contributor (human or AI) from producing convention-adherent contributions on day one. Treat any AI-assistance configuration listed in the doc corpus (e.g. ` + "`CLAUDE.md`, `AGENTS.md`, rules, skills, instructions" + `) as documentation for the purposes of this audit.

The goal is not technical completeness. It is identifying where **missing or misleading guidance would cause someone to produce wrong, broken, or off-pattern output** — and proposing the smallest concrete artifact (a new doc or a doc update) that closes the gap.

## Project Configuration

- **Target directory:** ` + "`{{.TargetDir}}`" + `
- **Doc corpus:** {{formatList .DocRoots}} (supports glob patterns, e.g. ` + "`**/AGENTS.md`" + `)
{{- if .AnalyzeExclude}}
- **Analyze excludes:** {{formatList .AnalyzeExclude}}
{{- end}}

## Audience and style

**Audience:** senior engineers who can capture context immediately. Write so a reader skimming a table or code block gets the answer; prose is the fallback.

**Defaults:**

- Lead with the table, code example, or checklist. Prose only if the structure doesn't carry the meaning.
- Max 2 sentences of context per section before the concrete artifact.
- One "must / always / never" per doc maximum.
- No "Use when" columns that restate the name.
- No tutorial framing ("This document outlines...", "It's important to understand...").
- No standalone "See also" footer. Inline links where useful, nowhere else.
- Code examples ≤ 15 lines; cut comments that restate the code.
- File:line references over descriptions of file structure.

**AGENTS.md contract:** AGENTS.md files are pure reading-order lists. 3–5 links with a one-line "what's here". No examples, no conventions, no explanations — those live in the SoT docs the AGENTS.md points to.

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

> **Immutable-document rule:** ADRs, RFCs, design docs, postmortems, and migration guides are point-in-time records. **Never flag them as "Docs Needing Updates."** If an ADR/RFC describes a decision that current code has superseded, report it under **Superseded Records** (Step 5) — the only valid action is adding a status annotation (e.g. "Status: Superseded by ADR-XXX") at the top of the file. Detect these by path pattern (e.g. ` + "`ADR/`, `RFC/`, `adr-`, `rfc-`" + `) or by title/heading conventions.

For each **mutable** doc in the corpus, check:

1. Does the doc mention wrappers Task B identified for libraries it covers? If a wrapper exists and the doc covers the underlying library without naming the wrapper → **Doc Needs Update**.
2. Does the doc's described pattern match what Task C found in actual usage? If the codebase has evolved a more specific convention → **Doc Needs Update**.
3. Does the doc claim broad coverage in its title/intro but only address one narrow aspect? → **Doc Needs Update** (false-coverage docs are worse than missing docs).
4. Use freshness signal as a tiebreaker: stale + hot = highest-confidence update target.
5. **Example validation against SoT docs.** For any example in the doc, check whether it conforms to the canonical pattern documented elsewhere (validation tags, route patterns, error handling, codegen contracts). If the doc's own example contradicts a SoT doc (e.g. a route example that defines search-param defaults the BFF struct should own), flag this as a **Doc Contradiction** under Docs Needing Updates. Existing examples are evidence of past mistakes, not authority.

#### 4e — Dependency conventions (Undocumented Dependency Conventions)

For each key dependency from Step 2 with no corresponding doc coverage — and especially those with project wrappers from Task B — produce an **Undocumented Dependency Convention** entry.

---

### Step 5 — Build the Single Source of Truth map

Before producing the Suggested Actions table, list every cross-cutting topic from Tasks B and C and assign **one** authoritative doc to each. Other docs may reference but **never restate** content owned by the SoT doc.

| Topic | SoT doc | Allowed references |
|---|---|---|
| Backend wrapper inventory | e.g. ` + "`BACKEND_PATTERNS.md`" + ` | ` + "`BFF.md`" + `, AGENTS.md |
| Validation tags / contract | e.g. ` + "`BFF_VALIDATION.md`" + ` | ` + "`BACKEND_PATTERNS.md`" + ` |
| Frontend wrapper inventory | e.g. ` + "`FRONTEND_WRAPPERS.md`" + ` | ` + "`CODE_STYLE.md`" + `, ` + "`API_DATA_FETCHING.md`" + ` |
| Claim-based authorization | e.g. ` + "`UI_AUTHORIZATION.md`" + ` | routing docs, ` + "`SECURITY.md`" + ` |
| ... | ... | ... |

Any Suggested Action that would write content into a non-SoT doc is rejected or rewritten as "link to SoT". If two docs both have a claim on the same topic, pick one and add a ` + "`consolidate`" + ` action to move content into it.

---

### Step 6 — Produce the gap report

Output a report using the format below. Skip sections that have no findings rather than emitting empty headers.

## Output Format

Start with a summary block, then the detailed sections.

### Summary

A compact overview so the reader immediately knows the scope. Use this exact structure:

` + "```" + `
## Summary

| Category | Count |
|----------|-------|
| Contributor Blockers | N |
| Undocumented Contracts | N |
| Complexity Traps | N |
| Docs Needing Updates | N |
| Superseded Records | N |
| Undocumented Dependency Conventions | N |

**Headline:** <one sentence describing the single most impactful gap>
` + "```" + `

---

### Contributor Blockers

Use this structure per finding:

` + "```" + `
#### <N>. <Cluster / directory name>

| Field | Value |
|-------|-------|
| Files | <directly-editable files in this cluster> |
| Gap | <what a contributor needs to know to work here safely — 1–2 sentences max> |
| Suggested doc | <path to create> |
| Rationale | <one line: what goes wrong without this> |
` + "```" + `

After the table, optionally include a bullet list of specific wrappers, components, or conventions (max 8 items) — one line each, no prose.

---

### Undocumented Contracts

` + "```" + `
#### <N>. <file path>

| Field | Value |
|-------|-------|
| Contract | <what it defines — interface, format, wiring sequence> |
| Must-know | <the specific steps or rules contributors must follow — brief> |
| Suggested target | <doc path to update or create> |
| Rationale | <what breaks if ignored — one line> |
` + "```" + `

---

### Complexity Traps

` + "```" + `
#### <N>. <file path> (<churn>× churn, <lines> lines)

| Field | Value |
|-------|-------|
| Pitfall | <the non-obvious behavior, edge case, or ordering rule — 1–2 sentences> |
| Risk | <what a contributor gets wrong without this context> |
| Suggested target | <doc path> |
` + "```" + `

---

### Docs Needing Updates

` + "```" + `
#### <N>. <doc file path> (<age> days old)

| Field | Value |
|-------|-------|
| Covers | <what the doc correctly describes — brief> |
| Divergences | <bullet list of specific mismatches with file:line evidence> |
| Omits | <what the doc should cover but doesn't> |
| Rationale | <what a contributor gets wrong by following the doc as-is — one line> |
` + "```" + `

Keep each divergence to one line with a ` + "`file:line`" + ` reference. Use a bullet list, not a paragraph.

---

### Superseded Records

` + "```" + `
#### <N>. <file path>

| Field | Value |
|-------|-------|
| Records | <the original decision — one line> |
| Superseded by | <newer ADR/RFC number or current pattern> |
| Evidence | <file:line showing the codebase no longer follows this decision> |
| Suggested annotation | "Status: Superseded by <X>" |
` + "```" + `

Do NOT suggest content edits to ADRs/RFCs — only a status annotation at the top of the file.

---

### Undocumented Dependency Conventions

` + "```" + `
#### <N>. <library name>

| Field | Value |
|-------|-------|
| Convention | <what the library imposes on contributors — one line> |
| Wrapper | <path> (documented: yes/no) |
| Misuse risk | <what a contributor does wrong without guidance — one line> |
| Suggested target | <doc path> |
` + "```" + `

---

### Suggested Actions

A prioritized table grouped by effort. List quick wins first (` + "`annotate status`" + `), then updates, then new docs. Each row must include **type**, **effort**, and **target surface**.

| # | Action | Type | Effort | Target surface | Rationale |
|---|--------|------|--------|----------------|-----------|
| … | … | …  | … | … | … |

**Type values:** ` + "`new doc`, `update doc`, `annotate status`, `consolidate`, `reduce`" + `.

- ` + "`consolidate`" + ` — move content from doc X into SoT doc Y; replace doc X's section with a one-line link.
- ` + "`reduce`" + ` — cut existing verbosity without adding content. Use when a doc exceeds its length budget.

**Effort values:** ` + "`trivial`" + ` (< 5 min, e.g. adding a status line), ` + "`small`" + ` (< 30 min, e.g. fixing a section), ` + "`medium`" + ` (1–2 hours, e.g. writing a new focused guide), ` + "`large`" + ` (half day+, e.g. comprehensive new doc with examples).

**Target surface values:** the path the change lands in (any path under the configured doc corpus, e.g. ` + "`docs/`, `README.md`, `.claude/rules/`, `.github/instructions/`, `CLAUDE.md`, `AGENTS.md`" + `).

---

### Step 7 — Offer to apply fixes

After presenting the full gap report, ask the user:

> Would you like me to apply these fixes now? I can:
>
> 1. **Apply all** — create new docs and update existing ones for every item in the Suggested Actions table.
> 2. **Apply selected** — let you pick which actions to apply (by number from the table).
> 3. **Skip** — end the audit here; no files will be created or modified.

Wait for the user's response before proceeding. If the user selects option 1 or 2:

**Pre-write validation gate (run for every action before writing):**

1. **SoT check.** Does the content belong in the SoT doc per the Step 4.5 map? If yes and this isn't the SoT doc, replace the content with a 1-line link.
2. **Example validation.** If the action includes a code example, identify every SoT doc the example touches (validation tags → validation doc, route pattern → auth doc, error handling → patterns doc). Read those docs and verify the example conforms. **Do not copy from existing examples in the doc being updated — they may themselves violate the SoT.**
3. **Length budget** (reject if exceeded without explicit justification):
   - AGENTS.md ≤ 15 lines (pointers only, no tutorials, no examples)
   - Reference doc ≤ 150 lines
   - New focused guide ≤ 100 lines
   - Section in existing doc ≤ 30 lines

**Then write:**

- ` + "`new doc`" + `: create the file at the suggested path. Apply the Audience and style defaults.
- ` + "`update doc`" + `: edit the existing file to address the divergences and omissions identified. Preserve the doc's existing structure and voice; only add or correct the specific sections that are stale or missing.
- ` + "`annotate status`" + `: prepend the status annotation line at the top of the file (below any frontmatter).
- ` + "`consolidate`" + `: move content into the SoT doc; replace the original location with a single-line link.
- ` + "`reduce`" + `: cut verbosity per the Audience and style defaults; do not add new content.

After applying, output a summary of files created and modified.

---

### Step 8 — Self-review

After applying fixes, re-read each touched doc and answer:

1. Does any other doc now contain the same content? If yes → ` + "`consolidate`" + `.
2. Could a senior engineer skim this doc in 60 seconds and act? If no → ` + "`reduce`" + `.
3. Does every code example conform to the SoT docs? Spot-check validation tags, route patterns, error handling, contract sources.
4. Are there standalone "See also" / "Related Documentation" sections that only restate top-of-doc context? Remove them.
5. Did any AGENTS.md grow beyond the 15-line pointer contract? Trim to pointers.

Report any issues found and apply additional cuts before reporting done.
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
