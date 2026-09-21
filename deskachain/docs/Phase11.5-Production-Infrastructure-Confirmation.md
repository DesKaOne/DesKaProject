# Phase 11.5 — Production Infrastructure Confirmation

## Purpose

Convert the conceptual production topology from Phase 10.2 into an explicit deployment and ownership record.

This phase confirms infrastructure boundaries; it does not authorize launch.

## Production roles

| Role | Required confirmation |
|---|---|
| Seed/Bootstrap | Host, endpoint, peer policy, owner, recovery |
| Full Node | Host, persistent datadir, P2P identity, owner |
| Miner/Operator | Private control path, key custody, owner |
| Explorer/Indexer | Host, storage, rebuild procedure, owner |
| Public RPC | Approved endpoints, read-only surface, exposure policy |
| Backup/Recovery | Storage, retention, restore owner, verification |
| Monitoring/Alerting | Metrics, alerts, escalation owner |
| Access Control | Accounts, privileges, secret handling, offboarding |

## Topology invariants

- Seed is discovery/bootstrap aid, not consensus authority.
- Every independent node has its own P2P identity and persistent state.
- Mining control is not exposed through public read RPC.
- Explorer/indexer is a rebuildable read model and does not participate in consensus.
- Public RPC exposes only explicitly approved surfaces.
- Mainnet and testnet infrastructure are isolated.

## Deployment evidence

Record for each production node/service:

- role;
- environment;
- hostname/endpoint;
- network and chain identity;
- software/release version;
- persistent storage boundary;
- P2P/RPC bindings;
- owner;
- backup/recovery relationship;
- monitoring relationship;
- access-control policy.

Do not store private credentials in this document.

## Failure and recovery

For each critical role define:

- failure detection;
- replacement procedure;
- state restoration boundary;
- identity restoration rules;
- operator escalation;
- validation after recovery.

Do not clone a node's P2P identity into an independent replacement node unless the documented recovery procedure explicitly requires identity restoration for that same node.

## Acceptance criteria

- Production roles have explicit owners.
- Network exposure is approved per role.
- Persistent state and P2P identity boundaries are explicit.
- Backup/recovery and monitoring ownership are confirmed.
- Testnet configuration cannot silently become production configuration.
- No launch authorization is implied.
