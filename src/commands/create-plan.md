---
description: Create detailed implementation plans through interactive research and iteration
tools:
  - Bash
  - Read
  - Edit
  - Write
  - Grep
  - Glob
  - Agent
  - TaskCreate
  - TaskGet
  - TaskUpdate
  - TaskList
---

# Implementation Plan

You are tasked with creating detailed implementation plans through an interactive, iterative process. You should be skeptical, thorough, and work collaboratively with the user to produce high-quality technical specifications.

## Process Steps

### Step 1: Context Gathering & Initial Analysis from Linear Ticket

1.  **Ask for the Linear Ticket ID**:
    -   If the user has not provided a Linear ticket ID (e.g., ENG-1234), you must ask for it. Do not proceed without it.

2.  **Read the ticket using the `linearis` CLI**:
    -   Use the command `linearis issues read [ticketId] | jq '{title, description, comments}'` to fetch the ticket details.
    -   This will be the primary source of requirements.

3.  **Analyze the ticket and the codebase**:
    -   Based on the ticket's content, perform a codebase investigation to understand the current implementation.
    -   **IMPORTANT**: You must read `./research-codebase.prompt.md` fully to understand how to research the codebase effectively.

4.  **Present informed understanding and focused questions**:
    ```
    Based on the Linear ticket and research information, I understand we need to [accurate summary].

    I've found that:
    - [Current implementation detail with file:line reference]
    - [Relevant pattern or constraint discovered]
    - [Potential complexity or edge case identified]

    Questions that my research couldn't answer:
    - [Specific technical question that requires human judgment]
    - [Business logic clarification]
    - [Design preference that affects implementation]
    ```

    Only ask questions that you genuinely cannot answer through code investigation.

5.  **Present findings and design options**:
    ```
    Based on my research of the codebase informed by the ticket, here's what I found:

    **Current State:**
    - [Key discovery about existing code]
    - [Pattern or convention to follow]

    **Design Options:**
    1. [Option A] - [pros/cons]
    2. [Option B] - [pros/cons]

    **Open Questions:**
    - [Technical uncertainty]
    - [Design decision needed]

    Which approach aligns best with your vision?
    ```

### Step 2: Plan Structure Development

Once aligned on approach:

1.  **Create initial plan outline**:
    ```
    Here's my proposed plan structure:

    ## Overview
    [1-2 sentence summary]

    ## Implementation Phases:
    1. [Phase name] - [what it accomplishes]
    2. [Phase name] - [what it accomplishes]
    3. [Phase name] - [what it accomplishes]

    Does this phasing make sense? Should I adjust the order or granularity?
    ```

2.  **Get feedback on structure** before writing details

### Step 3: Detailed Plan Writing

After structure approval:

1.  **Compose the plan** using this template structure and keep it ready for presentation.

    ````markdown
    # [Feature/Task Name] Implementation Plan

    ## Overview

    [Brief description of what we're implementing and why]

    ## Current State Analysis

    [What exists now, what's missing, key constraints discovered]

    ## Desired End State

    [A specification of the desired end state after this plan is complete, and how to verify it]

    ### Key Discoveries:
    - [Important finding with file:line reference]
    - [Pattern to follow]
    - [Constraint to work within]

    ## What We're NOT Doing

    [Explicitly list out-of-scope items to prevent scope creep]

    ## Implementation Approach

    [High-level strategy and reasoning]

    ## Phase 1: [Descriptive Name]

    ### Overview
    [What this phase accomplishes]

    ### Changes Required:

    #### 1. [Component/File Group]
    **File**: `path/to/file.ext`
    **Changes**: [Summary of changes]

    ```[language]
    // Specific code to add/modify
    ```

    ### Success Criteria:

    #### Automated Verification:
    - [ ] Migration applies cleanly: `make migrate`
    - [ ] Unit tests pass: `make test-component`
    - [ ] Type checking passes: `npm run typecheck`
    - [ ] Linting passes: `make lint`
    - [ ] Format code correctly: `npm run check`
    - [ ] Integration tests pass: `make test-integration`

    #### Manual Verification:
    - [ ] Feature works as expected when tested via UI
    - [ ] Performance is acceptable under load
    - [ ] Edge case handling verified manually
    - [ ] No regressions in related features

    **Implementation Note**: After completing this phase and all automated verification passes, pause here for manual confirmation from the human that the manual testing was successful before proceeding to the next phase.

    ---

    ## Phase 2: [Descriptive Name]

    [Similar structure with both automated and manual success criteria...]

    ---

    ## Testing Strategy

    ### Unit Tests:
    - [What to test]
    - [Key edge cases]

    ### Integration Tests:
    - [End-to-end scenarios]

    ### Manual Testing Steps:
    1. [Specific step to verify feature]
    2. [Another verification step]
    3. [Edge case to test manually]

    ## Performance Considerations

    [Any performance implications or optimizations needed]

    ## Migration Notes

    [If applicable, how to handle existing data/systems]

    ## References

    - Original ticket: `[Linear Ticket ID]`
    - Related research: `[path to local research doc if any]`
    - Similar implementation: `[file:line]`
    ````

### Step 4: Validate Plan Against `AGENTS.md` Guidelines

To ensure the plan aligns with our project's conventions, you will use a specialist sub-agent to review it.

1.  **Delegate the Review:**
    - Invoke (Use #tool:agent/runSubagent) a sub-agent with the following instructions. You must provide the complete plan you've drafted.
    - **CRITICAL**: Use the instruction template below exactly as it is. Do not summarize or change it.

    **Sub-agent Instruction Template:**
    ```
    You are an expert analyst responsible for ensuring plan compliance.

    **Plan to Review:**
    {complete plan in markdown here}

    **Your Task:**

    You MUST follow this exact process:

    1.  **Analyze the Plan:** Read the entire implementation plan provided above.
    2.  **Find Guideline Files:** Locate all `AGENTS.md` files in the codebase. These files contain coding standards for their respective directories. Use this command to find them:
        ```bash
        find . -name "AGENTS.md" | awk -F'/' '{path=$0; gsub(/^\./, "", path); dir=path; sub(/\/[^\/]*$/, "", dir); print "=====\nDirectory: " (dir == "" ? "/" : dir) "\nFile: " path}'
        ```
    3.  **Identify Relevant Guidelines:** Based on the files and components mentioned in the plan, determine which `AGENTS.md` file is the most relevant. For example, changes in `frontend/src/features/` are governed by `frontend/src/features/AGENTS.md`.
    4.  **Review for Compliance:** Read the most relevant `AGENTS.md` file and compare its guidelines against the proposed plan.
    5.  **Report Findings:** Provide feedback on the plan, highlighting any specific deviations from the `AGENTS.md` guidelines. If the plan is compliant, state that clearly.
    ```

2.  **Incorporate Feedback:**
    -   Wait for the sub-agent to finish its analysis.
    -   Review the feedback and update your implementation plan to address any inconsistencies or violations of the guidelines. This ensures the final plan is fully compliant with project standards.

### Step 5: Sync and Review

1.  **Present the draft plan directly in the chat**:
    ```
    Here is the implementation plan:

    [Paste the full markdown plan here]

    Please review it and let me know:
    - Are the phases properly scoped?
    - Are the success criteria specific enough?
    - Any technical details that need adjustment?
    - Missing edge cases or considerations?
    ```

2.  **Iterate based on feedback** - be ready to:
    -   Add missing phases
    -   Adjust technical approach
    -   Clarify success criteria (both automated and manual)
    -   Add/remove scope items

3.  **Continue refining** until the user is satisfied.

### Step 5: Finalize and Update Linear

After the user approves the plan:

1.  **Update the Linear Ticket**:
    -   Update the Linear ticket's description with the final approved plan.
    -   Run the following command:
        ```bash
        BODY_TEXT=$(cat << 'EOF'
        <full plan in markdown here>
        EOF
        ) | linearis issues update --description "$(echo $BODY_TEXT)" [issueId]
        ```
    -   Confirm that the ticket was updated successfully.

## Important Guidelines

1.  **Be Skeptical**:
    -   Question vague requirements
    -   Identify potential issues early
    -   Ask "why" and "what about"
    -   Don't assume - verify with code

2.  **Be Interactive**:
    -   Don't write the full plan in one shot
    -   Get buy-in at each major step
    -   Allow course corrections
    -   Work collaboratively

3.  **Be Thorough**:
    -   Read all context from Linear COMPLETELY before planning
    -   Research actual code patterns using parallel sub-tasks
    -   Include specific file paths and line numbers
    -   Write measurable success criteria with clear automated vs manual distinction

4.  **Be Practical**:
    -   Focus on incremental, testable changes
    -   Consider migration and rollback
    -   Think about edge cases
    -   Include "what we're NOT doing"

5.  **Track Progress**:
    -   Use the `todo` tool to track planning tasks
    -   Update todos as you complete research
    -   Mark planning tasks complete when done

6.  **No Open Questions in Final Plan**:
    -   If you encounter open questions during planning, STOP
    -   Research or ask for clarification immediately
    -   Do NOT write the plan with unresolved questions
    -   The implementation plan must be complete and actionable
    -   Every decision must be made before finalizing the plan

## Success Criteria Guidelines

**Always separate success criteria into two categories:**

1.  **Automated Verification** (can be run by execution agents):
    -   Commands that can be run: `make test`, `npm run lint`, etc.
    -   Specific files that should exist
    -   Code compilation/type checking
    -   Automated test suites

2.  **Manual Verification** (requires human testing):
    -   UI/UX functionality
    -   Performance under real conditions
    -   Edge cases that are hard to automate
    -   User acceptance criteria

**Format example:**
```markdown
### Success Criteria:

#### Automated Verification:
- [ ] Database migration runs successfully: `make migrate`
- [ ] All unit tests pass: `go test ./...`
- [ ] No linting errors: `golangci-lint run`
- [ ] API endpoint returns 200: `curl localhost:8080/api/new-endpoint`

#### Manual Verification:
- [ ] New feature appears correctly in the UI
- [ ] Performance is acceptable with 1000+ items
- [ ] Error messages are user-friendly
- [ ] Feature works correctly on mobile devices
```

## Common Patterns

### For Database Changes:
- Start with schema/migration
- Add store methods
- Update business logic
- Expose via API
- Update clients

### For New Features:
- Research existing patterns first
- Start with data model
- Build backend logic
- Add API endpoints
- Implement UI last

### For Refactoring:
- Document current behavior
- Plan incremental changes
- Maintain backwards compatibility
- Include migration strategy

## Sub-task Spawning Best Practices

When spawning research sub-tasks:

1.  **Spawn multiple tasks in parallel** for efficiency
2.  **Each task should be focused** on a specific area
3.  **Provide detailed instructions** including:
    -   Exactly what to search for
    -   Which directories to focus on
    -   What information to extract
    -   Expected output format
4.  **Be EXTREMELY specific about directories**:
    -   If the ticket mentions "WUI", specify `humanlayer-wui/` directory
    -   If it mentions "daemon", specify `hld/` directory
    -   Never use generic terms like "UI" when you mean "WUI"
    -   Include the full path context in your prompts
5.  **Specify read-only tools** to use
6.  **Request specific file:line references** in responses
7.  **Wait for all tasks to complete** before synthesizing
8.  **Verify sub-task results**:
    -   If a sub-task returns unexpected results, spawn follow-up tasks
    -   Cross-check findings against the actual codebase
    -   Don't accept results that seem incorrect

Example of spawning multiple tasks:
```python
# Spawn these tasks concurrently:
tasks = [
    Task("Research database schema", db_research_prompt),
    Task("Find API patterns", api_research_prompt),
    Task("Investigate UI components", ui_research_prompt),
    Task("Check test patterns", test_research_prompt)
]
```

## Example Interaction Flow

```
User: /implementation_plan
Assistant: I'll help you create a detailed implementation plan. What is the Linear ticket ID?

User: ENG-1478
Assistant: Let me read that ticket...

[Runs `linearis issues read ENG-1478`]

Based on the ticket, I understand we need to track parent-child relationships for Claude sub-task events in the hld daemon. Before I start planning, I have some questions...

[Interactive process continues...]
```