# DesKa Ecosystem Roadmap

> Current development roadmap from IndoChain and IndoScan through DesKaCash.
>
> Current branch checkpoint: **Phase 10 — Pre-Mainnet / Mainnet Readiness Evidence**.
>
> Phase 8 testnet engineering and IndoScan foundations are complete. Phase 9 testnet operations/evidence scope is complete. Phase 10 readiness controls and reproducible snapshot validation are complete as an engineering gate; production/mainnet approval, release authority, and final production identifiers remain separately gated.

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

**Status: ✅ Complete**

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
**Status: ✅ Complete — engineering scope**

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
**Status: ✅ Complete — engineering/UI foundation scope**

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
**Status: ✅ Complete — integration foundation/evidence scope**

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
**Status: ✅ Complete — engineering hardening scope**

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
**Status: ✅ Complete — documented/testnet operations scope**

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

## Phase 9 — IndoChain Testnet Operations & Pre-Mainnet Readiness

**Status: 🟢 Engineering/documentation scope complete on `phase-9-testnet`**

Phase 9 operationalizes the existing testnet and collects evidence for a later pre-mainnet decision. It does **not** authorize mainnet, production deployment, regulatory approval, or external-partner readiness.

### 9.1 — Testnet Operations Baseline
**Status: ✅ Complete**

- Node roles and startup order
- Persistent-state boundaries
- Network invariants
- Operational health gate

Evidence: `deskachain/docs/Phase9.1-Testnet-Operations-Baseline.md`

### 9.2 — Network Bootstrap & Node Provisioning
**Status: ✅ Complete**

- Fresh-node provisioning
- Bootstrap/seed topology
- Persistent node identity
- Network/genesis invariants

Evidence: `deskachain/docs/Phase9.2-Network-Bootstrap-Provisioning.md`

### 9.3 — Operational Smoke & Health
**Status: ✅ Complete**

- Repeatable operator smoke flow
- RPC/P2P/Explorer readiness checks
- Read-only monitoring validation

Evidence: `deskachain/docs/Phase9.3-Operational-Smoke-Health.md`

### 9.4 — Backup, Restore & Data Recovery
**Status: ✅ Complete**

- Stopped-node backup boundary
- Restore into a new datadir
- Explorer rebuild vs canonical state recovery
- Recovery evidence requirements

Evidence: `deskachain/docs/Phase9.4-Backup-Restore-Recovery.md`

### 9.5 — Upgrade & Compatibility Drill
**Status: ✅ Documented / Evidence-gated**

- Controlled binary upgrade
- Configuration compatibility
- Restart/resynchronization checks
- Rollback boundary

Evidence: `deskachain/docs/Phase9.5-Upgrade-Compatibility-Drill.md`

### 9.6 — Security & Operational Hardening
**Status: ✅ Complete as engineering review scope**

- RPC/P2P exposure review
- Secrets and permissions boundaries
- Backup/recovery safety
- Read-only Explorer/monitoring checks
- Operator control hardening

This is an engineering hardening gate, not a formal security audit.

Evidence: `deskachain/docs/Phase9.6-Security-Operational-Hardening.md`

### 9.7 — Long-Running Testnet Observation
**Status: ✅ Complete as tooling/documentation scope**

- Multi-node observation window
- JSONL operational evidence
- Chain/peer/mempool/mining/Explorer observations
- Incident/recovery recording

Evidence: `deskachain/docs/Phase9.7-LongRunning-Observation.md`

### 9.8 — Pre-Mainnet Readiness Evidence
**Status: ✅ Complete as evidence-consolidation scope**

- Consolidated Phase 8 + Phase 9 evidence
- Explicit engineering exit criteria
- Remaining pre-mainnet questions carried forward
- Mainnet authority kept separate

Evidence: `deskachain/docs/Phase9.8-PreMainnet-Readiness-Evidence.md`

### Phase 9 boundary

Phase 9 completion does **not** mean mainnet approval. Production identity, economics/protocol, security assessment, infrastructure, external dependencies, production genesis, release candidate, and final release authority remain subject to the separate Phase 11 gates.

## Phase 10 — Pre-Mainnet / Mainnet Readiness

**Status: ✅ Engineering gate complete / production decision-gated**

Phase 10 prepares and mechanically validates the production-readiness evidence and decision gates after testnet operations. It does **not** launch mainnet and does not create production identifiers by assumption.

### 10.1 — Network Identity & Genesis Decision
**Status: 🟡 Open — explicit production decision still required**

- Mainnet network ID remains TBD
- Mainnet chain ID remains TBD
- Production genesis artifact/hash remains TBD
- Protocol/release version remains TBD until approved
- Testnet/mainnet isolation guard is enforced

Evidence: `deskachain/docs/Phase10.1-Network-Identity-Genesis-Decision.md`

### 10.2 — Production Topology & Roles
**Status: ✅ Documented / Review-gated**

- Seed/bootstrap
- Full nodes
- Miner/operator
- Explorer/indexer
- Public RPC
- Recovery ownership

Evidence: `deskachain/docs/Phase10.2-Production-Topology-Roles.md`

### 10.3 — Release & Artifact Control
**Status: ✅ Documented / Review-gated**

- Traceable release identity
- Reproducible-build expectations
- Checksums/signing boundary
- Configuration provenance
- Migration and rollback

Evidence: `deskachain/docs/Phase10.3-Release-Artifact-Control.md`

### 10.4 — Security & Threat-Model Gate
**Status: 🟡 Review-gated**

- Trust boundaries
- RPC/P2P exposure
- Key and backup custody
- Threat evidence lifecycle
- Independent review requirements

Evidence: `deskachain/docs/Phase10.4-Security-Threat-Model-Gate.md`

### 10.5 — Economic & Protocol Assumptions
**Status: 🟡 Review-gated**

- Consensus assumptions
- Mining/difficulty
- Fee behavior
- dIDR accounting
- Issued-asset boundaries

Evidence: `deskachain/docs/Phase10.5-Economic-Protocol-Assumptions.md`

### 10.6 — Operational Readiness
**Status: 🟡 Review-gated**

- Provisioning
- Health/smoke
- Backup/restore
- Upgrade/rollback
- Monitoring
- Incident response

Evidence: `deskachain/docs/Phase10.6-Operational-Readiness.md`

### 10.7 — External Dependencies
**Status: 🟡 Review-gated**

- Infrastructure
- DNS/RPC/hosting
- Monitoring/backups
- Wallet/payment integrations
- Legal/regulatory dependencies

Evidence: `deskachain/docs/Phase10.7-External-Dependencies.md`

### 10.8 — Mainnet Release Gate
**Status: 🔒 Approval-gated**

- Consolidated evidence package
- Explicit blockers
- Release authority
- Controlled release/rollback
- Post-release verification

Evidence: `deskachain/docs/Phase10.8-Mainnet-Release-Gate.md`

**Milestone:** Pre-Mainnet / Mainnet Readiness Evidence

> Important: Phase 10 completion is not mainnet approval or release. Mainnet identifiers, genesis, security decisions, production infrastructure, dependencies, and release authority remain separately gated.

## Phase 11 — IndoChainWallet

**Status: ⏳ Next after Phase 10 readiness gates**

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

**Milestone:** IndoChainWallet usable on IndoChain Testnet first, with production/mainnet network selection kept explicit and isolated.

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
