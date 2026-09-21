# Phase 9 — Testnet Operations & Pre-Mainnet Readiness

## Purpose

Phase 9 starts after the Phase 8 testnet engineering gate. It focuses on operating IndoChain as a repeatable testnet system and collecting evidence needed for a later pre-mainnet decision.

Phase 9 does not authorize mainnet launch, production deployment, regulatory approval, economic-security claims, or external-partner readiness.

## Scope

### 9.1 Testnet Operations Baseline
- Define repeatable node roles, data directories, ports, identities, and startup order.
- Establish operator-facing configuration and environment documentation.
- Keep testnet deployment isolated from main.

### 9.2 Network Bootstrap & Node Provisioning
- Standardize bootstrap/seed configuration.
- Make fresh-node provisioning reproducible.
- Verify persistent identity and chain/network invariants.

### 9.3 Operational Smoke & Health
- Extend existing health checks into a repeatable operator smoke flow.
- Verify RPC, P2P, chain tip, peer connectivity, mempool, mining, and Explorer/indexer readiness.

### 9.4 Backup, Restore & Data Recovery
- Define what persistent state is required for recovery.
- Test backup/restore boundaries for chain, node identity, mempool, and Explorer/index data.
- Document rebuild-vs-restore expectations.

### 9.5 Upgrade & Compatibility Drill
- Exercise controlled binary/config upgrades on a testnet node.
- Verify restart, resynchronization, schema compatibility, and rollback boundaries.

### 9.6 Security & Operational Hardening
- Review exposed RPC/P2P surfaces, default bindings, secrets, permissions, and operator controls.
- Separate development conveniences from testnet-safe defaults.
- Record findings without treating this phase as a formal security audit.

### 9.7 Long-Running Testnet Observation
- Run repeatable multi-node observation windows.
- Capture node, chain, peer, mempool, mining, and Explorer metrics.
- Record incidents and recovery actions.

### 9.8 Pre-Mainnet Readiness Evidence
- Consolidate operational evidence and unresolved risks.
- Define explicit engineering exit criteria.
- Keep mainnet launch as a separate future gate.

## Working principles

1. main remains untouched until a separately approved integration/release decision.
2. Monitoring and Explorer remain read-only observability layers.
3. Consensus behavior is not changed merely to satisfy operational tooling.
4. Every operational claim must be backed by a reproducible test or documented evidence.
5. Development-only shortcuts must be clearly separated from testnet operation paths.

## Initial exit condition

Phase 9 is complete only when the scoped operational checks have reproducible evidence and the remaining gaps are explicitly documented for the next gate.
