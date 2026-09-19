# DesKaChain Asset & Fee Model

## Native asset

The DesKaChain v3 asset model defines **IDR** as the native asset:

- asset ID: `IDR`
- symbol: `IDR`
- native decimals: 0
- protocol fee asset: `IDR`

IDR is the native unit used by DesKaBank, DesKaWallet, DesKaPay, DesKaBusiness, and other DesKa services for protocol fees and internal settlement.

Legacy v1/v2 transactions remain available during the migration. Their historical DKC amount representation is kept so existing chain-hardening tests and old blocks remain compatible.

## Issued tokens

DesKaChain v3 also reserves explicit issued fungible assets. A token definition contains its issuer, symbol, decimals, supply policy, and status.

Examples include:

- USDT/USDC-style stablecoin tokens
- utility tokens
- application tokens
- other cryptocurrency assets

For a token that claims a fiat or commodity peg, backing, redemption, custody, and issuer controls are external business/financial functions. Consensus records token ownership and authorized token issuance operations; it does not independently prove an off-chain reserve.

The native IDR asset cannot be recreated as an issued token.

## Fee payment

The protocol separates the **asset being transferred** from the **asset used to pay gas/protocol fees**.

For all v3 transactions:

`Fee asset = IDR`

By default:

`FeePayer = From`

Therefore a token holder sending:

`100 USDT + 2 IDR fee`

must have enough of both assets:

- at least 100 USDT
- at least 2 IDR

### Paymaster

A paymaster may sponsor the fee:

`User -> token transfer`
`Paymaster -> IDR fee`

The user still signs the transaction. The paymaster adds a separate authorization signature. The paymaster authorization is intentionally excluded from the transaction ID so sponsorship can be attached after the user's transaction payload is fixed.

The active consensus/execution layer must later verify:

1. the token owner is authorized to spend the token;
2. the effective fee payer has enough spendable IDR;
3. a different fee payer supplied a valid authorization;
4. the fee is charged exactly once in native IDR.

## Asset transaction types

The v3 transaction envelope reserves:

- `asset_create`
- `asset_mint`
- `asset_burn`
- `transfer` with an explicit `asset_id`

Asset creation uses a deterministic ID derived from the create transaction ID.

## Migration status

Phase 5.16 introduces the protocol envelope and amount primitives for:

- native IDR
- issued asset metadata
- explicit asset IDs
- native-IDR fee payer
- paymaster authorization
- asset-specific decimal formatting/parsing

Phase 5.17 now includes the multi-asset state/execution foundation: per-asset balances are part of the deterministic state snapshot and StateRoot, persisted in bbolt, queryable through the chain/RPC layer, and executed through the v3 ledger path. Consensus activation remains gated by the network transaction-version profile.


## Phase 5.18 fee / gas policy

v3 protocol fees are paid exclusively in native **IDR**. The transaction `Fee` field is an integer IDR amount.

The active network profile defines a deterministic gas policy:

- base gas depends on the transaction type;
- encoded v3 signing bytes are charged in `BytesPerGas` units;
- `GasUsed = BaseGas + ceil(SigningBytes / BytesPerGas)`;
- `MinimumFee = max(MinFee, GasUsed * MinGasPrice)`;
- `MaxGasPerTx` is a consensus limit.

The gas estimate uses the chain-bound transaction signing preimage, so the result is stable before the sender signature and after any paymaster authorization is attached.

For an issued-token transfer, the token owner pays the IDR fee by default. When `FeePayer` is a different address, the user signs the transaction and the external paymaster supplies its own authorization signature.

Fees are not newly minted by the v3 coinbase. During block execution the fee is debited from the effective fee payer, held in the protocol fee collector, and settled to the block miner. The v3 coinbase is therefore limited to the configured block subsidy.

### Fee RPC

`GET /fee/policy` exposes the active fee schedule and confirms that the fee asset is IDR.

`POST /fee/estimate` accepts a transaction envelope and returns gas units, encoded signing-byte size, minimum fee, fee asset, and whether the requested fee meets policy.

Example:

```json
{
  "transaction": {
    "version": 3,
    "type": "transfer",
    "from": "<FROM_ADDR>",
    "to": "<TO_ADDR>",
    "asset_id": "asset:example",
    "amount": 100,
    "fee": 5,
    "nonce": 1,
    "timestamp": 1780000000
  }
}
```

Migration remains profile-gated: built-in localnet, testnet, and mainnet configurations currently keep v1 transactions active, so the v3 fee/gas rules are not yet the default network consensus.


## Protocol fee system

V3 fees are denominated exclusively in native `IDR`. The protocol separates the transferred asset from the fee asset.

For a normal transaction:

`sender -> token/IDR transfer + IDR protocol fee`

For a sponsored transaction:

`sender -> token/IDR transfer`
`paymaster -> IDR protocol fee`

The fee policy is deterministic and transaction-type aware. A fee quote combines a base-gas schedule with the canonical unsigned envelope size. The configured `MinFee` is the minimum payable fee, while `MinGasPrice` can make the minimum scale with gas usage.

The current fee lifecycle is:

`payer IDR`
`    -> protocol fee pool`
`    -> block producer settlement`

V3 uses zero block subsidy in the current economic profile, so fee income is not created as new IDR. It is transferred from the effective fee payer to the protocol fee pool and then settled to the block producer. The pool is represented as a protocol-owned asset balance and is not a normal user account.

Fee policy is exposed through RPC so DesKaWallet/DesKaBank can quote a fee before signing. Paymaster authorization does not alter the transaction ID.
