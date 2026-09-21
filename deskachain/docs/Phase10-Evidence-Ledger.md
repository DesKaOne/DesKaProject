# Phase 10 — Evidence Ledger

## Execution state

**OPEN / READINESS REVIEW**

This ledger records the current evidence state for Phase 10. Documentation is not treated as approval, execution proof, or mainnet authorization.

## Gate matrix

| Gate | Current state | Closure requirement |
|---|---|---|
| 10.1 Network identity & genesis | OPEN | Explicit approval plus exact artifact/hash verification |
| 10.2 Production topology & roles | DOCUMENTED / OPEN | Node inventory, independent identities, ownership and recovery evidence |
| 10.3 Release & artifact control | DOCUMENTED / OPEN | Concrete candidate artifact, provenance, checksum and verification |
| 10.4 Security & threat model | OPEN | Material-threat evidence, custody decisions, review requirements and residual-risk ownership |
| 10.5 Economic & protocol assumptions | OPEN | Explicit fee/economic decisions and supporting evidence |
| 10.6 Operational readiness | DOCUMENTED / OPEN | Actual drills with timestamped evidence |
| 10.7 External dependencies | OPEN | Provider/owner, environment, fallback and approval evidence |
| 10.8 Mainnet release gate | READY FOR REVIEW | Upstream blockers resolved and explicit release authority decision |

## Established semantics

- Native unit: **dIDR**
- Ecosystem accounting relationship: **1 dIDR = 1000 Rupiah**
- Issued assets remain separate from native dIDR.
- Testnet identity must not be promoted to mainnet.

## Testnet identity

| Parameter | Current testnet value | Mainnet state |
|---|---|---|
| Network | `testnet` | TBD / approval-gated |
| Network ID | `ind-testnet-1` | TBD / approval-gated |
| Chain ID | `777101` | TBD / approval-gated |
| Genesis | Testnet canonical genesis | TBD / production artifact |

## Evidence rules

1. A procedure document is not proof that the procedure was executed.
2. CI success is separate from operational, security and release evidence.
3. Testnet values are not production values.
4. Secrets, credentials, private keys and signing private material never belong in this ledger.
5. Unresolved prerequisites remain OPEN or BLOCKED until evidence supports closure.

## Exit condition

Phase 10 remains incomplete until the final gate records the required evidence, unresolved blockers, explicit release authority and a separate mainnet decision.

## Source documents

- `deskachain/docs/Phase10-PreMainnet-Mainnet-Readiness.md`
- `deskachain/docs/Phase10.1-Network-Identity-Genesis-Decision.md`
- `deskachain/docs/Phase10.2-Production-Topology-Roles.md`
- `deskachain/docs/Phase10.3-Release-Artifact-Control.md`
- `deskachain/docs/Phase10.4-Security-Threat-Model-Gate.md`
- `deskachain/docs/Phase10.5-Economic-Protocol-Assumptions.md`
- `deskachain/docs/Phase10.6-Operational-Readiness.md`
- `deskachain/docs/Phase10.7-External-Dependencies.md`
- `deskachain/docs/Phase10.8-Mainnet-Release-Gate.md`