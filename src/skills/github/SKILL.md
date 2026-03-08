---
name: github
description: Interact with GitHub via the `gh` CLI for pull requests, issues, checks, and releases. Use when creating, reading, updating, or searching GitHub resources. Provides consistent output and handles common GitHub workflows.
argument-hint: <action> [args]
allowed-tools:
  - Bash(gh *)
  - Bash(git *)
---

# GitHub CLI Skill

Use the `gh` CLI to interact with GitHub repositories, pull requests, checks, and releases.

## IMPORTANT RESTRICTIONS

**YOU MUST ONLY:**
- Operate on the current repository unless a `--repo` flag is explicitly provided
- Use `gh pr` commands for pull request operations
- Use `gh run` commands for CI/CD operations

**NEVER:**
- Use `gh issue` for issue management — use the Linear skill instead
- Push to `main` or `master` directly
- Force push unless the user explicitly requests it

---

## Pull Requests

### View a PR

```bash
gh pr view <number> --json title,body,state,author,reviews,statusCheckRollup
```

### List open PRs

```bash
gh pr list --state open --json number,title,author,headRefName
```

### Create a PR

```bash
gh pr create \
  --draft \
  --title "<title>" \
  --body "<body>"
```

Always create PRs in draft state unless the user says otherwise.

### Check PR status

```bash
gh pr checks <number>
```

### Merge a PR (squash)

```bash
gh pr merge <number> --squash --delete-branch
```

---

## CI / Workflow Runs

### List recent runs

```bash
gh run list --limit 10
```

### View a specific run

```bash
gh run view <run-id>
```

### Watch a run until completion

```bash
gh run watch <run-id>
```

### Download artifacts

```bash
gh run download <run-id>
```

---

## Releases

### List releases

```bash
gh release list
```

### View a release

```bash
gh release view <tag>
```

### Create a release

```bash
gh release create <tag> \
  --title "<title>" \
  --notes "<release notes>" \
  --draft
```

Always create releases as drafts unless the user says otherwise.

---

## Repositories

### View repo info

```bash
gh repo view --json name,description,defaultBranchRef,isPrivate
```

### Clone a repo

```bash
gh repo clone <owner>/<repo>
```

---

## Output Format

By default, return results as clean, readable text. When the user asks for structured output or when feeding results to another tool, use `--json` and format with `jq`:

```bash
gh pr list --json number,title,author | jq '.[] | "\(.number): \(.title) (@\(.author.login))"'
```

---

## Common Workflows

### Check if CI passes before merging

```bash
gh pr checks <number> --watch
gh pr merge <number> --squash --delete-branch
```

### Find failing checks on a PR

```bash
gh pr checks <number> --json name,state,conclusion \
  | jq '.[] | select(.conclusion == "FAILURE")'
```

### Get PR diff

```bash
gh pr diff <number>
```
