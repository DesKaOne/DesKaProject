# Phase 10.3 — Release & Artifact Control

## Purpose

Define the release controls required to move an approved IndoChain build from engineering state toward a controlled mainnet candidate.

This phase controls artifacts and release evidence. It does not authorize a mainnet launch.

## Release identity

Every candidate release must have a unique, traceable identity:
- semantic/project version;
- source commit;
- build timestamp;
- build environment/toolchain record;
- target platform/architecture;
- configuration profile;
- protocol version;
- release status.

The release record must identify exactly which source and configuration produced an artifact.

## Reproducible build expectation

Where supported, the same source and declared build inputs should produce equivalent artifacts.

Record compiler/toolchain version, dependency lock/module state, build flags, target OS/architecture, and generated files affecting the artifact. Known non-reproducible inputs must be documented.

## Artifact integrity

For every release candidate, record artifact filename, type, platform/architecture, cryptographic checksum, source commit, version, and release timestamp.

Checksums must be calculated from the exact distributed artifact.

If signing is introduced, record signing method, public verification material, signature artifact, verification procedure, and key custody/rotation policy.

Private signing keys must never be committed or included in release evidence.

## Configuration provenance

A release candidate must identify:
- binary and protocol version;
- network profile;
- configuration source;
- genesis artifact/hash;
- bootstrap/seed configuration;
- RPC/P2P binding policy.

Testnet configuration must not silently become mainnet configuration.

## Migration policy

If a release changes persistent on-disk formats:
1. document the migration;
2. define the pre-migration backup point;
3. define compatibility expectations;
4. define validation after migration;
5. define rollback/recovery behavior;
6. test on disposable state before production use.

Do not assume an older binary can safely open newer persistent state.

If no persistent-format migration exists, record that explicitly.

## Rollback policy

Rollback must identify the previous known-good binary/configuration, protected pre-upgrade backup, compatibility status, and restoration procedure.

For uncertain persistent-format compatibility:
- stop the node;
- preserve current state as evidence;
- restore the pre-upgrade backup into a new datadir;
- run the previous binary;
- validate chain/network/genesis identity;
- run health and smoke checks.

## Release evidence package

Each mainnet candidate should archive:
1. release metadata;
2. binary artifacts;
3. checksums/signatures when applicable;
4. source commit/reference;
5. build/toolchain record;
6. approved configuration reference;
7. genesis artifact/hash reference;
8. migration notes;
9. rollback procedure;
10. verification results;
11. known limitations and unresolved findings;
12. explicit approval status.

Do not put private keys, credentials, or secret configuration values in the archive.

## Verification gate

Before an artifact becomes a mainnet candidate:
- [ ] artifact runs on intended target;
- [ ] version/source commit match release record;
- [ ] checksum matches distributed artifact;
- [ ] configuration provenance is recorded;
- [ ] network/chain/genesis identity matches the approved decision record;
- [ ] persistent-state migration is understood;
- [ ] rollback path is documented;
- [ ] security/release findings are recorded;
- [ ] release evidence package is complete.

## Release state model

Use explicit states:

draft -> candidate -> verified -> approved -> released

A candidate must not be described as approved or released merely because it was built.

## Mainnet boundary

A technically complete artifact may still await mainnet identity/genesis approval, security review, operational approval, external dependency readiness, legal/regulatory review where applicable, and explicit release authority.

No state in this document constitutes launch authorization.

## Acceptance criteria

- Release artifacts are uniquely traceable.
- Build provenance is recorded.
- Checksums are part of release evidence.
- Signing requirements are defined without exposing private keys.
- Configuration provenance is explicit.
- Persistent-format migration and rollback boundaries are explicit.
- A complete release evidence package is defined.
- Release states distinguish candidate from approved/released.
- No consensus implementation change is required.
- CI status remains separate from artifact/release evidence.
