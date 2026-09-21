# Phase 11.7 — Mainnet Release Candidate

## Purpose

Define the controlled release-candidate package assembled after sufficient evidence exists across Phase 11.1–11.6.

A release candidate is not an approved release.

## Candidate identity

Record:

- project/release version;
- source commit;
- build timestamp;
- compiler/toolchain;
- target platform/architecture;
- protocol version;
- production configuration reference;
- genesis artifact reference;
- genesis hash.

## Artifact integrity

For every release artifact record:

- filename;
- artifact type;
- platform/architecture;
- checksum;
- optional signature and verification method;
- source reference;
- generation/build metadata.

Private signing keys must never be stored in the repository or release evidence.

## Required evidence

The candidate package should contain:

1. approved production identity;
2. verified production genesis artifact/hash;
3. economic/protocol decision record;
4. security review evidence;
5. production topology/ownership;
6. external dependency decisions;
7. migration/upgrade notes;
8. rollback procedure;
9. backup/recovery evidence;
10. verification results;
11. known residual risks and exceptions;
12. release authority record.

## Verification gate

Before approval, verify at minimum:

- binary starts with approved configuration;
- network/chain/genesis identity matches evidence;
- chain validation succeeds;
- node health and peer boundaries behave as expected;
- approved RPC exposure is enforced;
- Explorer/indexer operates as a read model;
- monitoring is read-only;
- backup/restore and rollback procedures are available;
- artifact checksums/signatures verify where used.

## Release states

`DRAFT → CANDIDATE → VERIFIED → APPROVED → RELEASED`

A candidate may return to `BLOCKED` when verification fails or a critical dependency changes.

## Stop conditions

Do not promote the candidate when:

- production identity is unresolved;
- genesis hash is unresolved or mismatched;
- critical security findings remain unresolved without explicit authority;
- required infrastructure is unavailable;
- critical external dependencies are blocked;
- rollback/recovery is unverified;
- release evidence is incomplete.

## Acceptance criteria

- Candidate is traceable to source and build metadata.
- Artifact integrity is verifiable.
- Production identity and genesis are traceable.
- Cross-phase evidence is assembled.
- Approval is still a separate decision from candidate creation.
