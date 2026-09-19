Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur project:

* node/

  * cmd/deskachain/
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

* Phase 1 sampai Phase 2.6 sudah selesai.
* go work sync berhasil.
* go test ./node/... pass.
* Fitur yang sudah ada:

  * local blockchain,
  * wallet CLI,
  * send transaction,
  * mempool,
  * mining,
  * RPC node,
  * P2P node,
  * block broadcast,
  * tx broadcast,
  * fork detection,
  * common ancestor,
  * safe reorg preview/apply,
  * reorg mempool recovery.
* Phase 2.6 sudah valid:

  * orphan tx valid bisa requeue ke mempool,
  * tx confirmed di peer branch tidak masuk mempool lagi,
  * tx conflict/double-spend drop invalid,
  * chain validate pass,
  * total supply canonical benar.

Nama patch:
DesKaChain Phase 2.6.5 — Runtime Stats & Safety Cleanup

Tujuan:
Membersihkan bug kecil dan memperkuat safety runtime sebelum masuk ke:

* Protocol Spec Freeze,
* IDR Base58 Address Migration,
* Difficulty Adjustment,
* Coinbase Maturity,
* Standalone Miner.

Aturan penting:

* Jangan implement address Base58.
* Jangan ubah format address.
* Jangan ganti key curve.
* Jangan implement difficulty adjustment.
* Jangan implement coinbase maturity.
* Jangan implement standalone miner.
* Jangan ubah consensus behavior besar.
* Jangan rewrite besar.
* Patch incremental dan aman.
* Semua command lama harus tetap bekerja.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...

==================================================

1. Fix chain info runtime transaction stats
   ==================================================

Masalah:
Saat node running dan chain info dipanggil lewat RPC/runtime state, statistik transaksi bisa salah karena memakai shortcut berdasarkan height.

Contoh bug yang harus diperbaiki:

* coinbase blocks dihitung dari height, masih oke untuk chain coinbase-only.
* total transactions bisa dianggap sama dengan height.
* normal transactions bisa tetap 0 walaupun chain punya transaksi normal.
* Setelah reorg dengan transaksi normal, chain info runtime bisa tidak mencerminkan isi block canonical yang sebenarnya.

Target:
`chain info` harus selalu menampilkan statistik berdasarkan canonical blocks asli.

Field yang harus benar:

* height
* tip hash
* total supply
* cumulative work
* pending tx count
* blocks
* coinbase blocks
* total transactions
* coinbase transactions
* normal transactions
* circulating supply
* datadir

Definisi:

* total transactions = semua transaksi dalam canonical blocks, termasuk coinbase.
* coinbase transactions = jumlah transaksi coinbase dalam canonical blocks.
* normal transactions = transaksi non-coinbase dalam canonical blocks.
* coinbase blocks = jumlah block canonical non-genesis yang punya coinbase tx.
* genesis block tidak otomatis dianggap coinbase jika txs kosong.

Implementasi:

* Buat helper reusable, misalnya:
  chain.CalculateChainStats(blocks)
  chain.BuildChainStats(store)
  rpc.BuildChainInfo(...)
* Jangan ada dua logic stats yang berbeda antara CLI local dan RPC remote.
* Local `chain info` dan remote `chain info` harus memakai helper yang sama atau hasil yang konsisten.
* Runtime state boleh tetap dipakai untuk height/tip cache, tapi transaction stats harus dihitung dari canonical block data atau cache stats yang benar-benar diupdate saat block import/mine/reorg.

==================================================
2. Update chain info after mining, send, and reorg
==================================================

Pastikan stats benar pada kondisi:

A. Chain kosong setelah genesis:

* height: 0
* blocks: 1
* total transactions: 0
* coinbase transactions: 0
* normal transactions: 0

B. Setelah mine 3 block coinbase-only:

* height: 3
* blocks: 4
* coinbase blocks: 3
* total transactions: 3
* coinbase transactions: 3
* normal transactions: 0

C. Setelah mine block yang berisi 1 normal tx + 1 coinbase:

* total transactions bertambah 2
* coinbase transactions bertambah 1
* normal transactions bertambah 1

D. Setelah reorg:

* stats mengikuti canonical branch baru.
* orphaned transactions tidak dihitung sebagai confirmed transactions.
* transaksi yang requeue ke mempool tidak dihitung dalam chain transactions.

==================================================
3. Improve .gitignore
=====================

Update root `.gitignore`.

Pastikan runtime/dev data tidak ikut commit:

data/
testdata/
tmp/
dist/
build/
*.zip
*.log

**/chain.db
**/*.db
**/wallets.json
**/mempool.json
**/peers.json
**/node.lock
**/node_id

Jangan ignore:

* docs/
* prompts/
* README.md
* Roadmap.md
* go.work
* node/go.mod
* node/go.sum
* source code

Jika ada `.gitignore` di node/, sinkronkan juga bila perlu.

==================================================
4. Mempool atomic write
=======================

Masalah:
Mempool saat ini disimpan ke file JSON, misalnya `mempool.json`.
Jika proses crash saat write, file bisa corrupt.
Jika dua operasi menulis bersamaan, data bisa overwrite.

Target:
Implement atomic write untuk mempool save.

Behavior:

* Saat save:

  1. marshal JSON ke bytes.
  2. tulis ke file temp, contoh:
     mempool.json.tmp
  3. flush/sync jika memungkinkan.
  4. rename temp ke mempool.json.
* Jika write temp gagal:

  * jangan rusak file lama.
* Jika rename gagal:

  * return error.
* Jika load menemukan file missing:

  * return empty mempool, bukan error fatal.
* Jika load menemukan JSON corrupt:

  * return error jelas:
    failed to load mempool: invalid json
  * jangan panic.

Tambahkan helper reusable:
atomicWriteFile(path string, data []byte, perm os.FileMode) error

Atau simpan di internal/storage/fileutil jika cocok.

==================================================
5. Mempool duplicate tx guard
=============================

Tambahkan guard agar txid duplicate tidak masuk mempool dua kali.

Behavior:

* Add(tx) jika txid belum ada:
  add success.
* Add(tx) jika txid sudah ada:
  tidak duplicate.
  return status already_exists atau error typed.
* Broadcast tx duplicate dari peer tidak membuat mempool count bertambah.
* Reorg requeue tx yang sudah ada di mempool tidak membuat duplicate.
* Mempool list tidak menampilkan txid sama lebih dari satu kali.

Pastikan duplicate check berdasarkan txid, bukan pointer/object equality.

==================================================
6. Basic mempool runtime mutex
==============================

Tambahkan perlindungan concurrency minimal untuk runtime mempool.

Masalah yang ingin dicegah:

* RPC send dan P2P tx masuk bersamaan.
* Mining sedang memilih tx dan remove confirmed tx.
* Reorg sedang revalidate mempool.
* Save/load mempool race.

Target:

* Semua operasi mutasi mempool harus lewat lock.
* Operasi baca list/detail juga aman.
* Jangan deadlock.
* Jangan lock chain runtime terlalu lama.
* Jangan melakukan network call saat memegang mempool lock.

Implementasi bisa di:

* mempool.Store
* node runtime
* atau service wrapper

Yang penting:

* Add
* Remove
* RemoveIDs
* Revalidate
* Load/Save mutating path
* List snapshot

harus aman secara concurrency.

==================================================
7. Mempool detail and tx count consistency
==========================================

Pastikan command:

mempool list
mempool list --detail

Local dan remote tetap bekerja.

Output minimal:

* pending tx count: N

Jika detail:

* txid
* from
* to
* amount
* fee
* nonce jika ada
* timestamp jika ada
* status pending

Pastikan:

* pending tx count sama dengan jumlah tx unique.
* Setelah duplicate Add, count tidak bertambah.
* Setelah mining tx, tx hilang dari mempool.
* Setelah reorg requeue valid tx, count benar.
* Setelah reorg drop invalid/confirmed tx, count benar.

==================================================
8. HTTP server timeout initial support
======================================

Audit RPC server dan P2P server.

Jika saat ini masih memakai:

http.ListenAndServe(addr, mux)

Ganti ke:

http.Server{
Addr: addr,
Handler: mux,
ReadHeaderTimeout: 5 * time.Second,
ReadTimeout: 15 * time.Second,
WriteTimeout: 30 * time.Second,
IdleTimeout: 60 * time.Second,
}

Aturan:

* Jangan ubah route.
* Jangan ubah response format.
* Jangan membuat timeout terlalu pendek untuk mining RPC yang memang bisa jalan lama.
* Jika `/mine` long-running via RPC bisa terdampak WriteTimeout, jangan paksa timeout global merusak mining.
* Jika perlu, beri catatan TODO untuk route long-running mining agar nanti menjadi job-based/polling penuh.
* P2P server wajib punya ReadHeaderTimeout untuk mengurangi slowloris risk.

Jika perubahan ini terlalu berisiko untuk `/mine`, minimal:

* Terapkan ke P2P server dulu.
* Tambahkan TODO jelas di RPC server.
* Tapi usahakan tetap aman.

==================================================
9. Request body size limit
==========================

Tambahkan limit body minimal pada endpoint yang menerima JSON.

Target awal:

* tx submit endpoint
* p2p tx receive
* p2p block receive
* common ancestor request
* reorg apply/preview request jika ada body
* peer connect/add jika body

Batas default localnet:

* max tx body: 1 MB
* max block body: 8 MB
* max generic JSON body: 1 MB

Implementasi:

* gunakan http.MaxBytesReader pada handler HTTP.
* Jika body terlalu besar:
  return HTTP 413
  response JSON error jelas jika pattern project sudah JSON.

Jangan over-engineer config dulu.
Boleh hardcode constants dengan TODO pindahkan ke network profile/config nanti.

==================================================
10. Error response consistency
==============================

Pastikan endpoint RPC/P2P tidak panic saat:

* JSON body invalid.
* body terlalu besar.
* required field kosong.
* peer URL invalid.
* tx duplicate.
* mempool file corrupt.

Response harus jelas.

Contoh:
{
"ok": false,
"error": "invalid json"
}

Atau sesuai format existing project.

Jangan ubah semua response besar-besaran.
Cukup rapikan endpoint yang disentuh.

==================================================
11. Runtime safety TODO notes
=============================

Tambahkan TODO atau docs kecil untuk public hardening future:

* stronger RPC rate limiting,
* peer ban threshold,
* per-IP limits,
* max mempool size,
* tx fee policy,
* nonce replacement policy,
* config file support,
* graceful shutdown consistency,
* atomic reorg apply transaction.

Bisa di:

* docs/RuntimeSafety.md
  atau README section.

Jangan implement semua sekarang.
Phase ini hanya cleanup kecil.

==================================================
12. Tests wajib
===============

Tambahkan/update tests:

1. Chain info genesis stats:

* height 0
* blocks 1
* total transactions 0
* coinbase transactions 0
* normal transactions 0

2. Chain info coinbase-only stats:

* mine 3 blocks
* total transactions 3
* coinbase transactions 3
* normal transactions 0

3. Chain info normal tx stats:

* mine funding blocks
* create normal tx
* mine tx
* total transactions includes coinbase + normal tx
* normal transactions >= 1

4. Chain info after reorg:

* use existing reorg tx scenario or helper
* after reorg canonical branch stats correct
* orphaned-only tx not counted as confirmed
* requeued mempool tx not counted as chain tx

5. RPC chain info runtime stats:

* start runtime/node handler if test infra exists
* call /chain/info
* ensure stats match canonical block scan

6. Atomic mempool save:

* save mempool
* file exists
* JSON valid
* temp file not left behind on success

7. Atomic mempool save preserves old file on failure:

* if easy to simulate with bad path/permission
* otherwise test helper separately

8. Mempool duplicate guard:

* add same tx twice
* count remains 1
* second add returns duplicate/already exists

9. Reorg requeue duplicate guard:

* tx already in mempool
* orphan extraction tries requeue same tx
* mempool count remains 1

10. Mempool concurrent add:

* add same tx from multiple goroutines
* count remains 1
* no race if run with:
  go test -race ./node/...
  if supported locally

11. Mempool list detail:

* detail output includes txid/from/to/amount/status
* pending count correct

12. Invalid JSON endpoint:

* handler returns error, not panic.

13. Body too large:

* handler returns 413 or clear error.

14. P2P/RPC server timeout config:

* if testable, assert server has ReadHeaderTimeout > 0.
* otherwise keep code simple and covered by compile.

==================================================
13. README / Roadmap update
===========================

Update README.md and/or Roadmap.md:

Current phase:
Phase 2.6.5 — Runtime Stats & Safety Cleanup

Add short notes:

* chain info stats now scan canonical blocks correctly,
* mempool writes are atomic,
* duplicate mempool txs are rejected,
* basic mempool runtime locking added,
* HTTP server timeout/body limit groundwork added.

Do not rewrite whole README.
Patch only relevant section.

==================================================
14. Expected final commands
===========================

After patch:

go work sync
go test ./node/...

Optional race test:

go test -race ./node/...

Manual quick test:

go run ./node/cmd/deskachain --datadir ./testdata/safety dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/safety init
go run ./node/cmd/deskachain --datadir ./testdata/safety wallet new

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/safety node start --rpc :8371 --p2p :9371 --advertise-p2p http://127.0.0.1:9371

Mine:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8371 mine --address <addr> --blocks 3

Check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8371 chain info

Expected:
height: 3
blocks: 4
coinbase blocks: 3
total transactions: 3
coinbase transactions: 3
normal transactions: 0
total supply: 150 IDR

Mempool duplicate/manual if possible:
send same tx twice or submit same tx twice.
Expected:
pending tx count does not duplicate.

Reorg tx scenarios from Phase 2.6 must still pass:
requeue-valid
confirmed-on-peer
conflict-double-spend

Jangan over-engineer.
Fokus Phase 2.6.5 hanya:

* runtime chain info stats correctness,
* .gitignore hygiene,
* mempool atomic write,
* mempool duplicate guard,
* basic mempool lock,
* safe HTTP timeout/body limit groundwork,
* tests.
