# IndoChain Controlled Testnet Runbook

Phase 3.6 prepares controlled/local and public-testnet bootstrap with seed peer support. This is not a public production mainnet launch.

Testnet dIDR has no monetary value. Staking remains collateral-only, has no APY, does not create validators, and does not mint staking rewards. Service points are simulation-only and are not spendable dIDR.

## Network Profile

- Network: `testnet`
- Network ID: `ind-testnet-1`
- Chain ID: `777101`
- Genesis hash: `db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4`
- Address format: `iND...` Base58Check
- Default RPC: `:18331`
- Default P2P: `:19331`

The testnet genesis is deterministic and differs from localnet. Datadirs initialized with `--network testnet` write `network.json`; later commands can resolve the active profile from that metadata.

The formal genesis candidate is recorded in `docs/TestnetGenesis.md`. Public-testnet deployment preparation is documented in `docs/DeployTestnet.md`, with preflight checks in `docs/Preflight.md`.

## Public-Safe Operator Mode

For a public read-only testnet node:

```powershell
go run ./node/cmd/indochain --datadir ./data/testnet-public --network testnet init
go run ./node/cmd/indochain --datadir ./data/testnet-public node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc
```

Public mode disables wallet/admin/miner/faucet/service write RPC by default. Add `--enable-miner-rpc` only for a miner-enabled public testnet node:

```powershell
go run ./node/cmd/indochain --datadir ./data/testnet-public node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --enable-miner-rpc
go run ./node/cmd/indominer --rpc-url http://<PUBLIC_HOST>:8811 --address <TESTNET_IND_ADDR> --threads 4
```

See `docs/Operator.md` for Linux systemd examples, Windows PowerShell examples, firewall notes, and preflight checks.

## Node A Bootstrap

```powershell
go run ./node/cmd/indochain --datadir ./testdata/tn1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/tn1 --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/tn1 wallet new
go run ./node/cmd/indochain --datadir ./testdata/tn1 node start --rpc :8611 --p2p :9611 --advertise-p2p http://127.0.0.1:9611
```

Check Node A:

```powershell
Invoke-RestMethod http://127.0.0.1:8611/health
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8611 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8611 stake info
```

Expected: `network=testnet`, `network_id=ind-testnet-1`, `chain_id=777101`, and a testnet genesis hash.

## Node B With Bootnode

```powershell
go run ./node/cmd/indochain --datadir ./testdata/tn2 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/tn2 --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/tn2 node start --rpc :8612 --p2p :9612 --advertise-p2p http://127.0.0.1:9612 --bootnode http://127.0.0.1:9611
```

Check peers:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8612 peer list
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8612 peer list --source
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8612 peer check http://127.0.0.1:9611
```

Expected: the bootnode is stored in `peers.json` as a normalized URL without a trailing slash, source `bootnode`, and no duplicate entry after restarting Node B with the same `--bootnode` flag.

## Node C With Seed Peers

Seed peers are equivalent startup hints for reusable public-testnet bootstrap lists. They can come from the testnet network profile, `IND_SEED_PEERS`, config `p2p.seed_peers`, `--seed-peer`, `--seed-peers`, or `--seed-file`.

Create a seed file:

```powershell
Set-Content -Path ./testdata/testnet-seeds.txt -Value @"
# One P2P URL per line.
http://127.0.0.1:9611
"@
```

Start Node C from that file:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/tn3 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/tn3 --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/tn3 node start --rpc :8613 --p2p :9613 --advertise-p2p http://127.0.0.1:9613 --seed-file ./testdata/testnet-seeds.txt
```

Check peers:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8613 peer list --source
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8613 peer check http://127.0.0.1:9611
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8613 peer sync http://127.0.0.1:9611
```

Expected: the seed peer is normalized, stored with source `seed`, not duplicated across restart, and ignored if it equals Node C's own `--advertise-p2p`. Offline seeds do not prevent startup; wrong-network or wrong-genesis seeds are rejected by `peer check` and `peer sync`.

## Mining And Sync

Mine on Node A:

```powershell
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8611 --address <NODE_A_IND_ADDR> --threads 4 --once
```

Sync Node B:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8612 peer sync http://127.0.0.1:9611
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8611 chain validate
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8612 chain validate
```

Expected: Node B reaches Node A height and both chains validate as testnet.

When sync imports blocks, output includes `imported blocks`, `height`, and `tip hash`. When the local node is already up to date or ahead of the peer, output should say so and keep `imported blocks: 0`.

## Dev Faucet Funding

The faucet is dev/testnet-only and disabled unless the node starts with `--enable-faucet-rpc`. It uses a normal wallet as source and creates a normal pending transaction; mine a block to confirm it.

```powershell
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn wallet new
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn wallet new
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn node start --rpc :8811 --p2p :9811 --advertise-p2p http://127.0.0.1:9811 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 100 --faucet-min-interval 1m
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --max-blocks 105
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 faucet info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 faucet request --address <RECIPIENT_ADDR>
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 balance <RECIPIENT_ADDR>
```

Expected: faucet request returns `status: pending`, recipient confirmed balance increases only after mining, and total supply changes only by normal coinbase rewards.

## Faucet To Stake To Service Eligibility

Use this controlled end-to-end flow when testing a service owner that needs the current testnet service collateral threshold.

Start a faucet node with a 1000 dIDR request amount:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/e2e_service dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/e2e_service --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/e2e_service wallet new
go run ./node/cmd/indochain --datadir ./testdata/e2e_service wallet new
go run ./node/cmd/indochain --datadir ./testdata/e2e_service node start --rpc :8911 --p2p :9911 --advertise-p2p http://127.0.0.1:9911 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 1000 --faucet-min-interval 1s --faucet-max-per-address 2000
```

Mine enough mature faucet balance, request funds, and mine one confirmation:

```powershell
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --max-blocks 105
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 faucet request --address <OWNER_ADDR>
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 balance <OWNER_ADDR>
```

Lock the service collateral and mine the stake transaction:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 stake lock --address <OWNER_ADDR> --amount 1000
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 stake list --address <OWNER_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 balance <OWNER_ADDR>
```

Register and run the service simulation once:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 service register --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971
go run ./node/cmd/indoservice --rpc-url http://127.0.0.1:8911 --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 service score --address <OWNER_ADDR>
```

Expected:

- required stake: `1000 dIDR`
- active stake: `1000 dIDR`
- stake eligible: `true`
- collateral status: `eligible`
- eligible simulated points: greater than `0`
- service points are simulation-only and are not spendable dIDR
- total supply changes only when blocks are mined and only by normal coinbase rewards

## Network Mismatch Check

Start a localnet node and check it from the testnet node:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/ln1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/ln1 --network localnet init
go run ./node/cmd/indochain --datadir ./testdata/ln1 node start --rpc :8621 --p2p :9621 --advertise-p2p http://127.0.0.1:9621
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8612 peer check http://127.0.0.1:9621
```

Expected: rejected with network, chain ID, or genesis mismatch.

An offline peer check records the peer as `offline` with `last_error`; a later successful check of the same normalized URL clears `last_error` and marks it `active`. Network/genesis mismatches and invalid block responses should not print `sync complete`.

## Controlled Service Node

Service RPC is for controlled testnet simulation. Do not expose it publicly without rate limits and abuse controls.

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8611 service register --address <NODE_A_IND_ADDR> --endpoint http://127.0.0.1:9701
go run ./node/cmd/indoservice --rpc-url http://127.0.0.1:8611 --address <NODE_A_IND_ADDR> --endpoint http://127.0.0.1:9701 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8611 service score --address <NODE_A_IND_ADDR>
```

Expected: service score uses testnet collateral parameters. Service points remain simulation-only.
