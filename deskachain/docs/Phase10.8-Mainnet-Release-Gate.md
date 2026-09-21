# Phase 10.8 — Mainnet Release Gate

## Purpose

Provide the final engineering release gate that consolidates Phase 10.1 through 10.7 evidence and clearly separates technical readiness from the actual mainnet launch decision.

This document is a release gate, not an automatic launch authorization.

## 10.8.1 Required evidence set

The final gate must reference Phase 10.1 Network Identity & Genesis Decision; 10.2 Production Topology & Roles; 10.3 Release & Artifact Control; 10.4 Security & Threat-Model Gate; 10.5 Economic & Protocol Assumptions; 10.6 Operational Readiness; and 10.7 External Dependencies.

A release candidate must identify the exact evidence revision used for the decision.

## 10.8.2 Gate status model

Use explicit states:
- **OPEN** — evidence or decision work remains;
- **READY FOR REVIEW** — required evidence is assembled for authority review;
- **APPROVED** — authorized release decision has been recorded;
- **RELEASED** — the approved artifact has actually been released through the controlled procedure;
- **BLOCKED** — a blocker prevents progression.

Engineering completion alone must never be represented as APPROVED or RELEASED.

## 10.8.3 Mainnet blockers

Before an APPROVED state, explicitly review final network ID, final chain ID, production genesis artifact/hash, production topology, final RPC/P2P exposure, key custody, independent security review requirements, economic/protocol assumptions, backup/rollback evidence, release artifact integrity, external infrastructure/partner readiness, applicable legal/regulatory requirements, incident-response ownership, and release authority.

Any unresolved release prerequisite remains BLOCKED rather than silently complete.

## 10.8.4 Release authority

The release record must identify the person or authorized body responsible for the final mainnet decision. Engineering implementation and documentation do not themselves grant release authority.

Record approved artifact/version, source commit, network/chain/genesis identity, evidence package revision, residual risks, exceptions, decision, and timestamp.

## 10.8.5 Controlled release boundary

Once approved, release execution must use the approved artifact and configuration exactly as recorded. Any mainnet-critical change after approval requires re-evaluation and may return the candidate to READY FOR REVIEW.

Preserve checksum/signature verification, configuration provenance, genesis verification, backup/recovery point, rollback path, and post-release validation.

## 10.8.6 Post-release verification

Check process availability, network/chain/genesis identity, peer connectivity, chain height/tip consistency, transaction/mempool behavior as applicable, monitoring, Explorer/indexer state when enabled, and unexpected configuration or artifact drift.

Record the verification result.

## 10.8.7 Stop conditions

Stop or return to review when artifact integrity fails, network/chain/genesis identity differs, configuration differs, persistent-state compatibility is uncertain, a release-blocking security finding is open, a required dependency is not production-ready, required legal/regulatory approval is absent where applicable, or post-release validation detects unexpected behavior.

## 10.8.8 Final decision record

| Field | Required |
|---|---|
| Release version | Yes |
| Source commit | Yes |
| Artifact checksum/signature | Yes, as applicable |
| Network ID | Yes |
| Chain ID | Yes |
| Genesis artifact/hash | Yes |
| Evidence package revision | Yes |
| Security status | Yes |
| Operational status | Yes |
| External dependency status | Yes |
| Known residual risks | Yes |
| Exceptions | If any |
| Release authority | Yes |
| Decision | Yes |
| Decision timestamp | Yes |

## 10.8.9 Current gate boundary

At completion of Phase 10 documentation, the project has defined the engineering gates but must not infer that all production decisions are approved.

Phase 10.1 records production network identity/genesis as not yet approved, while Phase 10.4–10.7 identify decisions and evidence requiring explicit review.

Therefore the documented state remains **READY FOR REVIEW**, not APPROVED or RELEASED, until the required authorities and evidence complete their respective decisions.

## Acceptance criteria

- Phase 10.1–10.7 evidence is connected to the final gate.
- Release states are unambiguous.
- Mainnet blockers are explicit.
- Release authority is explicit.
- Controlled release and rollback boundaries are defined.
- Post-release verification is defined.
- Stop conditions are explicit.
- Documentation does not falsely authorize mainnet launch.
- No consensus implementation change is required by this documentation phase.
- CI status remains separate from the release decision.
