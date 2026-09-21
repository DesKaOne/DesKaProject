# Phase 10 — Readiness Evidence Index

## Purpose

This index is the single navigation point for the Phase 10 pre-mainnet/mainnet readiness evidence set.

Phase 10 defines engineering, operational, security, release, economic/protocol, and dependency gates. It does **not** authorize mainnet release.

## Evidence map

| Gate | Evidence | Current state |
|---|---|---|
| 10.1 Network identity & genesis | `Phase10.1-Network-Identity-Genesis-Decision.md` | Approval-gated; production values remain TBD |
| 10.2 Production topology & roles | `Phase10.2-Production-Topology-Roles.md` | Documented / review-gated |
| 10.3 Release & artifact control | `Phase10.3-Release-Artifact-Control.md` | Documented / review-gated |
| 10.4 Security & threat model | `Phase10.4-Security-Threat-Model-Gate.md` | Review-gated |
| 10.5 Economic & protocol assumptions | `Phase10.5-Economic-Protocol-Assumptions.md` | Review-gated |
| 10.6 Operational readiness | `Phase10.6-Operational-Readiness.md` | Review-gated |
| 10.7 External dependencies | `Phase10.7-External-Dependencies.md` | Review-gated |
| 10.8 Mainnet release gate | `Phase10.8-Mainnet-Release-Gate.md` | Approval-gated |

## Evidence carried forward

Phase 9 evidence remains an input to Phase 10, including:

- testnet operations baseline;
- network bootstrap/provisioning;
- operational smoke/health;
- backup/restore/recovery;
- upgrade compatibility procedure and evidence template;
- operational hardening review;
- long-running observation tooling/evidence;
- pre-mainnet readiness consolidation.

## State rules

Phase 10 uses explicit readiness states.

- Documentation is not approval.
- A built artifact is not a released artifact.
- Testnet identity is not mainnet identity.
- Explorer/indexer state remains derived and rebuildable.
- CI status is separate from operational and release evidence.
- Secrets, private keys, credentials, and signing private material never belong in evidence.

## Current production boundary

The repository does not currently authorize a mainnet launch.

Mainnet network ID, chain ID, production genesis artifact/hash, final release identity, security approvals, infrastructure readiness, external dependencies, and release authority remain explicit decision gates.

## Exit linkage

The final Phase 10 state must be recorded by the release gate in:

`Phase10.8-Mainnet-Release-Gate.md`

Phase 11 production decision records remain a separate downstream decision package.
