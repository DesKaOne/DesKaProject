# WhitePaper DesKaChain

## Versi Testnet — Draft v0.1

**Status:** Draft awal untuk fase testnet
**Nama proyek:** DesKaChain
**Ticker sementara:** DKC
**Jenis jaringan:** Blockchain native berbasis komunitas
**Status ekonomi:** Testnet DKC tidak memiliki nilai ekonomi

---

# 1. Ringkasan

DesKaChain adalah proyek blockchain native yang dirancang sebagai jaringan terbuka, ringan, dan dapat diikuti oleh komunitas melalui node, mining, wallet, serta layanan jaringan tambahan seperti service node berbasis bandwidth.

Pada tahap awal, DesKaChain menggunakan mekanisme Proof-of-Work sebagai fondasi keamanan jaringan. Mining dilakukan untuk membuat block, memvalidasi transaksi, dan menjaga integritas chain. Selain mining block, DesKaChain juga merencanakan konsep service node yang dapat berkontribusi melalui uptime, latency, dan bandwidth yang diverifikasi.

Fase testnet digunakan untuk menguji stabilitas node, mining, wallet, transaksi, P2P network, fork handling, reorg, explorer, faucet, dan mekanisme reward testnet. Seluruh coin di testnet tidak memiliki nilai ekonomi, tidak dijanjikan dapat diperdagangkan, dan tidak boleh dianggap sebagai aset bernilai.

DesKaChain dirancang dengan prinsip jangka panjang: membangun jaringan yang sehat, transparan, dapat dimining, dapat digunakan, dan tidak bergantung pada skema harga, yield tetap, atau model ekonomi yang rawan runtuh.

---

# 2. Visi

Visi DesKaChain adalah membangun blockchain komunitas yang:

* dapat dijalankan secara terbuka,
* dapat dimining oleh perangkat umum,
* memiliki distribusi yang lebih adil,
* memiliki wallet dan explorer sendiri,
* dapat digunakan untuk eksperimen transaksi native coin,
* memiliki testnet publik yang aktif,
* dan berkembang secara bertahap menuju mainnet yang stabil.

DesKaChain tidak dirancang sebagai skema cepat kaya, token yield, atau stablecoin algoritmik. Nilai utama jaringan berasal dari utilitas, partisipasi node, mining, komunitas, dan pengembangan ekosistem.

---

# 3. Prinsip Dasar

DesKaChain mengikuti beberapa prinsip utama:

## 3.1 Fair Launch

Mainnet DesKaChain direncanakan menggunakan prinsip peluncuran yang transparan. Distribusi coin harus dijelaskan sejak awal melalui dokumen tokenomics dan genesis final.

## 3.2 Tidak Ada Janji Harga

DesKaChain tidak menjanjikan harga, keuntungan, APY tetap, atau return finansial apa pun. Harga pasar, jika suatu hari terbentuk, sepenuhnya ditentukan oleh mekanisme pasar dan utilitas jaringan.

## 3.3 Testnet Tidak Bernilai Ekonomi

Coin di testnet hanya digunakan untuk pengujian. Testnet DKC tidak memiliki harga, tidak boleh dijual, tidak boleh dianggap sebagai aset, dan tidak memiliki jaminan konversi otomatis ke mainnet.

## 3.3.1 Fondasi Protocol Phase 2.6.6

Address wallet baru DesKaChain memakai format `DKC` + Base58Check dengan payload `version byte + HASH160(compressed secp256k1 public key)`. Private key disimpan dan diexport sebagai raw 32-byte scalar hex. Address dev lama hanya didukung untuk kompatibilitas localnet/dev dan tidak menjadi format public testnet atau mainnet.

Network profile awal terdiri dari localnet, testnet, dan mainnet dengan chain id, network id, dan address version yang berbeda. Protocol version awal adalah `1`, P2P protocol version `dkc-p2p/1`, dan RPC API version `v1`.

## 3.4 Keamanan Sebelum Ekspansi

Fitur seperti wallet, explorer, miner desktop, bandwidth service node, staking, swap, dan bridge hanya dikembangkan setelah fondasi chain stabil.

## 3.5 Tidak Menggunakan Model Luna

DesKaChain tidak menggunakan stablecoin algoritmik, tidak menggunakan mint/burn untuk mempertahankan peg, tidak menjanjikan yield tinggi, dan tidak membuat sistem ekonomi yang bergantung pada pertumbuhan harga secara terus-menerus.

---

# 4. Masalah yang Ingin Diselesaikan

Banyak proyek blockchain kecil gagal karena terlalu cepat mengejar hype, listing, harga, atau skema reward tanpa fondasi teknis yang kuat. Akibatnya, jaringan tidak memiliki node sehat, wallet tidak stabil, explorer tidak tersedia, dan ekonomi token menjadi rapuh.

DesKaChain mencoba membangun dari bawah:

1. Core blockchain terlebih dahulu.
2. P2P network yang stabil.
3. Fork dan reorg yang aman.
4. Wallet dan miner yang mudah digunakan.
5. Explorer dan faucet untuk testnet.
6. Komunitas mining.
7. Service node dan bandwidth reward secara bertahap.
8. Mainnet hanya setelah testnet cukup matang.

---

# 5. Gambaran Teknis

DesKaChain adalah blockchain account-based yang menyimpan saldo berdasarkan address. Pada fase awal, jaringan memiliki fitur:

* genesis block,
* block validation,
* transaksi,
* saldo akun,
* nonce transaksi,
* mempool,
* mining,
* P2P block broadcast,
* P2P transaction broadcast,
* fork detection,
* common ancestor detection,
* safe reorg,
* chain validation,
* RPC node,
* dan CLI wallet.

Arsitektur awal terdiri dari:

* Node core,
* Wallet CLI,
* Mining module,
* P2P networking,
* RPC API,
* Storage,
* Ledger,
* Mempool,
* Reorg module,
* dan config network.

---

# 6. Consensus Awal

Pada tahap testnet, DesKaChain menggunakan Proof-of-Work sebagai mekanisme dasar untuk membuat block.

Proof-of-Work dipilih karena:

* sederhana untuk diverifikasi,
* cocok untuk fair mining,
* tidak membutuhkan validator set sejak awal,
* dapat diuji secara lokal maupun publik,
* dan menjadi fondasi yang lebih mudah dibanding langsung memulai dengan PoS penuh.

Difficulty mining dikembangkan bertahap. Sejak Phase 2.7, DesKaChain memakai difficulty adjustment sederhana berdasarkan target block time dan retarget window. Localnet memakai target block time 10 detik, retarget window 10 block, dan perubahan difficulty konservatif maksimal 1 per window.

---

# 7. Mining

DesKaChain merencanakan dua jenis kontribusi jaringan:

## 7.1 Block Mining

Block mining adalah proses membuat block baru menggunakan Proof-of-Work. Miner mencari nonce yang menghasilkan hash valid sesuai difficulty jaringan.

Block miner bertugas:

* mengambil transaksi dari mempool,
* membuat block candidate,
* mencari hash valid,
* mengirim block ke node,
* dan menerima block reward jika block diterima jaringan.

Pada fase awal, mining dilakukan melalui CLI dan RPC node. Ke depannya, mining akan dipisahkan menjadi standalone miner agar node dan miner dapat berjalan secara terpisah.

Target dukungan:

* Desktop Windows,
* Linux amd64,
* Linux arm64,
* dan perangkat server kecil.

GPU mining akan diteliti setelah CPU miner stabil dan algoritma PoW sudah lebih final.

## 7.2 Bandwidth / Service Node

Selain block mining, DesKaChain merencanakan service node berbasis kontribusi jaringan seperti:

* uptime,
* latency,
* bandwidth,
* availability,
* dan respons terhadap challenge jaringan.

Namun, bandwidth mining tidak akan langsung menjadi consensus block mining. Pada tahap awal, bandwidth mining akan diposisikan sebagai service reward layer, bukan penentu block canonical.

Hal ini penting untuk mencegah:

* fake traffic,
* self-farming,
* abuse proxy,
* farming multi-akun,
* dan risiko jaringan digunakan sebagai exit proxy publik yang tidak aman.

Service node akan diuji terlebih dahulu pada testnet dan reward-nya akan bersifat simulasi atau terbatas sesuai aturan testnet.

---

# 8. Staking dan PoS

DesKaChain mempertimbangkan staking sebagai fitur masa depan, tetapi tidak langsung menjadikan staking sebagai consensus utama.

Pada tahap awal, staking lebih cocok digunakan sebagai:

* collateral service node,
* syarat partisipasi reward tertentu,
* anti-abuse layer,
* dan fondasi riset hybrid PoW + PoS.

PoS penuh membutuhkan desain yang jauh lebih kompleks, termasuk validator set, slashing, epoch, finality, delegation, randomness, downtime penalty, dan perlindungan terhadap long-range attack.

Karena itu, DesKaChain akan memulai dari:

1. PoW sebagai fondasi chain.
2. Staking sebagai module ekonomi/collateral.
3. PoS atau hybrid consensus sebagai riset lanjutan.

Staking DesKaChain tidak boleh dipasarkan sebagai janji profit atau APY tetap.

---

# 9. Address dan Wallet

DesKaChain akan menggunakan format address final sebelum public testnet.

Rencana address:

* prefix: `DKC`,
* encoding: Base58Check,
* payload: version byte + public key hash + checksum,
* private key: raw 32-byte hex.

Format address dev lama hanya digunakan pada fase dev/localnet awal dan dapat dipertahankan sementara untuk migrasi atau testing internal.

Wallet akan dikembangkan bertahap:

1. Wallet CLI.
2. Wallet desktop.
3. Wallet mobile.
4. Wallet web.
5. Wallet SDK.

Sebelum wallet publik dirilis, format address harus sudah final agar tidak terjadi migrasi besar setelah pengguna mulai memakai wallet.

---

# 10. Tokenomics Awal

Tokenomics final DesKaChain akan ditentukan sebelum mainnet. Pada fase testnet, angka-angka berikut masih dapat berubah.

Prinsip tokenomics:

* block reward jelas,
* supply schedule transparan,
* tidak ada mint tidak terbatas untuk mempertahankan harga,
* tidak ada stablecoin algoritmik,
* tidak ada APY tetap tinggi,
* tidak ada mekanisme yang bergantung pada user baru,
* dan tidak ada janji harga.

Reward awal localnet/testnet digunakan hanya untuk pengujian.

Contoh parameter sementara:

* block reward: 50 DKC,
* decimals: 8,
* coinbase maturity: aktif sejak Phase 2.8,
* dynamic difficulty: aktif sejak Phase 2.7 untuk localnet/testnet,
* total supply final: TBD,
* emission schedule final: TBD,
* dev fund atau treasury: TBD dan harus transparan jika digunakan.

---

# 11. Testnet

Testnet DesKaChain adalah jaringan pengujian publik sebelum mainnet. Tujuannya adalah menguji:

* node,
* mining,
* transaksi,
* mempool,
* fork,
* reorg,
* wallet,
* explorer,
* faucet,
* public RPC,
* service node,
* dan partisipasi komunitas.

Testnet dapat mengalami reset, hard fork, perubahan genesis, perubahan difficulty, perubahan address format, atau perubahan aturan lain sesuai kebutuhan pengembangan.

Coin testnet tidak memiliki nilai ekonomi.

---

# 12. Program Klaim Mainnet Terbatas

DesKaChain dapat menyediakan program klaim mainnet terbatas untuk menghargai partisipasi komunitas pada fase testnet.

Program ini bukan berarti coin testnet memiliki harga. Testnet DKC tetap tidak bernilai ekonomi.

Program klaim dapat menggunakan prinsip:

* snapshot aktivitas testnet,
* batas maksimal klaim per wallet,
* batas maksimal total claim pool,
* periode klaim terbatas,
* anti-abuse checks,
* signature verification,
* dan expiration setelah periode klaim selesai.

Contoh alur:

1. Komunitas menjalankan node atau mining di testnet.
2. Aktivitas testnet dicatat sampai snapshot height tertentu.
3. Setelah mainnet launch, pengguna yang memenuhi syarat dapat melakukan klaim terbatas.
4. Pengguna harus membuktikan kepemilikan address testnet melalui signature.
5. Klaim dikirim ke address mainnet.
6. Setelah periode klaim selesai, eligibility yang tidak diklaim akan hangus.
7. Coin testnet tetap tidak dapat ditukar dan tidak memiliki nilai.

Aturan detail program klaim akan diumumkan sebelum public testnet atau sebelum mainnet, bukan setelahnya.

---

# 13. Anti-Abuse

Karena testnet mining dapat dilakukan secara bebas, DesKaChain perlu menerapkan anti-abuse untuk mencegah farming tidak sehat.

Potensi abuse:

* multi-wallet farming,
* spam transaksi,
* fake mining activity,
* fake bandwidth proof,
* node palsu,
* self-farming,
* dan eksploitasi faucet.

Mitigasi awal:

* cap per wallet,
* cap total reward,
* snapshot height,
* signature claim,
* activity scoring,
* blacklist obvious abuse,
* rate limit faucet,
* peer scoring,
* dan service node verification.

Anti-abuse tidak bertujuan membuat jaringan tertutup, tetapi menjaga agar partisipasi komunitas lebih adil.

---

# 14. Roadmap Testnet

## Phase 1 — Local Blockchain Core

Status: selesai.

* Genesis block
* Block validation
* Local mining
* Wallet generate
* Balance check
* Send transaction
* Mempool basic
* Local storage
* Chain validate

## Phase 2 — P2P, Sync, Fork, and Reorg Foundation

Status: selesai.

* Multi-node localnet
* RPC server
* P2P server
* Peer sync
* Transaction broadcast
* Block broadcast
* Fork detection
* Common ancestor
* Safe reorg
* Reorg mempool recovery

## Phase 2.6.5 — Runtime Stats & Safety Cleanup

Status: rencana berikutnya.

* Chain info stats fix
* Mempool atomic write
* Duplicate transaction guard
* Runtime safety cleanup
* Improved `.gitignore`

## Phase 2.6.6 — Protocol Spec Freeze & DKC Address Migration

Status: implemented.

* DKC Base58Check address
* Private key hex
* Protocol versioning
* Network profiles
* Localnet/testnet/mainnet config

## Phase 2.7 — Difficulty Adjustment

Status: planned.

* Target block time
* Retarget window
* Dynamic difficulty
* Realistic cumulative work
* Difficulty validation

## Phase 2.8 — Coinbase Maturity

Status: implemented.

* Mature balance
* Immature balance
* Spendable balance
* Prevent spending immature rewards
* Safer reorg behavior
* Circulating supply based on mature coinbase rewards

## Phase 2.9 — Standalone CPU Miner CLI

Status: implemented.

* Miner separate from node
* RPC block template
* RPC submit block
* Hashrate
* Thread control
* Node-side block validation and broadcast

## Phase 2.10 — Node Public Hardening

Status: implemented.

* RPC/P2P HTTP timeout and max header hardening
* RPC body size limits with explicit JSON error responses
* Basic in-memory per-IP RPC rate limiting
* Public RPC mode with wallet/admin RPC disabled by default
* Explicit wallet, miner, and admin RPC gates
* CORS allowlist support
* Max peer defaults per network profile
* Health and readiness endpoints
* Graceful RPC/P2P shutdown
* JSON config file foundation

## Phase 3.x — Service Node, Staking, GPU Mining, and Pool

Status: research/planned.

Phase 3.0 adds a service-node research layer only. Service nodes can register, heartbeat, submit simulated challenge measurements, receive service scores, and accumulate simulated service points. Phase 3.1 adds a standalone `dkcservice` safe-mode agent that automates those RPC calls and stores local agent state. These points are not DKC, are not spendable, and do not affect consensus, PoW difficulty, cumulative work, supply, coinbase rewards, balances, or chain validation.

Phase 3.2 adds staking collateral for service-node eligibility. This staking module only locks and unlocks DKC through canonical chain transactions. It is not Proof-of-Stake, does not create validators, does not select block producers, does not mint staking rewards, and does not slash DKC in this phase. Active and unbonding stake reduce spendable balance until released.

* Bandwidth service node research
* Service reward simulation
* Staking collateral
* Hybrid PoS research
* GPU mining research
* Mining pool MVP

---

# 15. Risiko

DesKaChain masih berada pada tahap pengembangan awal. Risiko meliputi:

* bug pada consensus,
* bug pada reorg,
* bug pada wallet,
* kehilangan private key,
* perubahan address format sebelum final,
* reset testnet,
* abuse testnet,
* ketidakstabilan public RPC,
* risiko mining tidak seimbang,
* risiko service node dimanipulasi,
* dan risiko ekonomi jika tokenomics tidak dirancang dengan hati-hati.

Karena itu, seluruh fase testnet harus dianggap eksperimental.

---

# 16. Prinsip Anti-Luna

DesKaChain secara eksplisit menghindari model ekonomi yang rapuh.

Aturan dasar:

1. Tidak membuat stablecoin algoritmik pada fase awal.
2. Tidak membuat peg undercollateralized.
3. Tidak menggunakan mint tidak terbatas untuk mempertahankan harga.
4. Tidak menjanjikan APY tetap tinggi.
5. Tidak membayar reward dari user baru.
6. Tidak menjadikan DKC sebagai satu-satunya jaminan aset stabil.
7. Tidak memasarkan DKC sebagai pendapatan pasti.
8. Tidak membuat bandwidth mining sebagai passive income tanpa verifikasi.
9. Tidak membuat staking sebagai mesin profit otomatis.
10. Tidak menjadikan harga sebagai fondasi utama proyek.

DesKaChain fokus pada utilitas, jaringan, mining, wallet, explorer, dan komunitas.

---

# 17. Posisi Ekonomi DKC

DKC adalah native coin jaringan DesKaChain. Fungsi yang direncanakan:

* membayar transaksi,
* menerima block reward,
* digunakan dalam wallet,
* digunakan dalam service node reward,
* digunakan dalam staking/collateral masa depan,
* dan menjadi unit dasar ekosistem DesKaChain.

DKC bukan stablecoin, bukan yield token, bukan synthetic asset, dan bukan jaminan keuntungan.

---

# 18. Kesimpulan

DesKaChain adalah proyek blockchain native yang dibangun secara bertahap dari core chain, P2P network, mining, fork handling, reorg, wallet, explorer, service node, hingga public testnet dan mainnet.

Fokus utama DesKaChain adalah membangun jaringan yang dapat bertahan lama, bukan menciptakan hype jangka pendek. Testnet digunakan untuk menguji teknologi, membangun komunitas, dan memperbaiki sistem sebelum mainnet.

Coin testnet tidak memiliki nilai ekonomi. Jika program klaim mainnet diterapkan, program tersebut akan bersifat terbatas, memiliki cap, memiliki periode klaim tertentu, dan tunduk pada anti-abuse checks.

DesKaChain bertujuan menjadi jaringan yang sederhana, terbuka, dapat dimining, dan berkembang secara sehat bersama komunitas.
