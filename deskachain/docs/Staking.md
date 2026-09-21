# IndoChain Staking Collateral

Phase 3.2 adds staking as collateral for service-node eligibility.

This is not Proof-of-Stake. It does not create validators, does not select block proposers, does not mint dIDR rewards, and does not affect PoW consensus. PoW remains the only canonical block production mechanism.

Phase 3.2.1 adds regression tests and consensus safety checks for staking lifecycle, spendable balance accounting, chain validation, RPC safety, and service-node collateral eligibility. It does not add new staking economics.

Phase 3.2.2 keeps staking collateral-only while plumbing localnet/testnet profiles through runtime RPC, P2P, miner, and service collateral paths. Testnet staking parameters are distinct from localnet and remain placeholders until the official testnet profile is finalized.

TODO: add stake-specific reorg regression coverage when the reorg helper can exercise orphaned stake lock/unlock requeue and drop behavior without broad test scaffolding.

## What Staking Does

- Locks mature dIDR so it cannot be spent while active.
- Lets service nodes prove collateral eligibility.
- Starts an unbonding period when unlocked.
- Releases stake automatically after the unbonding period.

## What Staking Does Not Do

- No APY.
- No guaranteed profit.
- No validator set.
- No delegation.
- No slashing in Phase 3.2.
- No staking reward dIDR.
- No coinbase reward change.

## Localnet Parameters

- staking enabled: true
- minimum stake amount: 10 dIDR
- minimum service stake: 100 dIDR
- unbonding period: 10 blocks
- max active stakes per address: 10

## Testnet Parameters

- staking enabled: true
- minimum stake amount: 100 dIDR
- minimum service stake: 1000 dIDR
- unbonding period: 100 blocks

Faucet-funded testnet dIDR can be used to test stake lock and service-node collateral eligibility. The dev faucet still creates normal pending transfer transactions and those funds must be mined before they can be staked.

Mainnet staking parameters are not final.

## Commands

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake lock --address <IND_ADDR> --amount 10
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake list --address <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake unlock --address <IND_ADDR> --stake-id <STAKE_ID>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake status --stake-id <STAKE_ID>
```

Stake lock and unlock require the node wallet private key for signing, so public RPC mode disables those write endpoints by default. Read-only stake info/list/status remain available.

## Balance Behavior

Confirmed and mature balances do not change when stake is locked. Spendable balance changes:

```text
spendable = mature balance - active stake - unlocking stake - pending stake lock - pending outgoing transfer
```

Immature coinbase rewards cannot be staked. Released stake automatically becomes spendable again after the unbonding period.

`chain info` circulating supply is not the same thing as wallet spendable balance. Active, unlocking, and released stake still count as circulating supply because the coins are mature and owner-controlled collateral. Only spendable balance excludes active/unlocking stake.

## Service Node Eligibility

Service nodes are stake-eligible when active stake is at least the network `min_service_stake`. Service points are still simulation-only and are not spendable dIDR.

On testnet, the current service threshold is `1000 dIDR`. Locking less than that can create an active stake record, but service score remains `stake_eligible: false` and eligible simulated points stay at `0`. Unlocking a previously eligible stake moves it out of active collateral during the unbonding period.

## Multi-Host Notes

Stake lock and unlock transactions are normal chain transactions. After they are mined and peers sync, Host A and Host B should report the same active stake totals in `chain info`, `stake info`, and `stake list`.

Service-node registration and scoring are not consensus state in this phase. They are local simulation files on the RPC node handling service requests. This means service eligibility depends on chain-backed active stake, but the service node must register and submit challenge samples to each RPC node where the operator wants local service score visibility.
