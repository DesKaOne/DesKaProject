Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur saat ini:

* Core node berada di:
  node/
  cmd/deskachain/
  internal/
  go.mod
  go.sum
* Root repo punya:
  go.work
  README.md
  Roadmap.md
  testdata/
  data/
  prompts/

Status saat ini:

* Phase 1 sampai Phase 2.3.3 sudah selesai.
* go test ./node/... harus pass.
* Runtime hygiene sudah diperbaiki.
* Peer store sudah strict per datadir.
* dev reset membersihkan stale peer.
* difficulty mining sudah konsisten.
* Remote mining via RPC sudah tidak stuck.
* Broadcast tx dan block sudah berhasil.
* Node compare menunjukkan nodes in sync.
* Chain validate pass di node1 dan node2.

Nama patch:
DesKaChain Phase 2.4 — Fork Detection Test & Light Reorg Preparation

Tujuan utama:
Mempersiapkan DesKaChain untuk menghadapi fork ringan tanpa langsung implementasi reorg kompleks penuh.

Pada Phase 2.4:

* Deteksi fork harus jelas.
* Node bisa membandingkan chain dengan peer.
* Node bisa mencari common ancestor secara aman.
* Node bisa membuat block locator.
* Node bisa menolak fork yang tidak didukung reorg.
* Tambahkan command debug fork.
* Jangan aktifkan automatic reorg dulu kecuali mode eksperimen eksplisit.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public peer discovery.
* Jangan implementasi reorg otomatis penuh dulu.
* Jangan rewrite total project.
* Patch incremental.
* Pertahankan semua command lama.
* Semua tetap harus lolos:
  go work sync
  go test ./node/...

==================================================

1. Tambahkan Block Locator
   ==================================================

Tambahkan fungsi di package chain atau p2p:

BuildBlockLocator(chain) []BlockLocatorEntry

BlockLocatorEntry:

* height
* hash

Tujuan:
Dipakai untuk mencari common ancestor dengan peer.

Algoritma sederhana:

* Mulai dari tip.
* Ambil hash height saat ini.
* Mundur 1, 2, 4, 8, 16, dst.
* Selalu sertakan genesis.
* Untuk chain kecil, hasil boleh semua height.

Contoh pada height 10:
10, 9, 8, 6, 2, 0

Tambahkan endpoint P2P:

GET /p2p/locator

Response:
{
"height": 10,
"tip_hash": "...",
"locator": [
{"height": 10, "hash": "..."},
{"height": 9, "hash": "..."},
{"height": 8, "hash": "..."},
{"height": 6, "hash": "..."},
{"height": 2, "hash": "..."},
{"height": 0, "hash": "..."}
]
}

==================================================
2. Tambahkan endpoint find common ancestor
==========================================

Tambahkan endpoint P2P:

POST /p2p/common-ancestor

Body:
{
"locator": [
{"height": 10, "hash": "..."},
{"height": 9, "hash": "..."}
]
}

Response jika ditemukan:
{
"found": true,
"height": 6,
"hash": "..."
}

Response jika tidak ditemukan:
{
"found": false,
"error": "no common ancestor found"
}

Aturan:

* Cocokkan locator dari tinggi ke rendah.
* Return ancestor pertama yang hash-nya ada dan cocok pada height yang sama.
* Genesis harusnya cocok jika network/genesis sama.
* Jika genesis tidak cocok, return no common ancestor.

==================================================
3. Perkuat fork detection saat sync
===================================

Saat SyncFromPeer menemukan header pertama tidak extend local tip:

Sekarang mungkin hanya:
fork detected: peer does not extend local tip

Perkuat output internal dan response:

{
"error": "fork detected",
"local_height": 6,
"local_tip": "...",
"peer_height": 8,
"peer_next_height": 7,
"peer_previous_hash": "...",
"common_ancestor_height": 4,
"common_ancestor_hash": "...",
"reorg_supported": false
}

Perilaku:

* Jika fork terdeteksi, jangan import block.
* Cari common ancestor menggunakan locator.
* Laporkan common ancestor.
* Jangan reorg otomatis dulu.
* Tambahkan TODO:
  Phase 2.5 automatic safe reorg.

==================================================
4. Tambahkan command chain locator
==================================

Command local/remote:

chain locator

Local:
go run ./node/cmd/deskachain --datadir ./testdata/node1 chain locator

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain locator

Output:
height: 10
tip hash: ...
locator:
height=10 hash=...
height=9 hash=...
height=8 hash=...
height=6 hash=...
height=2 hash=...
height=0 hash=...

RPC:
GET /chain/locator

==================================================
5. Tambahkan command chain common-ancestor
==========================================

Command remote debug:

chain common-ancestor --peer http://127.0.0.1:9332

Perilaku:

* Ambil locator chain lokal.
* Kirim ke peer P2P /p2p/common-ancestor.
* Tampilkan hasil.

Remote mode:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332

Output:
common ancestor found
height: 4
hash: ...

Jika tidak:
common ancestor not found

==================================================
6. Tambahkan command fork check
===============================

Tambahkan command:

fork check --peer <p2p-url>

Local dan remote mode.

Contoh:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332

Perilaku:

1. Handshake peer.
2. Ambil status peer.
3. Jika height dan tip sama:
   Output:
   no fork detected
   nodes in sync
4. Jika local tip tidak sama:

   * Ambil locator lokal.
   * Cari common ancestor ke peer.
   * Tampilkan:
     fork detected
     local height: ...
     peer height: ...
     common ancestor height: ...
     local ahead blocks: ...
     peer ahead blocks: ...
     reorg supported: false

Output jika peer lebih tinggi tapi masih extend local:
no fork detected
peer is ahead
missing blocks: N

Output jika local lebih tinggi tapi peer ancestor:
no fork detected
local is ahead
peer missing blocks: N

==================================================
7. Tambahkan RPC debug fork
===========================

Tambahkan endpoint RPC:

POST /fork/check

Body:
{
"peer": "http://127.0.0.1:9332"
}

Response:
{
"fork_detected": true,
"in_sync": false,
"local_height": 6,
"peer_height": 8,
"common_ancestor_height": 4,
"common_ancestor_hash": "...",
"local_ahead_blocks": 2,
"peer_ahead_blocks": 4,
"reorg_supported": false
}

==================================================
8. Simulasi fork lokal untuk testing
====================================

Tambahkan command development-only:

dev fork-sim

Flags:
--datadir-a <path>
--datadir-b <path>
--miner-a <address>
--miner-b <address>
--blocks-a <n>
--blocks-b <n>

Atau jika terlalu besar, cukup tambahkan test helper internal.

Tujuan:
Membuat dua datadir dari genesis sama, lalu mining berbeda sehingga tip berbeda.

Contoh:

* nodeA mine 3 block ke minerA
* nodeB mine 3 block ke minerB
* Keduanya punya genesis sama tapi block height 1 beda.
* fork check harus mendeteksi common ancestor height 0.
* sync harus menolak otomatis karena fork.

Command ini development-only.
Jika implementasi CLI terlalu banyak, minimal buat unit/integration test.

==================================================
9. Sync behavior pada fork
==========================

Saat peer lebih tinggi tapi fork:

* Sync harus gagal aman.
* Tidak boleh import sebagian block.
* Tidak boleh mengubah tip.
* Tidak boleh mengubah total supply.
* Tidak boleh menghapus mempool.
* Error harus jelas:
  sync failed: fork detected
  common ancestor height: X
  automatic reorg: disabled

==================================================
10. Dokumentasi reorg future
============================

Tambahkan docs/ForkAndReorg.md

Isi:

* Apa itu fork.
* Kenapa fork bisa terjadi.
* Phase 2.4 hanya deteksi fork.
* Belum automatic reorg.
* Future Phase 2.5:

  * common ancestor search
  * reorg depth limit
  * rollback ledger
  * rollback mempool
  * apply peer branch
  * fork choice by total work
* Catatan:
  height paling panjang saja tidak cukup.
  Nanti perlu cumulative work/total difficulty.

==================================================
11. Tambahkan cumulative work placeholder
=========================================

Tambahkan field atau function placeholder:

CalculateBlockWork(difficulty uint32) uint64
CalculateCumulativeWork(chain) uint64

Untuk Phase 2.4:

* Boleh sederhana.
* Jangan ubah fork choice utama dulu.
* Gunakan untuk display/debug.

chain info bisa tambah:
cumulative work: <value>

node compare bisa tambah:
local work: ...
peer work: ...

Jika terlalu invasif, cukup helper + tests.

==================================================
12. Tests wajib
===============

Tambahkan/update tests:

1. Block locator:

* Chain height 0 locator berisi genesis.
* Chain height 10 locator berisi tip dan genesis.
* Locator height urut menurun.
* Tidak ada duplikat.

2. Common ancestor:

* Dua chain sama menemukan tip sebagai common ancestor.
* Dua chain fork dari genesis menemukan ancestor height 0.
* Locator tanpa match return not found.

3. Fork check:

* Nodes sama => no fork detected.
* Peer ahead dan extend local => no fork detected, peer ahead.
* Local ahead dan peer ancestor => no fork detected, local ahead.
* Chain berbeda pada height 1 => fork detected, common ancestor 0.

4. Sync fork safety:

* Jika fork detected, sync gagal.
* Local height/tip tidak berubah.
* Total supply tidak berubah.

5. RPC /chain/locator:

* Response valid.

6. RPC /fork/check:

* Response fork_detected benar untuk fork.
* Response in_sync benar untuk node sama.

7. Cumulative work placeholder:

* Work bertambah setelah block ditambah.
* Work genesis tidak membuat error.

==================================================
13. README update
=================

Update README:

Current phase:
Phase 2.4 — Fork Detection Test & Light Reorg Preparation

Tambahkan command:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain locator

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332

Tambahkan catatan:

* Phase 2.4 hanya mendeteksi fork.
* Automatic reorg belum aktif.
* Jika fork terjadi, node menolak sync otomatis.
* Reorg aman akan masuk Phase 2.5.

==================================================
14. Expected final commands
===========================

Setelah patch:

go work sync
go test ./node/...

Jalankan dua node normal:

go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

Remote:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain locator

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332

Expected jika sync:
no fork detected
nodes in sync

Jika sengaja fork:
fork detected
common ancestor height: 0 atau height tertentu
reorg supported: false

Jangan over-engineer.
Fokus patch ini hanya pada:

* block locator,
* common ancestor,
* fork check,
* fork-safe sync rejection,
* reorg preparation,
* dokumentasi future reorg.
