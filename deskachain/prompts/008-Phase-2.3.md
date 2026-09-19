Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 2.2.1 sudah selesai.
* go test ./... pass.
* Remote CLI via --rpc-url sudah berjalan.
* peer check remote berhasil.
* peer sync remote berhasil.
* peer status remote berhasil.
* node compare menunjukkan node1 dan node2 in sync.
* Node2 sudah bisa sync block dari node1.
* Datadir lock sudah bekerja benar.
* Remote peer command sudah bekerja.

Log terbaru:
peer ok
network id: dkc-local-1
chain id: 777001
height: 6
peer sync: local chain already up to date
peer status: active
score=787
node compare: nodes in sync

Masalah kecil yang ditemukan:
Peer score terlalu cepat naik sampai 787. Ini kemungkinan karena auto-sync/status memberi score +1 terus-menerus tanpa batas/cooldown.

Nama patch:
DesKaChain Phase 2.3 — Broadcast TX/Block End-to-End dan Peer Score Tuning

Tujuan utama:

1. Membuktikan broadcast transaksi antar node benar-benar jalan.
2. Membuktikan broadcast block hasil mining antar node benar-benar jalan.
3. Merapikan peer score agar bounded dan tidak naik liar.
4. Menambah command/debug agar alur P2P mudah diuji.
5. Menjaga node tetap sinkron setelah send + mine lewat RPC.

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

==================================================

1. Rapikan peer score agar bounded
   ==================================================

Saat ini peer score bisa naik sangat tinggi, contoh:
score=787

Ubah sistem peer score:

Konstanta:
PeerScoreMax = 100
PeerScoreMin = -100
PeerScoreBadThreshold = -20
PeerScoreExcellentThreshold = 50

Aturan:

* Semua perubahan score harus lewat helper:
  AdjustPeerScore(peer, delta, reason)
* Score tidak boleh lebih dari +100.
* Score tidak boleh kurang dari -100.
* Simpan reason terakhir jika mudah:
  last_score_reason

Aturan scoring baru:

* Handshake sukses pertama kali: +5
* Peer add sukses: +5
* Sync sukses dan import block > 0: +10
* Sync sukses tapi up to date: +1, maksimal 1 kali per 60 detik per peer
* Peer status/health sukses: +1, maksimal 1 kali per 60 detik per peer
* Broadcast tx sukses: +2
* Broadcast block sukses: +3
* Request timeout/gagal koneksi: -5
* Handshake mismatch: -20
* Invalid header/block/tx dari peer: -30
* Fork detected dari peer: -10

Status:

* score <= -20 => status "bad"
* score > -20 dan peer responsif => status "active"
* peer belum dicek => status "unknown"

Jangan auto-delete bad peer.
Bad peer hanya di-skip saat sync/broadcast kecuali include_bad=true.

==================================================
2. Peer status output lebih informatif
======================================

Update command:

peer status

Local dan remote output:

checked peers: 1
active: 1
bad: 0
unknown: 0
url=http://127.0.0.1:9331 status=active height=6 score=42 last_seen=<time> reason="sync up to date"

Jika score sebelumnya besar dari versi lama, normalisasi saat load:

* Jika score > 100, ubah menjadi 100.
* Jika score < -100, ubah menjadi -100.

==================================================
3. Tambahkan command p2p test broadcast
=======================================

Tambahkan command debug:

p2p test-broadcast --from <addressA> --to <addressB> --amount 10 --miner <addressA>

Mode local:
go run ./node/cmd/deskachain --datadir ./testdata/node1 p2p test-broadcast ...

Mode remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p test-broadcast ...

Untuk Phase 2.3, remote mode lebih penting.

Perilaku command:

1. Cek node status.
2. Cek peers.
3. Kirim transaksi dari addressA ke addressB.
4. Pastikan tx masuk mempool node utama.
5. Tunggu sebentar atau panggil peer status.
6. Cek mempool peer jika peer RPC tersedia atau jika ada flag --peer-rpc.
7. Mine 1 block oleh miner.
8. Tunggu sebentar agar block broadcast.
9. Compare node utama dengan peer jika --peer-rpc diberikan.
10. Tampilkan ringkasan.

Flags:
--from <address>
--to <address>
--amount <amount>
--miner <address>
--peer-rpc <url optional>
--wait <duration optional default 2s>

Output sukses:
p2p broadcast test started
tx created: <txid>
local mempool: 1
tx broadcast: success=1 failed=0
mined block: height=7 hash=<hash> txs=2
block broadcast: success=1 failed=0
peer balance: 10 DKC
compare: nodes in sync
p2p broadcast test passed

Jika --peer-rpc tidak diberikan:

* Tetap jalankan send + mine.
* Tampilkan broadcast summary dari response.
* Jangan gagal hanya karena tidak bisa cek peer RPC.

==================================================
4. Perkuat broadcast tx
=======================

Pastikan saat RPC /send dipanggil pada node1:

* Tx dibuat.
* Tx masuk mempool node1.
* Tx dibroadcast ke semua peers active.
* Peer node2 menerima POST /p2p/tx.
* Node2 memvalidasi tx:

  * signature valid
  * sender balance cukup berdasarkan chain node2
  * nonce sesuai confirmed + pending
  * txid belum ada di mempool
  * txid belum confirmed di chain
* Jika valid, node2 memasukkan ke mempool.
* Response broadcast summary harus jelas.

RPC /send response:
{
"status": "pending",
"id": "...",
"from": "...",
"to": "...",
"amount": "10 DKC",
"fee": "0 DKC",
"nonce": 1,
"broadcast": {
"peers": 1,
"success": 1,
"failed": 0,
"results": [
{
"peer": "http://127.0.0.1:9332",
"ok": true,
"message": "accepted"
}
]
}
}

Catatan penting:

* Jika node1 hanya tahu peer node2 dari peers.json/runtime, broadcast harus menggunakan P2P URL node2.
* Jika node2 hanya tahu node1 tetapi node1 tidak tahu node2, broadcast dari node1 ke node2 tidak akan terjadi.
* Tambahkan dokumentasi bahwa untuk broadcast dua arah, kedua node harus saling add peer atau node start harus menyimpan inbound peer jika memungkinkan.

==================================================
5. Tambahkan inbound peer learning sederhana
============================================

Saat node menerima request P2P dari peer:

* Jika header request memiliki informasi peer URL/node ID, boleh simpan sebagai known peer.
* Jika tidak ada, jangan memaksa.

Tambahkan optional header untuk semua P2P request dari client:
X-DKC-Node-ID
X-DKC-P2P-URL
X-DKC-Network-ID

Saat menerima request valid:

* Jika X-DKC-P2P-URL ada dan valid, tambahkan/update peer metadata.
* Jangan tambahkan URL kosong.
* Jangan tambahkan self URL.
* Jangan gagal jika header tidak ada.

Tujuan:
Agar node1 bisa belajar node2 ketika node2 connect/sync, sehingga broadcast node1 ke node2 lebih mudah.

Tetap pertahankan peer add manual sebagai cara utama.

==================================================
6. Perkuat broadcast block
==========================

Saat RPC /mine dipanggil pada node1:

* Block baru ditambahkan ke chain node1.
* Block dibroadcast ke semua peers active.
* Peer node2 menerima POST /p2p/block.
* Jika block extend tip node2, append.
* Jika node2 tertinggal, tolak dengan:
  "block does not extend local tip"
  lalu sync loop berikutnya boleh sync.
* Jika block valid dan diterima:

  * hapus tx dalam block dari mempool node2
  * revalidate mempool
  * update height/tip
* Broadcast summary harus jelas.

RPC /mine response:
{
"mined_blocks": 1,
"new_height": 7,
"miner_balance": "350 DKC",
"blocks": [
{
"height": 7,
"hash": "...",
"txs": 2,
"reward": "50 DKC",
"difficulty": 4,
"nonce": 123
}
],
"broadcast": {
"peers": 1,
"success": 1,
"failed": 0,
"results": [
{
"peer": "http://127.0.0.1:9332",
"ok": true,
"message": "accepted"
}
]
}
}

==================================================
7. Tambahkan RPC endpoint mempool detail jika belum lengkap
===========================================================

Pastikan endpoint:

GET /mempool

Response:
{
"pending_tx_count": 1,
"transactions": [
{
"id": "...",
"from": "...",
"to": "...",
"amount": "10 DKC",
"fee": "0 DKC",
"nonce": 1
}
]
}

Remote CLI:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list

Harus bisa menampilkan tx hasil broadcast dari node1.

==================================================
8. Tambahkan command peer connect dua arah
==========================================

Tambahkan command:

peer connect <p2p-url>

Perilaku:

* Sama seperti peer add, tetapi setelah add sukses, coba panggil endpoint peer untuk mengenalkan diri.
* Jika memungkinkan, peer remote juga menyimpan node lokal sebagai peer.

Endpoint baru P2P:

POST /p2p/peer

Body:
{
"url": "http://127.0.0.1:9332",
"node_id": "...",
"network_id": "dkc-local-1",
"chain_id": 777001
}

Response:
{
"accepted": true
}

Aturan:

* Validasi handshake/network.
* Jangan tambahkan self.
* Jangan duplikat.

Contoh:
Node2 connect ke node1:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331

Expected:

* Node2 menyimpan node1.
* Node1 juga mengenal node2 jika node2 punya p2p listen URL di runtime.

Ini membantu broadcast block/tx dari node1 ke node2.

==================================================
9. Tambahkan self P2P URL di node runtime
=========================================

Saat node start:

* Simpan/ketahui own P2P advertised URL.

Flag baru:
--advertise-p2p <url>

Default:

* Jika p2p listen ":9331", advertise boleh menjadi:
  http://127.0.0.1:9331
  untuk localnet.
* Untuk future public node, user bisa isi:
  --advertise-p2p http://192.168.1.5:9331

Node status harus menampilkan:
p2p listen: :9331
p2p advertise: http://127.0.0.1:9331

Handshake response juga harus menampilkan:
p2p_advertise: "http://127.0.0.1:9331"

==================================================
10. Manual test flow wajib di README
====================================

Update README dengan alur final Phase 2.3.

Setup bersih:

go run ./node/cmd/deskachain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node2 dev reset --yes

go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node2 init

go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new
go run ./node/cmd/deskachain --datadir ./testdata/node2 wallet new

Terminal 1:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

Terminal 2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

Terminal 3:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 peer list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer list

Mine di node1:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3

Sync node2:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

Send dari node1 ke wallet node2:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10

Cek mempool node2:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list

Expected:
pending tx count: 1

Mine node1 lagi:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1

Cek node2:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 balance <walletB>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332

Expected:

* Mempool node2 kosong.
* Balance walletB = 10 DKC.
* Nodes in sync.

==================================================
11. Tests wajib
===============

Tambahkan/update tests:

1. Peer score:

* Score tidak bisa melebihi 100.
* Score tidak bisa kurang dari -100.
* Up to date sync tidak menaikkan score terus tanpa cooldown.
* Status bad jika score <= -20.

2. Advertise P2P:

* node start config punya p2p advertise URL.
* handshake menampilkan p2p_advertise.

3. Peer connect:

* peer connect menambahkan peer lokal.
* peer connect mencoba mengenalkan diri ke remote.
* peer connect tidak membuat duplikat.

4. Broadcast tx:

* Node1 dan node2 saling mengenal.
* Node1 send tx.
* Node2 menerima tx di mempool.
* Duplicate tx tidak masuk dua kali.

5. Broadcast block:

* Node1 mining block berisi tx.
* Node2 menerima block.
* Node2 mempool bersih.
* Balance walletB benar.
* Tip node1 dan node2 sama.

6. Remote mempool:

* remote mempool list menampilkan tx pending dari RPC.

7. RPC /mine broadcast summary:

* Response /mine punya field broadcast.
* success/failed sesuai peer.

8. RPC /send broadcast summary:

* Response /send punya field broadcast.
* success/failed sesuai peer.

==================================================
12. Expected final commands
===========================

Setelah patch:

go mod tidy
go test ./...

Manual test:

go run ./node/cmd/deskachain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/node2 dev reset --yes

go run ./node/cmd/deskachain --datadir ./testdata/node1 init
go run ./node/cmd/deskachain --datadir ./testdata/node2 init

go run ./node/cmd/deskachain --datadir ./testdata/node1 wallet new
go run ./node/cmd/deskachain --datadir ./testdata/node2 wallet new

Terminal 1:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

Terminal 2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

Terminal 3:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 peer list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer list

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer sync

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 balance <walletB>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain validate

Expected:

* peer score maksimal 100, tidak naik liar.
* node1 dan node2 saling mengenal peer.
* send dari node1 broadcast ke node2.
* mempool node2 berisi tx pending.
* mine node1 broadcast block ke node2.
* mempool node2 kosong setelah block diterima.
* balance walletB = 10 DKC.
* nodes in sync.
* chain validate pass di kedua node.

Jangan over-engineer.
Fokus Phase 2.3 pada:

* peer score tuning,
* advertise p2p URL,
* peer connect dua arah,
* broadcast tx end-to-end,
* broadcast block end-to-end,
* command debug P2P yang enak dipakai.
