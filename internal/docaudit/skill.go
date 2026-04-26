package docaudit

import (
	"bytes"
	"strings"
	"text/template"
)

// Config holds the project-specific settings baked into the generated skill.
type Config struct {
	TargetDir      string
	ProjectName    string
	DocRoots       []string
	AnalyzeExclude []string
}

const skillTemplate = `---
name: doc-audit
description: Audit {{.ProjectName}} for external documentation gaps
allowed-tools:
  - Bash
  - Read
---

# Doc Audit: {{.ProjectName}}

Surface documentation gaps that would block a new contributor — human or AI — from making correct changes to this codebase. The goal is not technical completeness; it is identifying where missing docs would cause someone to produce wrong, broken, or inconsistent output.

## Project Configuration

- **Target directory:** ` + "`{{.TargetDir}}`" + `
- **Doc paths:** {{formatList .DocRoots}}
{{- if .AnalyzeExclude}}
- **Analyze excludes:** {{formatList .AnalyzeExclude}}
{{- end}}

## Process

Follow these steps in order.

**Step 1 — Analyze the codebase**

Run the following command and read the full output:

` + "```bash" + `
aicfg analyze {{.TargetDir}}
` + "```" + `

The report contains hub files, hotspots, and clusters. These are your primary inputs for the cross-reference in Step 3.

**Step 2 — Read existing documentation and dependency manifests**

Glob and read all files under the configured doc paths:
{{range .DocRoots}}
- ` + "`{{.}}`" + `
{{- end}}

Then locate and read the dependency manifest for this project's language or package manager. The analyzer reports the tech stack at a high level; reading the manifest directly gives the full picture — specific libraries, versions, and combinations that deterministic detection cannot surface. Common manifests by ecosystem:

| Ecosystem | Files to read |
|-----------|---------------|
| JavaScript / TypeScript | ` + "`package.json`" + ` |
| Go | ` + "`go.mod`" + ` |
| Ruby | ` + "`Gemfile`" + ` |
| Elixir | ` + "`mix.exs`" + ` |
| Python | ` + "`requirements.txt`" + `, ` + "`pyproject.toml`" + `, ` + "`Pipfile`" + ` |
| Rust | ` + "`Cargo.toml`" + ` |
| Java / Kotlin | ` + "`pom.xml`" + `, ` + "`build.gradle`" + ` |
| PHP | ` + "`composer.json`" + ` |
| .NET | ` + "`*.csproj`" + `, ` + "`*.fsproj`" + ` |

Read whichever are present. For each dependency found, ask:
- Does this library impose conventions a contributor must follow (e.g. routing patterns, state management rules, ORM query style, test structure)?
- Are those conventions documented anywhere in the doc paths?
- Does a combination of libraries carry an implicit architectural decision that is not made explicit in any doc?

If a key library has no corresponding documentation coverage, mark it as a **Undocumented Dependency Convention** in the gap report.

**Step 3 — Cross-reference**

The guiding question for every check: *"If a new contributor starts working in this area today, would they have enough context to make a correct change without breaking something?"*

For each **cluster** in the analysis:
- Does the cluster contain files that contributors directly author or edit (e.g. config files, input formats, rule definitions, source templates)?
- If yes: is there a doc explaining the **authoring conventions** — what a valid file looks like, what fields are required, what common mistakes to avoid?
- If not: mark it as **Contributor Blocker**
- Skip clusters that are pure internal implementation with no contributor-facing interface

For each **hub file** in the analysis:
- Does it define a **contract, format, or interface** that contributors must follow? (e.g. a schema, a rule structure, a config format)
- If yes: is that contract documented in accessible, actionable terms — not just as code comments?
- If not: mark it as **Undocumented Contract**

For each **hotspot** in the analysis:
- Does its high churn and size suggest **non-obvious complexity** — ordering rules, edge cases, platform differences, pitfalls?
- If yes: is there a doc explaining the key behaviors or what to watch out for when making changes?
- If not: mark it as a **Complexity Trap**

**Step 4 — Produce the gap report**

Output a report using the format defined in the Output Format section below.

## Output Format

### Contributor Blockers

For each cluster where authoring conventions are undocumented:
- Cluster name and the directly-editable files it contains
- What a contributor would need to know to work there safely
- Suggested doc file to create (e.g. ` + "`docs/billing.md`" + `)
- One-line rationale

### Undocumented Contracts

For each hub file whose contract or format is not documented:
- File path and the contract it defines
- What a contributor must know in order to comply with it
- Which doc file should cover it
- One-line rationale

### Complexity Traps

For each hotspot with non-obvious behavior and no guidance:
- File path, churn count, and line count
- The specific behavior, edge case, or pitfall that needs to be documented
- Risk to a contributor making changes without that context

### Undocumented Dependency Conventions

For each key library or framework with no documentation coverage:
- Library name and the conventions it imposes on contributors
- What a contributor would likely get wrong without guidance
- Suggested doc file to create or section to add

### Suggested Actions

A prioritized list of specific, actionable documentation tasks. Prioritize by contributor impact — which gaps are hit first and most often — then by blast radius (fan-in) and churn.
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
