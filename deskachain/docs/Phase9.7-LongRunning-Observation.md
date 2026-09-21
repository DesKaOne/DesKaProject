# Phase 9.7 — Long-Running Testnet Observation

## Objective

Define a repeatable observation window for a running multi-node testnet and capture read-only evidence from existing monitoring and health surfaces.

This phase observes the system; it does not introduce a new monitoring store and does not change consensus state.

## Observation sources

Use existing:
- `/node/metrics`
- `/health`
- `/peer/health`
- `/explorer/indexer/stats`
- `/explorer/status`

Observation records are operator artifacts, not consensus data.

## Observation script

`scripts/testnet-observe.sh` polls one or more node RPC endpoints and writes JSON Lines records.

Example:

```sh
./scripts/testnet-observe.sh \
  --node http://127.0.0.1:9311 \
  --node http://127.0.0.1:9321 \
  --node http://127.0.0.1:9331 \
  --interval-seconds 30 \
  --duration-seconds 3600 \
  --output ./observations/phase9.7.jsonl
```

Each record captures timestamp, node endpoint, metrics, health, peer, and Explorer indexer observations.

## Observation window

Recommended baseline:
- minimum 30 minutes for a short operational window;
- 60 minutes or longer for stronger Phase 9 evidence;
- all selected nodes observed at the same interval.

Record start/end timestamp, node endpoints/roles, network/chain identity, interval, duration, output file, incidents, recovery actions, and final health result.

## What to look for

### Chain
- height advances when mining is intentionally enabled;
- no unexpected tip divergence among converged nodes;
- chain validation remains healthy.

### Peers
- active peers remain within the intended topology;
- failures/reconnections are explainable;
- persistent connectivity loss is recorded.

### Mempool
- pending count behaves as expected;
- persistent growth without confirmation is recorded as an incident.

### Mining
- block intervals remain observable;
- difficulty/retarget observations remain internally consistent;
- stalled mining is distinguished from intentionally idle mining.

### Explorer
- indexer remains ready or recovers within documented tolerance;
- lag does not silently grow;
- Explorer failure remains isolated from canonical chain state.

## Incident and recovery record

For each incident capture timestamp, node, symptom, last known healthy observation, operator action, recovery result, and final chain/tip/indexer state.

Do not put private keys, wallet secrets, or credentials in observation artifacts.

## Acceptance criteria

- Repeatable multi-node observation command exists.
- Observation data is append-only JSONL and easy to archive.
- Existing read-only metrics are reused.
- Chain, peer, mempool, mining, and Explorer observations are captured.
- Incident/recovery evidence is explicitly documented.
- Observation tooling does not modify node or consensus state.
- A Phase 9.7 observation window can be reproduced by an operator.
