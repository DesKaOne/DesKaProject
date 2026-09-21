# Phase 10 — Pre-Mainnet / Mainnet Readiness

## Purpose

Phase 10 is the decision-preparation layer after Phase 8 testnet engineering and Phase 9 testnet operations.

It does not launch mainnet. It establishes the evidence, decisions, release controls, and unresolved questions required before a separate mainnet release gate.

## Scope

### 10.1 Network identity and genesis decision
- Finalize the intended mainnet network ID and chain ID.
- Define the production genesis artifact and immutable genesis identity.
- Record native unit semantics: dIDR remains the IndoChain native unit; the existing project decision is 1 dIDR = 1000 Rupiah for ecosystem accounting.
- Keep issued assets separate from native dIDR.

### 10.2 Production topology and roles
Define:
- seed/bootstrap nodes;
- full nodes;
- miner/operator roles;
- Explorer/indexer nodes;
- RPC/public-facing boundaries;
- redundancy and recovery ownership.

### 10.3 Release and artifact control
Define:
- versioning;
- reproducible build expectations;
- release artifact checksums/signatures;
- configuration provenance;
- migration policy;
- rollback policy;
- release evidence archive.

### 10.4 Security and threat-model gate
Consolidate:
- threat model;
- key custody;
- private-key handling;
- RPC/P2P exposure review;
- formal security review/audit requirements;
- unresolved security findings.

### 10.5 Economic and protocol assumptions
Document:
- consensus assumptions;
- mining/difficulty assumptions;
- transaction fee semantics;
- native dIDR accounting;
- issued-asset boundaries;
- known limitations and operational assumptions.

This is documentation/evidence only until separately approved.

### 10.6 Operational readiness
Define final:
- deployment procedure;
- backup/restore;
- upgrade/rollback;
- monitoring;
- incident response;
- operator access;
- observation and alerting.

### 10.7 External dependencies
Inventory:
- infrastructure providers;
- RPC/hosting;
- Explorer hosting;
- DNS/domain;
- wallet/client integrations;
- payment/financial integrations;
- third-party services;
- partner dependencies.

Each dependency gets owner, purpose, failure mode, and readiness status.

### 10.8 Mainnet release gate
Create a final evidence package containing:
- approved decisions;
- verified artifacts;
- unresolved risks;
- operational runbooks;
- security evidence;
- dependency status;
- rollback plan;
- explicit release authority.

## Phase 10 working principles

1. Testnet evidence is not automatically mainnet approval.
2. Mainnet identifiers and genesis must be explicit and immutable after approval.
3. Consensus changes require their own implementation and test gate.
4. Monitoring and Explorer remain non-consensus read models.
5. Secrets and private keys never belong in repository evidence.
6. Mainnet release requires an explicit approval gate; no document in this phase is itself authorization.
7. CI status and operational evidence remain separate records.

## Initial exit criteria

Phase 10 should not be marked complete merely because documents exist.

The gate requires:
- network/genesis decisions recorded;
- production topology approved;
- release artifact controls defined;
- security/threat-model evidence recorded;
- economic/protocol assumptions documented;
- operational runbooks reviewed;
- external dependencies inventoried;
- unresolved blockers explicitly tracked;
- final release authority identified;
- a separate mainnet release decision recorded.

## Phase 10.1 immediate deliverable

The first deliverable is the network identity/genesis decision record. Until approved, production values remain placeholders and testnet identity must not be reused accidentally.
