# Difficulty Observation

Difficulty observation is informational only. Phase 4.10 does not change the difficulty formula, PoW consensus, genesis, block reward, or block format.

Read current mining metrics:

```sh
./deskachain --rpc-url http://127.0.0.1:9311 mining status
./deskachain --rpc-url http://127.0.0.1:9311 mining difficulty
./deskachain --rpc-url http://127.0.0.1:9311 mining blocks --limit 30
```

Read through RPC:

```text
GET /mining/status
GET /mining/stats
GET /mining/difficulty
GET /mining/blocks?limit=30
```

Metrics include network, chain ID, height, current difficulty, next difficulty, target block time, retarget window, blocks until retarget, recent intervals, average/min/max interval, recent difficulties, total supply, coinbase maturity, pending transaction count, and peer count.

Retarget direction is a projection for operators. Consensus validation remains the authority.

