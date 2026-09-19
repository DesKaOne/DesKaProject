Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 2.3 sudah selesai.
* go test ./... pass.
* Remote CLI via --rpc-url sudah berjalan.
* node1 dan node2 bisa peer connect dua arah.
* Broadcast tx berhasil:
  node1 send tx ke wallet node2.
  node2 mempool list menampilkan pending tx count: 1.
* Broadcast block berhasil:
  node1 mine block berisi tx.
  node2 menerima block.
  node2 mempool kosong.
  walletB balance = 10 IDR.
  node compare menunjukkan nodes in sync.
  chain validate pass di node1 dan node2.

Masalah yang ditemukan:
Walaupun broadcast tx/block berhasil, log P2P auto-sync sering timeout pada endpoint /p2p/status:

sync failed peer=http://127.0.0.1:9332 error=Get "http://127.0.0.1:9332/p2p/status": context deadline exceeded (Client.Timeout exceeded while awaiting headers)

Dan sebaliknya:

sync failed peer=http://127.0.0.1:9331 error=Get "http://127.0.0.1:9331/p2p/status": context deadline exceeded (Client.Timeout exceeded while awaiting headers)

Anehnya:

* POST /p2p/block sukses.
* POST /p2p/tx sukses.
* Node compare via RPC sukses.
* Chain valid.

Kesimpulan awal:
Kemungkinan ada lock contention/deadlock kecil atau handler /p2p/status melakukan operasi terlalu berat / menunggu lock yang sedang dipegang sync loop.

Nama patch:
DesKaChain Phase 2.3.1 — P2P Status Timeout & Runtime Concurrency Fix

Tujuan utama:

1. Membuat endpoint /p2p/status, /p2p/tip, /p2p/health, dan /p2p/handshake selalu cepat.
2. Menghindari sync loop menahan lock saat melakukan HTTP request keluar.
3. Menghindari deadlock antar node ketika dua node saling sync bersamaan.
4. Membuat auto-sync tidak spam timeout jika peer sebenarnya sehat.
5. Menambahkan debug endpoint/command untuk latency P2P.

Aturan penting:

* Jangan tambah smart contract.
* Jangan tambah EVM.
* Jangan tambah mining pool.
* Jangan tambah public discovery dulu.
* Jangan rewrite total project.
* Patch incremental.
* Pertahankan command lama.
* Semua tetap harus lolos:
  go mod tidy
  go test ./...

==================================================

1. Audit lock/mutex di runtime node, peer manager, chain, mempool
   ==================================================

Cari semua penggunaan mutex/lock pada:

* node runtime
* p2p server
* p2p sync loop
* peer manager
* chain storage
* mempool
* RPC handler

Pastikan aturan berikut diterapkan:

Aturan emas:

* Jangan pernah menahan mutex saat melakukan HTTP request keluar.
* Jangan pernah menahan mutex saat melakukan operasi mining panjang.
* Jangan pernah menahan mutex saat melakukan full chain validation berat.
* Jangan pernah menahan mutex peer store saat memanggil peer remote.
* Jangan memanggil callback yang bisa network I/O saat lock masih dipegang.

Pola yang benar:

1. Ambil snapshot data yang dibutuhkan di bawah lock.
2. Lepas lock.
3. Lakukan HTTP request keluar.
4. Ambil lock lagi hanya untuk update state/metadata.

Contoh buruk yang harus dihindari:
peerManager.mu.Lock()
defer peerManager.mu.Unlock()
http.Get(peer.URL + "/p2p/status")

Contoh benar:
peers := peerManager.Snapshot()
for _, peer := range peers {
status := client.GetStatus(peer.URL)
peerManager.Update(peer.URL, status)
}

==================================================
2. Buat endpoint status/tip/health/handshake super ringan
=========================================================

Endpoint ini harus cepat dan tidak melakukan replay chain penuh:

GET /p2p/health
GET /p2p/status
GET /p2p/tip
GET /p2p/handshake

Pastikan endpoint tersebut:

* Tidak menjalankan chain validate.
* Tidak replay ledger penuh jika tidak perlu.
* Tidak melakukan peer sync.
* Tidak melakukan HTTP request keluar.
* Tidak menunggu operasi mining selesai kecuali perlu membaca tip atomik.
* Tidak menahan lock lama.
* Response ideal di bawah 50ms pada localnet kecil.

Untuk /p2p/status cukup ambil:

* network id
* chain id
* height
* tip hash
* difficulty
* total supply cached atau cepat
* mempool count

Jika total supply saat ini dihitung dengan replay chain dan membuat lambat, tambahkan cache runtime:

* height
* tip hash
* total supply
* difficulty
* mempool count

Cache harus diupdate setelah append block/mine/receive block.

==================================================
3. Tambahkan runtime chain state cache
======================================

Tambahkan struktur runtime state cache, misalnya:

type ChainRuntimeState struct {
Height uint64
TipHash string
Difficulty uint32
TotalSupply uint64
MempoolCount int
UpdatedAt time.Time
}

Aturan:

* Diupdate saat node start dari storage.
* Diupdate setelah block lokal ditambang.
* Diupdate setelah block dari peer diterima.
* Diupdate setelah sync import block.
* Diupdate setelah mempool berubah.
* Dibaca oleh /p2p/status, /p2p/tip, /node/status, /chain/info jika remote runtime.

Gunakan RWMutex kecil atau atomic snapshot.
Jangan jadikan cache sumber kebenaran final untuk validasi block.
Storage/chain tetap sumber kebenaran.

==================================================
4. Perbaiki sync loop agar tidak saling deadlock
================================================

Auto-sync loop saat ini bisa membuat dua node saling request /p2p/status pada waktu yang sama.

Perbaiki dengan:

* Gunakan per-peer sync lock, bukan global lock panjang.
* Jika sync ke peer sedang berjalan, skip interval berikutnya untuk peer itu.
* Tambahkan jitter kecil ke interval auto-sync agar node tidak selalu sync bersamaan.
  Contoh:
  interval 5s + random 0-1000ms
* Tambahkan timeout pendek untuk status:
  status timeout 2s
* Tambahkan timeout lebih panjang untuk block fetch:
  block fetch timeout 5s atau 10s
* Jika peer status timeout, jangan langsung spam log setiap interval.

Output log:

* Jika peer timeout pertama kali, log warning.
* Jika timeout berulang, log maksimal 1 kali per 60 detik per peer.
* Jika peer pulih, log:
  peer recovered url=<url>

==================================================
5. Tambahkan endpoint debug latency
===================================

Tambahkan RPC endpoint:

GET /debug/p2p

Response:
{
"node_id": "...",
"height": 4,
"tip_hash": "...",
"peers": [
{
"url": "http://127.0.0.1:9332",
"status": "active",
"score": 21,
"last_seen_at": "...",
"last_error": "",
"last_latency_ms": 12,
"last_status_check_at": "..."
}
]
}

Tambahkan command:

p2p debug

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p debug

Output:
node id: <id>
height: 4
tip hash: <hash>
peers: 1
url=http://127.0.0.1:9332 status=active score=21 latency=12ms last_error=""

==================================================
6. Tambahkan command p2p ping
=============================

Tambahkan command:

p2p ping <p2p-url>

Local atau remote mode:

* Jika remote mode, RPC node yang melakukan ping.
* Jika local mode, CLI langsung ping P2P URL.

Endpoint yang diping:

* /p2p/health
* /p2p/status
* /p2p/handshake

Output:
p2p ping: http://127.0.0.1:9332
health: ok latency=3ms
status: ok latency=4ms height=4 tip=<hash>
handshake: ok latency=5ms node_id=<id>
result: ok

Jika gagal:
result: failed
status error: context deadline exceeded

RPC endpoint untuk remote ping:

POST /debug/p2p/ping

Body:
{
"url": "http://127.0.0.1:9332"
}

Response:
{
"ok": true,
"url": "...",
"health_latency_ms": 3,
"status_latency_ms": 4,
"handshake_latency_ms": 5,
"height": 4,
"tip_hash": "..."
}

==================================================
7. Tambahkan health check manual untuk endpoint P2P
===================================================

Pastikan command berikut bisa digunakan saat node berjalan:

curl http://127.0.0.1:9331/p2p/health
curl http://127.0.0.1:9331/p2p/status
curl http://127.0.0.1:9331/p2p/tip
curl http://127.0.0.1:9331/p2p/handshake

Di Windows PowerShell, dokumentasikan:

Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9332/p2p/status

Expected:

* Response cepat.
* Tidak timeout.
* Tidak tergantung sync loop.

==================================================
8. Broadcast tidak boleh memblokir status endpoint
==================================================

Saat mining broadcast 3 block:

* Broadcast boleh jalan sequential atau concurrent.
* Tapi broadcast tidak boleh menahan lock yang membuat /p2p/status timeout.
* Jika broadcast menggunakan peer manager, ambil snapshot peers dulu, lalu lakukan request.
* Update score/metadata setelah response selesai.

Pastikan:

* /p2p/status tetap bisa dijawab saat broadcast sedang berjalan.
* /p2p/tx tetap bisa diterima saat sync loop berjalan.
* /p2p/block tetap bisa diterima saat sync loop berjalan.

==================================================
9. Peer score update tidak boleh membuat lock panjang
=====================================================

Peer score update harus cepat.
Jangan simpan peers.json terlalu sering.

Tambahkan debounce persist peer metadata:

* Update metadata in-memory boleh sering.
* Persist peers.json maksimal 1 kali per beberapa detik, atau saat perubahan penting:

  * peer add/remove/connect
  * status berubah active/bad
  * node shutdown
* Untuk patch ini, implementasi sederhana cukup:

  * Jangan write file peers.json pada setiap up-to-date sync.
  * Simpan hanya jika height/tip berubah, status berubah, atau score melewati threshold tertentu.

Tujuan:
Mengurangi disk I/O dan potensi lock.

==================================================
10. Node shutdown cleanup
=========================

Pastikan saat node berhenti normal:

* lock file dihapus.
* peer metadata terakhir disimpan.
* log:
  node stopped

Tangani Ctrl+C / SIGINT / SIGTERM.

Di Windows, Ctrl+C harus mencoba cleanup lock file.

==================================================
11. Tests wajib
===============

Tambahkan/update tests:

1. Status endpoint cepat:

* Start test node.
* Call /p2p/status.
* Pastikan response tidak memanggil chain validate berat.
* Test dengan timeout kecil.

2. No lock during outbound HTTP:

* Peer manager Snapshot tidak menahan lock saat client request.
* Bisa dites dengan fake peer server yang delay.
* Selama sync ke fake peer delay, peer list/status lokal tetap bisa dibaca.

3. Concurrent status during broadcast:

* Saat broadcast block ke fake peer lambat, /p2p/status tetap respond.

4. Sync loop skip if already running:

* Jika sync peer masih berjalan, interval berikutnya skip dan tidak membuat goroutine numpuk.

5. Log throttle:

* Timeout berulang tidak menyebabkan log setiap 5 detik.
* Jika terlalu sulit dites, cukup implement helper dan unit test helper ShouldLogPeerError.

6. Runtime state cache:

* Setelah mining, cache height/tip berubah.
* Setelah receive block, cache height/tip berubah.
* /p2p/status memakai cache.

7. Peer metadata persist:

* Up-to-date sync tidak menulis peers.json terus-menerus.
* Score tetap bounded 100/-100.

8. Shutdown:

* Lock file cleanup function bisa dipanggil.
* Tidak perlu test OS signal penuh jika sulit.

==================================================
12. README update
=================

Update README:

Current phase:
Phase 2.3.1 — P2P Status Timeout & Runtime Concurrency Fix

Tambahkan troubleshooting:

Jika muncul:
context deadline exceeded saat GET /p2p/status

Cek:
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9332/p2p/status

Gunakan:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p ping http://127.0.0.1:9332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p debug

Expected:

* status endpoint cepat.
* p2p ping result ok.
* p2p debug menampilkan latency dan last_error.

Tambahkan catatan:

* Sync loop tidak boleh menahan lock saat request keluar.
* Endpoint status/health/handshake harus ringan.
* Broadcast tx/block tidak boleh memblokir status endpoint.

==================================================
13. Expected final commands
===========================

Setelah patch:

go mod tidy
go test ./...

Setup 2 node seperti biasa.

Terminal 1:
go run ./node/cmd/deskachain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

Terminal 2:
go run ./node/cmd/deskachain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

Terminal 3:
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9332/p2p/status

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p ping http://127.0.0.1:9332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 p2p ping http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 p2p debug
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 p2p debug

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8332 chain validate

Expected:

* Tidak ada spam context deadline exceeded untuk /p2p/status.
* Invoke-RestMethod /p2p/status cepat.
* p2p ping result ok.
* p2p debug menampilkan latency normal.
* Broadcast tx tetap jalan.
* Broadcast block tetap jalan.
* Node compare tetap nodes in sync.
* Chain validate pass di kedua node.

Jangan over-engineer.
Fokus patch ini hanya pada:

* menghilangkan timeout /p2p/status,
* memperbaiki lock/concurrency runtime,
* status endpoint ringan,
* sync loop tidak deadlock,
* debug latency P2P.
