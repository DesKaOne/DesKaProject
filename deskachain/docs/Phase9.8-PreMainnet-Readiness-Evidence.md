# Phase 9.8 — Pre-Mainnet Readiness Evidence

## Purpose

This document consolidates the engineering evidence produced by Phase 8 and Phase 9 for a later pre-mainnet decision.

It is an evidence gate, not a mainnet launch approval.

Phase 9 does not authorize production deployment, regulatory approval, economic-security claims, external-partner readiness, or a mainnet release.

## Evidence baseline

### Phase 8 — Testnet engineering

The Phase 8 final readiness gate records:

- multi-node convergence and health convergence;
- controlled fork/reorg coverage and operator health checks;
- bounded soak coverage;
- transaction, mining, mempool-pressure, restart/recovery, peer-churn, and Explorer indexer recovery coverage;
- node, chain, peer, mining, mempool, indexer, runtime, and monitoring API statistics;
- Explorer Monitoring dashboard backed by the versioned `/node/metrics` contract;
- consolidated Node → Chain → Mempool → Explorer Indexer → Explorer API → Monitoring → Explorer UI integration coverage.

The Phase 8 gate also explicitly keeps production/mainnet readiness as a separate decision.

### Phase 9 — Testnet operations

Phase 9 evidence now covers:

- operations baseline and repeatable node roles;
- bootstrap and fresh-node provisioning;
- operator smoke and health checks;
- backup, restore, and recovery boundaries;
- upgrade and compatibility drill boundaries;
- security and operational hardening review;
- long-running observation and incident/recovery evidence.

## Evidence artifact index

| Phase | Artifact | Status |
|---|---|---|
| 9.1 | Phase9.1-Testnet-Operations-Baseline.md | Complete |
| 9.2 | Phase9.2-Network-Bootstrap-Provisioning.md | Complete |
| 9.3 | Phase9.3-Operational-Smoke-Health.md | Complete |
| 9.4 | Phase9.4-Backup-Restore-Recovery.md | Complete |
| 9.5 | Phase9.5-Upgrade-Compatibility-Drill.md + Phase9.5-Upgrade-Compatibility-Evidence.md | Procedure + execution record template |
| 9.6 | Phase9.6-Security-Operational-Hardening.md | Engineering review scope |
| 9.7 | Phase9.7-LongRunning-Observation.md | Observation procedure |
| 9.8 | Phase9.8-PreMainnet-Readiness-Evidence.md | Consolidation gate |

## Evidence index

| Area | Evidence | Gate |
|---|---|---|
| Multi-node behavior | Phase 8.5 tests/docs | Engineering evidence |
| Soak/load/recovery | Phase 8.6 tests/docs | Engineering evidence |
| Monitoring/observability | Phase 8.7 API/UI/tests/docs | Engineering evidence |
| Integrated flow | Phase 8.8 integration/docs | Engineering evidence |
| Operations baseline | Phase 9.1 docs | Operational evidence |
| Provisioning/bootstrap | Phase 9.2 docs | Operational evidence |
| Smoke/health | Phase 9.3 scripts/docs | Operational evidence |
| Backup/recovery | Phase 9.4 scripts/docs | Operational evidence |
| Upgrade compatibility | Phase 9.5 procedure + Phase9.5-Upgrade-Compatibility-Evidence.md | Execution evidence only after operator drill |
| Hardening | Phase 9.6 review/docs | Operational evidence |
| Long-running observation | Phase 9.7 script/docs | Observation evidence |

## Engineering exit criteria

Phase 9 engineering work can be considered complete when:

- [x] Phase 8 testnet engineering gate is complete.
- [x] Repeatable testnet roles, provisioning, and bootstrap are documented.
- [x] Operator smoke/health checks are reproducible.
- [x] Backup/restore/recovery boundaries are documented.
- [x] Upgrade/compatibility and rollback boundaries are documented.
- [x] Security/operational hardening findings have a classification and remediation/mitigation path.
- [x] Long-running observation can produce archiveable evidence.
- [x] Remaining gaps are explicitly recorded rather than silently treated as complete.
- [x] Main branch remains outside this testnet work.

## Remaining pre-mainnet questions

The following are deliberately outside the automatic completion claim and require separate evidence/approval:

1. What exact mainnet network and chain identifiers will be approved?
2. What production node topology and operator roles will be used?
3. What final RPC/P2P exposure policy will be approved?
4. What key custody, backup retention, and recovery policy will be approved?
5. What formal security review or audit is required?
6. What economic/security assumptions and threat model are accepted?
7. What external infrastructure, partners, services, or operational dependencies are required?
8. What release artifact, versioning, migration, and rollback policy will be approved?
9. What regulatory/legal/compliance requirements apply before deployment?
10. Who has final authority to approve a mainnet release?

These questions must not be answered by inference from testnet evidence alone.

## Final Phase 9 state

Phase 9 testnet operations and pre-mainnet evidence scope is complete at the engineering/documentation level when the evidence above is archived and any unresolved findings are explicitly carried into the next gate.

The next stage should be a separately approved **Pre-Mainnet / Mainnet Readiness** phase.

No mainnet launch is implied by this document.

## CI tracking

The repository contains `.github/workflows/go-test.yml`. The Phase 9 branch now participates in the workflow for push/PR coverage, but this document does not claim a successful run unless GitHub reports a completed passing workflow for the exact commit under review.

Project progress status and GitHub Actions execution status remain separate evidence.
