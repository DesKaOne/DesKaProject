# Phase 9.1 — Testnet Operations Baseline

## Node roles

| Role | Purpose | Required state |
|---|---|---|
| Seed/Bootstrap | Provides initial peer discovery/connectivity | Persistent identity, reachable P2P endpoint |
| Validator/Full node | Maintains canonical chain state and serves RPC | Persistent chain data and node identity |
| Miner node | Produces testnet blocks when mining is enabled | Full-node state plus mining capability |
| Explorer node | Serves Explorer/API read models | Full-node state plus Explorer indexer |

A single development process may temporarily combine roles, but operational tests should identify the intended role explicitly.

## Persistent boundaries

The operationally important state is separated conceptually into:

- chain/block storage;
- node/P2P identity;
- mempool state;
- Explorer index data;
- operator configuration.

Deleting Explorer index data must be recoverable through index rebuild. Deleting chain or node identity data is a different recovery event and must not be treated as an Explorer rebuild.

## Network invariants

Every testnet node participating in the same network must agree on the configured network identity, chain identity, genesis identity, and protocol expectations before it is considered healthy.

Peer health is observational; a node must not infer consensus from the Explorer indexer alone.

## Startup order

1. Prepare persistent directories.
2. Start bootstrap/seed connectivity where required.
3. Start full nodes with persistent identities.
4. Verify network and chain invariants.
5. Verify P2P connectivity and chain convergence.
6. Start or verify Explorer indexing.
7. Run the read-only health and monitoring checks.
8. Enable mining only for the intended mining node(s).

## Operational health gate

A node is operationally healthy when the existing testnet health checks can verify the node's RPC/P2P reachability, chain tip, peer state, and configured Explorer/indexer readiness when Explorer service is part of the role.

The node metrics endpoint remains read-only and is evidence for observation, not a control plane.

## Safety boundaries

- Do not expose development-only RPC bindings as a production default.
- Do not reuse node identities between independent nodes.
- Do not copy private keys or identity files into source control.
- Do not treat a green CI run as evidence of production readiness.
- Do not modify consensus rules solely for operational convenience.

## 9.1 acceptance criteria

- [x] Node roles are explicitly documented.
- [x] Persistent-state boundaries are documented.
- [x] Network invariants and startup order are documented.
- [x] Existing health/metrics tooling is identified as the operational observation layer.
- [x] Main remains outside the Phase 9 work branch.
