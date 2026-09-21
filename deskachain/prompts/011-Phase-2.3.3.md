Kamu sedang bekerja pada project Go yang sudah ada: IndoChain.

Status saat ini:

* Phase 1 sampai Phase 2.3.2 sudah selesai.
* go test ./node/... pass.
* Remote mining via --rpc-url sudah tidak stuck.
* Mining job log sudah berjalan.
* Broadcast block berhasil.
* Broadcast tx berhasil.
* Node2 menerima block dari node1.
* Node2 menerima tx pending dari node1.
* Setelah block berikutnya diterima, mempool node2 kosong.
* Wallet node2 menerima 10 dIDR.
* node compare menunjukkan nodes in sync.
* chain validate pass di node1 dan node2.

Log sukses terbaru:

* mining started
* mining block started
* mining block found
* mining block committed
* block broadcast success
* tx broadcast success
* block received
* tx received
* chain valid

Masalah kecil yang ditemukan:

1. Setelah dev reset dan init ulang, node1 start tanpa flag --peers tetapi output:
   peers: 1
   Lalu node1 mencoba sync ke:
   http://127.0.0.1:9332
   Padahal pada command node1:
   go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331
   tidak ada --peers.

   Ini kemungkinan:

   * peers.json tidak ikut terhapus oleh dev reset,
   * peer store tersimpan di lokasi yang salah,
   * node menggunakan default datadir untuk peers,
   * peer metadata stale dari runtime sebelumnya,
   * atau peer bootstrap/load peer mencampur peers lama.

2. Saat mining block pertama setelah genesis, log:
   mining block started target_height=1 difficulty=0 pending_txs=0
   Tetapi output mined block:
   difficulty=4
   Artinya log difficulty saat start mining tidak konsisten dengan difficulty aktual block.

Nama patch:
IndoChain Phase 2.3.3 — Runtime Hygiene, Peer Store Reset, dan Difficulty Consistency

Tujuan utama:

1. Memastikan dev reset benar-benar membersihkan semua data node pada datadir.
2. Memastikan peer store selalu memakai datadir yang benar.
3. Menghilangkan stale peers setelah reset.
4. Membuat difficulty log konsisten dengan difficulty aktual block.
5. Merapikan startup peer bootstrap agar jelas.
6. Menambah command debug untuk inspect file runtime.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public peer discovery.
* Jangan rewrite total project.
* Patch incremental.
* Pertahankan command lama.
* Semua tetap harus lolos:
  go mod tidy
  go test ./...

==================================================

1. Audit semua file yang berada di datadir
   ==================================================

Pastikan semua file berikut benar-benar berada di selected datadir:

* chain database
* wallet store
* mempool store jika persistent
* peers.json
* node_id
* node.lock
* runtime metadata jika ada
* debug/cache file jika ada

Jangan ada komponen yang diam-diam memakai:
./data
saat user menjalankan:
--datadir ./testdata/node1

Tambahkan helper path terpusat jika belum ada:

internal/config/paths.go

Contoh:
DataDir()
ChainDBPath()
WalletPath()
PeersPath()
NodeIDPath()
LockPath()
MempoolPath()

Tujuan:
Semua package mengambil path dari satu sumber, bukan membuat path manual sendiri.

==================================================
2. Perbaiki dev reset agar benar-benar bersih
=============================================

Command:
dev reset --yes

Harus menghapus seluruh selected datadir, termasuk:

* peers.json
* node_id
* node.lock
* wallet file
* chain db
* mempool file
* runtime cache file

Setelah:
go run ./node/cmd/indochain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/node1 init
go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331

Expected:
peers: 0

Kecuali jika user memberi:
--peers ...

Tambahkan test:

* Buat datadir temp.
* Buat peers.json berisi peer.
* Jalankan dev reset.
* Init ulang.
* Load peer store.
* Peer count harus 0.

==================================================
3. Peer store harus strict per datadir
======================================

Audit PeerStore/PeerManager.

Pastikan:

* peers.json dibaca dari selected datadir.
* peers.json ditulis ke selected datadir.
* peer list remote membaca peer runtime dari node yang benar.
* peer list local membaca peers.json dari datadir yang dipilih.
* Tidak ada fallback ke ./data/peers.json jika --datadir diberikan.
* Tidak ada global package variable yang menyimpan path lama antar command/test.

Tambahkan log saat node start:
peer store: <absolute-or-clean-path-to-peers.json>

Contoh:
peer store: testdata\node1\peers.json

==================================================
4. Startup peer source harus jelas
==================================

Saat node start, tampilkan rincian peer source:

Jika tidak ada peer:
peers: 0

Jika peers dari file:
peers: 1
peer source: peers.json

Jika peers dari flag:
peers: 1
peer source: flag

Jika gabungan:
peers: 2
peer source: peers.json + flag

Jika peer duplicate dari file dan flag:

* Deduplicate.
* Output tetap peers unik.

Tambahkan command:
peer list --source

Output:
peers: 1
url=http://127.0.0.1:9332 source=peers.json status=active score=5

Untuk remote mode:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 peer list --source

RPC response /peers harus bisa menyertakan source jika mudah.

==================================================
5. Tambahkan command dev inspect
================================

Tambahkan command:

dev inspect

Contoh:
go run ./node/cmd/indochain --datadir ./testdata/node1 dev inspect

Output:
datadir: testdata\node1
chain db: exists
wallets: exists
peers.json: missing
node_id: exists
node.lock: missing
mempool: missing
height: 0
peers: 0
wallets: 1

Jika node sedang locked, command ini read-only dan boleh berjalan.
Jika tidak bisa baca file, tampilkan error jelas, jangan panic.

Remote mode optional:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 dev inspect

Jika remote mode terlalu aneh untuk nama dev inspect, boleh return:
remote mode not supported for dev inspect

==================================================
6. Difficulty consistency saat mining
=====================================

Masalah:
Log:
mining block started target_height=1 difficulty=0

Padahal block yang dimining memakai difficulty 4.

Perbaiki agar:

* Difficulty yang ditampilkan saat "mining block started" adalah difficulty aktual yang akan dipakai block.
* chain info difficulty juga konsisten:

  * genesis height 0 boleh difficulty 0.
  * next block difficulty harus initial difficulty 4.
* p2p status difficulty pada height 0 boleh menampilkan:
  difficulty: 0
  next_difficulty: 4
  atau tampilkan difficulty saat tip dan next difficulty secara terpisah.

Tambahkan ke /p2p/status dan /chain/info jika memungkinkan:
current difficulty: <tip difficulty>
next difficulty: <next block difficulty>

CLI chain info output:
difficulty: 4
atau:
current difficulty: 0
next difficulty: 4

Pilih format yang tidak merusak terlalu banyak command lama.
Rekomendasi:

* Tetap pertahankan field lama:
  difficulty: <next difficulty>
* Tambahkan:
  tip difficulty: <tip difficulty>
  next difficulty: <next difficulty>

Untuk height 0:
tip difficulty: 0
next difficulty: 4

Untuk height > 0:
tip difficulty: 4
next difficulty: 4

==================================================
7. Mining log difficulty harus pakai next difficulty
====================================================

Saat mulai mining block:

mining block started target_height=1 difficulty=4 pending_txs=0

Bukan difficulty 0.

Pastikan difficulty di:

* block header
* PoW validation
* mining output
* mining log
* chain info
* p2p status
  konsisten.

Tambahkan test:

* Setelah genesis, CalculateNextDifficulty menghasilkan 4.
* Log atau mining result block height 1 difficulty 4.
* chain info height 0 menampilkan next difficulty 4 jika testable.
* p2p status height 0 punya next difficulty 4 jika endpoint response ditest.

==================================================
8. Peer bootstrap tidak boleh spam connect refused
==================================================

Pada fresh start, jika peer dari flag/file belum hidup:
connectex: No connection could be made

Itu normal, tetapi jangan spam.

Pastikan:

* Error pertama boleh log.
* Error berikutnya throttle minimal 30-60 detik.
* Jika peer recovered, log:
  peer recovered url=<url>
* Jika peer dari stale file dan gagal terus, peer score turun bounded.
* Jika status bad, auto-sync skip bad peer kecuali include_bad.

==================================================
9. Tambahkan command peer clear
===============================

Tambahkan command:

peer clear --yes

Local:
go run ./node/cmd/indochain --datadir ./testdata/node1 peer clear --yes

Remote:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 peer clear --yes

Behavior:

* Menghapus semua peers dari peer store/runtime.
* Requires --yes.
* Jangan hapus chain/wallet.
* Output:
  peers cleared

Jika tanpa --yes:
refusing to clear peers without --yes

Tujuan:
Agar saat testing P2P, user bisa membersihkan peer stale tanpa reset full datadir.

RPC:
POST /peers/clear
Body:
{
"yes": true
}

==================================================
10. Tambahkan tests wajib
=========================

Tambahkan/update tests:

1. Dev reset clears peers:

* Buat peers.json.
* dev reset.
* init ulang.
* peer count 0.

2. Peer store path:

* Node1 temp datadir punya peer A.
* Node2 temp datadir punya peer B.
* Load peer store node1 tidak melihat peer B.
* Load peer store node2 tidak melihat peer A.

3. Node start fresh peers:

* Datadir fresh tanpa peers.json.
* Node config tanpa --peers.
* Runtime peer count 0.

4. Node start with flag peers:

* Datadir fresh.
* Start config dengan --peers peer1.
* Runtime peer count 1.
* Source = flag.

5. Node start with file peers:

* Datadir punya peers.json peer1.
* Start tanpa --peers.
* Runtime peer count 1.
* Source = peers.json.

6. Peer clear:

* Peer store berisi peer.
* peer clear --yes.
* Peer count 0.
* Chain/wallet tidak terhapus.

7. Difficulty:

* Genesis tip difficulty 0.
* Next difficulty setelah genesis 4.
* Mined block height 1 difficulty 4.
* Mining log/message memakai difficulty 4 jika mudah dites melalui return struct.

8. p2p status:

* Pada height 0 response punya tip difficulty 0 dan next difficulty 4.
* Setelah mining response tip difficulty 4 dan next difficulty 4.

9. Error throttle:

* Repeated peer connection refused tidak log terus setiap interval.
* Jika sulit test log langsung, test helper ShouldLogPeerError.

==================================================
11. README update
=================

Update README:

Current phase:
Phase 2.3.3 — Runtime Hygiene, Peer Store Reset, dan Difficulty Consistency

Tambahkan troubleshooting:

Jika setelah dev reset node masih menampilkan peers: 1:
go run ./node/cmd/indochain --datadir ./testdata/node1 dev inspect
go run ./node/cmd/indochain --datadir ./testdata/node1 peer list --source
go run ./node/cmd/indochain --datadir ./testdata/node1 peer clear --yes

Expected setelah reset:
peers: 0

Tambahkan penjelasan:

* peers.json harus berada di datadir masing-masing node.
* dev reset menghapus peers.json.
* peer clear hanya menghapus peer, tidak menghapus chain/wallet.
* difficulty pada genesis berbeda dari next difficulty.
* mining block pertama harus memakai next difficulty 4.

==================================================
12. Expected final commands
===========================

Setelah patch:

go work sync
go test ./node/...

Test peer reset:

go run ./node/cmd/indochain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/node1 init
go run ./node/cmd/indochain --datadir ./testdata/node1 dev inspect
go run ./node/cmd/indochain --datadir ./testdata/node1 peer list --source

Expected:
peers: 0
peers.json: missing atau peers: 0

Start node1 fresh:

go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

Expected:
peers: 0

Test difficulty:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain info

Expected pada height 0:
tip difficulty: 0
next difficulty: 4

Mine 1 block:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1

Expected log:
mining block started target_height=1 difficulty=4

Expected CLI:
mined block height=1 ... difficulty=4

Test peer clear:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 peer clear --yes
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 peer list

Expected:
peers: 0

Jangan over-engineer.
Fokus patch ini hanya:

* dev reset bersih,
* peer store strict per datadir,
* stale peer hilang,
* peer source jelas,
* difficulty log konsisten,
* peer clear,
* dev inspect.
