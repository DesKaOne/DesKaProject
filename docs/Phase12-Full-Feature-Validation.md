# Phase 12 — Full Feature Validation Checklist

> Checkpoint for validating **IndoChain + IndoScan** before VPS private-testnet deployment.
>
> **IndoChainWallet is intentionally frozen and outside the Phase 12 application scope.**
>
> Source baseline:
> - `docs/IndoChainFeatures.md`
> - `docs/IndoChainNetworkAccess.md`

## Status Legend
- ⬜ Not validated
- 🟡 In progress / partial
- 🟢 Validated
- 🔴 Failed / blocking
- ⚪ Not applicable

## 1. IndoChain Core
| ID | Feature | Status | Evidence / Notes |
|---|---|---:|---|
| IC-01 | Proof-of-Work canonical block | ⬜ | |
| IC-02 | CPU mining | ⬜ | |
| IC-03 | Dynamic difficulty | ⬜ | |
| IC-04 | Account + nonce | ⬜ | |
| IC-05 | Transaction + mempool | ⬜ | |
| IC-06 | Coinbase maturity | ⬜ | |
| IC-07 | Chain validation | ⬜ | |
| IC-08 | Fork detection | ⬜ | |
| IC-09 | Safe reorg | ⬜ | |
| IC-10 | Mempool recovery after reorg | ⬜ | |
| IC-11 | dIDR balance / transfer | ⬜ | |
| IC-12 | Address validation | ⬜ | |
| IC-13 | Transaction signing primitives | ⬜ | |
| IC-14 | Standalone `indominer` | ⬜ | |
| IC-15 | Testnet faucet | ⬜ | |
| IC-16 | Staking collateral | ⬜ | |
| IC-17 | Service-node simulation | ⬜ | |

## 2. P2P Network
| ID | Feature | Status | Evidence / Notes |
|---|---|---:|---|
| P2P-01 | Handshake | 🟢 | Local two-node validation passed |
| P2P-02 | Network / genesis guard | ⬜ | |
| P2P-03 | Transaction broadcast | ⬜ | |
| P2P-04 | Block broadcast | ⬜ | |
| P2P-05 | Header-first sync | ⬜ | |
| P2P-06 | Peer connect / list / status | 🟢 | Peer status/recovery validated |
| P2P-07 | Bootstrap / seed | ⬜ | |
| P2P-08 | Peer gossip / discovery | ⬜ | |
| P2P-09 | Retry / bounded backoff | 🟢 | Recovery behavior observed |
| P2P-10 | Peer TTL / pruning | ⬜ | |
| P2P-11 | Peer reputation / scoring | 🟢 | Score/liveness separation validated |
| P2P-12 | Recovery cooldown | 🟢 | Recovery maintenance validated |
| P2P-13 | Peer recovery | 🟢 | Authenticated peer recovered to active |

## 3. RPC / API
| ID | Feature | Status | Evidence / Notes |
|---|---|---:|---|
| RPC-01 | Chain API | ⬜ | |
| RPC-02 | Block API | ⬜ | |
| RPC-03 | Transaction API | ⬜ | |
| RPC-04 | Address / balance API | ⬜ | |
| RPC-05 | Mempool API | ⬜ | |
| RPC-06 | Mining API | ⬜ | |
| RPC-07 | Faucet API | ⬜ | |
| RPC-08 | Staking API | ⬜ | |
| RPC-09 | Service-node API | ⬜ | |
| RPC-10 | Peer / P2P API | 🟢 | Peer checks/status validated |
| RPC-11 | Health / status API | ⬜ | |
| RPC-12 | Explorer API | ⬜ | |
| RPC-13 | Public-RPC hardening | ⬜ | |

## 4. IndoScan
| ID | Feature | Status | Evidence / Notes |
|---|---|---:|---|
| IS-01 | Dashboard | ⬜ | |
| IS-02 | Block list | ⬜ | |
| IS-03 | Block detail | ⬜ | |
| IS-04 | Transaction detail | ⬜ | |
| IS-05 | Address history | ⬜ | |
| IS-06 | Stake record | ⬜ | |
| IS-07 | Service-node information | ⬜ | |
| IS-08 | Search | ⬜ | |
| IS-09 | Pagination | ⬜ | |
| IS-10 | Read-only API | ⬜ | |
| IS-11 | Read-only Web UI | ⬜ | |
| IS-12 | RPC / API consistency | ⬜ | |

## 5. Network Profiles
| ID | Profile | Expected identity | Status | Evidence / Notes |
|---|---|---|---:|---|
| NET-01 | localnet | chain 777001 / `ind-local-1` | ⬜ | |
| NET-02 | testnet | chain 777101 / `ind-testnet-1` | ⬜ | |
| NET-03 | mainnet | chain 777000 / `ind-main-1` | ⬜ | |
| NET-04 | Genesis compatibility | Matching genesis required | ⬜ | |
| NET-05 | Address version separation | Network-specific address version | ⬜ | |
| NET-06 | Peer compatibility guard | Network/chain/genesis guard | ⬜ | |
| NET-07 | Environment guard | No accidental cross-network operation | ⬜ | |

## 6. Network Access / Security Matrix

Validate behavior described in `docs/IndoChainNetworkAccess.md`.

### Localnet
- [ ] Full development feature set available where intended
- [ ] Wallet/private-key tooling available for development
- [ ] Mining available
- [ ] Faucet available
- [ ] Staking available
- [ ] Service-node simulation available
- [ ] Admin/debug available
- [ ] Reorg tooling available
- [ ] RPC write available
- [ ] P2P available
- [ ] Explorer available

### Testnet
- [ ] Functional end-to-end feature set available for testing
- [ ] Mining available
- [ ] Faucet available
- [ ] Wallet/RPC flow available
- [ ] Staking collateral flow available
- [ ] Service-node simulation available
- [ ] P2P available
- [ ] Explorer available
- [ ] Operator admin/debug available on intended operator node
- [ ] Test-only reorg tooling available where intended
- [ ] Required RPC writes available

### Mainnet public boundary
- [ ] Chain info = public read
- [ ] Block lookup = public read
- [ ] Transaction lookup = public read
- [ ] Address/balance = public read
- [ ] Safe mempool information = public read
- [ ] Network/health information = public read
- [ ] Safe P2P metadata = public read
- [ ] Explorer = public
- [ ] P2P protocol = public
- [ ] Wallet management = closed
- [ ] Private-key operations = closed
- [ ] Admin/debug = closed
- [ ] Faucet = closed
- [ ] Miner write RPC = restricted
- [ ] Sensitive staking operations = restricted
- [ ] Service-node write = restricted/closed until production-ready
- [ ] Reorg apply = operator-only
- [ ] Public RPC is not an admin console or wallet

## 7. Persistence / Recovery
| ID | Drill | Status | Evidence / Notes |
|---|---|---:|---|
| REC-01 | Stop/start node | ⬜ | |
| REC-02 | Datadir persistence | ⬜ | |
| REC-03 | Node identity persistence | ⬜ | |
| REC-04 | Chain state recovery | ⬜ | |
| REC-05 | Stale datadir lock recovery | ⬜ | |
| REC-06 | Active-node lock protection | ⬜ | |
| REC-07 | P2P reconnect after restart | ⬜ | |
| REC-08 | RPC reconnect after restart | ⬜ | |
| REC-09 | IndoScan consistency after restart | ⬜ | |

## 8. Local E2E — Mini PC ↔ Main PC
| ID | Scenario | Status | Evidence / Notes |
|---|---|---:|---|
| E2E-01 | Start server node on Mini PC | ⬜ | |
| E2E-02 | Main PC reaches RPC | ⬜ | |
| E2E-03 | Main PC reaches P2P | 🟢 | |
| E2E-04 | Network / chain / genesis match | 🟢 | |
| E2E-05 | Chain height consistency | 🟢 | |
| E2E-06 | Transaction submission | ⬜ | |
| E2E-07 | Transaction propagation | ⬜ | |
| E2E-08 | Transaction confirmation | ⬜ | |
| E2E-09 | IndoScan reads canonical data | ⬜ | |
| E2E-10 | Restart and reconnect | ⬜ | |
| E2E-11 | Full local deployment drill has no blocking errors | ⬜ | |

## 9. CI / Regression
| ID | Check | Status | Evidence / Notes |
|---|---|---:|---|
| CI-01 | Existing unit/regression tests pass | ⬜ | |
| CI-02 | P2P recovery regression test passes | 🟢 | Added for peer status recovery |
| CI-03 | Stale-lock regression test passes | ⬜ | |
| CI-04 | Full CI workflow green | 🔴 | Current checkpoint requires CI failure investigation |
| CI-05 | No known blocking CI failure remains | ⬜ | |

> A branch being mergeable / having no base-branch conflict is not equivalent to green CI.

## 10. Phase 12 Exit Gate

Phase 12 may be marked **🟢 Complete** only when:
- [ ] IndoChain core validation is complete
- [ ] P2P validation is complete
- [ ] RPC/API validation is complete
- [ ] IndoScan validation is complete
- [ ] Network profile validation is complete
- [ ] Access/security matrix validation is complete
- [ ] Persistence/recovery drill is complete
- [ ] Mini PC ↔ Main PC E2E is complete
- [ ] CI is green
- [ ] No blocking node/client errors remain
- [ ] Evidence/checkpoints are recorded in this document

### Current checkpoint

**Phase 12: 🟡 In Progress**

Known green area:
- P2P peer recovery / liveness recovery

Known work:
- Full IndoChain feature validation
- Full IndoScan validation
- Persistence/restart drill
- Stale-lock validation
- CI failure investigation
- Complete local E2E

## 11. Scope Boundary

### Included in Phase 12
```
IndoChain
    +
IndoScan
    +
Local deployment validation
```

### Frozen / deferred
```
IndoChainWallet
```

IndoChainWallet application development remains frozen until IndoChain + IndoScan infrastructure is stable and ready for VPS deployment.

Intended sequence:
```
Phase 12
IndoChain + IndoScan
        │
        ▼
Phase 13
Private IndoChain Testnet on VPS
        │
        ▼
Stable RPC / API domains
        │
        ▼
IndoChainWallet
```

This keeps wallet integration independent from temporary LAN IP addresses and allows the wallet to consume stable deployment endpoints.
