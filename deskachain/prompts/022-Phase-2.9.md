Kamu sedang bekerja pada project Go monorepo IndoChain.

Struktur project:

* node/

  * cmd/indochain/
  * internal/
  * go.mod
  * go.sum
* root repo punya:

  * go.work
  * README.md
  * Roadmap.md
  * docs/
  * prompts/
  * testdata/
  * data/

Status saat ini:

* Phase 1 sampai Phase 2.8 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... pass.
* Address final sudah aktif:

  * wallet baru menghasilkan address `iND...`
  * format address: `iND` + Base58Check
  * private key raw 32-byte hex
* Dynamic difficulty sudah aktif:

  * localnet target block time 10s
  * retarget window 10
  * next difficulty calculation valid
  * cumulative work memakai `16^difficulty`
* Coinbase maturity sudah aktif:

  * localnet maturity 10 blocks
  * mining reward belum langsung spendable
  * balance punya confirmed/mature/immature/spendable
  * send memakai spendable balance
  * circulating supply berdasarkan mature coinbase
* Fitur yang sudah ada:

  * local blockchain,
  * account ledger,
  * wallet CLI,
  * send transaction,
  * nonce,
  * mempool,
  * mining built-in node,
  * RPC node,
  * P2P node,
  * block broadcast,
  * tx broadcast,
  * fork detection,
  * common ancestor,
  * safe reorg preview/apply,
  * reorg mempool recovery,
  * runtime stats cleanup,
  * iND Base58Check address,
  * network profile foundation,
  * protocol metadata,
  * dynamic difficulty,
  * realistic cumulative work,
  * coinbase maturity.

Nama patch:
IndoChain Phase 2.9 — Standalone CPU Miner CLI

Tujuan:
Memisahkan proses mining dari node agar miner bisa berjalan sebagai aplikasi/CLI terpisah.

Saat ini mining masih dilakukan lewat command node:

indochain mine --address <ADDR> --blocks N

Phase 2.9 harus menambahkan:

* RPC block template,
* RPC submit block,
* standalone CPU miner CLI,
* multi-thread CPU mining,
* hashrate reporting,
* stale template handling,
* reconnect/retry,
* clean shutdown,
* block validation tetap di node,
* block broadcast setelah submit diterima.

Aturan penting:

* Jangan ubah address format `iND...`.
* Jangan rollback Base58Check.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah cumulative work formula.
* Jangan ubah coinbase maturity.
* Jangan implement GPU miner.
* Jangan implement mining pool.
* Jangan implement staking.
* Jangan implement PoS.
* Jangan implement bandwidth mining.
* Jangan implement explorer.
* Jangan rewrite besar.
* Patch incremental.
* Semua command lama harus tetap bekerja.
* Built-in mining command lama boleh tetap ada.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...

==================================================

1. Tambahkan RPC block template
   ==================================================

Tambahkan endpoint RPC untuk miner mengambil template block:

GET /miner/template?address=<IND_ADDRESS>

atau jika pattern RPC existing memakai POST, boleh:

POST /miner/template
{
"address": "iND..."
}

Pilih style yang konsisten dengan RPC existing.

Behavior:

* Validate reward address.
* Ambil canonical tip terbaru.
* Hitung next height.
* Hitung next difficulty dari Phase 2.7.
* Ambil pending transactions dari mempool.
* Filter tx yang valid terhadap current canonical state.
* Buat block candidate/template dengan:

  * height,
  * previous hash,
  * difficulty,
  * timestamp,
  * coinbase tx ke reward address,
  * selected mempool txs,
  * tx root / merkle root / transaction digest sesuai desain existing,
  * network id,
  * chain id,
  * protocol version,
  * template id.

Template response minimal:

{
"template_id": "hex-or-string",
"network": "localnet",
"chain_id": 777001,
"height": 12,
"previous_hash": "...",
"difficulty": 4,
"target": "0000ffffffff...",
"reward_address": "iND...",
"coinbase_reward": "50",
"timestamp": 1781910000,
"transactions": [...],
"tx_count": 1,
"header": {...}
}

Catatan:

* Jangan kirim private key ke miner.
* Miner hanya butuh reward address.
* Node tetap pihak yang memvalidasi submitted block.
* Template boleh berisi full tx serialized atau block candidate serialized, sesuai desain existing.
* Jika block struct existing sudah mudah dikirim full, boleh return unsigned/unmined block candidate dengan nonce 0.
* Jangan expose wallet/private key via miner endpoint.

==================================================
2. Tambahkan RPC submit block
=============================

Tambahkan endpoint:

POST /miner/submit

Request minimal:

{
"template_id": "...",
"block": {...}
}

atau jika lebih cocok:

{
"template_id": "...",
"nonce": 123456,
"timestamp": 1781910000,
"hash": "0000..."
}

Pilih format yang paling aman dan konsisten dengan existing block validation.

Behavior:

* Decode submitted block.
* Validasi block penuh:

  * previous hash harus sama dengan current tip atau masih valid untuk tip saat submit.
  * height benar.
  * expected difficulty benar.
  * PoW valid.
  * timestamp valid.
  * coinbase valid.
  * tx valid.
  * tx tidak double-spend.
  * tx tidak spend immature coinbase.
  * address reward valid.
* Jika valid:

  * simpan block ke canonical chain.
  * hapus tx yang confirmed dari mempool.
  * update runtime state.
  * broadcast block ke peers.
  * return accepted true.
* Jika stale karena tip sudah berubah:

  * return accepted false
  * reason: stale template
  * current height/tip
* Jika invalid PoW:

  * return accepted false
  * reason: invalid proof of work
* Jika invalid difficulty:

  * return accepted false
  * reason: invalid difficulty expected X got Y

Response:

{
"accepted": true,
"height": 12,
"hash": "0000...",
"difficulty": 4,
"tx_count": 2,
"broadcast_success": 0,
"broadcast_failed": 0
}

Jika stale:

{
"accepted": false,
"reason": "stale template",
"current_height": 13,
"current_tip": "..."
}

==================================================
3. Template staleness rules
===========================

Miner harus sadar bahwa template bisa stale.

Template dianggap stale jika:

* current tip hash berbeda dari template previous hash.
* current height sudah berubah.
* expected next difficulty berubah.
* mempool tx selected ada yang tidak valid lagi.

Node submit endpoint harus menolak stale template secara aman.

Miner behavior:

* Jika submit response stale:

  * discard current job.
  * fetch template baru.
  * lanjut mining.
* Miner juga boleh polling template baru secara berkala.

==================================================
4. Tambahkan standalone miner CLI
=================================

Tambahkan binary baru atau subcommand baru.

Pilihan disarankan:

* Tambahkan binary baru:
  node/cmd/indominer/
  sehingga bisa build:
  go build -o indominer ./node/cmd/indominer

Tetap boleh juga tambahkan:
indochain miner start

Tapi minimal wajib ada command standalone yang bisa dijalankan tanpa membuka datadir chain lokal.

Command target:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8391 --address <IND_ADDR> --threads 4

Flags:

* --rpc-url string, required.
* --address string, required.
* --threads int, default runtime.NumCPU().
* --poll-interval duration, default 5s.
* --log-interval duration, default 10s.
* --once, mine one block then exit.
* --max-blocks int, optional, 0 means unlimited.
* --retry-interval duration, default 3s.
* --user-agent string, default indominer/<version>.
* --network optional, default inferred from node/template if possible.

Output example:

IndoChain CPU Miner
rpc: http://127.0.0.1:8391
reward address: iND...
threads: 4

new job height=12 difficulty=4 txs=1 prev=...
hashrate: 245.30 KH/s total_hashes=2453000 elapsed=10s
block found height=12 hash=0000... nonce=123456 thread=2
submit accepted height=12 hash=0000...
new job height=13 difficulty=4 txs=0 prev=...

==================================================
5. CPU multi-thread mining
==========================

Implement CPU mining dengan beberapa goroutine.

Rules:

* Setiap thread/goroutine mencari nonce range berbeda.
* Jangan semua thread mulai dari nonce yang sama.
* Gunakan atomic counters untuk total hashes.
* Gunakan context cancellation saat:

  * block found,
  * template stale,
  * process interrupted,
  * max blocks reached,
  * --once done.
* Jangan membuat goroutine leak.
* Jangan terlalu banyak log per hash.
* Jangan mengunci mutex berat di hot loop.
* Hashrate dihitung dari total hashes / elapsed.

Nonce strategy:

* Thread i mulai dari nonce offset i.
* Step = thread count.
* Contoh:
  thread 0: 0, 4, 8, ...
  thread 1: 1, 5, 9, ...
  thread 2: 2, 6, 10, ...
  thread 3: 3, 7, 11, ...
* Jika nonce overflow, fetch template baru.

Timestamp:

* Template timestamp boleh diperbarui oleh miner jika existing block hash memasukkan timestamp.
* Jika miner update timestamp, node submit harus tetap validasi timestamp.
* Jika lebih mudah, miner memakai template timestamp saja untuk Phase 2.9.
* Tambahkan TODO untuk rolling timestamp jika perlu.

==================================================
6. Miner tidak boleh membutuhkan private key
============================================

Standalone miner hanya membutuhkan:

* rpc-url node,
* reward address.

Miner tidak boleh:

* membaca wallet file,
* meminta private key,
* signing tx,
* mengakses datadir node,
* menyimpan wallet,
* membuat address.

Ini penting agar mining bisa dijalankan di mesin lain tanpa membawa private key.

==================================================
7. Integrasi dengan existing built-in mine command
==================================================

Command lama:

indochain mine --address <ADDR> --blocks N

harus tetap bekerja.

Boleh refactor mining logic agar:

* built-in mine menggunakan service internal yang sama,
* standalone miner menggunakan template/submit RPC,
* tetapi jangan rewrite besar kalau berisiko.

Yang penting:

* Tidak ada regression pada command lama.
* Mining built-in tetap valid dengan difficulty dan maturity.
* Standalone miner valid dengan template/submit.

==================================================
8. Submit accepted block harus broadcast ke peers
=================================================

Jika standalone miner menemukan block dan submit diterima:

* node harus broadcast block ke peers sama seperti built-in mine.
* Output response harus mencantumkan broadcast success/failed.
* Peer yang menerima block harus validasi normal.

Jangan broadcast dari miner langsung.
Miner hanya submit ke node.
Node yang broadcast.

==================================================
9. Mempool integration
======================

Template harus mengambil pending tx dari mempool.

Rules:

* Jika mempool kosong:
  block txs = coinbase only.
* Jika mempool ada tx valid:
  block berisi coinbase + selected tx.
* Jika tx invalid karena maturity/spendable berubah:
  jangan masukkan ke template.
* Setelah block accepted:
  confirmed tx dihapus dari mempool.
* Jika submit stale:
  mempool tidak berubah.
* Jika tx di template sudah tidak valid saat submit:
  block ditolak atau tx difilter sebelum template.
  Lebih aman: block ditolak dengan reason tx invalid.

==================================================
10. Coinbase maturity tetap berlaku
===================================

Standalone miner reward:

* masuk confirmed balance,
* belum langsung mature,
* maturity sesuai network profile.

Manual test:

* Standalone miner mine 3 blocks.
* balance miner:
  confirmed 150
  mature 0
  immature 150
  spendable 0
* Setelah height 11:
  mature 50.

Jangan bypass maturity pada submit block.

==================================================
11. Dynamic difficulty tetap berlaku
====================================

Template harus memakai next difficulty dari current chain.

Manual test:

* Pada height 0 template difficulty 4.
* Setelah fast blocks melewati retarget boundary, template difficulty bisa naik ke 5.
* Miner harus mine sesuai difficulty template.
* Node submit harus reject jika difficulty salah.

==================================================
12. RPC safety
==============

Tambahkan body size limit untuk submit block jika belum ada.

* /miner/template body kecil.
* /miner/submit body max block size, contoh 8 MB.

Error harus jelas:

* invalid json
* invalid address
* invalid proof of work
* invalid difficulty
* stale template
* block too large
* tx invalid
* internal error

Jangan panic.

==================================================
13. Miner reconnect/retry
=========================

Standalone miner harus tahan jika node belum hidup atau restart.

Behavior:

* Jika template fetch gagal:
  log warning.
  sleep retry interval.
  coba lagi.
* Jika submit gagal karena network:
  log warning.
  fetch template baru.
* Jika RPC returns stale:
  fetch template baru.
* Jika RPC returns invalid:
  log error.
  fetch template baru atau exit jika invalid config.
* Ctrl+C harus stop bersih.

Output contoh:

template fetch failed: connection refused; retrying in 3s
stale template; refreshing job
interrupted, shutting down...

==================================================
14. Miner status endpoint optional
==================================

Tidak wajib.

Jika mudah, tambahkan local miner-only status log saja.
Jangan buat server miner.

Phase ini cukup CLI miner.

==================================================
15. Tests wajib
===============

Tambahkan/update tests:

1. Template endpoint at genesis:

* GET /miner/template address iND valid.
* returns height 1.
* previous hash genesis.
* difficulty initial 4.
* tx_count 1 coinbase only.

2. Template invalid address:

* returns error invalid address.

3. Submit valid block:

* fetch template.
* mine nonce in test helper.
* submit block.
* accepted true.
* chain height increases.
* balance confirmed increases.
* immature balance increases.
* chain valid.

4. Submit invalid PoW:

* tamper nonce/hash.
* accepted false.
* height unchanged.

5. Submit stale template:

* fetch template A at height N.
* mine/commit another block first.
* submit template A block.
* rejected stale.
* height unchanged.

6. Submit wrong difficulty:

* block difficulty not expected.
* rejected.

7. Template includes mempool tx:

* create matured spendable funds.
* send tx pending.
* fetch template.
* tx_count includes coinbase + pending tx.

8. Submit block with mempool tx:

* submit accepted.
* mempool count becomes 0.
* receiver balance updated.
* normal tx count increases.

9. Coinbase maturity with standalone submit:

* after submit mined block, reward immature.
* spendable remains 0 until maturity.

10. Dynamic difficulty template:

* after fast retarget scenario, template difficulty reflects next difficulty.

11. Miner internal nonce partition:

* with threads >1, workers search different nonce ranges.
* no duplicate first nonce among workers.

12. Miner stops on block found:

* workers cancel after one worker finds valid nonce.
* no goroutine leak if testable.

13. Miner --once:

* mines one block and exits cleanly.

14. Miner retry:

* RPC unavailable returns retry behavior if unit-testable.
* Otherwise test client function returns error gracefully.

15. Existing built-in mine:

* still works.
* chain validate pass.
* maturity balance still correct.

16. Existing tests still pass:

* address tests.
* difficulty tests.
* maturity tests.
* reorg tests.
* mempool tests.
* P2P tests.

==================================================
16. README / docs update
========================

Update README.md and/or docs:

Current phase:
Phase 2.9 — Standalone CPU Miner CLI

Tambahkan dokumentasi:

1. Start node:

go run ./node/cmd/indochain --datadir ./testdata/miner node start --rpc :8401 --p2p :9401 --advertise-p2p http://127.0.0.1:9401

2. Create wallet:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 wallet new

3. Start standalone miner:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4

4. Mine one block:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4 --once

5. Build binary:

go build -o indominer ./node/cmd/indominer

6. Notes:

* Miner does not need private key.
* Miner only needs reward address.
* Node validates all submitted blocks.
* Coinbase rewards are immature until coinbase maturity.
* Public RPC nodes should be careful with mining submit limits.
* GPU miner and mining pool are future phases.

==================================================
17. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/indochain --datadir ./testdata/miner dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/miner init
go run ./node/cmd/indochain --datadir ./testdata/miner wallet new

Start node:

go run ./node/cmd/indochain --datadir ./testdata/miner node start --rpc :8401 --p2p :9401 --advertise-p2p http://127.0.0.1:9401

Check template:

Invoke-RestMethod "http://127.0.0.1:8401/miner/template?address=<IND_ADDR>"

Expected:
height: 1
difficulty: 4
previous_hash: genesis hash
tx_count: 1

Run standalone miner once:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4 --once

Expected:
new job height=1 difficulty=4
block found height=1 hash=0000...
submit accepted height=1

Check chain:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 balance <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 chain validate

Expected:
height: 1
total supply: 50 dIDR
confirmed balance: 50
mature balance: 0
immature balance: 50
spendable balance: 0
chain valid

Mine 3 blocks:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4 --max-blocks 3

Expected:
accepted blocks 3
height increases by 3
hashrate logs appear

Mempool tx test:

* mine enough blocks until maturity.
* create receiver wallet.
* send 10 dIDR.
* run standalone miner --once.
* expected tx confirmed and mempool count 0.

Jangan over-engineer.
Fokus Phase 2.9 hanya:

* miner template RPC,
* miner submit RPC,
* standalone CPU miner CLI,
* multi-thread nonce search,
* hashrate logs,
* stale template handling,
* reconnect/retry,
* node-side validation,
* maturity/difficulty compatibility.
