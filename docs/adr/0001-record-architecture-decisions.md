# 0001. Record architecture decisions

- Status: accepted
- Date: 2026-10-20

## Context

Decisions for the MVP were made in design discussions and during implementation.
Without a record, nobody can tell later why `gd` behaves the way it does.

## Decision

Every decision that changes behaviour, file formats or interfaces is recorded as
an ADR in `docs/adr/`, numbered, using [template.md](template.md). Records are
immutable; a changed decision supersedes the old record with a new one.

Design specs and plans (`docs/superpowers/`) describe *what to build*; ADRs
describe *why it is this way*.

## Consequences

One small file per decision. Reviews of PRs that change behaviour should ask for
an ADR.
