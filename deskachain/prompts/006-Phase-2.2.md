Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:

* Phase 1 MVP selesai.
* Phase 1.5 stabilization selesai.
* Phase 1.6 transfer dan mempool hardening selesai.
* Phase 2 local HTTP P2P multi node sync selesai.
* Phase 2.1 node runtime hardening dan remote CLI selesai.
* Remote mining via --rpc-url sudah berhasil.
* Node2 sudah berhasil auto-sync block height 4, 5, 6 dari node1.
* go test ./... harus tetap pass.

Nama patch:
DesKaChain Phase 2.2 — P2P Handshake, Network Guard, Header Sync, dan Fork Safety

Tujuan utama:
Memperkuat P2P sebelum masuk ke phase discovery publik atau mining pool. Node harus bisa:

* mengenali network yang sama,
* menolak peer dari network/genesis berbeda,
* melakukan handshake,
* sync memakai block header dulu,
* mendeteksi fork dengan lebih jelas,
* menyediakan debug command yang enak.

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

1. Tambahkan Network Config
   ==================================================

Tambahkan config jaringan di internal/config.

Field wajib:

* NetworkName
* NetworkID
* ChainID
* GenesisHash
* ProtocolVersion
* MinProtocolVersion

Default localnet:
NetworkName: "deskachain-local"
NetworkID: "dkc-local-1"
ChainID: 777001
ProtocolVersion: 1
MinProtocolVersion: 1

Command:
network info

Output:
network name: deskachain-local
network id: dkc-local-1
chain id: 777001
protocol version: 1
min protocol version: 1
genesis hash: <hash>

Tambahkan global flag:
--network <name>

Untuk Phase 2.2 cukup dukung:
localnet

Default:
localnet

Jika network tidak dikenal:
unknown network: <name>

==================================================
2. P2P Handshake
================

Tambahkan endpoint:

GET /p2p/handshake

Response:
{
"network_name": "deskachain-local",
"network_id": "dkc-local-1",
"chain_id": 777001,
"protocol_version": 1,
"min_protocol_version": 1,
"genesis_hash": "...",
"height": 6,
"tip_hash": "...",
"node_id": "...",
"p2p_listen": ":9331"
}

NodeID:

* Buat node ID lokal.
* Simpan di: <datadir>/node_id
* Format boleh hex random 32 byte.
* Node ID dibuat sekali, lalu dipakai ulang.
* Jangan pakai wallet address sebagai node ID.

Handshake validation:
Saat peer add atau sync:

* Ambil /p2p/handshake dari peer.
* Validasi:

  * network_id sama
  * chain_id sama
  * genesis_hash sama
  * protocol_version peer >= MinProtocolVersion lokal
  * protocol_version lokal >= min_protocol_version peer
* Jika gagal, peer harus ditolak.

Error contoh:
peer rejected: network id mismatch
peer rejected: chain id mismatch
peer rejected: genesis hash mismatch
peer rejected: incompatible protocol version

peer add harus melakukan handshake sebelum menyimpan peer.

==================================================
3. Peer metadata
================

Ubah peers.json dari list string sederhana menjadi metadata.

Format baru:
{
"peers": [
{
"url": "http://127.0.0.1:9331",
"node_id": "...",
"network_id": "dkc-local-1",
"chain_id": 777001,
"last_height": 6,
"last_tip_hash": "...",
"last_seen_at": "2026-06-17T04:16:55+07:00",
"status": "active"
}
]
}

Aturan migrasi:

* Jika peers.json lama berisi list string, load tetap bisa.
* Saat peer add/sync berhasil, simpan ulang ke format baru.
* Jangan crash kalau file peers.json rusak; return error jelas.

Command:
peer list

Output:
peers: 1
url=http://127.0.0.1:9331 node_id=<id> height=6 status=active last_seen=<time>

==================================================
4. Header endpoint
==================

Tambahkan tipe BlockHeader.

Isi minimal:

* height
* hash
* previous_hash
* timestamp
* difficulty
* merkle_root
* tx_count
* miner_address

Tambahkan endpoint:

GET /p2p/headers?from=<height>&limit=<n>

Response:
{
"headers": [
{
"height": 1,
"hash": "...",
"previous_hash": "...",
"timestamp": 123,
"difficulty": 4,
"merkle_root": "...",
"tx_count": 1,
"miner_address": "dkc1..."
}
]
}

Limit default:
100

Max limit:
500

Tujuan:
Sync tidak langsung mengambil full block semua. Ambil header dulu, cek continuity, baru ambil block yang dibutuhkan.

==================================================
5. Header-first sync
====================

Update SyncFromPeer agar:

1. Handshake peer.
2. Ambil status peer.
3. Jika peer height <= local height, update metadata peer lalu selesai.
4. Jika peer lebih tinggi:

   * Ambil header dari localHeight+1.
   * Validasi header:

     * height urut
     * previous_hash header pertama harus sama dengan local tip
     * setiap previous_hash berikutnya harus sama dengan hash header sebelumnya
     * difficulty masuk akal
     * hash tidak kosong
   * Jika header valid, ambil full block berdasarkan height.
   * Validasi full block sama seperti sebelumnya.
   * Append block satu per satu.
5. Setelah sync, jalankan chain validate ringan atau minimal validasi tip.
6. Update peer metadata.

Output manual:
sync started
peer: http://127.0.0.1:9331
local height: 3
peer height: 6
headers checked: 3
imported block height=4 hash=<hash>
imported block height=5 hash=<hash>
imported block height=6 hash=<hash>
sync complete
new height: 6

==================================================
6. Fork detection lebih jelas
=============================

Untuk Phase 2.2, belum perlu reorg kompleks.

Jika header pertama dari peer tidak extend local tip:

* Jangan import block.
* Return error:
  fork detected: peer does not extend local tip

Tambahkan detail:
local height: <height>
local tip: <hash>
peer next height: <height>
peer previous hash: <hash>

Command output:
sync failed: fork detected
local height: 6
local tip: ...
peer next height: 7
peer previous hash: ...

Tambahkan TODO:

* Phase 2.5: block locator
* Phase 2.5: common ancestor search
* Phase 2.5: longest valid chain
* Phase 2.5: safe reorg

==================================================
7. Peer score sederhana
=======================

Tambahkan field peer score di metadata.

Aturan sederhana:

* Default score: 0
* Handshake sukses: +1
* Sync sukses/import block: +2
* Peer up to date dan responsif: +1 maksimal per beberapa menit
* Request gagal: -1
* Handshake mismatch: -10
* Invalid block/header: -20
* Jika score <= -20, status menjadi "bad"

Command:
peer list

Tampilkan:
score=<score> status=<active|bad|unknown>

Untuk Phase 2.2:

* Jangan auto-ban permanen.
* Bad peer boleh tetap ada, tapi sync harus skip bad peer kecuali flag:
  --include-bad

==================================================
8. Tambahkan command peer check
===============================

Command:
peer check <url>

Perilaku:

* Lakukan handshake.
* Tampilkan hasil validasi.

Output sukses:
peer ok
url: http://127.0.0.1:9331
node id: <id>
network id: dkc-local-1
chain id: 777001
height: 6
tip hash: ...

Output gagal:
peer rejected: genesis hash mismatch

Command ini tidak wajib menyimpan peer.

==================================================
9. Tambahkan command peer status
================================

Command:
peer status

Perilaku:

* Cek semua peer di peers.json.
* Update last_seen, height, tip, status, score.
* Jangan import block.
* Hanya health/handshake check.

Output:
checked peers: 2
active: 1
bad: 1

Detail:
url=http://127.0.0.1:9331 status=active height=6 score=5
url=http://127.0.0.1:9999 status=bad error="connection refused" score=-1

==================================================
10. Node compare diperkuat
==========================

Update command:

node compare --peer <rpc-url>

Agar juga membandingkan:

* network_id
* chain_id
* genesis_hash
* protocol_version
* height
* tip_hash
* total_supply
* pending_tx_count

Output jika sama:
nodes in sync
network id: dkc-local-1
chain id: 777001
height: 6
tip hash: ...

Output jika beda:
nodes differ
network id: local=<id> peer=<id>
chain id: local=<id> peer=<id>
height: local=6 peer=5
tip: local=<hash> peer=<hash>

==================================================
11. RPC endpoint network/node debug
===================================

Tambahkan endpoint RPC:

GET /network/info

Response sama dengan command network info.

GET /node/id

Response:
{
"node_id": "..."
}

GET /node/compare?peer=<rpc-url>

Response:
{
"in_sync": true,
"local": {
"network_id": "...",
"chain_id": 777001,
"height": 6,
"tip_hash": "..."
},
"peer": {
"network_id": "...",
"chain_id": 777001,
"height": 6,
"tip_hash": "..."
}
}

==================================================
12. Kurangi risiko duplicate import
===================================

Pastikan saat sync:

* Jika block height sudah ada dan hash sama, skip.
* Jika block height sudah ada tapi hash beda, return fork detected.
* Jangan append block duplikat.
* Jangan mengubah total supply dua kali.
* chain info setelah sync ulang tetap sama.

Test manual:
peer sync
peer sync
peer sync

Expected:

* height tidak bertambah jika sudah up to date.
* total supply tidak berubah.
* tip tetap sama.

==================================================
13. Tests wajib
===============

Tambahkan/update tests:

1. Network info:

* localnet config valid.
* unknown network ditolak.

2. Node ID:

* node_id dibuat.
* node_id persistent setelah restart.

3. Handshake:

* handshake sukses jika network/genesis sama.
* handshake gagal jika network_id beda.
* handshake gagal jika genesis_hash beda.
* handshake gagal jika protocol incompatible.

4. Peer add:

* peer add melakukan handshake.
* peer add menyimpan metadata.
* peer add tidak menggandakan peer.
* peer add menolak peer invalid.

5. Header endpoint:

* /p2p/headers mengembalikan header sesuai height.
* limit max diterapkan.

6. Header-first sync:

* nodeB sync dari nodeA memakai header.
* height dan tip sama setelah sync.
* sync ulang tidak duplicate import.

7. Fork detection:

* Jika header previous_hash tidak sama dengan local tip, sync gagal dengan fork detected.

8. Peer score:

* handshake sukses menaikkan score.
* request gagal menurunkan score.
* invalid peer menjadi status bad.

9. Node compare:

* node compare in_sync true jika height/tip/network sama.
* node compare false jika height beda.

==================================================
14. README update
=================

Update README.md:

Current phase:
Phase 2.2 — P2P Handshake, Network Guard, Header Sync, dan Fork Safety

Tambahkan penjelasan:

* Node sekarang punya node_id.
* Peer add melakukan handshake.
* Peer beda network/genesis ditolak.
* Sync sekarang header-first.
* Fork belum otomatis reorg; hanya dideteksi dan ditolak.
* Bad peer belum diban permanen.

Tambahkan command contoh:

go run ./node/cmd/deskachain network info

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node status

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer check http://127.0.0.1:9331

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer add http://127.0.0.1:9331

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer status

go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332

Tambahkan test sync ulang:
peer sync
peer sync
chain info

Expected:

* Height tetap.
* Total supply tetap.
* Tip hash tetap.

==================================================
15. Expected final commands
===========================

Setelah patch selesai:

go mod tidy
go test ./...

Setup bersih:

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
go run ./node/cmd/deskachain network info
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer check http://127.0.0.1:9331
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer sync
go run ./node/cmd/deskachain --datadir ./testdata/node2 peer status
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain validate

Expected:

* peer check sukses.
* peer sync memakai handshake dan header-first sync.
* sync kedua tidak menambah duplicate block.
* node compare menampilkan nodes in sync.
* chain validate pass di node1 dan node2.
* peer status menampilkan peer active.
* network/genesis mismatch ditolak.

Jangan over-engineer.
Fokus Phase 2.2 pada:

* network identity,
* node identity,
* handshake,
* peer metadata,
* header-first sync,
* fork detection,
* peer score sederhana,
* debug command.
