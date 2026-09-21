# Phase 10.6 — Operational Readiness

## Purpose

Consolidate the operational procedures established during Phases 8 and 9 into a mainnet-oriented readiness gate.

This phase defines operational evidence and runbook requirements. It does not authorize a mainnet launch.

## 10.6.1 Node lifecycle

Each production node must have an identified:
- role;
- persistent datadir;
- P2P identity;
- network/chain configuration;
- bootstrap/seed configuration;
- operator owner;
- recovery owner.

Node startup must verify network, chain, and genesis identity before normal operation.

## 10.6.2 Deployment and provisioning

The Phase 9.2 provisioning boundary remains the baseline:

1. create a new persistent datadir;
2. establish a unique node/P2P identity;
3. apply the intended network and chain configuration;
4. configure approved bootstrap/seed endpoints;
5. start the node;
6. verify health;
7. verify peer health/list;
8. validate chain state;
9. verify Explorer/indexer when enabled;
10. record node identity and endpoint evidence.

Production provisioning must not reuse testnet identity or persistent chain state.

## 10.6.3 Health and smoke gate

The operational health gate should cover, as applicable:

- /health;
- /peer/health;
- /peer/list;
- /node/metrics;
- /explorer/status;
- /explorer/indexer/stats;
- chain information;
- chain validation.

The established testnet health tooling may be reused as the operational baseline, with production endpoints and thresholds explicitly configured before mainnet use.

## 10.6.4 Backup and recovery

The Phase 9.4 boundary remains mandatory:

- stop the source node before backup;
- keep backups outside the live datadir;
- protect backup access;
- restore into a new empty datadir;
- validate network/chain/genesis identity;
- run health and smoke checks;
- preserve failed state as evidence when investigation is required.

Explorer/indexer data remains rebuildable read-model state.

Never copy one node's P2P identity to another independent node.

## 10.6.5 Upgrade and rollback

The Phase 9.5 and 10.3 boundaries apply:

- record pre-upgrade state;
- create the protected backup point;
- verify binary/configuration compatibility;
- replace the intended artifact only;
- restart cleanly;
- validate chain and network identity;
- verify Explorer/indexer and monitoring;
- retain a known-good rollback artifact and backup.

If persistent-format compatibility is uncertain, restore the protected backup into a new datadir rather than attempting an unsafe downgrade.

## 10.6.6 Monitoring and observation

Operational monitoring should retain the existing read-only surfaces:

- node runtime;
- chain height/tip;
- peer health;
- mempool;
- mining/difficulty;
- Explorer indexer state.

Phase 9.7 observation tooling provides a JSONL evidence path for repeated read-only sampling.

Monitoring failure must not be treated as consensus failure without separate chain evidence.

## 10.6.7 Incident response

Before mainnet approval, define an operator response for at least:

- node unavailable;
- peer isolation or abnormal peer failure;
- unexpected chain divergence;
- indexer lag/failure;
- mempool/resource pressure;
- corrupted or inconsistent persistent state;
- failed upgrade;
- suspected key/credential compromise;
- backup/restore failure.

For an incident:
1. identify affected node and role;
2. preserve relevant evidence;
3. avoid destructive recovery actions until state is captured when practical;
4. isolate the affected surface if required;
5. validate canonical chain state independently;
6. recover using the approved runbook;
7. record cause, impact, action, and verification.

## 10.6.8 Access and operator controls

Operational access must be assigned by role.

At minimum distinguish:
- node administration;
- mining/operator control;
- backup/recovery;
- release/artifact handling;
- security/incident response.

Risky actions must identify the target node, datadir, and network before execution.

Secrets and private keys must never be placed in logs, documentation examples, or evidence archives.

## 10.6.9 Operational evidence package

Before the final release gate, retain:

1. node inventory and roles;
2. production topology reference;
3. provisioning evidence;
4. health/smoke evidence;
5. backup/restore evidence;
6. upgrade/rollback evidence;
7. monitoring/observation evidence;
8. incident-response runbook;
9. access/control ownership;
10. known limitations and unresolved findings.

The package must identify evidence timestamps and relevant release/configuration references without exposing secrets.

## 10.6.10 Readiness checklist

- [ ] Production node roles are assigned.
- [ ] Production datadirs and identities are isolated.
- [ ] Provisioning procedure is verified.
- [ ] Health/smoke procedure is verified.
- [ ] Backup and restore procedure is verified.
- [ ] Upgrade and rollback procedure is verified.
- [ ] Monitoring and observation are operational.
- [ ] Incident response ownership is assigned.
- [ ] Operator access boundaries are defined.
- [ ] Operational evidence package is complete.

Unchecked items remain open readiness items.

## Acceptance criteria

- Existing Phase 9 operational controls are consolidated into a mainnet-oriented runbook.
- Deployment, recovery, upgrade, rollback, monitoring, and incident procedures are explicit.
- Evidence requirements are defined.
- Production/testnet isolation remains explicit.
- No consensus implementation change is required.
- No mainnet launch authorization is implied.
- CI status remains separate from operational evidence.
