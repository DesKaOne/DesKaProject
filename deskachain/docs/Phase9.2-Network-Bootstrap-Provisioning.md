# Phase 9.2 — Network Bootstrap & Node Provisioning

## Objective

Make a fresh IndoChain testnet node reproducible without changing consensus behavior. Bootstrap configuration is an operational concern: it establishes how nodes discover and reach peers, while chain validation remains authoritative for canonical state.

## Provisioning model

Each node receives:

1. a unique persistent datadir;
2. a unique persistent P2P identity;
3. the same testnet network identity and chain identity;
4. the canonical genesis configuration/state;
5. explicit RPC and P2P listen addresses;
6. bootstrap/seed peer information where discovery requires it;
7. an operator-defined role such as seed, full node, miner, or Explorer node.

Node identities must never be copied between independent nodes. A cloned datadir is not a valid substitute for provisioning a fresh identity.

## Bootstrap topology

The baseline topology is:

```text
                 ┌──────────────┐
                 │  Seed Node   │
                 │ discovery    │
                 └──────┬───────┘
                        │
             ┌──────────┼──────────┐
             │          │          │
          Node A      Node B     Node C
          full/miner  full       full/explorer
             │          │          │
             └──────────┴──────────┘
                  peer sync
```

The seed is a discovery/bootstrap aid, not a source of truth for consensus. Nodes independently validate network ID, chain ID, protocol compatibility, and genesis identity during peer interaction.

## Fresh-node procedure

1. Create a new persistent datadir.
2. Generate/initialize the node's P2P identity in that datadir.
3. Apply the intended testnet network/chain configuration.
4. Configure the bootstrap/seed endpoint(s).
5. Start the node.
6. Confirm `/health` reports the expected network identity.
7. Confirm `/peer/list` and `/peer/health` show the intended topology.
8. Confirm `chain info` and `chain validate` identify a valid canonical chain.
9. If Explorer is enabled, confirm `/explorer/status` and `/explorer/indexer/stats` converge independently.
10. Record the resulting node identity and operational endpoint for the testnet inventory.

## Bootstrap failure handling

A bootstrap/seed outage must not be interpreted as a chain failure. If a node already has usable peer metadata, it may continue according to the existing P2P behavior. If a fresh node cannot discover peers, the operator should repair bootstrap reachability or provide an approved peer introduction path; consensus rules are not modified to compensate.

## Provisioning acceptance matrix

| Check | Fresh node | Expected |
|---|---:|---|
| Unique datadir | yes | no shared writable state |
| Unique P2P identity | yes | persists across restart |
| Network identity | testnet | matches existing nodes |
| Chain identity | testnet chain | matches existing nodes |
| Genesis identity | canonical | matches existing nodes |
| Bootstrap peers | configured | reachable or recoverable |
| Peer health | observable | expected topology |
| Chain validation | pass | canonical chain valid |
| Explorer indexer | optional by role | catches up independently |
| Public RPC | read-only | wallet/admin disabled on public-safe node |

## Operator evidence

Provisioning evidence should record the node role, datadir location, node identity fingerprint/identifier, network identity, chain identity, bootstrap peers, RPC/P2P endpoints, initial height/tip, and health-check result. Private keys and secret material must never be committed to source control or copied into issue/CI logs.

## 9.2 acceptance criteria

- [x] Fresh-node provisioning procedure is documented.
- [x] Bootstrap topology and responsibilities are explicit.
- [x] Seed/bootstrap is separated from consensus authority.
- [x] Identity, datadir, and secret-handling boundaries are explicit.
- [x] Existing health, peer, chain, and Explorer checks form the acceptance evidence path.
