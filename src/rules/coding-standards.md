# Coding Standards

This file defines coding standards that apply across all services in the repository. These rules are enforced during code review and CI.

## General Principles

- **Clarity over cleverness** — Code is read far more than it is written. Prefer the obvious solution.
- **Fail loudly** — Errors should surface, not be silently swallowed or converted to zero values.
- **Small, focused units** — Functions and modules should do one thing well.
- **No premature abstraction** — Don't create a generic utility for a single use case. Abstract when the third instance appears.

## Naming

- Names should describe what something **is** or **does**, not how it works internally.
- Boolean variables/functions: use `is`, `has`, `can`, `should` prefixes (`isLoading`, `hasPermission`).
- Avoid abbreviations unless they are universally understood in context (`url`, `id`, `ctx` are fine; `rq`, `hdlr` are not).
- Avoid generic names like `data`, `info`, `thing`, `obj`, `tmp` unless scope is extremely narrow.

## Error Handling

- Never ignore returned errors.
- Wrap errors with context: `fmt.Errorf("loading user %s: %w", id, err)` (Go), `throw new Error("loading user: " + cause)` (TS).
- Distinguish between user-facing errors (safe to expose) and internal errors (must not leak internals).
- Avoid `panic` in Go except at program startup for unrecoverable misconfigurations.

## Functions

- Keep functions short — if a function doesn't fit on one screen, consider splitting it.
- Functions should have a single level of abstraction per body (don't mix high-level orchestration with low-level detail in the same function).
- Prefer explicit return values over modifying input arguments.

## Comments

- Don't comment **what** code does — the code should explain itself.
- Do comment **why** when a decision is non-obvious or counterintuitive.
- TODO comments must include a ticket reference: `// TODO(EXT-123): remove once migration is complete`.

## Testing

- Tests live alongside source files or in a `__tests__` / `_test.go` sibling.
- Test names must describe the behavior being tested, not the function name: `"returns 404 when user does not exist"` not `"TestGetUser"`.
- Avoid testing implementation details — test inputs and outputs, not internal state.
- Each test should be independent. Do not rely on execution order.

## File Organization

- One primary export per file (TypeScript).
- Group related functionality — don't scatter helpers across unrelated files.
- Keep files under ~300 lines. If a file grows beyond this, evaluate whether it's doing too much.

## Dependencies

- Do not add a dependency to solve a problem that can be solved with 10 lines of code.
- All new dependencies must be reviewed for maintenance status, license, and security track record.
- Pin exact versions in TypeScript (`"1.2.3"`, not `"^1.2.3"`).
