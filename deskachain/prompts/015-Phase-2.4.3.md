Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur:

* node/

  * cmd/deskachain/
  * internal/
  * go.mod
  * go.sum
* root punya go.work

Status saat ini:

* Phase 2.4.2 sudah berjalan.
* go work sync berhasil.
* go test ./node/... pass.
* dev fork-sim berhasil membuat dua chain fork:

  * forkA height 3 tip berbeda
  * forkB height 3 tip berbeda
  * genesis sama
  * common ancestor height 0
* fork inspect local berhasil:
  fork detected
  common ancestor height: 0
* fork check via P2P berhasil:
  fork detected
  common ancestor height: 0
  reorg supported: false
* chain validate forkA pass.

Bug yang ditemukan:
Command:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer sync

Pada kondisi forkA dan forkB punya height sama tetapi tip berbeda, output saat ini:

sync complete
local chain already up to date
height: 3
imported blocks: 0

Ini salah.

Expected:
sync failed: fork detected
common ancestor height: 0
automatic reorg: disabled

Penyebab kemungkinan:
Sync logic hanya mengecek:

if peerHeight <= localHeight {
return up to date
}

Padahal jika peerHeight == localHeight tetapi peerTip != localTip, itu fork.

Bug tambahan:
chain info pada forkA menampilkan:

coinbase blocks: 3
total transactions: 0

Padahal setiap mined block punya coinbase transaction. Jika total transactions memang didefinisikan termasuk coinbase, maka harusnya:

total transactions: 3

Nama patch:
DesKaChain Phase 2.4.3 — Same Height Fork Sync Fix & Chain Info Tx Count

Tujuan:

1. Memperbaiki peer sync agar mendeteksi fork pada height sama tapi tip berbeda.
2. Memastikan sync tidak pernah menyebut up-to-date jika tip hash berbeda.
3. Memastikan response RPC /peers/sync membawa error fork detected.
4. Memperbaiki chain info total transactions agar konsisten.
5. Menambah tests agar bug ini tidak balik lagi.

Aturan penting:

* Jangan implementasi automatic reorg dulu.
* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan rewrite besar.
* Patch incremental.
* Semua tetap harus lolos:

  go work sync
  go test ./node/...

==================================================

1. Perbaiki SyncFromPeer same-height fork
   ==================================================

Audit fungsi sync, kemungkinan bernama:

* SyncFromPeer
* SyncAllPeers
* PeerSync
* p2p sync service

Tambahkan logic wajib:

Ambil status lokal:

* localHeight
* localTip

Ambil status peer:

* peerHeight
* peerTip

Rules:

Case 1:
peerHeight == localHeight && peerTip == localTip
Result:
up to date

Case 2:
peerHeight == localHeight && peerTip != localTip
Result:
fork detected
cari common ancestor
return error
jangan import block

Case 3:
peerHeight < localHeight
Jika peer tip sama dengan block lokal pada height peer:
local ahead, no import needed
Jika peer tip beda dengan block lokal pada height peer:
fork detected
cari common ancestor
return error

Case 4:
peerHeight > localHeight
Ambil header localHeight+1.
Jika previous_hash header pertama == localTip:
sync boleh lanjut
Jika previous_hash beda:
fork detected
cari common ancestor
return error

==================================================
2. Error detail fork harus konsisten
====================================

Saat fork detected, error detail minimal:

sync failed: fork detected
local height: 3
local tip: <localTip>
peer height: 3
peer tip: <peerTip>
common ancestor height: 0
common ancestor hash: <genesis>
automatic reorg: disabled
TODO Phase 2.5 automatic safe reorg

Untuk CLI remote peer sync, output harus jelas:

sync failed: fork detected
local height: 3
peer height: 3
common ancestor height: 0
automatic reorg: disabled

Jangan output:
local chain already up to date
jika tip berbeda.

==================================================
3. RPC /peers/sync response fork
================================

Jika fork detected, response /peers/sync harus:

{
"synced": false,
"error": "fork detected",
"local_height_before": 3,
"local_height_after": 3,
"imported_blocks": 0,
"peers_checked": 1,
"common_ancestor_height": 0,
"common_ancestor_hash": "...",
"reorg_supported": false,
"results": [
{
"peer": "http://127.0.0.1:9342",
"ok": false,
"height_before": 3,
"height_after": 3,
"imported_blocks": 0,
"message": "fork detected",
"common_ancestor_height": 0,
"common_ancestor_hash": "..."
}
]
}

Jika ada beberapa peers:

* Jika satu peer fork, result peer tersebut ok=false.
* Jangan ubah chain lokal.
* Jika semua peers gagal/fork, synced=false.
* Jika sebagian peers sukses import dan sebagian fork, synced=true boleh, tapi result harus jelas.
  Untuk Phase 2.4.3, cukup pastikan single peer fork menghasilkan synced=false.

==================================================
4. Peer score saat fork
=======================

Saat sync mendeteksi fork:

* Peer score boleh dikurangi, misalnya -10.
* Jangan langsung bad jika hanya fork normal.
* Status peer tetap active atau unknown, kecuali score turun melewati bad threshold.
* Reason:
  "fork detected"

Jangan auto-delete peer.

==================================================
5. Chain state tidak boleh berubah saat same-height fork sync
=============================================================

Pada kondisi forkA height 3 dan forkB height 3:

Sebelum sync:

* height A = 3
* tip A = hashA
* total supply A = 150 DKC

Setelah peer sync ke forkB yang fork:

* height A tetap 3
* tip A tetap hashA
* total supply A tetap 150 DKC
* chain validate A tetap pass
* mempool A tidak berubah

Tambahkan guard/test untuk ini.

==================================================
6. Perbaiki chain info total transactions
=========================================

Saat chain punya:

* genesis txs=0
* block 1 coinbase txs=1
* block 2 coinbase txs=1
* block 3 coinbase txs=1

chain info harus menampilkan:

coinbase blocks: 3
total transactions: 3

Definisi total transactions:

* Total semua transaksi dalam semua block, termasuk coinbase.
* Jika ingin menampilkan normal tx terpisah, tambahkan field:
  normal transactions: 0
  coinbase transactions: 3

Rekomendasi output chain info:
total transactions: 3
coinbase transactions: 3
normal transactions: 0

Jangan merusak field lama.
Tambahkan field baru boleh.

RPC /chain/info juga harus konsisten.

==================================================
7. Tests wajib
==============

Tambahkan/update tests:

1. Same height same tip:

* Chain A dan B sama height 3 dan tip sama.
* Sync result up to date.
* No fork.

2. Same height different tip:

* Chain A dan B fork dari genesis.
* Height sama 3.
* Tip beda.
* Sync result fork detected.
* Tidak boleh up to date.
* Common ancestor height 0.

3. Peer lower height but matching ancestor:

* Local height 3.
* Peer height 2.
* Peer tip sama dengan local block height 2.
* Result local ahead, no fork.

4. Peer lower height but different tip:

* Local height 3.
* Peer height 2 tapi hash height 2 beda.
* Fork detected.

5. Peer higher extends local:

* Local height 1.
* Peer height 3 dan header height 2 previous_hash == local tip.
* Sync import block.

6. Peer higher fork:

* Local height 1.
* Peer height 3 tapi header height 2 previous_hash != local tip.
* Fork detected, no import.

7. RPC peer sync fork response:

* Single peer fork.
* /peers/sync returns synced=false.
* error contains fork detected.
* common_ancestor_height == 0.

8. CLI remote peer sync fork:

* Output does not contain:
  local chain already up to date
* Output contains:
  sync failed: fork detected

9. Chain info tx count:

* After mining 3 coinbase blocks:
  total transactions == 3
  coinbase transactions == 3
  normal transactions == 0

10. Chain unchanged after failed fork sync:

* height unchanged.
* tip unchanged.
* total supply unchanged.
* chain validate pass.

==================================================
8. README update
================

Update README Phase 2.4.3:

Tambahkan catatan:

* Same height does not mean same chain.
* Nodes are only up to date if height and tip hash match.
* If height sama tapi tip beda, itu fork.
* Sync will reject fork until Phase 2.5 safe reorg.

Tambahkan contoh:

go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer add http://127.0.0.1:9342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer sync

Expected:
sync failed: fork detected
common ancestor height: 0
automatic reorg: disabled

==================================================
9. Expected final commands
==========================

Setelah patch:

go work sync
go test ./node/...

Simulasi:

go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

Terminal A:
go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

Terminal B:
go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

Terminal kontrol:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 fork check --peer http://127.0.0.1:9342

Expected:
fork detected
common ancestor height: 0

Sync:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer add http://127.0.0.1:9342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer sync

Expected:
sync failed: fork detected
common ancestor height: 0
automatic reorg: disabled

Verify unchanged:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain validate

Expected:
height: 3
tip hash: original forkA tip
total supply: 150 DKC
total transactions: 3
coinbase transactions: 3
normal transactions: 0
chain valid

Jangan over-engineer.
Fokus patch ini hanya:

* same-height fork detection,
* lower-height fork detection,
* sync rejection output,
* RPC sync response,
* chain info tx count consistency.
