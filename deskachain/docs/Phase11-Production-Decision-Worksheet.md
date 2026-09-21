# Phase 11 — Production Decision Worksheet

## Purpose

Provide the single controlled worksheet for converting Phase 11 documentation into explicit production decisions and evidence.

This worksheet does not pre-approve any value. Every production value must be explicitly entered, reviewed, and approved by the designated authority.

## A. Network identity

| Field | Current value | Decision |
|---|---|---|
| Production network name | TBD | TBD |
| Network ID | TBD | TBD |
| Chain ID | TBD | TBD |
| Protocol version | TBD | TBD |
| P2P boundary | TBD | TBD |
| RPC boundary | TBD | TBD |

### Identity approval

- [ ] Collision/isolation check completed
- [ ] Testnet/mainnet separation verified
- [ ] Production identity approved
- [ ] Approver recorded
- [ ] Decision timestamp recorded

## B. Genesis-critical parameters

| Parameter | Value | Evidence |
|---|---|---|
| Genesis time | TBD | TBD |
| Initial state | TBD | TBD |
| Initial difficulty | TBD | TBD |
| Block/reward parameters | TBD | TBD |
| Native dIDR initial state | TBD | TBD |
| Other genesis-critical inputs | TBD | TBD |

Genesis remains blocked until Section A is approved.

## C. Economic/protocol decisions

| Decision | Value | Evidence | Approval |
|---|---|---|---|
| Fee unit | TBD | TBD | TBD |
| Fee calculation | TBD | TBD | TBD |
| Minimum/required fee | TBD | TBD | TBD |
| Mining/block reward | TBD | TBD | TBD |
| Difficulty/retarget | TBD | TBD | TBD |
| Issuance rules | TBD | TBD | TBD |
| Issued-asset governance | TBD | TBD | TBD |
| Supply/accounting assumptions | TBD | TBD | TBD |

## D. dIDR boundary

Established ecosystem definition:

**1 dIDR = 1000 Rupiah**

This worksheet does not infer reserve, redemption, deposit, legal-tender, or freely-tradable-asset properties from that definition.

Any production monetary or financial behavior requires a separate explicit decision and appropriate external approvals where applicable.

## E. Security gate

- [ ] Review scope finalized
- [ ] Assessment/reviewer identified
- [ ] Assessed source/artifact recorded
- [ ] Findings recorded
- [ ] Blocker/High findings resolved or explicitly accepted by authorized owner
- [ ] Residual risks documented
- [ ] Key custody decision recorded
- [ ] Operator access decision recorded
- [ ] Public RPC/P2P exposure reviewed
- [ ] Backup/signing-key protection reviewed

Security approval: **TBD**

## F. Production infrastructure

- [ ] Seed/bootstrap hosts confirmed
- [ ] Full-node hosts confirmed
- [ ] Miner/operator path confirmed
- [ ] Explorer/indexer confirmed
- [ ] Public RPC exposure confirmed
- [ ] Backup/recovery storage confirmed
- [ ] Monitoring/alerting confirmed
- [ ] Access-control owners confirmed
- [ ] Recovery procedures tested

## G. External dependencies

For every dependency use: `Identified`, `Evaluating`, `Sandbox/Tested`, `Production-ready`, `Approved`, `Blocked`, or `Not required`.

| Dependency | State | Owner | Evidence | Blocker |
|---|---|---|---|---|
| Infrastructure/hosting | TBD | TBD | TBD | TBD |
| DNS/endpoints | TBD | TBD | TBD | TBD |
| Monitoring | TBD | TBD | TBD | TBD |
| Backup storage | TBD | TBD | TBD | TBD |
| Wallet/payment integration | TBD | TBD | TBD | TBD |
| Banking/card/payment partner | TBD | TBD | TBD | TBD |
| KYC/compliance | TBD | TBD | TBD | TBD |
| Legal/regulatory | TBD | TBD | TBD | TBD |

Sandbox availability must not be treated as production approval.

## H. Release candidate

| Field | Value |
|---|---|
| Release version | TBD |
| Source commit | TBD |
| Toolchain | TBD |
| Platform/architecture | TBD |
| Artifact checksum | TBD |
| Signature/verification | TBD |
| Approved config reference | TBD |
| Genesis artifact/hash | TBD |
| Migration notes | TBD |
| Rollback reference | TBD |
| Verification result | TBD |

## I. Final decision

Allowed states:

- READY FOR REVIEW
- APPROVED
- RELEASED
- BLOCKED

Current state: **READY FOR REVIEW framework only**.

### Required before APPROVED

- [ ] Production identity approved
- [ ] Genesis independently verified and approved
- [ ] Economic/protocol decisions approved
- [ ] Security gate approved
- [ ] Production infrastructure confirmed
- [ ] Critical external dependencies resolved/approved
- [ ] Release candidate verified
- [ ] Release authority explicitly identified
- [ ] Exceptions/blockers reviewed

### Release authority

Name/role: **TBD**

Decision timestamp: **TBD**

Decision record/reference: **TBD**

## CI tracking

No applicable CI workflow evidence is currently available for this documentation progression. Project tracking may use **GREEN** as the local progress convention.

GREEN does not mean GitHub Actions passed, security approved, legal/regulatory approval obtained, or mainnet released.

## Current boundary

This worksheet is intentionally incomplete until real production decisions and evidence are supplied. It is not a substitute for those decisions and does not authorize mainnet launch.
