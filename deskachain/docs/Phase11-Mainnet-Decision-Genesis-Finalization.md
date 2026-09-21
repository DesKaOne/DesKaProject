# Phase 11 — Mainnet Decision & Genesis Finalization

## Purpose

Move from documented readiness gates into explicit mainnet decision work. Phase 11 resolves the production items that Phase 10 intentionally left open.

This phase is decision and evidence work. It does not imply that mainnet is approved or launched.

## 11.1 Production network identity

Resolve and approve:
- final network name;
- final network ID;
- final chain ID;
- protocol version;
- production network boundaries;
- testnet/mainnet isolation rules.

No production node configuration should be generated from unresolved placeholders.

## 11.2 Genesis finalization

Produce the candidate production genesis from the approved configuration.

Evidence must include:
- deterministic generation inputs;
- genesis artifact;
- genesis hash;
- independent verification;
- immutable release reference.

Any change to genesis-critical inputs requires regeneration and re-verification.

## 11.3 Economic and protocol decision record

Resolve remaining mainnet-critical protocol/economic items, including where applicable:
- transaction fee policy;
- mining/block reward policy;
- difficulty parameters;
- issuance rules;
- native dIDR accounting boundaries;
- issued-asset governance;
- supply and accounting assumptions.

Undecided behavior remains explicitly TBD and cannot silently become a production guarantee.

## 11.4 Security review completion

Execute the security evidence required by Phase 10.4.

Record:
- review scope;
- reviewer or assessment authority;
- findings;
- remediation evidence;
- residual risks;
- accepted exceptions and owners.

A documented threat model is not by itself a formal security audit.

## 11.5 Production infrastructure confirmation

Confirm the actual production topology and operational ownership:
- seed/bootstrap nodes;
- full nodes;
- miner/operator role;
- Explorer/indexer;
- approved public RPC;
- backup/recovery infrastructure;
- monitoring/alerting;
- access control.

Each production node must have an independent identity and persistent state boundary.

## 11.6 External dependency decisions

Resolve each mainnet-critical external dependency as:
- production-ready;
- approved;
- blocked;
- not required.

Sandbox availability must not be treated as production partnership or approval.

Legal/regulatory requirements remain a separate decision track from engineering readiness.

## 11.7 Mainnet release candidate

Once 11.1–11.6 have sufficient evidence, assemble a release candidate containing:
- version;
- source commit;
- build/toolchain record;
- artifact checksum/signature where applicable;
- approved production configuration;
- genesis artifact/hash;
- migration/rollback procedure;
- verification evidence;
- known residual risks;
- approval status.

## 11.8 Final decision package

Prepare the package required by Phase 10.8:
- evidence revision references;
- blocker list;
- exception list;
- release authority;
- decision record;
- post-release verification plan.

The package must distinguish **READY FOR REVIEW**, **APPROVED**, and **RELEASED**.

## Current status

Phase 11 begins with unresolved production decisions inherited from Phase 10.1–10.8. Therefore the initial state is **OPEN**.

No mainnet launch is authorized by this document.

## Acceptance criteria

- Production identity is explicitly decided and approved by the appropriate authority.
- Genesis is generated, hashed, independently verified, and recorded.
- Mainnet-critical economic/protocol assumptions are explicitly decided.
- Required security review evidence is complete or its remaining blockers are explicit.
- Production infrastructure and ownership are confirmed.
- Critical external dependencies have explicit states.
- A traceable release candidate can be assembled.
- The final decision package is complete.
- No CI claim is inferred from documentation status.
