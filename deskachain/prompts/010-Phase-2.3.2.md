Kamu sedang bekerja pada project Go yang sudah ada: IndoChain.

Status saat ini:

* Phase 1 sampai Phase 2.3.1 sudah berjalan.
* go test ./... pass.
* /p2p/status di node1 dan node2 sudah cepat.
* node1 dan node2 bisa peer connect dua arah.
* peer list menunjukkan kedua node active.
* Masalah baru muncul saat remote mining:

Command:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3

Command tersebut stuck/hang tanpa output.

Kondisi sebelum stuck:

* node1 RPC :8331 aktif.
* node1 P2P :9331 aktif.
* node2 RPC :8332 aktif.
* node2 P2P :9332 aktif.
* peer connect sukses.
* /p2p/status kedua node cepat.
* peer list node1 dan node2 active.

Kemungkinan penyebab:

* RPC /mine handler menahan lock runtime terlalu lama.
* Mining/append block/cache update/broadcast memerlukan lock yang sama.
* Broadcast block dilakukan saat lock masih dipegang.
* Peer manager lock tertahan saat broadcast.
* Mempool/chain lock terjadi nested lock.
* HTTP response /mine menunggu semua proses termasuk broadcast peer yang bisa macet.
* Tidak ada timeout/context pada mining job atau broadcast.

Nama patch:
IndoChain Phase 2.3.2 — RPC Mine Deadlock Fix & Mining Job Runtime

Tujuan utama:

1. Membuat remote mine tidak hang.
2. Memastikan mining tidak menahan global runtime lock.
3. Memastikan broadcast block dilakukan setelah block committed dan di luar lock berat.
4. Menambahkan mining job state agar status mining bisa dipantau.
5. Menambahkan timeout dan progress log untuk mining.
6. Menjaga semua fitur lama tetap jalan.

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

1. Audit dan perbaiki RPC /mine handler
   ==================================================

Cari handler RPC:

POST /mine

Pastikan handler ini TIDAK:

* memegang global node mutex selama seluruh mining,
* memegang peer manager lock saat mining,
* memegang mempool lock saat proof-of-work loop,
* memegang chain/storage lock saat broadcast,
* melakukan broadcast saat lock chain/runtime masih dipegang,
* menjalankan chain validate penuh saat mining masih di lock.

Pola yang benar:

1. Validasi request.
2. Ambil snapshot kecil:

   * miner address
   * blocks count
   * peers snapshot
   * pending tx snapshot jika perlu
3. Lepas semua lock runtime panjang.
4. Mine block.
5. Saat block sudah ketemu:

   * ambil lock chain/storage sebentar
   * validasi block terhadap current tip
   * append block
   * update runtime state cache
   * hapus tx dari mempool
   * lepas lock
6. Broadcast block ke peers di luar lock.
7. Update peer score/metadata di luar lock panjang.
8. Lanjut block berikutnya.

Jangan pernah melakukan:
runtime.mu.Lock()
defer runtime.mu.Unlock()
MineNBlocks(...)
BroadcastBlock(...)

==================================================
2. Tambahkan MiningService
==========================

Tambahkan package/komponen jika belum ada:

internal/mining/
service.go
job.go

Atau jika lebih cocok di package chain/node runtime, tetap jaga dependency bersih.

Struktur minimal:

type MiningService struct {
mu sync.Mutex
running bool
currentJob *MiningJob
}

type MiningJob struct {
ID string
MinerAddress string
RequestedBlocks int
MinedBlocks int
StartedAt time.Time
UpdatedAt time.Time
Status string
LastHeight uint64
LastHash string
Error string
}

Status:

* idle
* running
* completed
* failed
* cancelled

Aturan:

* Hanya 1 mining job aktif per node untuk Phase 2.3.2.
* Jika /mine dipanggil saat mining masih running, return error jelas:
  mining already running
* Jangan membuat goroutine numpuk.
* Mining job harus bisa dibaca oleh /mine/status.

==================================================
3. Mode /mine tetap synchronous, tapi dengan lock aman
======================================================

Untuk kompatibilitas, POST /mine tetap synchronous:

* Request masuk.
* Mining dijalankan.
* Response dikembalikan setelah selesai.
* Tapi job status diperbarui selama proses berjalan.
* Endpoint /mine/status bisa dipanggil dari terminal lain saat mining berjalan.

Tambahkan RPC endpoint:

GET /mine/status

Response jika idle:
{
"status": "idle"
}

Response jika running:
{
"status": "running",
"job_id": "...",
"miner_address": "...",
"requested_blocks": 3,
"mined_blocks": 1,
"last_height": 1,
"last_hash": "...",
"started_at": "...",
"updated_at": "..."
}

Response completed:
{
"status": "completed",
"job_id": "...",
"requested_blocks": 3,
"mined_blocks": 3,
"last_height": 3,
"last_hash": "..."
}

Tambahkan CLI:

mine status

Remote:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine status

Output:
mining status: running
job id: <id>
mined blocks: 1/3
last height: 1
last hash: <hash>

==================================================
4. Tambahkan timeout dan context pada mining RPC
================================================

Tambahkan flag untuk mine command:

--timeout <duration>

Default:
0 atau no timeout untuk local CLI lama.
Untuk remote CLI, default timeout HTTP harus cukup besar:
5m

Contoh:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 3 --timeout 5m

RPC server harus memakai request context:

* Jika client disconnect/cancel, mining boleh:

  * lanjut sampai block sekarang selesai lalu stop, atau
  * cancel jika PoW loop mendukung context.
* Implementasi terbaik:
  PoW loop mengecek ctx.Done() setiap beberapa ribu nonce.

Tambahkan error:
mining cancelled

==================================================
5. Tambahkan progress log di node runtime saat mining
=====================================================

Saat /mine mulai:

mining started job=<id> miner=<addr> blocks=3

Saat mulai block:
mining block started target_height=1 difficulty=4 pending_txs=0

Saat block ditemukan:
mining block found height=1 hash=<hash> nonce=<nonce> elapsed=<duration>

Saat block committed:
mining block committed height=1 hash=<hash>

Saat broadcast selesai:
mining block broadcast complete height=1 success=<n> failed=<n>

Saat selesai:
mining completed job=<id> mined_blocks=3 new_height=3 elapsed=<duration>

Jika gagal:
mining failed job=<id> error=<err>

Dengan log ini, kalau stuck lagi, titik macet kelihatan.

==================================================
6. Perbaiki broadcast block agar tidak bisa menggantung mining
==============================================================

Broadcast block tidak boleh membuat /mine menggantung terlalu lama.

Aturan:

* Ambil snapshot peers active.
* Broadcast ke peers dengan timeout per peer:
  default 3s
* Jika peer timeout/gagal, catat failed, lanjut.
* Jangan retry panjang di dalam /mine.
* Jangan menunggu auto-sync peer.
* Jangan menahan lock peer store saat request keluar.

Jika ada 1 peer lambat:

* Mining tetap selesai.
* Response /mine tetap keluar.
* Broadcast summary mencatat failed.

Response /mine:
{
"mined_blocks": 3,
"new_height": 3,
"miner_balance": "150 dIDR",
"blocks": [...],
"broadcast": {
"peers": 1,
"success": 0,
"failed": 1,
"results": [
{
"peer": "http://127.0.0.1:9332",
"ok": false,
"message": "timeout"
}
]
}
}

==================================================
7. PoW loop harus support context cancel
========================================

Cari fungsi mining/ProofOfWork yang mencari nonce.

Ubah agar menerima context.Context:

MineBlock(ctx context.Context, ...)

Dalam loop nonce:

* Setiap 1000 atau 10000 iterasi cek:
  select {
  case <-ctx.Done():
  return error ctx.Err()
  default:
  }

Jangan cek ctx setiap iterasi kalau membuat lambat.

Jika context cancel:
return mining cancelled

==================================================
8. Tambahkan safety untuk difficulty lokal
==========================================

Karena localnet, difficulty 4 masih oke.
Tapi tambahkan guard:

* Jika difficulty terlalu tinggi untuk localnet, mine bisa terlihat stuck.
* Tambahkan flag:
  --max-nonce <n optional>

Untuk testing:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 1 --max-nonce 5000000

Jika nonce melewati max:
mining failed: max nonce reached

Default max-nonce:
0 berarti unlimited.

Tambahkan juga log setiap N nonce jika verbose mining aktif:

Flag:
--mine-verbose

Default false.

Jika true:
mining progress height=1 nonce=100000 hash_prefix=abcd

==================================================
9. Hindari nested lock chain/mempool
====================================

Pastikan urutan lock konsisten jika memang harus memakai banyak lock.

Aturan lock order:

1. chain/storage lock
2. mempool lock
3. state cache lock
4. peer metadata lock

Tapi lebih baik:

* Ambil snapshot mempool tanpa chain lock.
* Validasi tx saat commit block dengan lock minimal.
* Update cache setelah commit.
* Broadcast setelah semua lock dilepas.

Dokumentasikan di komentar:
Do not hold chain or mempool locks while performing network I/O.

==================================================
10. Tambahkan debug command untuk mine stuck
============================================

Tambahkan command:

debug locks

Remote:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 debug locks

Response minimal:
{
"mining_running": true,
"mining_job_id": "...",
"height": 0,
"tip_hash": "...",
"mempool_count": 0,
"peer_count": 1
}

Tidak perlu introspeksi mutex beneran kalau sulit.
Tujuannya hanya runtime snapshot.

Tambahkan RPC:
GET /debug/locks

==================================================
11. Tests wajib
===============

Tambahkan/update tests:

1. Remote mine tidak deadlock tanpa peer:

* Start node tanpa peer.
* POST /mine blocks=1.
* Response sukses dalam timeout test.
* Height naik.

2. Remote mine tidak deadlock dengan slow peer:

* Buat fake peer yang delay saat /p2p/block.
* POST /mine blocks=1.
* Response tetap sukses/gagal broadcast timeout, tapi mining tidak hang.
* Height lokal naik.

3. /mine/status saat mining running:

* Jalankan mining dengan difficulty test atau fake slow miner.
* Panggil /mine/status.
* Status running terbaca.

4. Mining already running:

* Jika job running, request mine kedua return error mining already running.

5. PoW context cancel:

* Context cancelled menyebabkan mining return error.

6. Broadcast outside lock:

* Saat broadcast ke fake peer lambat, /p2p/status lokal tetap respons cepat.

7. Peer manager snapshot:

* Snapshot peers tidak menahan lock saat request keluar.

8. Max nonce:

* max nonce kecil menyebabkan mining failed: max nonce reached.

==================================================
12. README update
=================

Update README:

Current phase:
Phase 2.3.2 — RPC Mine Deadlock Fix & Mining Job Runtime

Tambahkan troubleshooting:

Jika command ini terlihat stuck:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 3

Cek dari terminal lain:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine status
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 debug locks
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p debug
Invoke-RestMethod http://127.0.0.1:9331/p2p/status

Tambahkan contoh:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 3 --timeout 5m

Tambahkan catatan:

* Mining via RPC sekarang memakai mining job.
* Broadcast peer timeout tidak boleh menggantung mining.
* Hanya satu mining job aktif per node untuk Phase 2.3.2.
* PoW loop support context cancel.

==================================================
13. Expected final commands
===========================

Setelah patch:

go mod tidy
go test ./...

Setup 2 node seperti biasa:

go run ./node/cmd/indochain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/node2 dev reset --yes

go run ./node/cmd/indochain --datadir ./testdata/node1 init
go run ./node/cmd/indochain --datadir ./testdata/node2 init

go run ./node/cmd/indochain --datadir ./testdata/node1 wallet new
go run ./node/cmd/indochain --datadir ./testdata/node2 wallet new

Terminal 1:
go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

Terminal 2:
go run ./node/cmd/indochain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

Terminal 3:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3 --timeout 5m

Jika command mining terlihat lama, terminal lain:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine status
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 debug locks

Expected:

* Mining tidak hang.
* Log node menampilkan:
  mining started
  mining block started
  mining block found
  mining block committed
  mining block broadcast complete
  mining completed
* CLI mine mengeluarkan hasil mined block.
* Jika peer lambat, mining tetap selesai dengan broadcast failed/timeout.
* Node1 height naik.
* Node2 menerima block atau bisa sync setelahnya.
* node compare tetap bisa in sync.
* chain validate pass di node1 dan node2.

Jangan over-engineer.
Fokus patch ini hanya:

* memperbaiki remote mine stuck,
* mining job status,
* lock aman,
* broadcast timeout,
* PoW context cancel,
* debug ketika mining macet.
