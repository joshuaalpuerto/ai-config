---
name: doc-audit
description: Audit ai-config for external documentation gaps
---

# Doc Audit: ai-config

Surface missing or outdated external documentation by cross-referencing the structural analysis of the codebase against existing docs.

## Project Configuration

- **Target directory:** `/workspaces/ai-config`
- **Doc paths:** `docs/`, `README.md`
- **Analyze excludes:** `.claude`, `.github`

## Process

Follow these steps in order.

**Step 1 — Analyze the codebase**

Run the following command and read the full output:

```bash
aicfg analyze /workspaces/ai-config
```

The report contains hub files, hotspots, and clusters. These are your primary inputs for the cross-reference in Step 3.

**Step 2 — Read existing documentation**

Glob and read all files under the configured doc paths:

- `docs/`
- `README.md`

**Step 3 — Cross-reference**

For each **cluster** in the analysis:
- Check whether any doc file covers that cluster's domain (by folder name, exported symbols, or topic)
- If not: mark it as **Missing Doc**

For each **hub file** in the analysis:
- Check whether any doc file mentions, describes, or links to that file or its exports
- If not: mark it as **Undocumented Key File**

For each **hotspot** in the analysis:
- Check whether any doc file describes the purpose, behavior, or usage of that file
- If not: note it as a documentation coverage risk

**Step 4 — Produce the gap report**

Output a report using the format defined in the Output Format section below.

## Output Format

### Missing Docs

For each cluster with no documentation coverage:
- Cluster name and the files it contains
- Suggested doc file to create (e.g. `docs/billing.md`)
- One-line rationale

### Undocumented Key Files

For each hub file not mentioned in any doc:
- File path and its exported symbols
- Which doc file should cover it
- One-line rationale

### Suggested Actions

A prioritized list of specific, actionable documentation tasks. Prioritize by blast radius (fan-in) and churn.
