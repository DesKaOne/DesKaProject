# IndoChain — Fitur Utama

Dokumen ini merangkum fitur utama IndoChain/DesKaChain berdasarkan implementasi dan dokumentasi project saat ini.

## 1. Consensus dan Blockchain Core

- **Proof-of-Work (PoW)** sebagai mekanisme pembuatan canonical block.
- **CPU mining** dan standalone miner `indominer`.
- **Dynamic difficulty** dengan target block time, retarget window, batas minimum/maksimum difficulty, dan cumulative work.
- **Account-based ledger** dengan nonce.
- **Transaction + mempool** untuk transaksi sebelum masuk block.
- **Coinbase maturity**: reward mining memiliki status confirmed, mature, immature, dan spendable.
- **Chain validation** untuk memeriksa integritas canonical chain.
- **Fork detection** dan fondasi safe reorg eksperimental berbasis cumulative work.
- **Mempool recovery** saat reorg untuk transaksi normal yang masih valid.

## 2. Native Coin

Native coin IndoChain adalah **dIDR**.

- Decimal: 8.
- 1 dIDR = 100.000.000 unit terkecil.
- Digunakan untuk reward PoW, transfer, faucet testnet, dan collateral staking.
- Supply canonical berasal dari reward coinbase yang masuk ke chain; faucet testnet menggunakan transaksi normal dari wallet faucet.

## 3. Wallet dan Address

- Pembuatan wallet/address.
- Format address baru `iND...` berbasis Base58Check.
- secp256k1 untuk key/signature.
- Balance inspection.
- Address validation.
- Export/import private key untuk workflow development.
- Transaction signing.

> Wallet RPC dan private-key management bukan komponen yang boleh dibuka ke public mainnet RPC.

## 4. Mining

### Node mining

Node dapat membuat block melalui RPC/CLI mining.

### Standalone miner

`indominer` bekerja terpisah dari node:

```text
Node -> block template
Miner -> CPU PoW
Miner -> submit block
Node -> validate + commit + broadcast
```

Miner tidak perlu membaca private key wallet.

Mining juga memiliki retry/backoff, stale-job detection, submit timeout control, statistik mining, dan diagnostik read-only.

## 5. Faucet Testnet

Faucet tersedia untuk **development/testnet**.

Flow:

```text
Faucet request
    -> faucet wallet
    -> signed normal transaction
    -> mempool
    -> mining
    -> confirmed
```

Faucet tidak mencetak supply secara langsung.

**Mainnet: faucet harus disabled/closed.**

## 6. Staking Collateral

IndoChain memiliki staking sebagai **collateral**, bukan PoS.

- Mature dIDR dapat dikunci sebagai collateral.
- Stake memiliki status/info yang dapat dibaca.
- Collateral dapat menjadi syarat eligibility service node.
- Pada desain saat ini staking tidak menghasilkan staking reward/APY.
- Staking tidak membuat validator dan tidak menggantikan PoW.

## 7. Service Node

Fondasi service node mencakup:

- registration;
- heartbeat;
- simulated challenge;
- challenge submission;
- scoring;
- eligibility;
- service/reward information.

Pada implementasi saat ini, service-node contribution masih berupa **simulation/research layer**. Service points bukan dIDR dan tidak mempengaruhi supply, difficulty, cumulative work, coinbase reward, balance, atau consensus.

## 8. P2P Network

IndoChain memiliki jaringan P2P dengan:

- handshake dan network/genesis guard;
- transaction broadcast;
- block broadcast;
- header-first sync;
- peer connect/list/status;
- seed peer/bootstrap;
- peer gossip/discovery;
- retry dengan bounded exponential backoff;
- peer TTL/pruning;
- peer reputation/scoring;
- recovery cooldown;
- diagnostik P2P.

Node memvalidasi network ID, chain ID, genesis hash, dan kompatibilitas protocol peer sebelum sync.

## 9. RPC API

RPC menyediakan akses ke:

- chain;
- block;
- transaction;
- address/balance;
- mempool;
- mining;
- faucet;
- staking;
- service node;
- peer/P2P;
- node health/status;
- explorer.

Public RPC memiliki mode hardening untuk membatasi endpoint sensitif.

## 10. IndoScan / Explorer

Explorer terdiri dari API read-only dan Web UI read-only.

Fitur utamanya:

- dashboard;
- block list/detail;
- transaction detail;
- address history;
- stake record;
- service-node simulation information;
- search;
- pagination.

Explorer dirancang sebagai lapisan observability publik terhadap blockchain, bukan sebagai wallet/admin interface.

## 11. Network Profile

IndoChain membedakan:

| Network | Chain ID | Network ID | Tujuan |
|---|---:|---|---|
| localnet | 777001 | ind-local-1 | Development |
| testnet | 777101 | ind-testnet-1 | Testing/public testnet |
| mainnet | 777000 | ind-main-1 | Production |

Address version juga dipisahkan berdasarkan network.

## 12. Public RPC Security

Untuk node yang diekspos ke internet, `--public-rpc` dirancang agar endpoint sensitif tidak tersedia secara default.

Prinsipnya:

- public node = **read-first/read-only**;
- wallet management = private;
- private key = private;
- admin/debug = private;
- faucet write = testnet/private;
- miner write = restricted;
- staking write = melalui signed transaction/wallet;
- service-node write = restricted sampai production protocol tersedia.

## 13. Release dan Operational Tooling

Project juga menyediakan fondasi operasional:

- build/package binary;
- checksum;
- release artifact;
- GitHub Actions CI/release workflow;
- public-testnet quickstart;
- seed-node deployment;
- multi-host testnet;
- restart recovery;
- long-running testnet operations;
- health/readiness checks.

---

## Ringkasan Arsitektur

```text
                    IndoChain
                       |
       +---------------+----------------+
       |               |                |
      PoW              P2P             RPC
       |               |                |
    Mining        Sync/Discovery      Apps
       |               |                |
       +---------------+----------------+
                       |
          +------------+-------------+
          |            |             |
         dIDR       Staking      Service Node
          |
       Faucet
       (testnet)

                       |
                  IndoScan
               Explorer read-only
```

## Prinsip Utama

**PoW tetap menjadi consensus canonical block.** Staking collateral dan service-node simulation tidak mengubah IndoChain menjadi PoS.

**dIDR adalah native coin.** Asset/token lain merupakan lapisan terpisah dari native coin.

**Public infrastructure tidak sama dengan wallet.** Node publik menyediakan data dan protocol yang diperlukan network; private key dan wallet management harus tetap berada pada client/operator yang tepat.
