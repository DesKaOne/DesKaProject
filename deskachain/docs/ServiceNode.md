# DesKaChain Service Node Simulation

Phase 3.0 introduces a research-only service node layer for bandwidth and uptime contribution experiments.

This layer is not consensus mining. PoW remains the only canonical block creation mechanism. Service node points are not DKC, are not spendable, and do not change total supply, circulating supply, balances, difficulty, cumulative work, coinbase rewards, or chain validation.

Since Phase 3.2, a service node can become stake-eligible when its owner address has active staking collateral at or above the configured required stake. This collateral only affects eligibility metadata and simulated service points; it does not make the node a validator and does not create spendable DKC rewards.

Phase 3.2.1 adds regression tests around this collateral eligibility. Staking remains collateral only: no PoS, no validator set, no staking APY, no DKC staking reward, and no slashing in this phase.

Phase 3.4.1 adds an end-to-end controlled testnet scenario where faucet-funded testnet DKC is locked as service-node collateral before running the service simulation. This proves the faucet, stake, and service layers agree on the same testnet profile and address.

Phase 3.5 documents service-node operator mode in `docs/Operator.md`. Service write RPC should stay disabled on public read-only nodes unless the node is intentionally running a controlled verifier/test setup.

Phase 3.9 adds deployment preparation docs in `docs/DeployTestnet.md`. The service owner wallet should be backed up, and the service agent should use its own state file such as `/var/lib/deskachain/service-agent-state.json`.

Phase 4.2 documents the public-testnet multi-host faucet, staking collateral, and service simulation flow. Stake collateral is chain-backed and syncs between nodes. The service-node registry, challenges, and score samples are still local simulation state in `service_nodes.json`, `service_challenges.json`, and `service_rewards.json` on the RPC node that the agent talks to.

## What It Tracks

- Service node registration tied to a `DKC...` owner address.
- Heartbeats and recent uptime.
- Local verification challenge simulation.
- Latency and bandwidth samples submitted to a challenge.
- Service score components.
- Anti-abuse flags and penalties.
- Daily simulated service points.

## What It Does Not Do

- No real payout.
- No mainnet reward guarantee.
- No public exit proxy.
- No VPN or public relay.
- No bandwidth selling.
- No staking or PoS.
- No staking rewards, validator role, or DKC slashing in Phase 3.2.
- No GPU mining.
- No mining pool.
- No DKC balance mutation.

## Scoring

The Phase 3.0 score is intentionally simple:

- uptime score: active heartbeat within 5 minutes scores 100, within 30 minutes scores 50, older scores 0.
- latency score: lower submitted latency scores higher.
- bandwidth score: uploaded plus downloaded bytes are mapped into coarse tiers.
- reliability score: passed challenges divided by completed challenges. No challenge scores 0 because the node is unverified.
- abuse penalty: suspicious behavior reduces the final score.

Final score:

```text
raw = uptime * 0.30 + latency * 0.20 + bandwidth * 0.30 + reliability * 0.20
service_score = clamp(raw - abuse_penalty, 0, 100)
```

Simulated points:

```text
simulated_points = service_score * 10
```

The epoch is the UTC date (`YYYY-MM-DD`). Points are for research/testnet accounting and leaderboards only.

## Anti-Abuse Flags

Initial flags include:

- `heartbeat_spam`
- `impossible_bandwidth`
- `repeated_identical_samples`
- `challenge_failed`
- `challenge_expired`
- `endpoint_changed_too_often`

Phase 3.0 applies penalties instead of permanent bans by default.

## RPC Safety

Read-only service score and rewards endpoints are available like other read-only RPC. Service write endpoints are disabled by default when `--public-rpc` is enabled. Enable them only in controlled verifier/test setups:

```powershell
go run ./node/cmd/deskachain node start --public-rpc --enable-service-rpc=true
```

## Testnet Collateral Flow

For a local controlled testnet, fund the service owner with the dev faucet, mine the faucet transaction, lock `1000 DKC`, mine the stake transaction, then register and run the service agent:

```powershell
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 faucet request --address <OWNER_ADDR>
go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 stake lock --address <OWNER_ADDR> --amount 1000
go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 service register --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971
go run ./node/cmd/dkcservice --rpc-url http://127.0.0.1:8911 --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971 --once
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8911 service score --address <OWNER_ADDR>
```

Expected service score metadata:

- required stake: `1000 DKC`
- active stake: `1000 DKC`
- stake eligible: `true`
- collateral status: `eligible`
- eligible simulated points: greater than `0`

Before the stake lock is mined, the same service owner remains ineligible with active stake `0`. If the owner unlocks the stake, the collateral status becomes `unlocking` and eligible simulated points return to `0`.

For public-testnet deployment prep, use `examples/testnet/service-node.env` for the node RPC profile and `examples/testnet/service-agent.env` for the agent. Neither file contains private keys.

## Multi-Host Public Testnet Flow

Host B can run the service owner workflow against its local synced node:

```sh
./deskachain --datadir ./data/wallet_b --network testnet init
OWNER_ADDR="$(./deskachain --datadir ./data/wallet_b wallet new)"
./deskachain --rpc-url http://<HOST_A_RPC_IP>:9311 faucet request --address "$OWNER_ADDR"
./dkcminer --rpc-url http://127.0.0.1:9311 --address <MINER_ADDR> --threads 2 --once
./deskachain --rpc-url http://127.0.0.1:9312 balance "$OWNER_ADDR"
./deskachain --rpc-url http://127.0.0.1:9312 stake lock --address "$OWNER_ADDR" --amount 1000
./dkcminer --rpc-url http://127.0.0.1:9312 --address "$OWNER_ADDR" --threads 2 --once
./deskachain --rpc-url http://127.0.0.1:9312 stake list --address "$OWNER_ADDR"
./deskachain --rpc-url http://127.0.0.1:9312 service register --address "$OWNER_ADDR" --endpoint http://<HOST_B_REACHABLE_IP>:9971
./dkcservice --rpc-url http://127.0.0.1:9312 --address "$OWNER_ADDR" --endpoint http://<HOST_B_REACHABLE_IP>:9971 --once
./deskachain --rpc-url http://127.0.0.1:9312 service score --address "$OWNER_ADDR"
```

Expected:

- active stake: `1000 DKC`.
- required stake: `1000 DKC`.
- stake eligible: `true`.
- collateral status: `eligible`.
- eligible simulated points: greater than `0` after a successful challenge.
- service points remain simulation-only and do not mint spendable DKC.

Cross-host visibility:

- Chain-backed stake transactions, active stake totals, total supply, height, and tip hash should match on Host A and Host B after sync.
- Service registration, challenge samples, and reward rows are local to the RPC node that received them. If Host B runs `dkcservice --rpc-url http://127.0.0.1:9312`, Host B's `service score` has the local simulation record. Host A will see the same active stake after sync, but it will not automatically have Host B's local `service_nodes.json` registration unless the service owner also registers against Host A with service RPC deliberately enabled there.
- Service RPC disabled should fail clearly. Enable `--enable-service-rpc` only for controlled verifier/test nodes; it does not enable wallet/admin RPC.

## Future Phases

- Real verifier network.
- Signed service-node registration.
- Mobile safe mode.
- Staking or collateral research.
- Capped testnet reward pool.
- Public testnet leaderboard.
