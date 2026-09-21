# IndoChain Dev/Testnet Faucet

Phase 3.4 adds a dev/testnet faucet for controlled funding flows. Testnet dIDR has no monetary value.

The faucet does not mint silently and does not credit balances directly. It signs a normal transfer from a configured faucet wallet, adds the transaction to the mempool, and requires a mined block before the recipient has confirmed funds.

For operator setup, see `docs/Operator.md`, `docs/DeployTestnet.md`, and `docs/MultiHostTestnet.md`. Do not run a public faucet by default; keep faucet RPC limited to controlled testnet environments.

## Safety Model

- Faucet RPC is disabled by default.
- Faucet requests are testnet-only.
- Public RPC does not enable faucet unless `--enable-faucet-rpc` is passed explicitly.
- The faucet source address must be a wallet in the node datadir.
- The faucet uses mature spendable balance only.
- Private keys are never returned by faucet RPC and are not stored in `faucet_state.json`.
- Rate limits are tracked per recipient address in `faucet_state.json`.
- Back up `faucet_state.json` if the operator wants rate-limit continuity across restarts or migrations.

## Start A Faucet Node

```powershell
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn wallet new
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn wallet new
```

Use the first wallet as `<FAUCET_ADDR>` and the second as `<RECIPIENT_ADDR>`.

```powershell
go run ./node/cmd/indochain --datadir ./testdata/faucet_tn node start --rpc :8811 --p2p :9811 --advertise-p2p http://127.0.0.1:9811 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 100 --faucet-min-interval 1m
```

## Fund The Faucet Wallet

Testnet coinbase maturity is 100 blocks, so mine enough blocks before requesting faucet funds.

```powershell
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --max-blocks 105
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 balance <FAUCET_ADDR>
```

Expected: mature/spendable balance is enough for the faucet amount.

## Request Funds

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 faucet info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 faucet request --address <RECIPIENT_ADDR>
```

The request returns a pending transaction ID. Mine one more block to confirm it:

```powershell
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8811 balance <RECIPIENT_ADDR>
```

The recipient can then use confirmed funds for transfers, staking collateral tests, and service-node eligibility tests. Staking remains collateral-only; faucet transfers do not create staking rewards, service payouts, slashing, or PoS behavior.

## Using Faucet Funds For Service-Node Collateral

Testnet service-node eligibility currently requires `1000 dIDR` active stake. For a controlled local testnet, start the faucet with a 1000 dIDR request amount and a daily cap above that amount:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/e2e_service node start --rpc :8911 --p2p :9911 --advertise-p2p http://127.0.0.1:9911 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 1000 --faucet-min-interval 1s --faucet-max-per-address 2000
```

After mining enough mature balance to `<FAUCET_ADDR>`, request and confirm the funding transaction:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 faucet request --address <OWNER_ADDR>
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 balance <OWNER_ADDR>
```

Then lock collateral and confirm the stake transaction:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 stake lock --address <OWNER_ADDR> --amount 1000
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8911 --address <FAUCET_ADDR> --threads 4 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 stake list --address <OWNER_ADDR>
```

Register the service node, run the service agent once, and check the score:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 service register --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971
go run ./node/cmd/indoservice --rpc-url http://127.0.0.1:8911 --address <OWNER_ADDR> --endpoint http://127.0.0.1:9971 --once
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8911 service score --address <OWNER_ADDR>
```

Expected score metadata:

- required stake: `1000 dIDR`
- active stake: `1000 dIDR`
- stake eligible: `true`
- collateral status: `eligible`
- eligible simulated points: greater than `0`

Testnet dIDR has no monetary value. The 1000 dIDR amount is only for controlled service-collateral testing. Service points are simulation-only and are not spendable dIDR. Staking is collateral-only and does not create APY, staking rewards, slashing, validators, or PoS block production.

## Rate Limit State

`faucet_state.json` lives in the selected datadir and records recipient request timestamps, daily requested amount, total requested amount, and pending transaction IDs. Corrupt state returns a clear error so test operators can inspect or reset the datadir intentionally.

## Multi-Host Public Testnet Faucet

Host A is the controlled operator node. Host B is a Windows/Linux user, staker, or service-owner node joined through Host A's seed peer. Testnet dIDR has no monetary value.

Host A responsibilities:

- run a testnet node with reachable P2P and intentionally enabled faucet RPC.
- keep wallet/admin RPC disabled on public RPC.
- use an operator-owned faucet address from the node datadir.
- fund that faucet wallet with mature mined testnet dIDR.
- back up `wallets.json` and `faucet_state.json` if continuity matters.
- never commit faucet wallet files, private keys, runtime state, or datadirs.

Host B responsibilities:

- join testnet with `--seed-peer http://<HOST_A_REACHABLE_IP>:10311/`.
- create a testnet owner wallet in a separate local datadir.
- request faucet funds from Host A.
- mine or wait for a mined block to confirm the faucet transaction.
- use confirmed spendable funds for service collateral tests if needed.

Prepare the faucet wallet on Host A. You may create it in a separate operator wallet datadir, but the private key must be imported into the seed/faucet node datadir before faucet RPC can spend from it:

```sh
./indochain --datadir ./data/faucet_wallet --network testnet init
FAUCET_ADDR="$(./indochain --datadir ./data/faucet_wallet wallet new)"
FAUCET_PRIV="$(./indochain --datadir ./data/faucet_wallet wallet export --address "$FAUCET_ADDR" --show-private-key | awk '/private_key:/ {print $2}')"
./indochain --datadir ./data/seed --network testnet init
./indochain --datadir ./data/seed wallet import --private-key "$FAUCET_PRIV"
```

The faucet source address must exist in the faucet node's wallet store. Mining 120 blocks gives room beyond the current 100-block testnet coinbase maturity:

```sh
./indominer --rpc-url http://127.0.0.1:9311 --address "$FAUCET_ADDR" --threads 2 --max-blocks 120
./indochain --rpc-url http://127.0.0.1:9311 balance "$FAUCET_ADDR"
```

Start or restart Host A deliberately as a faucet operator:

```sh
./indochain --datadir ./data/seed node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<HOST_A_REACHABLE_IP>:10311 \
  --public-rpc \
  --enable-miner-rpc \
  --enable-faucet-rpc \
  --faucet-address "$FAUCET_ADDR" \
  --faucet-amount 1000 \
  --faucet-min-interval 1h \
  --faucet-max-per-address 2000
```

Host B creates an owner address and requests controlled funding:

```sh
./indochain --datadir ./data/wallet_b --network testnet init
OWNER_ADDR="$(./indochain --datadir ./data/wallet_b wallet new)"
./indochain --rpc-url http://<HOST_A_RPC_IP>:9311 faucet info
./indochain --rpc-url http://<HOST_A_RPC_IP>:9311 faucet request --address "$OWNER_ADDR"
```

Confirm the normal faucet transaction by mining one block on a miner-enabled node:

```sh
./indominer --rpc-url http://127.0.0.1:9311 --address "$FAUCET_ADDR" --threads 2 --once
./indochain --rpc-url http://127.0.0.1:9312 balance "$OWNER_ADDR"
```

Expected result: Host B sees `1000 dIDR` confirmed and spendable after sync. Total supply increases only through mined coinbase rewards; the faucet transfer itself only moves existing testnet dIDR from the faucet wallet to the recipient.
