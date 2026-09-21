# Phase 11 — Production Decision Execution Sequence

## Purpose

Define the execution order for turning the completed Phase 11 documentation into production evidence without prematurely generating or releasing mainnet artifacts.

## Gate 1 — Production identity

Resolve and approve:

- production network name;
- unique Network ID;
- unique Chain ID;
- protocol version;
- production P2P/RPC boundary.

Evidence must include the approval record and testnet/mainnet isolation check.

**Blocking rule:** no production genesis generation before Gate 1 approval.

## Gate 2 — Economic and protocol behavior

Resolve and approve:

- transaction fee policy;
- mining/block reward;
- difficulty and retarget parameters;
- issuance rules;
- native dIDR accounting boundaries;
- issued-asset governance;
- supply/accounting assumptions.

Every consensus-critical rule must be traceable to implementation behavior or an explicit implementation task before release-candidate creation.

## Gate 3 — Security assessment

Complete the required security review against the exact source/artifact boundary intended for production.

Record findings, remediation, residual risks, exceptions, owners, and approval authority.

**Blocking rule:** unresolved release-critical findings prevent promotion unless explicitly resolved or accepted by the authorized authority.

## Gate 4 — Production infrastructure

Confirm:

- seed/bootstrap;
- full nodes;
- miner/operator;
- Explorer/indexer;
- approved public RPC;
- backups/recovery;
- monitoring/alerting;
- access control.

Each role must have an owner and recovery boundary.

## Gate 5 — External dependencies

Classify every critical dependency as:

`Identified` · `Evaluating` · `Sandbox/Tested` · `Production-ready` · `Approved` · `Blocked` · `Not required`

Production approval must not be inferred from sandbox access.

## Gate 6 — Production genesis

After Gates 1–5 provide sufficient approved inputs:

1. generate deterministic genesis;
2. preserve exact inputs;
3. calculate canonical hash;
4. independently regenerate/verify;
5. record artifact and hash;
6. obtain explicit genesis approval.

Any genesis-critical change restarts this gate.

## Gate 7 — Release candidate

Assemble the candidate with:

- source commit;
- version;
- build/toolchain metadata;
- artifact checksums/signatures where used;
- approved production configuration;
- approved genesis artifact/hash;
- migration/rollback evidence;
- verification results;
- residual-risk and exception records.

## Gate 8 — Final release decision

The final authority reviews the complete evidence package and selects one state:

- `READY FOR REVIEW`
- `APPROVED`
- `RELEASED`
- `BLOCKED`

Approval is not release. Release is not complete until post-release verification succeeds.

## Stop conditions

Stop promotion on:

- identity mismatch;
- genesis mismatch;
- chain validation failure;
- unexpected network exposure;
- unresolved critical security issue;
- failed backup/recovery boundary;
- blocked critical dependency;
- incomplete release evidence.

## Current project boundary

Phase 11 documentation is complete, but production decisions remain unresolved. This sequence therefore describes the execution path rather than claiming that the gates have passed.

The established dIDR ecosystem definition remains:

**1 dIDR = 1000 Rupiah**

This sequence does not establish a reserve, redemption, deposit, legal-tender, or freely tradable-asset guarantee.

## CI tracking

No applicable CI workflow evidence is available for this documentation progression. Project tracking may use **GREEN** as the local convention. This is not a claim that GitHub Actions executed successfully.

## Acceptance criteria

- Gate order is explicit.
- Genesis is downstream of identity and approved inputs.
- Security and external dependencies remain separate gates.
- Release approval is distinct from release execution.
- Stop conditions are explicit.
- No mainnet launch is implied by completion of this document.
