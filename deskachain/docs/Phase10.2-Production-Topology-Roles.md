# Phase 10.2 — Production Topology & Roles

## Purpose

Define the intended mainnet node topology and operational roles without treating the topology as a launch authorization.

Mainnet values, endpoints, and operators remain subject to the Phase 10.1 identity/genesis approval.

## Role model

| Role | Primary responsibility | Main boundary |
|---|---|---|
| Seed / Bootstrap | Peer discovery and initial connectivity | Not a consensus authority |
| Full Node | Canonical chain validation and network participation | Persistent chain state |
| Miner / Operator | Intentionally produces blocks according to the approved protocol | Mining control must not be exposed unintentionally |
| Explorer / Indexer | Builds read-only chain read model | Non-consensus, rebuildable |
| Public RPC | Serves explicitly approved read/API surfaces | Minimize exposed methods |
| Operations / Recovery | Backup, restore, upgrade, incident response | Access-controlled |

A deployment may combine roles only when the operational and security boundaries remain explicit.

## Recommended topology

The production topology should provide:

1. multiple independent full nodes;
2. more than one seed/bootstrap path where practical;
3. miner/operator separation from public-facing RPC where practical;
4. dedicated or isolated Explorer/indexer capacity;
5. no dependency on a single Explorer instance for chain validity;
6. documented recovery ownership for each persistent node.

Conceptual flow:

    Seed(s)
      |
      +---- Full Node A ----+
      +---- Full Node B ----+---- Miner / Operator
      +---- Full Node C ----+
                    |
                    +---- Explorer / Indexer
                    |
                    +---- Approved Read RPC

Seeds assist discovery and synchronization. They do not decide the canonical chain.

## Node identity and persistence

Each independent production node must have:
- its own persistent datadir;
- its own P2P identity;
- its own chain/block state;
- its own operator configuration;
- independently managed backup/recovery evidence.

One node's P2P identity must never be cloned into another independent node.

Testnet datadirs and identities must remain physically and operationally separate from mainnet.

## RPC exposure boundary

Public-facing nodes should expose only explicitly approved endpoints.

Baseline from testnet operations:
- health/observability surfaces are read-only;
- Explorer APIs are read-only;
- monitoring is read-only;
- wallet/admin/mining-control/write surfaces remain operator-only unless separately approved.

A public RPC node must not become an implicit administration or wallet endpoint.

## Seed/bootstrap boundary

Seed/bootstrap nodes:
- provide peer discovery/initial connectivity;
- may help a fresh node find approved peers;
- do not become trusted consensus authorities;
- must not be the sole route to network participation where redundancy is required.

Fresh nodes must still validate network ID, chain ID, genesis identity, and protocol compatibility.

## Miner/operator boundary

Mining operations require explicit operator control.

Recommended separation:
- mining RPC/control remains private;
- public RPC serves only approved read/API surfaces;
- miner state and keys are protected;
- mining availability is observable through existing monitoring.

The exact number and ownership of miners remains a separate approval decision.

## Explorer/indexer boundary

Explorer/indexer:
- consumes canonical chain state;
- maintains a rebuildable read model;
- may be restarted or rebuilt without changing consensus;
- should be isolated from public write/control surfaces;
- should have its own storage and backup policy where operationally required.

Loss of Explorer state must not be treated as loss of canonical chain state.

## Redundancy and recovery

For each production role, document:
- primary node/operator;
- backup node/operator;
- datadir location;
- backup location;
- recovery owner;
- expected recovery procedure;
- dependency on other nodes/services.

Critical paths should avoid a single operational dependency where practical.

## Mainnet isolation checklist

- [ ] Fresh mainnet datadirs.
- [ ] Fresh/approved mainnet P2P identities.
- [ ] Approved mainnet network/chain/genesis identity.
- [ ] Testnet bootstrap endpoints excluded.
- [ ] Testnet ports/config excluded.
- [ ] Public RPC methods reviewed.
- [ ] Mining control isolated.
- [ ] Explorer storage isolated.
- [ ] Backup/recovery ownership assigned.
- [ ] Seed/bootstrap redundancy reviewed.
- [ ] No role is treated as consensus authority merely because it is operationally central.

## Acceptance criteria

- Production roles are explicitly defined.
- A multi-node topology is documented.
- Seed, full-node, miner, Explorer, and public-RPC boundaries are explicit.
- Independent node identity and persistence are required.
- Testnet/mainnet separation is explicit.
- Recovery ownership is part of the topology record.
- Public exposure is minimized by design.
- No consensus implementation change is required.
- This document does not constitute mainnet release approval.
