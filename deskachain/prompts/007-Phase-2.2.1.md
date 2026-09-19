Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 2.2 sudah berjalan.
* Node runtime lock sudah aktif.
* Remote mining via --rpc-url sudah berhasil.
* Node2 auto-sync dari node1 sudah berhasil.
* node compare via RPC menunjukkan:
  nodes in sync
  network id: dkc-local-1
  chain id: 777001
  height: 6
  tip hash sama.

Masalah yang ditemukan:
Saat node2 sedang berjalan:

go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --peers http://127.0.0.1:9331

Command berikut gagal karena datadir terkunci:

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer check http://127.0.0.1:9331
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync

Output:
error: datadir is locked by running node
use --rpc-url http://127.0.0.1:8332 to control the running node

Ini benar secara konsep, tetapi remote CLI untuk peer command harus dilengkapi dan README/test command harus diperbaiki.

Nama patch:
DesKaChain Phase 2.2.1 — Remote Peer Commands & Lock UX Fix

Tujuan:
Merapikan command peer saat node sedang berjalan, sehingga semua operasi peer yang menyentuh runtime node bisa dilakukan lewat --rpc-url.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan rewrite total project.
* Patch incremental.
* Pertahankan semua command lama.
* Semua tetap harus lolos:
  go mod tidy
  go test ./...

==================================================

1. Remote peer command wajib didukung
   ==================================================

Pastikan command berikut bisa berjalan dalam remote mode:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer list

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer add http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer remove http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer status

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync --peer http://127.0.0.1:9331

Remote mode berarti:

* CLI tidak membuka datadir lokal.
* CLI tidak membaca/menulis peers.json langsung.
* CLI hanya memanggil RPC node berjalan.
* Node runtime yang melakukan handshake, peer add/remove/status/sync.

==================================================
2. Tambahkan atau perbaiki RPC endpoint peer check
==================================================

Tambahkan endpoint RPC:

POST /peers/check

Body:
{
"url": "http://127.0.0.1:9331"
}

Response sukses:
{
"ok": true,
"url": "http://127.0.0.1:9331",
"node_id": "...",
"network_id": "dkc-local-1",
"chain_id": 777001,
"height": 6,
"tip_hash": "...",
"genesis_hash": "...",
"protocol_version": 1
}

Response gagal:
{
"ok": false,
"url": "http://127.0.0.1:9331",
"error": "peer rejected: genesis hash mismatch"
}

Catatan:

* peer check tidak wajib menyimpan peer.
* peer check hanya handshake dan validasi network/genesis/protocol.

==================================================
3. Tambahkan atau perbaiki RPC endpoint peer status
===================================================

Tambahkan endpoint RPC:

POST /peers/status

Body optional:
{
"include_bad": false
}

Response:
{
"checked_peers": 1,
"active": 1,
"bad": 0,
"peers": [
{
"url": "http://127.0.0.1:9331",
"node_id": "...",
"network_id": "dkc-local-1",
"chain_id": 777001,
"height": 6,
"tip_hash": "...",
"status": "active",
"score": 5,
"last_seen_at": "..."
}
]
}

Perilaku:

* Node runtime melakukan handshake/health check ke semua peers.
* Update metadata peers.json melalui runtime node.
* CLI remote hanya menampilkan hasil.

==================================================
4. Perbaiki RPC endpoint peer sync
==================================

Pastikan endpoint:

POST /peers/sync

Mendukung body:
{
"peer": "http://127.0.0.1:9331",
"include_bad": false
}

Jika peer kosong:

* sync semua peer yang dikenal.

Response:
{
"synced": true,
"local_height_before": 3,
"local_height_after": 6,
"imported_blocks": 3,
"peers_checked": 1,
"results": [
{
"peer": "http://127.0.0.1:9331",
"ok": true,
"height_before": 3,
"height_after": 6,
"imported_blocks": 3,
"message": "sync complete"
}
]
}

Jika sudah up to date:
{
"synced": true,
"local_height_before": 6,
"local_height_after": 6,
"imported_blocks": 0,
"peers_checked": 1,
"results": [
{
"peer": "http://127.0.0.1:9331",
"ok": true,
"height_before": 6,
"height_after": 6,
"imported_blocks": 0,
"message": "local chain already up to date"
}
]
}

Jika gagal:
{
"synced": false,
"error": "fork detected: peer does not extend local tip",
"results": [...]
}

==================================================
5. Remote CLI output harus sama enaknya dengan local CLI
========================================================

Remote:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331

Output sukses:
peer ok
url: http://127.0.0.1:9331
node id: <id>
network id: dkc-local-1
chain id: 777001
height: 6
tip hash: <hash>

Remote:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

Output jika up to date:
sync complete
local chain already up to date
height: 6
imported blocks: 0

Output jika import:
sync started
peers checked: 1
imported blocks: 3
old height: 3
new height: 6
sync complete

Remote:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer status

Output:
checked peers: 1
active: 1
bad: 0
url=http://127.0.0.1:9331 status=active height=6 score=5

==================================================
6. Perbaiki lock UX message
===========================

Saat command local write ditolak karena datadir locked, tampilkan command alternatif yang tepat.

Contoh untuk:

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync

Output:
error: datadir is locked by running node
datadir: testdata\node2
rpc: :8332
p2p: :9332
use:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

Contoh untuk mine:
use:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <address> --blocks <n>

Contoh untuk send:
use:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <from> --to <to> --amount <amount>

Catatan:

* Jika rpc di lock file hanya ":8332", ubah menjadi URL default:
  http://127.0.0.1:8332
* Jika rpc sudah full URL, gunakan apa adanya.

==================================================
7. Peer check local saat datadir locked
=======================================

Tentukan perilaku final:

Opsi yang dipilih:

* Jika command memakai --datadir dan datadir sedang locked, peer check juga harus diarahkan ke remote mode.
* Alasannya: peer check memakai network config/node runtime dan lebih konsisten dilakukan node berjalan.
* Jadi peer check local tetap ditolak saat locked, tetapi pesan error harus memberi command remote yang benar.

==================================================
8. Update README
================

Perbaiki semua contoh command saat node sedang berjalan.

Jangan lagi contohkan:

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync

ketika node2 sedang hidup.

Ganti dengan:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

Bagian manual test Phase 2.2.1:

Setup:

go run ./node/cmd/deskachain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node2 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node2 init
go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new
go run ./node/cmd/deskachain --datadir ./testdata/node2 wallet new

Terminal 1:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

Terminal 2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --peers http://127.0.0.1:9331

Terminal 3:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332

Expected:

* peer check sukses.
* remote peer sync sukses atau up to date.
* peer status active.
* node compare nodes in sync.

==================================================
9. Tests wajib
==============

Tambahkan/update tests:

1. Remote peer check:

* CLI remote peer check memanggil RPC /peers/check.
* Output sukses berisi peer ok.

2. Remote peer status:

* CLI remote peer status memanggil RPC /peers/status.
* Output menampilkan checked peers.

3. Remote peer sync:

* CLI remote peer sync memanggil RPC /peers/sync.
* Jika up to date, output imported blocks 0.
* Jika import block, output imported blocks sesuai.

4. Lock UX:

* Saat datadir locked, local peer sync ditolak.
* Error menyertakan command alternatif --rpc-url yang benar.
* rpc address ":8332" diubah menjadi "http://127.0.0.1:8332".

5. README examples:

* Tidak perlu test file README jika belum ada infra, tapi pastikan dokumentasi diperbarui.

==================================================
10. Expected final commands
===========================

Setelah patch:

go mod tidy
go test ./...

Dengan node1 dan node2 sedang running:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer status

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332

Expected:

* peer check sukses.
* peer sync remote tidak kena datadir lock.
* peer status tampil.
* node compare tetap nodes in sync.

Jangan over-engineer.
Fokus patch ini hanya pada:

* remote peer command,
* RPC peer check/status/sync,
* lock UX yang lebih jelas,
* README command yang benar.
