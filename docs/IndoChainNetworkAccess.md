# IndoChain — Network Access Policy

Dokumen ini menetapkan pemisahan akses fitur IndoChain berdasarkan environment.

## Prinsip

- **Localnet:** semua fitur boleh dibuka untuk development/debugging.
- **Testnet:** fitur operasional dibuka agar seluruh flow dapat diuji.
- **Mainnet:** public interface bersifat read-first/read-only; operasi sensitif dibatasi ke wallet, miner, atau operator channel yang sesuai.

> **Mainnet public RPC bukan admin console dan bukan wallet.**

## Access Matrix

| Fitur | Localnet | Testnet | Mainnet |
|---|:---:|:---:|:---:|
| Chain info | OPEN | OPEN | PUBLIC READ |
| Block lookup | OPEN | OPEN | PUBLIC READ |
| Transaction lookup | OPEN | OPEN | PUBLIC READ |
| Address/balance | OPEN | OPEN | PUBLIC READ |
| Mempool read | OPEN | OPEN | PUBLIC READ* |
| P2P | OPEN | OPEN | OPEN |
| Peer discovery/bootstrap | OPEN | OPEN | OPEN |
| Chain validation | OPEN | OPEN | PUBLIC READ |
| Health/readiness | OPEN | OPEN | PUBLIC READ |
| Explorer/IndoScan | OPEN | OPEN | PUBLIC |
| CPU mining | OPEN | OPEN | OPEN* |
| Miner template/submit RPC | OPEN | OPEN | RESTRICTED |
| Wallet management RPC | OPEN | OPEN | CLOSED |
| Private-key operations | OPEN | OPEN | CLOSED |
| Send/write transaction API | OPEN | OPEN | RESTRICTED |
| Faucet | OPEN | OPEN | CLOSED |
| Faucet info | OPEN | OPEN | CLOSED/disabled |
| Stake info/status | OPEN | OPEN | PUBLIC READ |
| Stake lock/unlock | OPEN | OPEN | RESTRICTED |
| Service read/score | OPEN | OPEN | PUBLIC READ* |
| Service register/heartbeat/challenge write | OPEN | OPEN | CLOSED/RESTRICTED* |
| Admin RPC | OPEN | OPEN | CLOSED |
| Debug RPC | OPEN | OPEN | CLOSED |
| Mempool clear | OPEN | OPEN | CLOSED |
| Reorg preview | OPEN | OPEN | OPERATOR ONLY |
| Reorg apply | OPEN | OPEN | CLOSED/OPERATOR ONLY |

* Hanya endpoint/informasi yang memang aman untuk publik; detail implementasi dapat diperketat sebelum mainnet.

## Localnet

Localnet adalah environment development.

Boleh membuka:

- wallet;
- private-key tooling;
- mining;
- faucet;
- staking;
- service node;
- admin;
- debug;
- reorg tooling;
- RPC write;
- P2P;
- explorer.

Tujuan localnet adalah iterasi cepat dan pengujian internal, bukan security boundary production.

## Testnet

Testnet harus mempertahankan fitur secara fungsional agar flow end-to-end dapat diuji.

Boleh membuka:

- mining;
- faucet;
- wallet/RPC;
- staking collateral;
- service-node simulation;
- P2P;
- explorer;
- admin/debug pada node operator;
- reorg tooling untuk pengujian;
- RPC write yang dibutuhkan test scenario.

Tetap gunakan guard network/testnet dan jangan memperlakukan dIDR testnet sebagai aset bernilai ekonomi.

## Mainnet Public

Public mainnet membuka data blockchain yang dibutuhkan untuk transparansi dan integrasi:

### Public read

- chain info;
- block;
- transaction;
- address;
- balance;
- safe mempool information;
- network information;
- health/readiness;
- safe P2P metadata;
- explorer.

### P2P

P2P tetap terbuka karena merupakan bagian dari network consensus/propagation:

- handshake;
- headers;
- block propagation;
- transaction propagation;
- peer discovery/bootstrap;
- sync.

Endpoint P2P harus tetap memiliki validation, rate limiting, ukuran request yang wajar, dan guard terhadap peer/network yang incompatible.

## Mainnet Restricted

### Mining

PoW mining **tetap aktif** di mainnet.

Namun miner sebaiknya menggunakan jalur mining yang terpisah/restricted. Jangan mengekspos miner RPC secara bebas pada public read RPC.

Contoh arsitektur:

```text
Public RPC -> read-only
Miner      -> restricted miner RPC
P2P        -> public network protocol
```

### Transaction submission

Pengguna tetap harus dapat mengirim transaksi mainnet.

Private key harus berada di wallet/client.

Model yang disarankan:

```text
Wallet
  -> sign locally
  -> signed transaction
  -> broadcast
  -> mempool
  -> mining
```

Node tidak seharusnya menerima private key user melalui public RPC.

### Staking

Read:

- stake info;
- stake list;
- stake status.

Write:

- stake lock;
- stake unlock.

Operasi write harus melalui transaksi yang signed dan channel yang sesuai, bukan endpoint admin terbuka.

## Mainnet Closed

### Faucet

Mainnet tidak menyediakan faucet.

### Wallet management

Public mainnet node tidak boleh:

- membuat wallet user;
- menyimpan private key user;
- mengekspos private key;
- menyediakan wallet export melalui public RPC.

### Admin/debug

Tutup dari public internet:

- debug;
- lock inspection;
- admin mutation;
- peer clear;
- mempool clear;
- development reset;
- internal diagnostics yang membocorkan informasi sensitif.

### Reorg apply

Reorg apply bukan public operation.

Jika diperlukan untuk incident/operator procedure, jalankan melalui operator-controlled infrastructure dengan explicit confirmation.

## Service Node

Service-node feature saat ini masih memiliki simulation/research components.

Karena itu:

- localnet: open;
- testnet: open untuk pengujian;
- mainnet: jangan mengaktifkan payout/service-write production sebelum protocol service node benar-benar production-ready;
- public read hanya untuk data yang aman dipublikasikan.

Service points tidak boleh diperlakukan sebagai dIDR.

## Security Rule

Untuk mainnet:

```text
PUBLIC
  |
  +-- READ blockchain data
  +-- READ safe network data
  +-- P2P protocol
  +-- Explorer
  |
PRIVATE / RESTRICTED
  |
  +-- Wallet management
  +-- Private keys
  +-- Admin
  +-- Debug
  +-- Faucet
  +-- Reorg apply
  +-- Miner write RPC
  +-- Service write
  +-- Sensitive staking operations
```

## Recommended Mainnet Topology

```text
                    INTERNET
                       |
          +------------+------------+
          |                         |
       IndoScan                  Wallet
          |                         |
          v                         v
    Public Read RPC          Signed TX Broadcast
          |                         |
          +------------+------------+
                       |
                  Mainnet Node
                       |
                 +-----+-----+
                 |           |
                P2P         PoW
                 |           |
              Network      Miners

        Operator / Admin Network
                  |
          Restricted RPC only
```

## Final Rule

**Localnet = full open.**

**Testnet = full functional/open untuk pengujian.**

**Mainnet = public read + P2P + kebutuhan protocol; semua operasi sensitif restricted/private.**

Policy ini harus menjadi baseline ketika endpoint baru ditambahkan: setiap endpoint baru wajib dikategorikan sebagai **PUBLIC READ**, **PUBLIC PROTOCOL**, **RESTRICTED WRITE**, atau **PRIVATE/OPERATOR** sebelum diaktifkan pada mainnet.
