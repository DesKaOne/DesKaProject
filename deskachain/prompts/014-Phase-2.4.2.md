Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 2.4.1 sudah selesai.
* Common ancestor endpoint sudah fixed.
* Command berikut berhasil:

  go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332

Output:
common ancestor found
height: 3
hash: <tip hash>

* Direct endpoint /p2p/common-ancestor juga berhasil.
* Dua node yang sedang in sync bisa menemukan common ancestor di tip height.

Nama patch:
DesKaChain Phase 2.4.2 — Fork Simulation & Sync Rejection Hardening

Tujuan:
Menguji dan memperkuat fork detection pada kondisi fork nyata, bukan hanya node yang sudah sinkron.

Skenario utama:

* Node A dan Node B punya genesis sama.
* Node A mining block height 1, 2, 3 ke miner A.
* Node B mining block height 1, 2, 3 ke miner B secara terpisah.
* Keduanya punya chain berbeda mulai dari height 1.
* Common ancestor harus ditemukan di height 0.
* fork check harus menampilkan fork detected.
* peer sync harus menolak import otomatis.
* Chain lokal tidak boleh berubah setelah sync fork gagal.

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

1. Tambahkan command dev fork-sim
   ==================================================

Tambahkan command development-only:

dev fork-sim

Flags:
--datadir-a <path>
--datadir-b <path>
--blocks-a <n>
--blocks-b <n>

Opsional:
--miner-a <address>
--miner-b <address>

Jika miner address tidak diberikan:

* Buat wallet baru di masing-masing datadir.
* Gunakan wallet tersebut sebagai miner.

Perilaku:

1. Reset datadir A dan B.
2. Init chain A dan B.
3. Buat wallet miner A dan B jika belum diberikan.
4. Mine blocks-a di chain A.
5. Mine blocks-b di chain B.
6. Print ringkasan:

fork simulation complete
chain A:
datadir: ./testdata/forkA
height: 3
tip: <hashA>
miner: <minerA>
chain B:
datadir: ./testdata/forkB
height: 3
tip: <hashB>
miner: <minerB>
common ancestor expected: height 0

Pastikan:

* Hash height 1 chain A beda dengan hash height 1 chain B.
* Genesis hash sama.

==================================================
2. Tambahkan command fork inspect local datadir
===============================================

Tambahkan command:

fork inspect --other-datadir <path>

Contoh:
go run ./node/cmd/deskachain --datadir ./testdata/forkA fork inspect --other-datadir ./testdata/forkB

Output jika fork:
fork detected
local height: 3
other height: 3
common ancestor height: 0
common ancestor hash: <genesis>
local ahead blocks: 3
other ahead blocks: 3
reorg supported: false

Output jika sama:
no fork detected
chains in sync

Tujuan:
Agar fork bisa dites tanpa harus menjalankan dua node P2P.

==================================================
3. Perkuat fork check via P2P
=============================

Command yang sudah ada:

fork check --peer <p2p-url>

Pastikan jika peer fork dari genesis:

* Output:

  fork detected
  local height: 3
  peer height: 3
  common ancestor height: 0
  common ancestor hash: <genesis hash>
  local ahead blocks: 3
  peer ahead blocks: 3
  reorg supported: false

Jika peer lebih tinggi tapi fork:
fork detected
peer is ahead but does not extend local tip
common ancestor height: <n>
automatic reorg: disabled

==================================================
4. Sync harus aman saat fork
============================

Jika node lokal mencoba:

peer sync

ke peer yang fork:

* Sync harus gagal aman.
* Jangan import block dari peer.
* Jangan mengubah tip lokal.
* Jangan mengubah height lokal.
* Jangan mengubah total supply lokal.
* Jangan menghapus mempool lokal.

Output:
sync failed: fork detected
common ancestor height: 0
automatic reorg: disabled

RPC response /peers/sync juga harus menyertakan:
{
"synced": false,
"error": "fork detected",
"common_ancestor_height": 0,
"reorg_supported": false
}

==================================================
5. Tambahkan endpoint debug fork detail
=======================================

Tambahkan RPC endpoint:

POST /fork/inspect-datadir

Body:
{
"other_datadir": "./testdata/forkB"
}

Response:
{
"fork_detected": true,
"local_height": 3,
"other_height": 3,
"common_ancestor_height": 0,
"common_ancestor_hash": "...",
"local_ahead_blocks": 3,
"other_ahead_blocks": 3,
"reorg_supported": false
}

Endpoint ini development-only.
Boleh hanya local command kalau RPC untuk datadir dianggap terlalu aneh.

==================================================
6. Tambahkan tests wajib
========================

Tambahkan/update tests:

1. Fork sim creates real fork:

* Buat chain A dan B dari genesis sama.
* Mine block berbeda di masing-masing.
* Height sama.
* Tip beda.
* Height 1 hash beda.
* Genesis hash sama.

2. Common ancestor on fork:

* Locator A dikirim ke chain B.
* Common ancestor ditemukan di height 0.

3. Fork inspect local:

* fork inspect A vs B menghasilkan fork detected.
* common ancestor height 0.

4. Fork check P2P:

* Node A dan Node B fork.
* fork check ke peer menghasilkan fork detected.
* common ancestor height 0.

5. Sync rejection:

* Node A sync dari Node B yang fork.
* Sync gagal.
* Height A tetap.
* Tip A tetap.
* Total supply A tetap.
* Chain validate A tetap pass.

6. Same chain still works:

* Dua node in sync.
* fork check menghasilkan no fork detected.
* common ancestor tip.

==================================================
7. README update
================

Update README:

Current phase:
Phase 2.4.2 — Fork Simulation & Sync Rejection Hardening

Tambahkan contoh:

go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

go run ./node/cmd/deskachain --datadir ./testdata/forkA fork inspect --other-datadir ./testdata/forkB

Expected:
fork detected
common ancestor height: 0
reorg supported: false

Tambahkan catatan:

* Phase 2.4.2 hanya memastikan fork terdeteksi dan sync ditolak aman.
* Automatic reorg belum aktif.
* Reorg aman akan masuk Phase 2.5.

==================================================
8. Expected final commands
==========================

Setelah patch:

go work sync
go test ./node/...

Simulasi fork lokal:

go run ./node/cmd/deskachain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

go run ./node/cmd/deskachain --datadir ./testdata/forkA fork inspect --other-datadir ./testdata/forkB

Expected:
fork detected
common ancestor height: 0
reorg supported: false

Jalankan node forkA dan forkB:

go run ./node/cmd/deskachain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/deskachain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

Fork check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 fork check --peer http://127.0.0.1:9342

Expected:
fork detected
common ancestor height: 0
reorg supported: false

Sync rejection:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer add http://127.0.0.1:9342
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 peer sync

Expected:
sync failed: fork detected
automatic reorg: disabled

After failed sync:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8341 chain info

Expected:
chain valid
height unchanged
tip unchanged
total supply unchanged

Jangan over-engineer.
Fokus patch ini hanya:

* membuat simulasi fork nyata,
* inspect fork,
* fork check jelas,
* sync rejection aman,
* tests untuk fork sebelum automatic reorg Phase 2.5.
