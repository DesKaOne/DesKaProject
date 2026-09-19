Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:

* Phase 1 MVP selesai.
* Phase 1.5 stabilization selesai.
* Phase 1.6 transfer dan mempool hardening selesai.
* Phase 2 local HTTP P2P multi node sync sudah selesai.
* go test ./... pass.
* Dua node lokal sudah bisa sync block.
* Node2 berhasil import block height 1, 2, 3 dari node1.
* chain validate sudah pass.

Masalah/risiko yang ditemukan:
Saat node sedang berjalan dengan command:

go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

User masih bisa menjalankan command lokal lain yang memakai datadir sama, misalnya:

go run ./node/cmd/deskachain --datadir ./testdata/node1 mine --address <addr> --blocks 3

Untuk MVP ini kadang terlihat berhasil, tetapi secara desain ini tidak ideal karena dua proses berbeda bisa membuka storage/datadir yang sama. Ke depan ini rawan:

* race condition,
* database lock,
* state stale,
* node runtime tidak selalu sadar perubahan lokal,
* broadcast block/tx tidak konsisten.

Nama patch:
DesKaChain Phase 2.1 — Node Runtime Hardening, Remote CLI, dan Broadcast Stabilization

Tujuan utama:
Membuat node runtime menjadi sumber kebenaran saat node sedang berjalan, sehingga mining/send/peer sync bisa dilakukan lewat RPC node yang hidup, bukan dengan proses CLI lain yang menulis langsung ke datadir yang sama.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public peer discovery dulu.
* Jangan rewrite total project.
* Patch incremental saja.
* Pertahankan semua command lama.
* Semua tetap harus lolos:
  go mod tidy
  go test ./...
  go run ./node/cmd/deskachain

==================================================

1. Tambahkan datadir runtime lock
   ==================================================

Tambahkan mekanisme lock sederhana di datadir:

<datadir>/node.lock

Saat command:

node start

berjalan:

* Buat lock file node.lock.
* Isi lock file:

  * pid
  * rpc address
  * p2p address
  * started_at
* Saat node berhenti normal, hapus lock file.
* Jika proses crash dan lock tertinggal, command berikut harus bisa mendeteksi stale lock sebisa mungkin.

Aturan:

* Untuk Windows dan Linux, buat implementasi lock sederhana yang aman.
* Minimal:

  * Jika node.lock ada, baca isinya.
  * Jika command lokal yang akan menulis chain/mempool/wallet dijalankan, tampilkan warning/error.
* Jangan membuat implementasi terlalu kompleks.

Command lokal yang harus menolak jika datadir sedang dipakai node runtime:

* init jika perlu menulis genesis
* dev reset
* mine
* send
* mempool clear
* peer add/remove/sync jika command itu menulis peers atau melakukan import block
* wallet new/export jika wallet store dianggap bagian runtime

Output contoh:

datadir is locked by running node
datadir: ./testdata/node1
rpc: :8331
p2p: :9331
use --rpc-url http://127.0.0.1:8331 to control the running node

Tambahkan flag override khusus development:

--ignore-lock

Contoh:

go run ./node/cmd/deskachain --datadir ./testdata/node1 --ignore-lock chain info

Aturan:

* Read-only command boleh tetap jalan meski locked:

  * chain info
  * chain print
  * chain validate
  * balance
  * wallet list
  * peer list
* Write command harus menolak kecuali --ignore-lock dipakai.
* dev reset tetap harus menolak lock walaupun --ignore-lock, kecuali ada flag ekstra:
  --force
  Tapi default jangan force.

==================================================
2. Tambahkan remote CLI via --rpc-url
=====================================

Tambahkan global flag:

--rpc-url <url>

Jika --rpc-url diisi, command tertentu harus menggunakan RPC node berjalan, bukan storage lokal.

Contoh:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain info

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 3

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <a> --to <b> --amount 10

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

Remote command yang wajib didukung:

* chain info
* chain validate
* balance <address>
* mempool list
* send
* mine
* peer list
* peer add
* peer remove
* peer sync
* tx get
* address validate
* wallet inspect jika memungkinkan

Jika command belum didukung remote, tampilkan:

remote mode not supported for this command yet

Tujuan:
Agar saat node start hidup, user tidak perlu menjalankan proses lokal yang menulis langsung ke datadir.

==================================================
3. Tambahkan RPC endpoint untuk mining yang lebih lengkap
=========================================================

Pastikan RPC endpoint mining ada dan stabil:

POST /mine

Body:
{
"address": "<minerAddress>",
"blocks": 3
}

Response:
{
"mined_blocks": 3,
"new_height": 6,
"miner_balance": "300 IDR",
"blocks": [
{
"height": 4,
"hash": "...",
"txs": 2,
"reward": "50 IDR",
"difficulty": 4,
"nonce": 12345
}
]
}

Perilaku:

* Mining lewat RPC harus menambahkan block ke node runtime.
* Setelah block berhasil ditambahkan, broadcast block ke peers.
* Jangan perlu proses CLI lain menulis datadir.

==================================================
4. Tambahkan RPC endpoint send yang broadcast tx
================================================

Pastikan RPC endpoint send:

POST /send

Body:
{
"from": "<addressA>",
"to": "<addressB>",
"amount": "10"
}

Response:
{
"status": "pending",
"id": "<txid>",
"from": "<addressA>",
"to": "<addressB>",
"amount": "10 IDR",
"fee": "0 IDR",
"nonce": 1,
"broadcast": {
"peers": 1,
"success": 1,
"failed": 0
}
}

Perilaku:

* Send lewat RPC harus membuat tx pending di node runtime.
* Tx harus masuk mempool lokal.
* Tx harus dibroadcast ke peers.
* Jika broadcast gagal, tx lokal tetap pending.
* Error broadcast dikembalikan dalam field broadcast, bukan membuat send gagal total.

==================================================
5. Tambahkan RPC endpoint peer management
=========================================

Pastikan endpoint berikut ada dan dipakai oleh remote CLI:

GET /peers

Response:
{
"peers": [
"http://127.0.0.1:9331"
]
}

POST /peers

Body:
{
"url": "http://127.0.0.1:9331"
}

Response:
{
"added": true,
"url": "http://127.0.0.1:9331"
}

DELETE /peers

Body:
{
"url": "http://127.0.0.1:9331"
}

Response:
{
"removed": true,
"url": "http://127.0.0.1:9331"
}

POST /peers/sync

Body optional:
{
"peer": "http://127.0.0.1:9331"
}

Jika peer kosong:

* sync semua peer yang dikenal.

Response:
{
"synced": true,
"local_height_before": 0,
"local_height_after": 3,
"imported_blocks": 3,
"peers_checked": 1
}

==================================================
6. Kurangi noise log auto-sync
==============================

Saat node2 auto-sync setiap 5 detik, log sekarang terlalu ramai:

sync started
sync complete up_to_date=true

Ubah agar:

* Jika chain sudah up to date, jangan log setiap interval.
* Log up_to_date maksimal setiap 1 menit, atau hanya jika flag verbose aktif.
* Tambahkan flag node start:

  --verbose

Default:
false

Jika --verbose=false:

* Log hanya kejadian penting:

  * node started
  * peer added
  * imported block
  * received tx
  * received block
  * sync error
  * broadcast error

Jika --verbose=true:

* Boleh log sync started dan up_to_date setiap interval.

==================================================
7. Tambahkan command node status remote/lokal
=============================================

Command:

node status

Lokal:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node status

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node status

Output:
datadir: ./testdata/node1
rpc: :8331
p2p: :9331
height: 3
tip hash: ...
peers: 1
mempool pending: 0
locked: true

Jika lokal tidak ada node berjalan:
locked: false

==================================================
8. Pastikan node start memakai state terbaru
============================================

Saat node start berjalan:

* RPC dan P2P handler harus selalu membaca tip terbaru dari chain runtime.
* Jika ada mining lewat RPC, status P2P /p2p/status harus langsung mencerminkan height terbaru.
* Hindari cache height/tip yang stale.
* Jika memang ada cache, harus diupdate setelah append block.

Endpoint yang harus akurat:

* GET /node/status
* GET /chain/info
* GET /p2p/status
* GET /p2p/tip

==================================================
9. Broadcast tx end-to-end
==========================

Pastikan skenario ini berhasil:

Node1:
datadir ./testdata/node1
rpc :8331
p2p :9331

Node2:
datadir ./testdata/node2
rpc :8332
p2p :9332
peer http://127.0.0.1:9331

Alur:

1. Node1 mine 3 block lewat RPC remote CLI:
   go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3

2. Node2 sync otomatis atau manual dari node1.

3. Node1 send 10 IDR ke walletB lewat RPC remote CLI:
   go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10

Expected:

* Tx masuk mempool node1.
* Tx dibroadcast ke node2.
* Mempool node2 berisi tx yang sama.
* Command ini harus menampilkan tx pending di node2:

  go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list

==================================================
10. Broadcast block end-to-end
==============================

Lanjutan skenario:

1. Node1 mine 1 block lewat RPC remote CLI:
   go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1

Expected:

* Block berisi coinbase + tx pending.
* Block dibroadcast ke node2.
* Node2 menerima block.
* Node2 append block jika extend tip.
* Mempool node2 kosong setelah block diterima.
* Balance walletB di node2 menjadi 10 IDR.
* Tip hash node1 dan node2 sama.
* Chain validate node2 pass.

==================================================
11. Tambahkan command compare untuk debug dua node
==================================================

Tambahkan command:

node compare --peer <rpc-url>

Contoh:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332

Perilaku:

* Ambil /chain/info dari node lokal/remote utama.
* Ambil /chain/info dari peer RPC.
* Bandingkan:

  * height
  * tip hash
  * total supply
  * pending tx count

Output jika sama:
nodes in sync
height: 4
tip hash: ...

Output jika beda:
nodes differ
local height: 4
peer height: 3
local tip: ...
peer tip: ...

Command ini hanya untuk debug Phase 2.1.

==================================================
12. Tests wajib
===============

Tambahkan atau update tests:

1. Lock file:

* node lock dibuat.
* write command menolak saat lock ada.
* read command tetap boleh.
* stale lock bisa dibaca tanpa panic.

2. Remote CLI:

* remote chain info mengambil data dari RPC.
* remote mine memanggil RPC /mine.
* remote send memanggil RPC /send.
* remote mempool list mengambil dari RPC.

3. RPC mining:

* POST /mine menambah height.
* Response berisi mined blocks dan miner balance.
* Setelah mining, /p2p/status height ikut berubah.

4. RPC send:

* POST /send membuat tx pending.
* Tx masuk mempool.
* Response berisi broadcast summary.

5. Broadcast tx:

* Node1 send tx.
* Node2 menerima tx di mempool.

6. Broadcast block:

* Node1 mine block berisi tx.
* Node2 menerima block.
* Node2 mempool bersih.
* Balance penerima benar.
* Tip node1 dan node2 sama.

7. Log noise:

* Tidak perlu test terlalu detail, cukup pastikan verbose flag ada dan tidak merusak start.

==================================================
13. README update
=================

Update README.md:

Current phase:
Phase 2.1 — Node Runtime Hardening, Remote CLI, dan Broadcast Stabilization

Tambahkan penjelasan penting:

* Saat node start hidup, jangan menjalankan command lokal yang menulis ke datadir yang sama.
* Gunakan --rpc-url untuk mengontrol node yang sedang berjalan.
* Datadir lock membantu mencegah dua proses menulis storage yang sama.

Contoh alur baru yang benar:

Terminal 1:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

Terminal 2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --peers http://127.0.0.1:9331

Terminal 3:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 balance <walletB>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332

Expected:

* node2 menerima tx broadcast.
* node2 menerima block broadcast.
* walletB balance 10 IDR.
* node1 dan node2 in sync.
* chain validate pass di kedua node.

==================================================
14. Expected final commands
===========================

Setelah patch selesai:

go mod tidy
go test ./...

Setup:

go run ./node/cmd/deskachain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node2 dev reset --yes

go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node2 init

go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new
go run ./node/cmd/deskachain --datadir ./testdata/node2 wallet new

Start node:

Terminal 1:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

Terminal 2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --peers http://127.0.0.1:9331

Remote control:

Terminal 3:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 balance <walletB>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain validate

Expected:

* Remote mining sukses.
* Remote send sukses.
* Tx broadcast ke node2.
* Block broadcast ke node2.
* WalletB balance 10 IDR di node2.
* Node compare menunjukkan nodes in sync.
* Kedua node chain valid.

Jangan over-engineer.
Fokus Phase 2.1 pada:

* datadir lock,
* remote CLI via RPC,
* mining runtime,
* send runtime,
* broadcast tx,
* broadcast block,
* log auto-sync yang tidak berisik.
