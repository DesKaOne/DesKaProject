# Phase 11 — IndoChainWallet

## Purpose

Phase 11 starts the native blockchain wallet work on top of the completed IndoChain and IndoScan engineering foundations.

The first target is a **testnet-compatible wallet**. Production/mainnet identifiers, genesis, and release decisions remain explicit and are never inferred from testnet values.

## Scope

### 11.1 Wallet Core

- key generation
- address generation
- seed phrase lifecycle
- encrypted private-key storage
- transaction creation
- transaction signing
- network/chain validation
- nonce handling

### 11.2 Wallet Transactions

- send
- receive
- pending state
- confirmation state
- transaction history
- fee display

### 11.3 Network Integration

```
IndoChainWallet
       │
       ▼
IndoChain RPC
       │
       ▼
IndoChain
```

Wallet RPC integration must preserve the existing public/read-only Explorer boundary. Wallet/write capability must not be exposed through public RPC mode.

### 11.4 IndoScan Integration

```
Wallet
  │
  ├── transaction → IndoScan transaction view
  ├── address     → IndoScan address view
  └── block       → IndoScan block view
```

Integration is read-only on the Explorer side.

### 11.5 Wallet Security

- encrypted local storage
- explicit backup/export flow
- restore flow
- signing confirmation
- network validation
- transaction summary before signing
- protection against accidental mainnet/testnet mix-up

## Network separation

Current testnet identity remains:

- Network: `testnet`
- Network ID: `ind-testnet-1`
- Chain ID: `777101`

These values are testnet-only. Mainnet network ID, chain ID, genesis, and protocol release identity remain TBD until separately approved.

## Native unit

- Native IndoChain unit: **dIDR**
- Ecosystem accounting relation: **1 dIDR = 1000 Rupiah**
- This accounting relation does not by itself make dIDR a freely tradable asset.

## Wallet safety boundary

The wallet must never:

- expose private keys through logs or APIs;
- silently switch network;
- sign a transaction for a different network than the selected wallet network;
- rely on Explorer data as consensus truth;
- treat Explorer/indexer read models as authoritative for transaction validity.

## Phase 11 acceptance gates

### Gate A — Wallet Core Contract

- deterministic key/address model defined
- storage encryption model defined
- seed/restore lifecycle defined
- signing contract defined

### Gate B — Testnet RPC Integration

- wallet can query network identity
- wallet can query balance and nonce
- wallet can construct transactions using chain-native rules
- wallet can sign locally
- wallet can submit through an authenticated wallet/write boundary

### Gate C — IndoScan Integration

- transaction links open correctly
- address links open correctly
- block links open correctly
- Explorer remains read-only

### Gate D — Security

- private material excluded from logs
- network mismatch rejected
- signing requires explicit user action
- backup/restore tested

### Gate E — Release Boundary

- testnet wallet build identified
- source commit recorded
- configuration/network target recorded
- no mainnet launch is implied

## Current state

**Phase 11 specification started.**

Implementation work should proceed from the actual IndoChain transaction/address primitives and existing RPC contracts; no new consensus rules should be invented solely for the wallet layer.
