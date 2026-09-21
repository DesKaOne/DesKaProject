# Phase 11 — IndoChainWallet Readiness Checklist

| Area | Requirement | Status |
|---|---|---|
| Wallet core | Key/address model | OPEN |
| Wallet core | Seed/restore lifecycle | OPEN |
| Wallet core | Encrypted storage | OPEN |
| Wallet core | Transaction signing | OPEN |
| Wallet core | Nonce handling | OPEN |
| RPC | Network identity validation | OPEN |
| RPC | Balance/nonce queries | OPEN |
| RPC | Transaction submission boundary | OPEN |
| IndoScan | Transaction deep-link | OPEN |
| IndoScan | Address deep-link | OPEN |
| Security | Mainnet/testnet separation | OPEN |
| Security | No private-key leakage | OPEN |
| Security | Backup/restore test | OPEN |
| Release | Testnet artifact identity | OPEN |

## Rules

1. OPEN means implementation/evidence is still required.
2. A passing CI run does not close operational or security gates.
3. Testnet values are never copied into mainnet configuration.
4. Private keys, seed phrases, credentials, and signing secrets must not be stored in repository evidence.
5. Wallet implementation must consume existing IndoChain consensus/transaction rules rather than redefining them.
