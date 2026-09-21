Kamu sedang bekerja pada project Go monorepo IndoChain.

Status:

* Phase 1 sampai Phase 2.8 valid.
* Phase 2.9 Standalone CPU Miner CLI sudah dipatch sebagian.
* go work sync berhasil.
* go test ./node/... pass.
* Template endpoint sudah bekerja:
  GET /miner/template?address=<IND_ADDR>
  mengembalikan block template height=1 difficulty=4.
* indominer berhasil mengambil template dan menemukan valid nonce/hash.
* Tetapi submit block timeout dan setelah itu node RPC ikut macet.

Log manual:

Template OK:
GET http://127.0.0.1:8401/miner/template?address=iND...
returns height=1 difficulty=4 tx_count=1

Miner:
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address iND... --threads 4 --once

Output:
new job height=1 difficulty=4 txs=1 prev=6e1b3fed63a01109
block found height=1 hash=00006b03... nonce=167659 thread=3
submit failed: Post "http://127.0.0.1:8401/miner/submit": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
template fetch failed: Get "http://127.0.0.1:8401/miner/template?address=iND...": context deadline exceeded

Masalah:

* /miner/submit tidak mengembalikan response.
* Setelah submit timeout, endpoint /miner/template juga ikut timeout.
* Kemungkinan ada deadlock atau lock terlalu lama di runtime submit path.
* Kemungkinan submit handler memegang chain/runtime lock lalu memanggil fungsi lain yang mencoba lock yang sama, atau melakukan broadcast/network call saat lock masih dipegang.

Nama patch:
IndoChain Phase 2.9.1 — Miner Submit Deadlock & Timeout Fix

Tujuan:
Memperbaiki endpoint /miner/submit agar:

* valid block submit tidak hang,
* accepted/rejected response selalu kembali cepat,
* node RPC tetap responsif setelah submit,
* template endpoint tetap bisa dipanggil setelah submit,
* standalone miner --once bisa selesai normal.

Aturan penting:

* Jangan ubah address format iND.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah coinbase maturity.
* Jangan implement GPU miner.
* Jangan implement mining pool.
* Jangan rewrite besar.
* Semua command lama harus tetap bekerja.
* go test ./node/... harus pass.

==================================================

1. Audit /miner/submit handler
   ==================================================

Cari handler RPC:

/miner/submit

Audit alur:

* decode request,
* validate submitted block,
* commit block,
* update runtime state,
* remove confirmed tx from mempool,
* broadcast block,
* return response.

Pastikan tidak ada deadlock.

Larangan penting:

* Jangan melakukan peer broadcast saat memegang chain lock/runtime lock/mempool lock.
* Jangan melakukan HTTP/network call saat memegang lock.
* Jangan memanggil function yang mencoba mengambil lock yang sama saat lock sudah dipegang.
* Jangan memanggil chain info/balance/stats builder di dalam lock jika builder itu juga lock storage/runtime.
* Jangan hold lock selama PoW validation jika tidak perlu.

==================================================
2. Gunakan pola commit seperti built-in mining
==============================================

Built-in command:

indochain mine --address <ADDR> --blocks N

sebelumnya sudah valid dan tidak hang.

Refactor submit block agar memakai commit path yang sama atau helper yang sama dengan built-in mining.

Target helper misalnya:

runtime.AcceptMinedBlock(ctx, block) (AcceptResult, error)

Behavior:

1. Ambil snapshot tip/current state dengan lock pendek.
2. Lepas lock.
3. Validasi PoW/difficulty/timestamp/tx.
4. Ambil lock lagi hanya saat commit final.
5. Recheck stale:

   * tip hash masih sama dengan block.PreviousHash.
   * expected height masih sama.
6. Commit block.
7. Remove confirmed tx dari mempool.
8. Update runtime tip/state.
9. Lepas semua lock.
10. Broadcast block ke peers di luar lock.
11. Return response.

Jika broadcast lambat/gagal:

* jangan membuat submit response hang.
* return accepted true dengan broadcast_success/broadcast_failed.
* Atau broadcast async dengan timeout pendek.

==================================================
3. Submit response timeout safety
=================================

/miner/submit harus selesai cepat untuk localnet.

Target:

* Valid block height 1 coinbase-only harus response < 2 detik.
* Invalid PoW harus response < 1 detik.
* Stale template harus response < 1 detik.

Tambahkan context-aware handling:

* Jika request context canceled, stop work jika memungkinkan.
* Jangan menunggu broadcast tanpa timeout.
* Jika ada peer offline, submit tetap response.

==================================================
4. Lock ordering
================

Tentukan lock ordering yang konsisten.

Jika ada lock:

* chain/runtime lock
* mempool lock
* peer lock
* wallet lock

Jangan ambil peer lock saat chain lock masih dipegang.

Rekomendasi:

1. chain/runtime lock
2. mempool lock
3. release all
4. peer broadcast separately

Atau:

* Satu runtime method internal yang mengatur commit secara atomik.
* Tidak boleh nested lock yang berpotensi reentrant.

Tambahkan komentar di code:
// Do not broadcast while holding runtime/chain locks.

==================================================
5. Instrumentasi log submit
===========================

Tambahkan log debug/info ringkas di node saat submit:

Saat request masuk:
miner submit received height=1 hash=... template=...

Setelah decode:
miner submit decoded height=1 txs=1

Saat validasi:
miner submit validating height=1 difficulty=4

Saat commit:
miner submit committed height=1 hash=...

Saat reject:
miner submit rejected reason="stale template"

Saat broadcast:
miner submit broadcast complete height=1 success=0 failed=0

Jangan terlalu noisy, tapi cukup untuk tahu macet di tahap mana.

==================================================
6. Fix /miner/template after failed submit
==========================================

Setelah submit invalid/stale/timeout, endpoint ini harus tetap responsif:

GET /miner/template?address=<IND_ADDR>

Jika submit gagal, tidak boleh meninggalkan lock terkunci.

Pastikan semua lock menggunakan defer Unlock dengan benar.

Jika submit panic, recover di HTTP handler jika pattern project sudah ada.

==================================================
7. Miner client timeout
=======================

indominer boleh punya client timeout, tapi jangan terlalu pendek.

Default:

* template fetch timeout: 10s
* submit timeout: 20s
* retry interval: 3s

Tetapi root fix harus di node submit, bukan hanya memperpanjang timeout.

Jika submit accepted tapi client timeout karena response lama, itu tetap bug node.

==================================================
8. Submit stale handling
========================

Jika block sudah valid PoW tapi tip sudah berubah:

* return JSON cepat:

{
"accepted": false,
"reason": "stale template",
"current_height": N,
"current_tip": "..."
}

Jangan hang.

Miner harus:

* log stale template,
* fetch template baru,
* lanjut.

==================================================
9. Tests wajib
==============

Tambahkan/update tests:

1. Submit valid block returns quickly:

* start RPC handler with httptest.
* fetch template at genesis.
* mine block in test helper.
* POST /miner/submit with client timeout 2s.
* expect accepted true.
* expect height 1.

2. Template still works after submit:

* after accepted submit, GET /miner/template again.
* expect height 2.
* no timeout.

3. Submit invalid PoW returns quickly:

* submit bad nonce/hash.
* expect accepted false or error.
* height unchanged.
* subsequent template still works.

4. Submit stale returns quickly:

* fetch template A.
* commit another block.
* submit A.
* expect stale template.
* subsequent template still works.

5. Submit does not broadcast under lock:

* If test infra can simulate slow peer/broadcast, add peer that delays.
* submit should still return or broadcast should timeout.
* No deadlock.

6. indominer --once integration:

* run miner against test RPC if integration test exists.
* miner exits after accepted block.
* no retry loop after accepted submit.

7. Existing built-in mine still works:

* indochain mine path unaffected.
* chain validate pass.

8. Existing tests still pass:

* address
* difficulty
* maturity
* mempool
* reorg
* p2p
* rpc
* cpuminer

==================================================
10. Manual expected commands
============================

After patch:

go work sync
go test ./node/...

Reset:

go run ./node/cmd/indochain --datadir ./testdata/miner dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/miner init
go run ./node/cmd/indochain --datadir ./testdata/miner wallet new

Start node:

go run ./node/cmd/indochain --datadir ./testdata/miner node start --rpc :8401 --p2p :9401 --advertise-p2p http://127.0.0.1:9401

Template:

Invoke-RestMethod "http://127.0.0.1:8401/miner/template?address=<IND_ADDR>"

Expected:
height: 1
difficulty: 4
tx_count: 1

Standalone miner once:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4 --once

Expected:
new job height=1 difficulty=4
block found height=1 hash=0000...
submit accepted height=1
miner exits cleanly

Check chain:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 balance <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 chain validate

Expected:
height: 1
total supply: 50 dIDR
confirmed balance: 50
mature balance: 0
immature balance: 50
spendable balance: 0
chain valid

Call template again:

Invoke-RestMethod "http://127.0.0.1:8401/miner/template?address=<IND_ADDR>"

Expected:
height: 2
no timeout

Jangan over-engineer.
Fokus patch ini hanya:

* fix /miner/submit hang/deadlock,
* release locks before broadcast,
* response cepat,
* template tetap responsif setelah submit,
* miner --once selesai normal.
