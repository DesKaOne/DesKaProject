# DesKaChain Service Node Agent

`dkcservice` is the Phase 3.1 service node agent for DesKaChain service simulation.

It automates the Phase 3.0 service-node RPC workflow:

- register a DKC address as a service node,
- send heartbeats,
- create simulated verification challenges,
- submit safe simulated measurements,
- fetch service score and simulated points,
- save local agent state.

It is not a public exit proxy. It does not open a VPN, relay, or public listener. It does not sell bandwidth. It does not mine PoW blocks. It does not earn spendable DKC. Service points are simulation-only research/testnet accounting.

Since Phase 3.2, the node may mark service reward simulation as stake-eligible when the configured owner address has enough active staking collateral. The agent reports that score metadata from RPC, but it does not lock or unlock stake by itself.

## Requirements

- A running DesKaChain node with service RPC enabled.
- A `DKC...` address.
- No private key is required by the agent.

## Run Once

```powershell
go run ./node/cmd/dkcservice --rpc-url http://127.0.0.1:8431 --address <DKC_ADDR> --endpoint http://127.0.0.1:9501 --once
```

Expected output includes:

```text
DesKaChain Service Node Agent
service agent registered id=...
heartbeat ok score=...
challenge created id=...
challenge submitted status=passed score=...
service score=...
state saved
```

## Run Loop

```powershell
go run ./node/cmd/dkcservice --rpc-url http://127.0.0.1:8431 --address <DKC_ADDR> --endpoint http://127.0.0.1:9501 --heartbeat-interval 30s --challenge-interval 60s
```

Stop with `Ctrl+C`. The agent saves state before exit.

## State

The default state file is:

```text
./dkcservice-state.json
```

Inspect it:

```powershell
go run ./node/cmd/dkcservice status --state ./dkcservice-state.json
```

The state stores service metadata and counters only. It does not store private keys.

## Safe Mode

Safe mode is default. In Phase 3.1, even `--safe-mode=false` keeps behavior safe and prints a warning. The agent does not:

- bind a public listener,
- run a proxy server,
- relay traffic,
- run external speed tests,
- generate large traffic,
- access private keys,
- connect to random third-party hosts.

It only calls the configured DesKaChain node RPC.

## Public RPC Warning

When a node is started with `--public-rpc`, service write endpoints are disabled by default. Start controlled test/verifier nodes with:

```powershell
go run ./node/cmd/deskachain node start --public-rpc --enable-service-rpc=true
```

Do not enable service RPC publicly without rate limits and abuse protection.

## Future Work

- Signed service registration.
- Real verifier protocol.
- Mobile safe mode checks for WiFi, charging, battery, and temperature.
- Staking/collateral research.
- Capped testnet reward pool.
- Public testnet leaderboard.
