# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records (ADRs) for the tviewmd project.

## What is an ADR?

An ADR is a document that captures an important architectural decision made in the project. The purpose is to:

- Document the context and problem being addressed
- Record the decision and its rationale
- Capture consequences and trade-offs
- Provide a historical record of why certain choices were made

## Format

ADRs in this project follow a consistent format:

```markdown
# ADR NNNN: Title

## Status

Accepted | Proposed | Deprecated | Superseded by ADR-NNNN

## Context

The problem being addressed and any background information.

## Problem

A clear statement of the problem or decision to be made.

## Decision

The chosen solution or approach.

## Consequences

### Positive
- Benefits of the decision

### Negative
- Trade-offs or drawbacks

## Alternatives Considered

Other options that were evaluated and why they were rejected.

## Related

Links to issues, PRs, or other relevant documents.

---

*Created: YYYY-MM-DD*
*Last Updated: YYYY-MM-DD*
*Status: ...*
```

## Numbering

ADRs are numbered sequentially with 4-digit numbers (0001, 0002, etc.).

## Lifecycle

1. **Proposed**: ADR is being discussed, not yet accepted
2. **Accepted**: ADR has been agreed upon and is being implemented
3. **Deprecated**: ADR has been superseded by a new decision
4. **Superseded**: ADR is replaced by another ADR

## Current ADRs

| Number | Title | Status | Date |
|--------|-------|--------|------|
| [0001](0001-streaming-support.md) | Streaming Markdown Rendering for LLM Responses | Accepted | 2026-08-07 |

## Creating a New ADR

1. Copy the template format from existing ADRs
2. Use the next available number
3. Fill in all sections
4. Submit as a PR for review
5. Update the table above

## Additional Resources

- [ADR GitHub Repository](https://github.com/joel-costigliola/adr-tag) - Original ADR specification
- [ADR Template](https://adr.github.io/madr/) - Markdown ADR template