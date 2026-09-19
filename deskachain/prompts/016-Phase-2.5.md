Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur:

* node/

  * cmd/deskachain/
  * internal/
  * go.mod
  * go.sum
* root punya go.work

Status saat ini:

* Phase 1 sampai Phase 2.4.3 sudah selesai.
* go work sync berhasil.
* go test ./node/... pass.
* Localnet node, RPC, P2P, mining, mempool, wallet CLI sudah berjalan.
* Broadcast tx dan block sudah berjalan.
* Runtime lock, remote CLI, peer store, dev reset, difficulty consistency sudah diperbaiki.
* Fork detection sudah berjalan.
* Common ancestor sudah berjalan.
* Same-height fork sync sudah diperbaiki.
* Jika forkA dan forkB sama height tapi tip beda, peer sync sekarang benar:
  sync failed: fork detected
  common ancestor height: 0
  automatic reorg: disabled
* Chain tidak berubah setelah failed fork sync.
* chain validate tetap pass.
* chain info tx count sudah benar:
  total transactions: 3
  coinbase transactions: 3
  normal transactions: 0

Nama patch:
DesKaChain Phase 2.5 — Safe Reorg Experimental

Tujuan utama:
Menambahkan fitur reorg aman secara eksperimental, tetapi tidak otomatis secara default.

Reorg harus:

* mendeteksi common ancestor,
* mengambil branch peer,
* memvalidasi branch peer penuh sebelum apply,
* membandingkan cumulative work,
* menghormati batas max reorg depth,
* punya mode preview/dry-run,
* hanya apply jika user eksplisit pakai --yes,
* menjaga chain tetap valid,
* menjaga supply/balance konsisten setelah reorg,
* mengembalikan transaksi normal dari orphaned blocks ke mempool jika masih valid.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public peer discovery.
* Jangan membuat auto reorg default pada peer sync biasa.
* Jangan rewrite besar.
* Patch incremental.
* Semua tetap harus lolos:

  go work sync
  go test ./node/...

==================================================

1. Prinsip reorg Phase 2.5
   ==================================================

Phase 2.5 adalah experimental safe reorg.

Default behavior tetap:

* peer sync tanpa flag reorg tetap menolak fork.
* Tidak ada reorg otomatis diam-diam.

Reorg hanya boleh terjadi melalui command eksplisit:

reorg preview --peer <p2p-url>
reorg apply --peer <p2p-url> --yes

Atau peer sync dengan flag eksplisit:

peer sync --allow-reorg --max-reorg-depth 10 --yes

Tapi untuk Phase 2.5, prioritas utama adalah command reorg preview/apply.

==================================================
2. Tambahkan konsep ReorgPlan
=============================

Tambahkan package atau file:

internal/reorg/
plan.go
validate.go
apply.go

Atau jika lebih cocok di package chain/p2p, tetap jaga dependency bersih.

Struktur minimal:

type ReorgPlan struct {
Peer string
LocalHeight uint64
LocalTip string
PeerHeight uint64
PeerTip string
CommonAncestorHeight uint64
CommonAncestorHash string
DisconnectBlocks []BlockSummary
ConnectBlocks []BlockSummary
ReorgDepth uint64
LocalWork uint64
PeerWork uint64
PeerHasMoreWork bool
MaxDepth uint64
Allowed bool
Reason string
}

BlockSummary:

* height
* hash
* tx_count
* miner_address
* difficulty

Definitions:

* DisconnectBlocks = block lokal dari tip turun sampai common ancestor + 1.
* ConnectBlocks = block peer dari common ancestor + 1 sampai peer tip.
* ReorgDepth = jumlah block lokal yang akan dilepas.
* PeerHasMoreWork = peer cumulative work > local cumulative work.

==================================================
3. Reorg preview
================

Tambahkan command:

reorg preview --peer <p2p-url>

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg preview --peer http://127.0.0.1:9342

Flags:
--max-depth <n> default 10

Behavior:

1. Handshake peer.
2. Ambil status peer.
3. Cari common ancestor.
4. Ambil block summaries lokal yang akan disconnect.
5. Ambil headers/block summaries peer yang akan connect.
6. Hitung local work dan peer work.
7. Tentukan apakah reorg boleh:

   * peer harus fork dari common ancestor yang sama,
   * peer harus punya cumulative work lebih besar,
   * reorg depth <= max-depth,
   * branch peer harus bisa divalidasi penuh.
8. Jangan mengubah chain.
9. Print plan.

Output jika peer tidak punya work lebih besar:
reorg preview
fork detected
common ancestor height: 0
local height: 3
peer height: 3
disconnect blocks: 3
connect blocks: 3
local work: 13
peer work: 13
allowed: false
reason: peer chain does not have more cumulative work

Output jika peer lebih kuat:
reorg preview
fork detected
common ancestor height: 0
local height: 3
peer height: 5
disconnect blocks: 3
connect blocks: 5
local work: 13
peer work: 21
allowed: true
reason: peer chain has more cumulative work and depth is within limit

==================================================
4. Cumulative work harus dipakai untuk keputusan reorg
======================================================

Saat ini chain info sudah punya:
cumulative work: 13

Pastikan cumulative work dipakai untuk reorg decision.

Rule:

* Jangan reorg hanya karena peer height lebih tinggi.
* Reorg hanya boleh jika peer cumulative work > local cumulative work.
* Jika work sama, jangan reorg.
* Jika peer work lebih rendah, jangan reorg.

Untuk Phase 2.5, CalculateBlockWork boleh tetap sederhana sesuai difficulty sekarang.
Tapi helper harus jelas dan dites.

==================================================
5. Validasi branch peer sebelum apply
=====================================

Sebelum apply reorg:

* Ambil full block peer dari commonAncestorHeight+1 sampai peerHeight.
* Validasi continuity:

  * block pertama previous_hash == common ancestor hash
  * block berikutnya previous_hash == hash block sebelumnya
  * height urut
  * hash benar
  * merkle root benar
  * PoW valid
  * coinbase valid
  * transaksi normal valid dengan replay ledger dari ancestor
* Jangan apply block satu per satu ke chain utama sebelum semua branch lolos validasi.

Jika branch invalid:
allowed: false
reason: peer branch validation failed: <error>

==================================================
6. Apply reorg
==============

Tambahkan command:

reorg apply --peer <p2p-url> --yes

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg apply --peer http://127.0.0.1:9342 --yes

Flags:
--max-depth <n> default 10
--yes required

Tanpa --yes:
refusing to apply reorg without --yes

Behavior:

1. Buat ReorgPlan.
2. Jika plan allowed false, stop dan tampilkan reason.
3. Validasi branch peer penuh.
4. Snapshot state penting jika perlu.
5. Disconnect local blocks dari tip ke common ancestor+1.
6. Connect peer blocks dari common ancestor+1 ke peer tip.
7. Rebuild/replay ledger atau pastikan chain validation pass.
8. Update tip, height, total supply, runtime state cache.
9. Revalidate mempool.
10. Return summary.

Output sukses:
reorg applied
common ancestor height: 0
old height: 3
new height: 5
disconnected blocks: 3
connected blocks: 5
old tip: <hashA>
new tip: <hashB>
chain valid: true

==================================================
7. Storage support untuk rollback/prune
=======================================

Audit storage saat ini.

Tambahkan fungsi jika belum ada:

* GetBlockByHeight(height)
* DeleteBlockByHeight(height)
* SetTip(height, hash)
* HasBlock(height, hash)
* GetHeight()
* GetTip()

Untuk reorg:

* Hapus block lokal height > common ancestor height.
* Tulis block peer height > common ancestor height.
* Update tip.

Penting:

* Jangan meninggalkan block lama sebagai canonical jika storage by height.
* Kalau block lama masih disimpan by hash untuk history, boleh, tapi canonical height harus menunjuk branch baru.
* Untuk Phase 2.5, sederhana saja: canonical by height diganti.

==================================================
8. Mempool handling saat reorg
==============================

Saat disconnect local blocks:

* Kumpulkan transaksi normal dari disconnected blocks.
* Jangan kumpulkan coinbase.
* Setelah reorg apply, coba masukkan kembali transaksi normal yang masih valid ke mempool.
* Transaksi yang sudah ada di connected branch jangan dimasukkan.
* Transaksi yang invalid karena balance/nonce berubah jangan dimasukkan.
* Setelah connect peer branch, hapus tx yang sudah confirmed dari mempool.
* Revalidate mempool.

Output summary:
requeued transactions: <n>
dropped transactions: <n>

==================================================
9. Safety limit
===============

Tambahkan batas:

* default max reorg depth: 10
* command flag:
  --max-depth <n>

Jika reorg depth > max depth:
allowed: false
reason: reorg depth exceeds max depth

Untuk localnet/testnet, user bisa set lebih tinggi:
--max-depth 100

==================================================
10. Peer sync dengan allow reorg
================================

Update command:

peer sync --allow-reorg --max-reorg-depth 10 --yes

Default tanpa --allow-reorg:

* tetap seperti Phase 2.4.3:
  sync failed: fork detected
  automatic reorg: disabled

Dengan --allow-reorg tapi tanpa --yes:
refusing to apply reorg without --yes

Dengan --allow-reorg --yes:

* Jika fork dan plan allowed true, apply reorg.
* Jika plan allowed false, tampilkan reason.
* Jika peer hanya extend local tip normal, sync biasa.

Untuk Phase 2.5, implementasi reorg apply via peer sync boleh memakai helper yang sama dengan command reorg apply.

==================================================
11. RPC endpoints
=================

Tambahkan RPC endpoint:

POST /reorg/preview

Body:
{
"peer": "http://127.0.0.1:9342",
"max_depth": 10
}

Response:
{
"allowed": false,
"reason": "peer chain does not have more cumulative work",
"local_height": 3,
"peer_height": 3,
"common_ancestor_height": 0,
"disconnect_blocks": 3,
"connect_blocks": 3,
"local_work": 13,
"peer_work": 13
}

Tambahkan:

POST /reorg/apply

Body:
{
"peer": "http://127.0.0.1:9342",
"max_depth": 10,
"yes": true
}

Response sukses:
{
"applied": true,
"old_height": 3,
"new_height": 5,
"old_tip": "...",
"new_tip": "...",
"disconnected_blocks": 3,
"connected_blocks": 5,
"requeued_transactions": 0,
"dropped_transactions": 0,
"chain_valid": true
}

Response gagal:
{
"applied": false,
"error": "peer chain does not have more cumulative work"
}

==================================================
12. Tests wajib
===============

Tambahkan/update tests:

1. Reorg preview same work:

* forkA height 3 dan forkB height 3.
* Work sama.
* Preview allowed false.
* Reason peer chain does not have more cumulative work.

2. Reorg preview peer more work:

* forkA height 3.
* forkB height 5.
* Peer work lebih besar.
* Common ancestor height 0.
* Preview allowed true.

3. Reorg apply requires yes:

* reorg apply tanpa --yes gagal.

4. Reorg apply peer more work:

* forkA height 3.
* forkB height 5.
* Apply reorg dari A ke B.
* A height menjadi 5.
* A tip sama dengan B.
* Chain validate A pass.
* Total supply A sesuai 5 block reward.

5. Reorg depth limit:

* forkA height 20.
* forkB height 30.
* max-depth 5.
* Preview allowed false karena depth exceeds max depth.

6. No reorg if peer work equal:

* forkA height 3.
* forkB height 3.
* Apply reorg gagal.
* A tip tetap.

7. No reorg if peer work lower:

* forkA height 5.
* forkB height 3.
* Apply reorg gagal.
* A tip tetap.

8. Invalid peer branch:

* Peer branch dengan block invalid.
* Preview/apply gagal.
* Chain lokal tidak berubah.

9. Mempool requeue:

* Local branch punya normal tx di disconnected block.
* Peer branch tidak punya tx itu.
* Setelah reorg, tx kembali ke mempool jika masih valid.

10. Mempool remove confirmed:

* Mempool punya tx yang ternyata ada di peer branch.
* Setelah reorg, tx dihapus dari mempool.

11. peer sync default still rejects fork:

* Tanpa --allow-reorg, fork tetap error.
* Chain lokal tidak berubah.

12. peer sync allow reorg:

* Dengan --allow-reorg --yes, peer more work menyebabkan reorg apply.
* Chain lokal mengikuti peer.

13. Runtime cache:

* Setelah reorg, /chain/info, /p2p/status, /node/status menampilkan height/tip baru.

==================================================
13. README update
=================

Update README:

Current phase:
Phase 2.5 — Safe Reorg Experimental

Tambahkan penjelasan:

* Reorg adalah proses mengganti canonical branch lokal dengan branch peer yang punya cumulative work lebih besar.
* Reorg tidak otomatis secara default.
* peer sync biasa tetap menolak fork.
* Reorg hanya dilakukan dengan command eksplisit dan --yes.
* DesKaChain memakai cumulative work, bukan height saja.
* Untuk Phase 2.5, reorg masih experimental localnet/testnet.

Tambahkan contoh:

Buat fork:
go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 5

Start:
go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

Preview:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg preview --peer http://127.0.0.1:9342

Apply:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg apply --peer http://127.0.0.1:9342 --yes

Verify:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 node compare --peer http://127.0.0.1:8342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain validate

Expected:
nodes in sync
chain valid

==================================================
14. Expected final commands
===========================

Setelah patch:

go work sync
go test ./node/...

Skenario 1: equal work, reorg harus ditolak

go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg preview --peer http://127.0.0.1:9342

Expected:
allowed: false
reason: peer chain does not have more cumulative work

Skenario 2: peer lebih kuat, reorg boleh

go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 5

go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg preview --peer http://127.0.0.1:9342

Expected:
allowed: true
peer chain has more cumulative work

Apply:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 reorg apply --peer http://127.0.0.1:9342 --yes

Expected:
reorg applied
old height: 3
new height: 5
chain valid: true

Verify:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 node compare --peer http://127.0.0.1:8342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain info

Expected:
nodes in sync
chain valid
height: 5
total supply: 250 DKC

Jangan over-engineer.
Fokus Phase 2.5 hanya pada:

* reorg preview,
* reorg apply eksplisit,
* cumulative work decision,
* max depth safety,
* branch validation before apply,
* rollback/connect canonical blocks,
* mempool cleanup/requeue,
* runtime cache update.
