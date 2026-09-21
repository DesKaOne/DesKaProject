# Phase 6 — Production Readiness

Phase 6 hardens the merged IndoChain v3 protocol before external testnet use.

## Frozen v3 invariants

- Native asset: **dIDR**
- Fee asset: **dIDR**
- Native asset precision: **8 decimal units**
- Transaction version: **3**
- Block version: **1**
- RPC API: **v1**
- P2P protocol: **ind-p2p/1**
- Economics: **fee-only blocks**
- Block subsidy: **0**
- User-issued assets: enabled
- Token transfers: enabled
- Paymaster: enabled
- Consensus and network request limits must be non-zero and bounded.

Built-in localnet, testnet, and mainnet profiles are validated through
config.ValidateNetworkProfile. This is a protocol guardrail, not a dynamic
configuration mechanism: changing a frozen invariant requires an explicit
protocol/version decision and corresponding migration or genesis treatment.

## Phase 6 sequence

1. Protocol/genesis freeze
2. Asset, transaction and signature security audit
3. State/storage crash-recovery hardening
4. P2P authentication and abuse resistance
5. RPC/API production hardening
6. Observability and operator diagnostics
7. Multi-node testnet soak/recovery testing
8. Release candidate

The current branch starts with step 1. No DesKaWallet/DesKaBank integration is
added in Phase 6 until the settlement layer passes these readiness gates.
