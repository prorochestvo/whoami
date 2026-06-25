# Agent memory

Persistent, project-scoped memory written by each subagent into its own subdirectory
(`site-architect/`, `site-engineer/`, `site-reviewer/`, `site-fixer/`).

## What lives here

Each subdirectory contains a `MEMORY.md` index plus per-topic files following the
memory schema (see the "Persistent Agent Memory" section inside each agent file).
Typical entry types:

- `user_*.md` — facts about the user (role, preferences, expertise)
- `feedback_*.md` — guidance the user has given about how to approach work
- `project_*.md` — ongoing initiatives, deadlines, in-flight refactors
- `reference_*.md` — pointers to external systems (Linear, Grafana, runbooks)

## Commit policy

**Commit `MEMORY.md` and per-topic files.** They encode shared team knowledge and
should travel with the repo.

**Do not commit** ad-hoc scratch files agents may write here for working state —
those belong under `scratch/` subdirectories ignored via `.gitignore`.
