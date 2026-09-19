Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:

* Phase 1 MVP selesai.
* Phase 1.5 stabilization selesai.
* Phase 1.6 transfer dan mempool hardening selesai.
* go test ./... pass.
* init, dev reset, wallet, mine, send, mempool, balance, chain info, chain validate sudah berjalan.
* Transfer antar wallet sudah terbukti:

  * Wallet A mining 3 block = 150 DKC.
  * Wallet A kirim 10 DKC ke Wallet B.
  * Wallet A mining 1 block lagi.
  * Wallet A akhir = 190 DKC.
  * Wallet B akhir = 10 DKC.
  * Total supply = 200 DKC.
  * Chain valid.

Nama patch:
DesKaChain Phase 2 — Local P2P Multi Node Sync

Tujuan utama:
Membuat beberapa node DesKaChain lokal dapat saling terhubung, sinkronisasi block, broadcast transaksi, dan broadcast block hasil mining.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public peer discovery dulu.
* Jangan rewrite total project.
* Patch incremental.
* Pertahankan semua command lama.
* Semua tetap harus lolos:
  go mod tidy
  go test ./...
  go run ./node/cmd/deskachain

==================================================

1. Konsep P2P Phase 2
   ==================================================

Untuk Phase 2, gunakan P2P sederhana berbasis HTTP antar node.

Alasan:

* Lebih mudah dites lokal.
* Lebih mudah debug.
* Cocok untuk MVP sebelum masuk TCP/libp2p/gossip yang lebih serius.

Setiap node punya:

* datadir berbeda
* RPC/API listen address berbeda
* P2P listen address berbeda
* daftar peers

Contoh:
Node 1:
datadir: ./testdata/node1
rpc: :8331
p2p: :9331

Node 2:
datadir: ./testdata/node2
rpc: :8332
p2p: :9332
peer: http://127.0.0.1:9331

Node 3:
datadir: ./testdata/node3
rpc: :8333
p2p: :9333
peer: [http://127.0.0.1:9331,http://127.0.0.1:9332](http://127.0.0.1:9331,http://127.0.0.1:9332)

==================================================
2. Struktur package baru
========================

Tambahkan package baru:

internal/p2p/
peer.go
server.go
client.go
sync.go
broadcast.go
types.go

Jaga dependency tetap bersih:

* p2p boleh memakai chain, storage, mempool, types, amount jika perlu.
* chain jangan bergantung langsung ke p2p.
* mining boleh memanggil broadcast melalui interface/callback agar tidak coupling parah.
* Hindari circular import.

==================================================
3. Command baru: node start
===========================

Tambahkan command:

node start

Flags:
--rpc <addr>
--p2p <addr>
--peers <comma-separated-peer-urls>
--mine-address <address optional>
--auto-mine false/true optional
--sync-interval <duration optional>

Default:
--rpc :8332
--p2p :9332
--peers ""
--auto-mine false
--sync-interval 5s

Contoh:

go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --peers http://127.0.0.1:9331

Perilaku:

* Jika chain belum init, otomatis init genesis.
* Start RPC server biasa.
* Start P2P server.
* Connect ke peers dari flag --peers.
* Jalankan background sync loop setiap sync interval.
* Log ringkas:
  node started
  datadir: ./testdata/node1
  rpc: :8331
  p2p: :9331
  peers: 0
  height: 0
  tip: <hash>

Catatan:

* Jangan hapus command rpc lama.
* node start adalah mode gabungan RPC + P2P.

==================================================
4. Command peer
===============

Tambahkan command:

peer list
peer add <url>
peer remove <url>
peer sync

Untuk Phase 2, peer persistence boleh disimpan dalam file:

<datadir>/peers.json

Format peers.json sederhana:
{
"peers": [
"http://127.0.0.1:9331",
"http://127.0.0.1:9332"
]
}

Aturan:

* peer add harus menolak URL kosong.
* peer add harus menolak duplikat.
* peer add harus menerima URL [http://host:port](http://host:port).
* peer remove menghapus dari peers.json.
* peer list menampilkan peers dari flag/runtime dan peers.json jika relevan.
* peer sync melakukan sync manual sekali dari peers yang tersimpan.

Output peer list jika kosong:
peers: 0

Output jika ada:
peers: 2
http://127.0.0.1:9331
http://127.0.0.1:9332

==================================================
5. Endpoint P2P internal
========================

Tambahkan HTTP endpoint pada P2P server:

GET /p2p/health
Response:
{
"ok": true,
"network": "deskachain-local",
"height": 4,
"tip_hash": "..."
}

GET /p2p/status
Response:
{
"network": "deskachain-local",
"height": 4,
"tip_hash": "...",
"difficulty": 4,
"total_supply": "200 DKC",
"mempool_count": 0
}

GET /p2p/tip
Response:
{
"height": 4,
"hash": "..."
}

GET /p2p/block/{height}
Response:

* JSON block pada height tersebut.
* Jika tidak ada:
  {
  "error": "block not found"
  }

GET /p2p/blocks?from=<height>&limit=<n>
Response:
{
"blocks": [...]
}

POST /p2p/tx
Body:

* JSON transaction.

Perilaku:

* Validasi tx secara dasar.
* Jika valid dan belum ada, masukkan ke mempool.
* Jangan langsung ubah confirmed balance.
* Return:
  {
  "accepted": true,
  "txid": "..."
  }

Jika invalid:
{
"accepted": false,
"error": "..."
}

POST /p2p/block
Body:

* JSON block.

Perilaku:

* Validasi block terhadap tip lokal.
* Jika previous_hash cocok dengan tip lokal dan block valid, append.
* Hapus tx yang masuk block dari mempool.
* Return:
  {
  "accepted": true,
  "height": 5,
  "hash": "..."
  }

Jika block tidak cocok karena node tertinggal atau previous hash beda:
{
"accepted": false,
"error": "block does not extend local tip",
"local_height": 4,
"local_tip": "..."
}

==================================================
6. Sinkronisasi chain dari peer
===============================

Implementasi sync sederhana:

Function:
SyncFromPeer(peerURL)

Perilaku:

1. Ambil /p2p/status dari peer.
2. Bandingkan height peer dengan height lokal.
3. Jika peer height <= local height:

   * tidak perlu sync.
4. Jika peer height > local height:

   * ambil block dari localHeight+1 sampai peerHeight.
   * Bisa pakai /p2p/block/{height} satu per satu.
   * Validasi setiap block sebelum append.
   * Jika block gagal validasi, stop dan return error jelas.
5. Setelah sync selesai, chain validate lokal harus tetap pass.

Output manual peer sync:
sync started
local height: 0
peer: http://127.0.0.1:9331
peer height: 4
imported block height=1 hash=<hash>
imported block height=2 hash=<hash>
imported block height=3 hash=<hash>
imported block height=4 hash=<hash>
sync complete
new height: 4

Jika sudah sama:
sync complete
local chain already up to date

==================================================
7. Broadcast transaksi
======================

Saat command send berhasil membuat tx pending lokal:

* Broadcast tx ke semua peers yang diketahui.
* Jika broadcast gagal ke salah satu peer, jangan gagalkan send lokal.
* Log warning saja.

Tambahkan function:
BroadcastTx(tx)

Aturan:

* Peer yang menerima tx harus memasukkan tx ke mempool jika valid.
* Jangan duplikat tx di mempool.
* Tx yang sudah confirmed jangan dimasukkan lagi ke mempool.
* Jika peer belum punya chain cukup untuk validasi balance, boleh reject dengan error jelas:
  "sender balance insufficient or chain not synced"

==================================================
8. Broadcast block hasil mining
===============================

Saat mine berhasil menambahkan block lokal:

* Broadcast block ke semua peers.
* Peer yang menerima block:

  * Jika block extend tip lokal, append.
  * Jika peer tertinggal, block bisa ditolak dengan pesan "block does not extend local tip".
  * Sync loop berikutnya akan mengambil block yang kurang.
* Jangan bikin mining gagal hanya karena broadcast gagal.

Tambahkan function:
BroadcastBlock(block)

==================================================
9. Mempool setelah block diterima
=================================

Saat node menerima block dari peer:

* Validasi block.
* Append ke chain jika valid.
* Hapus tx dalam block dari mempool.
* Jika ada pending tx lokal yang jadi invalid karena balance/nonce berubah, boleh:

  * hapus langsung, atau
  * tetap simpan tapi mining berikutnya akan skip.
    Untuk Phase 2, lebih baik lakukan cleanup sederhana:
    RevalidateMempoolAgainstChain()

==================================================
10. Konflik chain dan fork handling Phase 2
===========================================

Untuk Phase 2, fork handling dibuat sederhana.

Aturan:

* Node hanya menerima block yang extend tip lokal.
* Jika peer punya height lebih tinggi tapi block pada localHeight+1 previous_hash tidak cocok dengan tip lokal:

  * sync gagal dengan error:
    "fork detected: manual reset or future reorg needed"
* Jangan implementasi reorg kompleks dulu.
* Tambahkan TODO jelas untuk Phase 2.5/3:

  * longest valid chain
  * fork choice rule
  * block locator
  * reorg support
  * orphan block pool

==================================================
11. RPC tambahan untuk node/peer
================================

Pada RPC server biasa, tambahkan endpoint:

GET /node/status

Response:
{
"datadir": "...",
"height": 4,
"tip_hash": "...",
"peers": 2,
"p2p_listen": ":9332",
"rpc_listen": ":8332"
}

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

POST /peers/sync
Body optional:
{
"peer": "http://127.0.0.1:9331"
}

Jika peer kosong, sync semua peers.

==================================================
12. Logging
===========

Tambahkan logging sederhana dengan log.Printf.

Log penting:

* node start
* peer added
* peer removed
* sync started
* sync complete
* imported block
* tx received
* tx broadcast success/fail
* block received
* block broadcast success/fail
* validation error

Jangan terlalu noisy.

==================================================
13. Test wajib
==============

Tambahkan test untuk package p2p jika memungkinkan.

Gunakan temporary datadir dan httptest/server lokal jika lebih mudah.

Test minimal:

1. Dua chain lokal genesis sama

* Init node A dan node B.
* Genesis hash harus sama.

2. Sync block dari node A ke node B

* Node A mining 3 block.
* Node B sync dari node A.
* Height B harus sama dengan A.
* Tip hash B harus sama dengan A.
* chain validate B pass.

3. Broadcast tx

* Node A punya balance.
* Node A buat tx ke address B.
* Broadcast tx ke node B.
* Mempool node B berisi tx tersebut.

4. Broadcast block

* Node A mining block yang berisi tx.
* Broadcast block ke node B.
* Node B append block.
* Mempool node B kosong.
* Balance penerima benar.
* Tip sama.

5. Peer store

* peer add menyimpan peers.json.
* peer add duplikat tidak menggandakan.
* peer remove menghapus.

6. Fork sederhana

* Jika node B punya tip berbeda, sync harus gagal dengan error fork detected.
* Tidak perlu reorg dulu.

==================================================
14. README update
=================

Update README.md:

Current phase:
Phase 2 — Local P2P Multi Node Sync

Tambahkan contoh menjalankan 2 node lokal di Windows PowerShell.

Persiapan:

go run ./node/cmd/deskachain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node2 dev reset --yes

Init dan buat wallet:

go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new

Simpan address miner node1.

Start node1 terminal pertama:

go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

Start node2 terminal kedua:

go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --peers http://127.0.0.1:9331

Mining dari terminal ketiga:

go run ./node/cmd/deskachain --datadir ./testdata/node1 mine --address <walletNode1> --blocks 3

Manual sync node2:

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync

Cek node2:

go run ./node/cmd/deskachain --datadir ./testdata/node2 chain info
go run ./node/cmd/deskachain --datadir ./testdata/node2 chain validate

Expected:

* node2 height sama dengan node1.
* tip hash sama.
* chain valid.

Tambahkan contoh broadcast tx:

* Buat wallet di node2.
* Kirim tx dari wallet node1 ke wallet node2.
* Cek mempool node2.
* Mine block di node1.
* Sync node2.
* Balance node2 bertambah.

Tambahkan catatan:

* Phase 2 masih local HTTP P2P.
* Belum ada fork reorg.
* Belum ada NAT traversal.
* Belum ada peer discovery publik.
* Belum ada ban score/rate limit P2P.
* P2P ini belum production safe.

==================================================
15. Expected final commands
===========================

Setelah patch selesai, command ini harus berhasil:

go mod tidy
go test ./...

Manual test lokal:

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
go run ./node/cmd/deskachain --datadir ./testdata/node1 mine --address <walletNode1> --blocks 3

Sync node2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer add http://127.0.0.1:9331
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync

Cek:
go run ./node/cmd/deskachain --datadir ./testdata/node1 chain info
go run ./node/cmd/deskachain --datadir ./testdata/node2 chain info
go run ./node/cmd/deskachain --datadir ./testdata/node2 chain validate

Expected:

* node1 dan node2 punya height sama.
* node1 dan node2 punya tip hash sama.
* node2 chain valid.

Jangan over-engineer.
Fokus Phase 2 hanya:

* peer store,
* P2P HTTP endpoint,
* sync block lokal,
* broadcast tx,
* broadcast block,
* multi datadir node lokal.

Fork reorg, peer discovery publik, mining pool, ban score, dan NAT traversal masuk phase berikutnya.
