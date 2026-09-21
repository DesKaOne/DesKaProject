# Phase 9.5 — Upgrade & Compatibility Evidence Record

## Purpose

Record an operator-executed upgrade drill against a disposable or non-critical IndoChain testnet node.

This record is an evidence template. Filling it requires actual execution data; placeholders are not test results.

## Drill identity

| Field | Value |
|---|---|
| Drill ID | TBD |
| Date/time | TBD |
| Operator | TBD |
| Node role | TBD |
| Datadir | TBD |
| Network | testnet |
| Network ID | TBD |
| Chain ID | TBD |
| Genesis identity/hash | TBD |

## Pre-upgrade evidence

| Check | Result | Evidence/reference |
|---|---|---|
| Binary version/build | TBD | TBD |
| Config checksum | TBD | TBD |
| Node/P2P identity | TBD | TBD |
| Height | TBD | TBD |
| Tip hash | TBD | TBD |
| Peer count | TBD | TBD |
| Mempool pending count | TBD | TBD |
| Explorer index height/lag/readiness | TBD | TBD |
| `/node/metrics` schema | TBD | TBD |
| `chain validate` | TBD | TBD |
| Phase 9.3 smoke | TBD | TBD |
| Phase 9.4 backup | TBD | TBD |

## Upgrade execution

| Step | Result | Evidence/reference |
|---|---|---|
| Clean shutdown | TBD | TBD |
| Backup protected | TBD | TBD |
| Binary replaced | TBD | TBD |
| Reviewed config applied | TBD | TBD |
| Node restarted | TBD | TBD |
| Network/chain/genesis verified | TBD | TBD |
| Chain state opened/validated | TBD | TBD |
| Peer reconnection/sync | TBD | TBD |
| Explorer recovery/readiness | TBD | TBD |
| Metrics schema remains compatible | TBD | TBD |
| Post-upgrade smoke | TBD | TBD |
| Normal sync/recovery cycle observed | TBD | TBD |

## Post-upgrade comparison

| Field | Before | After | Expected |
|---|---|---|---|
| Network ID | TBD | TBD | unchanged |
| Chain ID | TBD | TBD | unchanged |
| Genesis identity | TBD | TBD | unchanged |
| Height | TBD | TBD | valid/converged |
| Tip hash | TBD | TBD | valid/converged |
| Chain validation | TBD | TBD | pass |
| Peer state | TBD | TBD | healthy/recoverable |
| Explorer index | TBD | TBD | ready/within tolerance |
| Metrics schema | TBD | TBD | compatible |
| Smoke result | TBD | TBD | pass |

## Rollback drill

Set this section to `NOT EXERCISED` when rollback was not executed.

| Field | Result |
|---|---|
| Rollback executed | TBD |
| Previous binary restored | TBD |
| Protected backup required | TBD |
| Restored height/tip | TBD |
| Chain validation | TBD |
| Network/chain/genesis identity | TBD |
| Health/smoke result | TBD |
| Rollback incident notes | TBD |

Do not perform an in-place downgrade against an unknown incompatible persistent format.

## Findings

| ID | Classification | Description | Remediation/mitigation | Owner | Status |
|---|---|---|---|---|---|
| F-001 | TBD | TBD | TBD | TBD | TBD |

Classifications used by Phase 9.6: Blocker, High, Medium, Low, Informational.

## Final drill state

Allowed states:

- PASS
- PASS WITH OPERATIONAL RECOVERY
- FAILED
- BLOCKED
- NOT EXECUTED

Current state: **NOT EXECUTED**

## Evidence rule

This record becomes evidence only after the operator fills it from an actual drill and attaches traceable artifacts/logs/checksums where appropriate.

No private keys, credentials, or secret values belong in this record.

## Boundary

A passing upgrade drill demonstrates the tested compatibility path for the observed node and artifact. It does not certify a production release, security audit, or mainnet approval.