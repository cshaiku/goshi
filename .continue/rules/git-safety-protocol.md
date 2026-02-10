---
globs: |-
  **/.git/*
  **/*.git
description: >-
  This rule ensures repository safety by requiring explicit user approval before
  executing any git commands that could modify the repository state or history.
  The agent can suggest git commands and explain their purpose but must wait for
  user confirmation before execution.


  Allowed without approval:

  - git status

  - git log

  - git diff

  - git branch -l

  - Other read-only git commands


  Requires explicit approval:

  - git commit

  - git push

  - git pull

  - git merge

  - git rebase

  - git checkout

  - Any command that modifies repository state
alwaysApply: true
---

Never auto-execute git commands that modify repository state (commit, push, pull, merge, rebase, etc.) without explicit user approval.