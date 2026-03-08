---
description: Create a well-structured git commit from staged changes. Analyzes the diff, writes a conventional commit message, and commits.
tools:
  - Bash
---

# Command: Commit

## Goal

Create a clear, descriptive git commit from the currently staged changes. The commit message should explain **why** the change was made, not just what changed.

## Process

### 1. Assess staged changes

```bash
git diff --cached --stat
git diff --cached
```

If nothing is staged, check for unstaged changes:

```bash
git status
```

If there are unstaged changes, ask the user which files to include before proceeding.

### 2. Analyze the diff

Identify:
- What type of change is this? (feat, fix, refactor, chore, docs, test, style)
- What is the primary motivation or intent?
- Are there any caveats or notable side effects?

### 3. Write the commit message

Follow Conventional Commits format:

```
<type>(<optional scope>): <short summary>

<optional body — explain WHY, not WHAT>
```

Rules:
- Summary line: max 72 characters, imperative mood ("add", not "added")
- Body: only if the change needs more context; explain the reasoning
- Do not mention file names or obvious implementation details in the summary

### 4. Commit

```bash
git commit -m "$(cat <<'EOF'
<your message here>

Co-Authored-By: Claude <noreply@anthropic.com>
EOF
)"
```

### 5. Confirm

Show the result of `git log -1 --oneline` so the user can verify the commit was created correctly.
