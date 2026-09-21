# DesKa Ecosystem Roadmap

> Current development roadmap from IndoChain and IndoScan through DesKaCash.
>
> Current checkpoint: **Phase 8 — IndoChain Testnet & IndoScan**.

## Roadmap Overview

```
IndoChain
   ↓
IndoScan
   ↓
IndoChainWallet
   ↓
DesKaPay
   ↓
DesKaCash
```

The dependency structure is:

```
IndoChain
   ├── IndoScan
   └── IndoChainWallet

DesKaPay
   └── DesKaCash
```

---

## Phase 8 — IndoChain + IndoScan

**Status: 🟡 Current**

Focus: testnet development, IndoScan integration, network hardening, and preparation for a usable public testnet.

### 8.1 — Explorer Architecture
**Status: ✅ Complete**

- Explorer architecture
- Core contracts/models
- Block data structure
- Transaction data structure
- Address data structure
- Chain information
- Initial API layer

### 8.2 — IndoScan Core
**Status: 🚧 In Progress**

- Block explorer
- Transaction explorer
- Address explorer
- Transaction details
- Block details
- Pagination
- Search
- Chain statistics
- Network status
- API integration

Target structure:

```
IndoScan
├── Blocks
├── Transactions
├── Addresses
├── Validators
├── Network
└── Search
```

### 8.3 — IndoScan UI / UX
**Status: 🚧 In Progress**

- Dashboard
- Latest blocks
- Latest transactions
- Block details
- Transaction details
- Address details
- Validator information
- Network statistics
- Responsive UI

### 8.4 — IndoScan Testnet Integration
**Status: 🟡 Current / Next**

Connect IndoScan to real IndoChain Testnet data through RPC/API.

Validation areas:

- Block synchronization
- Transaction indexing
- Address lookup
- Transaction lookup
- Confirmation status
- Chain height
- Network health
- Reorg handling
- API failure handling

### 8.5 — Testnet Hardening
**Status: ⏳ Planned**

- P2P stability
- Node synchronization
- Authentication
- Network limits
- Bounded responses
- Reorg protection
- Malformed request handling
- Restart recovery
- Database consistency
- CI regression testing

### 8.6 — Public Testnet
**Status: ⏳ Planned**

Target topology:

```
Node A
  │
Node B ─── IndoChain Testnet ─── Node C
  │
  └──────── IndoScan
```

Planned components:

- Bootstrap node
- Multiple full nodes
- Miner/validator/service node as applicable
- Public RPC
- IndoScan Testnet
- Monitoring
- Faucet, if required
- Testnet documentation

---

## Phase 9 — IndoChain Production Readiness

**Status: ⏳ Planned**

Prepare IndoChain for a production/mainnet environment.

### 9.1 — Consensus & Network Finalization

- Consensus behavior
- Block production
- Transaction validation
- Nonce
- Fee
- Gas
- Chain ID
- Network ID
- Genesis
- Finality
- Reorg behavior

### 9.2 — Security Hardening

- P2P security
- RPC security
- Rate limiting
- Authentication
- Replay protection
- Input validation
- DoS resistance
- Key management
- Wallet security considerations

### 9.3 — Node Operations

```
Full Node
Miner
Validator
RPC Node
Indexer
Explorer
```

Operational areas:

- systemd/service management
- Backup
- Restore
- Monitoring
- Logging
- Health checks
- Upgrade procedures

### 9.4 — Mainnet Preparation

- Genesis
- Chain parameters
- Network parameters
- Native coin configuration
- Fee model
- Initial accounts
- Node configuration
- Public RPC strategy
- Explorer infrastructure

### 9.5 — IndoChain Mainnet

**Milestone:** IndoChain Mainnet

```
IndoChain Mainnet
       │
       ├── Full Node
       ├── RPC
       ├── Consensus
       ├── Native IDR
       └── Public Network
```

---

## Phase 10 — IndoScan Mainnet

**Status: ⏳ Planned**

Move IndoScan from Testnet to Mainnet.

### 10.1 — Mainnet Explorer

- Mainnet network integration
- Public explorer deployment

### 10.2 — Production Indexer

- Block indexing
- Transaction indexing
- Address indexing
- Validator indexing
- Statistics
- Search
- Caching
- Database optimization

### 10.3 — Public IndoScan

Potential production domain:

```
indoscan.net
```

Development/staging may continue under the DesKa Ecosystem domain.

### 10.4 — Explorer API

IndoScan API can later be consumed by:

- IndoChainWallet
- DesKaPay
- DesKaCash
- Partners
- Developers

**Milestone:** IndoChain + IndoScan Production

---

## Phase 11 — IndoChainWallet

**Status: ⏳ Planned**

Build the native blockchain wallet after the chain and explorer foundations are stable.

### 11.1 — Wallet Core

- Key generation
- Address generation
- Private key protection
- Seed phrase
- Transaction signing
- Nonce handling
- Transaction creation

### 11.2 — Wallet Transactions

- Send
- Receive
- Transaction history
- Fee display
- Confirmation status

### 11.3 — Network Integration

```
IndoChainWallet
       │
       ▼
IndoChain RPC
       │
       ▼
IndoChain
```

### 11.4 — IndoScan Integration

- View transaction on IndoScan
- View address on IndoScan
- View block on IndoScan

### 11.5 — Wallet Security

- Encrypted storage
- Backup
- Restore
- Signing protection
- Network validation
- Transaction confirmation

**Milestone:** IndoChainWallet usable

---

## Phase 12 — DesKaPay

**Status: ⏳ Planned**

DesKaPay is the payment/merchant infrastructure layer, not the consumer e-wallet.

It is intended to provide the backend and APIs that can later be consumed by DesKaCash.

### 12.1 — DesKaPay Core

```
DesKaPay
├── User
├── Merchant
├── Account
├── Ledger
├── Balance
├── Transaction
├── Deposit
├── Withdrawal
├── Fee
└── Settlement
```

### 12.2 — Ledger

The ledger is the source of truth for financial balances and transaction state.

```
Account
   ↓
Ledger
   ↓
Balance
   ↓
Transaction
```

### 12.3 — Payment API

Planned API capabilities:

```
POST /payments
POST /transfers
POST /deposit
POST /withdraw
GET  /balance
GET  /transactions
```

### 12.4 — Virtual Account

Target flow:

```
Customer
   │
   ▼
Virtual Account
   │
   ▼
Partner Bank
   │
   ▼
DesKaPay
   │
   ▼
Ledger
```

### 12.5 — QRIS

Planned capabilities:

- QRIS receive
- QRIS payment
- QRIS status

### 12.6 — Payment Gateway / Partner Integration

Potential integration categories:

- Bank
- Payment gateway
- Virtual Account provider
- QRIS provider

### 12.7 — Callback / Webhook

```
Partner
   │
   ▼
Webhook
   │
   ▼
Validation
   │
   ▼
Idempotency
   │
   ▼
Ledger
   │
   ▼
Transaction
```

### 12.8 — Reconciliation

```
Partner Statement
        │
        ▼
DesKaPay
        │
        ▼
Reconciliation
        │
        ▼
Ledger
```

### 12.9 — Merchant API

DesKaPay can provide infrastructure for:

- Merchants
- Partners
- Businesses
- DesKaCash
- Future DesKa products

**Milestone:** DesKaPay Payment Infrastructure

---

## Phase 13 — DesKaCash

**Status: ⏳ Planned**

DesKaCash is the consumer-facing e-wallet layer built on top of DesKaPay infrastructure.

Architecture:

```
             DesKaCash
          Consumer Wallet
                │
                ▼
          DesKaCash API
                │
                ▼
            DesKaPay
       Payment Infrastructure
                │
       ┌────────┼─────────┐
       ▼        ▼         ▼
      VA       QRIS      Bank
       │        │         │
       └────────┼─────────┘
                ▼
             Ledger
```

### 13.1 — DesKaCash Account

- Registration
- Login
- User profile
- Wallet account
- Account security

### 13.2 — Balance

- IDR balance
- Balance sourced from the DesKaPay ledger

### 13.3 — Internal Transfer

```
User A
  │
  ▼
DesKaPay Ledger
  │
  ▼
User B
```

Internal user-to-user transfers use the internal ledger rather than VA-to-VA transfers.

### 13.4 — Deposit

```
Bank / VA
   ↓
DesKaPay
   ↓
Ledger
   ↓
DesKaCash
```

### 13.5 — Withdrawal

```
DesKaCash
   ↓
DesKaPay
   ↓
Partner Bank
```

### 13.6 — QRIS

```
DesKaCash
    ↓
DesKaPay
    ↓
QRIS
```

### 13.7 — Transaction History

- Transfer
- Deposit
- Withdrawal
- QRIS
- Fee
- Status
- Receipt

### 13.8 — Security

- PIN
- Biometric authentication
- Device binding
- OTP
- Transaction authentication
- Risk/fraud controls

**Milestone:** DesKaCash MVP

---

## Final Ecosystem Target

```
                    DESKA ECOSYSTEM
                           │
                           ▼
                 ┌───────────────────┐
                 │     INDOCHAIN     │
                 │   Blockchain Core │
                 └─────────┬─────────┘
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
       ┌──────────────┐        ┌──────────────────┐
       │   INDOSCAN   │        │ INDOCHAINWALLET  │
       │   Explorer   │        │ Blockchain Wallet│
       └──────────────┘        └────────┬─────────┘
                                        │
                                        ▼
                              ┌──────────────────┐
                              │     DESKAPAY     │
                              │ Payment Backend  │
                              │ Merchant / API   │
                              └────────┬─────────┘
                                       │
                                       ▼
                              ┌──────────────────┐
                              │    DESKACASH     │
                              │    E-Wallet      │
                              │ Consumer Layer   │
                              └──────────────────┘
```

### Milestones

1. **IndoChain Testnet + IndoScan**
2. **IndoChain Mainnet**
3. **IndoScan Mainnet**
4. **IndoChainWallet**
5. **DesKaPay Payment Infrastructure**
6. **DesKaCash MVP**

---

## Scope & Ownership

DesKa Ecosystem software, source code, architecture, and core technology are proprietary/internal development assets.

This roadmap describes the internal development direction and does not represent an offer to sell source code or software.

Partner integrations, infrastructure providers, business relationships, and regulatory requirements are separate workstreams and will be evaluated when the relevant product phase is reached.
