Kamu sedang bekerja pada project Go monorepo IndoChain.

Struktur:

* node/

  * cmd/indochain/
  * internal/
  * go.mod
  * go.sum
* root punya go.work

Status saat ini:

* Phase 2.4 sudah diterapkan sebagian.
* go work sync berhasil.
* go test ./node/... pass.
* Dua node lokal bisa berjalan.
* Node1 mining 3 block.
* Node2 menerima block broadcast height 1, 2, 3.
* chain locator di node1 berhasil.
* fork check dari node1 ke node2 berhasil:
  no fork detected
  nodes in sync
  local height: 3
  peer height: 3
* Tetapi command common ancestor gagal:

Command:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332

Output:
common ancestor not found
error: no common ancestor found

Padahal node1 dan node2 sedang in sync dengan tip hash sama:
height: 3
tip hash: 00004c32de7171fdb3d8335d171e86c89d0af1c3d202961426c52cf1590454a3

Masalah:
Endpoint /p2p/common-ancestor atau CLI chain common-ancestor tidak bekerja benar. Jika dua node punya chain sama, common ancestor harus ditemukan di tip tertinggi, bukan not found.

Nama patch:
IndoChain Phase 2.4.1 — Common Ancestor Endpoint Fix

Tujuan:
Memperbaiki command dan endpoint common ancestor agar:

* node yang sama menemukan common ancestor di tip,
* node yang fork menemukan common ancestor di genesis atau titik fork,
* locator JSON dikirim/dibaca dengan benar,
* hasil common ancestor bisa dipakai oleh fork check dan sync fork safety.

Aturan penting:

* Jangan implementasi automatic reorg dulu.
* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan rewrite total project.
* Patch incremental.
* Semua tetap harus lolos:
  go work sync
  go test ./node/...

==================================================

1. Audit tipe BlockLocatorEntry
   ==================================================

Pastikan tipe locator punya JSON tags eksplisit.

Contoh:

type BlockLocatorEntry struct {
Height uint64 `json:"height"`
Hash   string `json:"hash"`
}

Pastikan response /p2p/locator memakai format:

{
"height": 3,
"tip_hash": "...",
"locator": [
{"height": 3, "hash": "..."},
{"height": 2, "hash": "..."},
{"height": 1, "hash": "..."},
{"height": 0, "hash": "..."}
]
}

Pastikan request /p2p/common-ancestor memakai format:

{
"locator": [
{"height": 3, "hash": "..."},
{"height": 2, "hash": "..."},
{"height": 1, "hash": "..."},
{"height": 0, "hash": "..."}
]
}

==================================================
2. Perbaiki endpoint POST /p2p/common-ancestor
==============================================

Endpoint:

POST /p2p/common-ancestor

Body:
{
"locator": [...]
}

Behavior wajib:

* Decode JSON body dengan benar.

* Jika locator kosong:
  return:
  {
  "found": false,
  "error": "empty locator"
  }

* Untuk setiap entry locator dari urutan pertama sampai terakhir:

  * Ambil block lokal pada height entry.Height.
  * Jika block ada dan block.Hash == entry.Hash:
    return:
    {
    "found": true,
    "height": entry.Height,
    "hash": entry.Hash
    }

* Jika tidak ada match:
  return:
  {
  "found": false,
  "error": "no common ancestor found"
  }

Catatan:

* Jangan lookup hanya berdasarkan hash tanpa height kalau storage belum punya index hash.
* Height + hash adalah cara paling sederhana dan aman untuk Phase 2.4.1.
* Genesis harus match kalau network/genesis sama.

==================================================
3. Tambahkan logging debug ringan
=================================

Saat /p2p/common-ancestor dipanggil, log ringkas jika verbose aktif:

common ancestor request locator_count=4
common ancestor found height=3 hash=<hash>

Jika tidak ditemukan:
common ancestor not found locator_count=4 local_height=3

Jangan terlalu noisy pada default mode.

==================================================
4. Perbaiki CLI chain common-ancestor
=====================================

Command:

chain common-ancestor --peer <p2p-url>

Local dan remote mode harus benar.

Remote mode contoh:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332

Behavior:

1. CLI memanggil RPC node utama untuk mengambil locator lokal:
   GET /chain/locator
2. CLI atau RPC node utama mengirim locator itu ke peer P2P:
   POST http://127.0.0.1:9332/p2p/common-ancestor
3. Tampilkan hasil.

Jika common ancestor ditemukan:
common ancestor found
height: 3
hash: <hash>

Jika tidak:
common ancestor not found
error: no common ancestor found

Pastikan remote mode tidak salah mengambil locator dari datadir lokal CLI.

==================================================
5. Tambahkan atau perbaiki RPC /chain/common-ancestor
=====================================================

Jika command remote memakai RPC, tambahkan endpoint:

POST /chain/common-ancestor

Body:
{
"peer": "http://127.0.0.1:9332"
}

Behavior:

* Runtime node membuat locator dari chain runtime.
* Runtime node POST locator ke peer /p2p/common-ancestor.
* Return hasil.

Response sukses:
{
"found": true,
"height": 3,
"hash": "..."
}

Response gagal:
{
"found": false,
"error": "no common ancestor found"
}

Command remote harus memakai endpoint ini agar tidak membuka datadir lokal.

==================================================
6. Fork check harus memakai common ancestor helper yang sama
============================================================

Update fork check agar memakai helper common ancestor yang sama dengan command chain common-ancestor.

Jika nodes in sync:

* boleh tetap short-circuit,
* tetapi test tetap harus memastikan common ancestor endpoint bisa menemukan tip.

Jika fork:

* gunakan common ancestor helper.
* Jangan ada dua implementasi common ancestor yang beda logic.

==================================================
7. Tambahkan manual debug command output
========================================

Tambahkan flag optional:

chain common-ancestor --peer <p2p-url> --debug

Output debug:
local locator count: 4
local locator:
height=3 hash=...
height=2 hash=...
height=1 hash=...
height=0 hash=...
peer: http://127.0.0.1:9332
common ancestor found
height: 3
hash: ...

Tujuan:
Jika gagal lagi, kelihatan locator yang dikirim.

==================================================
8. Tests wajib
==============

Tambahkan/update tests:

1. Common ancestor same chain:

* Buat chain A dan chain B dengan block sama sampai height 3.
* Build locator dari chain A.
* Kirim ke common ancestor logic chain B.
* Expected:
  found true
  height 3
  hash tip hash.

2. Common ancestor genesis fork:

* Chain A dan B punya genesis sama.
* Chain A mine block height 1 dengan miner A.
* Chain B mine block height 1 dengan miner B.
* Locator A dikirim ke B.
* Expected:
  found true
  height 0
  hash genesis.

3. Common ancestor no match:

* Chain A dan B punya genesis berbeda atau mock hash berbeda.
* Expected:
  found false.

4. P2P endpoint /p2p/common-ancestor:

* Start test p2p server dengan chain height 3.
* POST locator yang cocok.
* Expected found true height 3.
* POST locator kosong.
* Expected found false error empty locator.

5. RPC /chain/common-ancestor:

* Runtime node A dan peer node B chain sama.
* Call endpoint.
* Expected found true height tip.

6. CLI remote chain common-ancestor:

* Jika CLI tests ada, pastikan remote command tidak membuka datadir lokal.
* Minimal test handler/client function.

7. Fork check still works:

* Nodes in sync tetap no fork detected.
* Fork dari genesis tetap fork detected dan common ancestor height 0.

==================================================
9. README update
================

Update README Phase 2.4.1:

Tambahkan troubleshooting:

Jika command:
chain common-ancestor --peer <p2p-url>

menghasilkan:
no common ancestor found

Padahal fork check menyatakan nodes in sync, jalankan:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain locator

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332 --debug

Expected untuk nodes in sync:
common ancestor found
height: <tip height>
hash: <tip hash>

==================================================
10. Expected final commands
===========================

Setelah patch:

go work sync
go test ./node/...

Dengan node1 dan node2 running serta sudah in sync di height 3:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain locator

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332

Expected:
common ancestor found
height: 3
hash: 00004c32de7171fdb3d8335d171e86c89d0af1c3d202961426c52cf1590454a3

Debug:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332 --debug

Expected:
local locator count: 4
common ancestor found
height: 3

Fork simulation expected:

* Jika chain fork dari genesis, common ancestor found height 0.
* Sync tetap menolak automatic reorg.
* fork check menampilkan fork detected.

Jangan over-engineer.
Fokus patch ini hanya memperbaiki common ancestor logic, endpoint, JSON schema, remote CLI, dan tests.
