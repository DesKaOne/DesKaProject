# Phase 11.4 — Security Review Completion

## Purpose

Define the evidence package required to close the security gate inherited from Phase 10.4.

A threat model and engineering hardening record are useful evidence, but they are not by themselves a formal independent security review or audit.

## Required review scope

The review must cover, as applicable:

- consensus and block validation;
- transaction validation and accounting;
- RPC exposure and authorization;
- P2P identity and network boundaries;
- wallet/private-key handling;
- mining/operator controls;
- Explorer/indexer read-model boundaries;
- backups and recovery;
- build and release artifacts;
- dependency and supply-chain exposure;
- operational access and secrets.

## Findings lifecycle

Use the existing lifecycle:

`Open — Blocker/High/Medium/Low` → `Mitigated — awaiting verification` → `Accepted — explicit owner` → `Closed — evidence verified`

Every finding should record severity, owner, affected component, remediation, verification evidence, and residual risk where applicable.

## Independent assessment

Record:

- reviewer or assessment authority;
- scope and methodology;
- assessment date/version;
- source commit or artifact assessed;
- findings;
- remediation evidence;
- exceptions and explicit owners;
- final assessment statement.

The assessed artifact must be identifiable and immutable enough to reproduce the review boundary.

## Mainnet security blockers

At minimum, explicitly resolve or own:

- key custody;
- operator access;
- public RPC/P2P exposure;
- backup protection and retention;
- release/signing key custody;
- critical dependency risk;
- residual high/blocker findings;
- incident escalation.

## Gate states

- **OPEN** — review not complete;
- **IN REVIEW** — assessment active;
- **REMEDIATION** — findings being addressed;
- **READY FOR APPROVAL** — evidence complete with explicit residual risks;
- **APPROVED** — authorized security gate;
- **BLOCKED** — unresolved critical issue.

## Acceptance criteria

- Review scope is explicit.
- Reviewer/assessment authority is recorded.
- Findings and remediation are traceable.
- Residual risks and exceptions have owners.
- No formal audit is claimed without corresponding evidence.
- Security approval remains separate from CI status and mainnet release authority.
