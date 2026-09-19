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
