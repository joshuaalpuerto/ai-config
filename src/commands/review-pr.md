---
description: Review an open pull request end-to-end. Fetches PR details, reads the diff, and produces structured feedback as a senior engineer.
tools:
  - Bash
  - Read
  - Grep
---

# Command: Review PR

## Goal

Provide a thorough, senior-level code review of an open pull request. Produce structured feedback grouped by severity. Be specific, cite line numbers, and distinguish blocking issues from suggestions.

## Usage

```
/review-pr <PR number or URL>
```

## Process

### 1. Fetch PR metadata

```bash
gh pr view <PR> --json title,body,author,baseRefName,headRefName,files
```

Read the title and description to understand what the PR is trying to accomplish. If no description is provided, note that as an observation.

### 2. Fetch the diff

```bash
gh pr diff <PR>
```

Read the full diff. For large PRs (>500 lines changed), prioritize:
1. Core logic files
2. New files (most likely to have issues)
3. Test files (validate coverage)

### 3. Explore context where needed

For any changed file, read surrounding code to understand:
- Existing patterns the PR should follow
- Whether the change fits the established architecture
- Whether tests reflect actual behavior

### 4. Identify issues

Categorize findings:

| Severity | When to use |
|----------|-------------|
| **Blocking** | Bugs, security issues, broken contracts, incorrect behavior |
| **Suggestion** | Better abstractions, naming, minor clarity improvements |
| **Observation** | Questions about intent, neutral notes for discussion |

### 5. Write the review

Structure output:

```
## PR Review: <title>

**Author:** @<author>
**Branch:** <head> → <base>

---

### Summary

<2-3 sentence overview of what the PR does and your overall assessment>

---

### Blocking Issues

- `path/to/file.ts:42` — <description of issue and why it matters>

### Suggestions

- `path/to/file.go:17` — <description of improvement>

### Observations

- <neutral note or question>

---

### Verdict

[ ] Approve  [ ] Request changes  [ ] Needs discussion
```

If there are zero blocking issues, state that clearly before the suggestions.
