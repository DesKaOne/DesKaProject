Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur:

* node/

  * cmd/deskachain/
  * internal/
  * go.mod
  * go.sum
* root punya go.work

Status saat ini:

* Phase 1 sampai Phase 2.5 sudah selesai.
* go work sync berhasil.
* go test ./node/... pass.
* Node, RPC, P2P, mining, wallet, mempool, broadcast, fork detection, common ancestor, fork rejection, dan safe reorg experimental sudah berjalan.
* Phase 2.5 test sudah valid:

  * Equal work fork 3 vs 3 ditolak:
    allowed: false
    reason: peer chain does not have more cumulative work
  * Peer more work fork 3 vs 5 diterima:
    allowed: true
    reason: peer chain has more cumulative work and depth is within limit
  * reorg apply --yes berhasil:
    old height: 3
    new height: 5
    chain valid: true
  * node compare setelah reorg:
    nodes in sync
  * chain info setelah reorg:
    height: 5
    total supply: 250 IDR
    total transactions: 5
    coinbase transactions: 5
    normal transactions: 0
  * reorg apply tanpa --yes ditolak.

Nama patch:
DesKaChain Phase 2.6 — Reorg Mempool Recovery & Transaction Conflict Tests

Tujuan utama:
Memastikan reorg aman ketika block yang dilepas dan branch peer berisi transaksi normal, bukan hanya coinbase.

Phase 2.5 sudah membuktikan reorg block kosong/coinbase berhasil.
Phase 2.6 harus membuktikan:

* transaksi normal dari disconnected/orphaned blocks bisa kembali ke mempool jika masih valid,
* transaksi yang sudah confirmed di branch baru tidak masuk mempool lagi,
* transaksi yang konflik/double-spend/drop invalid tidak masuk mempool,
* saldo ledger setelah reorg tetap konsisten,
* total supply tetap benar,
* mempool tidak duplicate,
* chain validate tetap pass.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public peer discovery.
* Jangan ubah default peer sync menjadi auto reorg.
* Reorg tetap harus eksplisit dengan --yes.
* Patch incremental.
* Semua tetap harus lolos:

  go work sync
  go test ./node/...

==================================================

1. Tambahkan helper reorg test scenario dengan transaksi
   ==================================================

Tambahkan command development-only:

dev reorg-tx-sim

Flags:
--datadir-a <path>
--datadir-b <path>
--scenario <name>

Skenario minimal:

1. requeue-valid
2. confirmed-on-peer
3. conflict-double-spend
4. invalid-after-reorg

Jika terlalu banyak untuk satu command, boleh implementasi sebagian sebagai integration test internal dulu, tetapi command CLI sangat membantu untuk manual testing.

==================================================
2. Scenario: requeue-valid
==========================

Tujuan:
Transaksi normal di branch lokal yang terlepas harus kembali ke mempool jika belum ada di branch peer dan masih valid setelah reorg.

Setup:

* Genesis sama.
* Buat wallet:

  * minerA
  * minerB
  * user1
  * user2
* Chain A:

  1. Mine 3 block ke minerA agar minerA punya 150 IDR.
  2. Buat tx normal dari minerA ke user1 sebesar 10 IDR.
  3. Mine tx tersebut ke block A4.
  4. Height A = 4.
* Chain B:

  1. Dari genesis yang sama, mine 6 block ke minerB.
  2. Tidak berisi tx minerA->user1.
  3. Height B = 6.
  4. Work B lebih besar dari A.

Reorg A ke B:

* Common ancestor height 0.
* Disconnect A1-A4.
* Connect B1-B6.
* Tx minerA->user1 dari orphaned block A4 harus dicek.
* Karena setelah reorg minerA tidak punya reward dari branch A, tx minerA->user1 kemungkinan menjadi invalid.
* Maka untuk scenario requeue-valid, funding harus dibuat agar tx tetap valid setelah reorg.

Agar tx tetap valid:

* Gunakan funding yang ada di common ancestor sebelum fork.
* Buat shared pre-fork chain dulu:

  * common chain height 3, minerCommon punya 150 IDR.
  * fork dari height 3.
  * Chain A membuat tx minerCommon -> user1 sebesar 10 IDR di A4.
  * Chain B mine block B4-B6 tanpa tx tersebut.
* Setelah reorg ke B, minerCommon masih punya 150 IDR dari common ancestor, jadi tx masih valid.
* Expected:
  requeued transactions: 1
  mempool count: 1
  tx minerCommon->user1 ada di mempool
  user1 balance setelah reorg: 0 IDR
  minerCommon balance belum berkurang di ledger canonical
  chain valid

==================================================
3. Scenario: confirmed-on-peer
==============================

Tujuan:
Jika transaksi dari disconnected branch juga ada di connected peer branch, jangan dimasukkan kembali ke mempool.

Setup:

* Common chain height 3, minerCommon punya saldo.
* Fork A:

  * tx1 minerCommon -> user1 sebesar 10 IDR masuk block A4.
* Fork B:

  * tx1 yang sama juga masuk block B4.
  * B lanjut mine sampai height 6.
* Reorg A ke B.

Expected:

* requeued transactions: 0
* dropped/confirmed transactions: 1
* mempool count: 0
* user1 balance setelah reorg: 10 IDR
* tx1 confirmed di canonical chain
* chain valid

==================================================
4. Scenario: conflict-double-spend
==================================

Tujuan:
Jika orphaned tx konflik dengan branch baru, tx harus drop, bukan requeue.

Setup:

* Common chain height 3, minerCommon punya 150 IDR.
* Fork A:

  * txA: minerCommon -> user1 sebesar 100 IDR masuk block A4.
* Fork B:

  * txB: minerCommon -> user2 sebesar 120 IDR masuk block B4.
  * B lanjut sampai height 6.
* Jika setelah branch B, saldo minerCommon tidak cukup untuk txA lagi atau nonce/sequence konflik, txA harus invalid.
* Reorg A ke B.

Expected:

* txA tidak masuk mempool.
* dropped transactions: 1
* mempool count: 0
* user1 balance setelah reorg: 0 IDR
* user2 balance setelah reorg: 120 IDR
* chain valid

Catatan:
Jika DesKaChain belum punya nonce, konflik minimal bisa berdasarkan insufficient balance setelah branch B.
Jika nonce sudah ada, gunakan nonce conflict juga.

==================================================
5. Scenario: mempool removes confirmed
======================================

Tujuan:
Jika sebelum reorg mempool lokal berisi tx yang kemudian confirmed di branch peer, setelah reorg tx tersebut harus hilang dari mempool.

Setup:

* Common chain height 3.
* Local A mempool berisi tx1.
* Peer B branch memasukkan tx1 di block B4.
* B lebih kuat.
* Reorg A ke B.

Expected:

* mempool count: 0
* tx1 tidak pending lagi.
* user balance sesuai tx1.
* chain valid.

==================================================
6. Tambahkan mempool revalidation helper
========================================

Tambahkan helper:

RevalidateMempoolAgainstLedger()

Behavior:

* Ambil semua tx pending.
* Buang tx yang sudah confirmed di canonical chain.
* Buang tx invalid karena balance kurang / signature invalid / duplicate / conflict.
* Pertahankan tx yang masih valid.
* Return summary:
  kept
  dropped_invalid
  dropped_confirmed
  dropped_duplicate

Pastikan helper dipanggil setelah reorg apply.

==================================================
7. Tambahkan orphan tx extraction
=================================

Saat reorg disconnect blocks:

* Ambil semua transaksi normal dari disconnected blocks.
* Jangan ambil coinbase.
* Jangan duplicate tx.
* Setelah branch peer connect dan ledger replay selesai:

  * Untuk setiap orphan tx:

    * jika sudah confirmed di canonical branch baru, count sebagai confirmed/dropped_confirmed.
    * jika masih valid, masukkan ke mempool.
    * jika invalid, count dropped_invalid.
* Summary:
  requeued transactions: N
  dropped transactions: N
  dropped confirmed transactions: N
  dropped invalid transactions: N

Output reorg apply sebaiknya menjadi:

reorg applied
old height: 4
new height: 6
disconnected blocks: 4
connected blocks: 6
requeued transactions: 1
dropped transactions: 0
dropped confirmed transactions: 0
dropped invalid transactions: 0
chain valid: true

==================================================
8. Tambahkan command mempool inspect detail
===========================================

Tambahkan atau perbaiki command:

mempool list --detail

Output:
pending tx count: 1
txid: ...
from: ...
to: ...
amount: 10 IDR
fee: ...
status: pending

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 mempool list --detail

Jika belum ada txid display, tambahkan.

==================================================
9. Balance checks
=================

Pastikan command balance bisa dipakai setelah reorg.

Contoh:

balance <address>

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 balance <address>

Pastikan setelah reorg:

* Saldo sesuai canonical branch baru.
* Saldo dari orphaned branch tidak ikut terbawa.
* Total supply sesuai jumlah canonical coinbase blocks.

==================================================
10. RPC endpoints update
========================

Update /reorg/apply response:

{
"applied": true,
"old_height": 4,
"new_height": 6,
"requeued_transactions": 1,
"dropped_transactions": 0,
"dropped_confirmed_transactions": 0,
"dropped_invalid_transactions": 0,
"mempool_count": 1,
"chain_valid": true
}

Update /mempool/list agar bisa detail:
GET /mempool/list?detail=true

Response tx harus include:

* txid
* from
* to
* amount
* fee jika ada
* timestamp jika ada

==================================================
11. Tests wajib
===============

Tambahkan/update tests:

1. Reorg requeues valid orphan tx:

* Common ancestor punya funding.
* Tx di branch lokal terlepas.
* Peer branch tidak punya tx.
* Setelah reorg, tx kembali ke mempool.
* Mempool count 1.
* Chain valid.

2. Reorg does not requeue tx confirmed on peer:

* Orphan tx juga ada di peer branch.
* Setelah reorg, mempool count 0.
* Balance recipient sesuai.
* Chain valid.

3. Reorg drops invalid orphan tx due insufficient balance:

* Orphan tx tidak valid setelah peer branch.
* Setelah reorg, mempool count 0.
* dropped_invalid_transactions 1.
* Chain valid.

4. Reorg removes mempool tx confirmed by new branch:

* Tx pending lokal ada di peer branch.
* Setelah reorg, tx hilang dari mempool.
* Chain valid.

5. No duplicate mempool tx:

* Orphan extraction menghasilkan tx yang sudah ada di mempool.
* Setelah reorg, hanya satu instance tx di mempool.

6. Coinbase never requeued:

* Disconnected coinbase tx tidak masuk mempool.
* requeued coinbase count 0.

7. Total supply after reorg:

* Canonical height 6 dengan reward 50 IDR.
* total supply 300 IDR.
* Orphaned coinbase tidak dihitung.

8. Balance after reorg:

* Recipient dari orphaned-only tx tidak menerima saldo.
* Recipient dari peer branch tx menerima saldo.
* Sender balance sesuai canonical ledger.

9. Reorg apply output summary:

* Summary count sesuai scenario.
* RPC response fields lengkap.

10. Chain validate after every reorg scenario:

* chain validate pass.

==================================================
12. README update
=================

Update README:

Current phase:
Phase 2.6 — Reorg Mempool Recovery & Transaction Conflict Tests

Tambahkan penjelasan:

* Saat reorg, block lama menjadi orphaned.
* Transaksi normal dari orphaned block bisa kembali ke mempool jika masih valid.
* Coinbase dari orphaned block tidak boleh kembali ke mempool.
* Transaksi yang sudah confirmed di branch baru tidak boleh pending lagi.
* Transaksi konflik atau invalid harus drop.
* Ledger mengikuti canonical branch baru.

Tambahkan contoh command:

go run ./node/cmd/deskachain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario requeue-valid

go run ./node/cmd/deskachain --datadir ./testdata/reorgTxA node start --rpc :8351 --p2p :9351 --advertise-p2p http://127.0.0.1:9351

go run ./node/cmd/deskachain --datadir ./testdata/reorgTxB node start --rpc :8352 --p2p :9352 --advertise-p2p http://127.0.0.1:9352

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 reorg preview --peer http://127.0.0.1:9352

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 reorg apply --peer http://127.0.0.1:9352 --yes

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 mempool list --detail

Expected untuk requeue-valid:
requeued transactions: 1
pending tx count: 1
chain valid: true

==================================================
13. Expected final commands
===========================

Setelah patch:

go work sync
go test ./node/...

Scenario requeue-valid:

go run ./node/cmd/deskachain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario requeue-valid

go run ./node/cmd/deskachain --datadir ./testdata/reorgTxA node start --rpc :8351 --p2p :9351 --advertise-p2p http://127.0.0.1:9351

go run ./node/cmd/deskachain --datadir ./testdata/reorgTxB node start --rpc :8352 --p2p :9352 --advertise-p2p http://127.0.0.1:9352

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 reorg preview --peer http://127.0.0.1:9352

Expected:
allowed: true

Apply:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 reorg apply --peer http://127.0.0.1:9352 --yes

Expected:
reorg applied
requeued transactions: 1
chain valid: true

Check mempool:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8351 mempool list --detail

Expected:
pending tx count: 1

Scenario confirmed-on-peer:

go run ./node/cmd/deskachain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario confirmed-on-peer

Expected after apply:
requeued transactions: 0
dropped confirmed transactions: 1
pending tx count: 0
chain valid: true

Scenario conflict-double-spend:

go run ./node/cmd/deskachain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario conflict-double-spend

Expected after apply:
requeued transactions: 0
dropped invalid transactions: 1
pending tx count: 0
chain valid: true

Jangan over-engineer.
Fokus Phase 2.6 hanya pada:

* orphan tx extraction,
* mempool requeue,
* confirmed tx cleanup,
* invalid/conflict tx drop,
* balance consistency,
* total supply consistency,
* tests untuk reorg dengan transaksi normal.
