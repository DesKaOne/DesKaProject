# Phase 11.8 — Final Mainnet Decision Package

## Purpose

Assemble the final evidence and decision package required to move IndoChain from technical readiness review toward an explicit mainnet release decision.

This package does not itself authorize release.

## Evidence index

The package must reference the final revisions of:

- Phase 11.1 production network identity;
- Phase 11.2 genesis finalization;
- Phase 11.3 economic/protocol decisions;
- Phase 11.4 security review;
- Phase 11.5 production infrastructure;
- Phase 11.6 external dependencies;
- Phase 11.7 release candidate;
- relevant Phase 8–10 operational, release, and readiness evidence.

## Blocker register

Record every unresolved item with:

- blocker ID;
- description;
- severity;
- owner;
- evidence/reference;
- required action;
- decision deadline if applicable;
- release impact.

No blocker should disappear merely because a release candidate exists.

## Exception register

For every accepted exception record:

- exception ID;
- affected area;
- rationale;
- residual risk;
- owner;
- approving authority;
- expiry/review condition where applicable.

## Release authority

The final decision record must identify the authority empowered to move the state from review to approval/release.

Engineering documentation alone does not substitute for explicit release authority.

## Decision states

Use exactly one final state at each decision point:

- **READY FOR REVIEW** — evidence assembled; approval not granted;
- **APPROVED** — authorized release decision recorded;
- **RELEASED** — controlled release executed and post-release checks completed;
- **BLOCKED** — a release-critical issue prevents promotion.

Do not use `APPROVED` or `RELEASED` as a shorthand for testnet readiness.

## Controlled release boundary

If approval is granted, the release procedure must identify:

1. exact artifact/version;
2. source commit;
3. production configuration;
4. genesis artifact/hash;
5. deployment order;
6. health and chain validation gates;
7. rollback trigger;
8. incident escalation;
9. post-release verification owner.

## Post-release verification

Immediately after a controlled release, verify:

- network and chain identity;
- genesis hash;
- peer topology;
- block production/validation;
- transaction processing;
- RPC exposure;
- monitoring;
- Explorer/indexer synchronization;
- backup/recovery visibility;
- incident/alert channels.

## Stop conditions

Release must stop on identity mismatch, genesis mismatch, chain validation failure, unexpected peer/network exposure, critical security event, failed recovery boundary, or any release-critical dependency failure.

## Current status

Phase 11 is an open decision process. Production identity and genesis remain unresolved until explicit approval. Therefore this package must not be interpreted as mainnet authorization.

## Acceptance criteria

- All Phase 11 evidence is traceable.
- Blockers and exceptions are explicit.
- Release authority is identified.
- Candidate and approval states are distinct.
- Post-release verification is defined.
- No mainnet launch is implied by documentation completion.
- CI status remains a separate tracking signal and is not represented as test execution evidence.
