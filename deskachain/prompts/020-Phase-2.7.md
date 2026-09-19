Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur project:

* node/

  * cmd/deskachain/
  * internal/
  * go.mod
  * go.sum
* root repo punya:

  * go.work
  * README.md
  * Roadmap.md
  * docs/
  * prompts/
  * testdata/
  * data/

Status saat ini:

* Phase 1 sampai Phase 2.6.6.1 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... harus pass.
* Address migration sudah selesai:

  * wallet baru menghasilkan address `IDR...`
  * format address final: `IDR` + Base58Check
  * private key tetap raw 32-byte hex
  * secp256k1 sudah dipakai jika Phase 2.6.6 menerapkannya
  * remote wallet new sudah valid
* Fitur yang sudah ada:

  * local blockchain,
  * account ledger,
  * wallet CLI,
  * send transaction,
  * nonce,
  * mempool,
  * mining,
  * RPC node,
  * P2P node,
  * block broadcast,
  * tx broadcast,
  * fork detection,
  * common ancestor,
  * safe reorg preview/apply,
  * reorg mempool recovery,
  * runtime stats cleanup,
  * IDR Base58Check address,
  * network profile foundation,
  * protocol metadata,
  * remote wallet command UX fix.

Nama patch:
DesKaChain Phase 2.7 — Difficulty Adjustment & Realistic Cumulative Work

Tujuan:
Mengganti difficulty fixed menjadi difficulty adjustment sederhana, aman, dan cocok untuk localnet/testnet.

Saat ini mining masih memakai difficulty tetap:
difficulty: 4
tip difficulty: 4
next difficulty: 4

Phase 2.7 harus menambahkan:

* target block time,
* retarget window,
* next difficulty calculation,
* block difficulty validation,
* timestamp sanity validation,
* realistic cumulative work,
* chain difficulty command,
* P2P/reorg compatibility dengan variable difficulty.

Aturan penting:

* Jangan ubah address format `IDR...`.
* Jangan rollback Base58Check.
* Jangan ubah private key format.
* Jangan implement coinbase maturity.
* Jangan implement standalone miner.
* Jangan implement staking.
* Jangan implement PoS.
* Jangan implement bandwidth mining.
* Jangan implement explorer.
* Jangan rewrite besar.
* Patch incremental.
* Semua command lama harus tetap bekerja.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...

==================================================

1. Tambahkan consensus difficulty params ke network profile
   ==================================================

Gunakan network profile yang sudah dibuat pada Phase 2.6.6.

Tambahkan DifficultyParams ke NetworkProfile atau config consensus.

Minimal:

type DifficultyParams struct {
InitialDifficulty uint32
MinDifficulty uint32
MaxDifficulty uint32
TargetBlockTimeSeconds int64
RetargetWindow uint64
MaxFutureDriftSeconds int64
}

Default localnet:

InitialDifficulty: 4
MinDifficulty: 1
MaxDifficulty: 8
TargetBlockTimeSeconds: 10
RetargetWindow: 10
MaxFutureDriftSeconds: 900

Default testnet sementara:

InitialDifficulty: 4
MinDifficulty: 1
MaxDifficulty: 12
TargetBlockTimeSeconds: 30
RetargetWindow: 30
MaxFutureDriftSeconds: 900

Default mainnet placeholder:

InitialDifficulty: 6
MinDifficulty: 1
MaxDifficulty: 24
TargetBlockTimeSeconds: 60
RetargetWindow: 60
MaxFutureDriftSeconds: 900

Catatan:

* Mainnet params masih placeholder.
* Jangan pakai mainnet untuk runtime default.
* Default CLI tetap localnet kecuali user memilih network lain.
* Jangan hardcode difficulty `4` tersebar di banyak file lagi.
* Semua mining/validation/chain info harus mengambil params dari network profile.

==================================================
2. Implement CalculateNextDifficulty
====================================

Tambahkan fungsi reusable:

CalculateNextDifficulty(storeOrBlocks, params) (uint32, error)

Atau:

CalculateNextDifficulty(blocks []types.Block, params DifficultyParams) uint32

Rules:

1. Jika chain hanya genesis / height 0:
   next difficulty = InitialDifficulty.

2. Jika current height < RetargetWindow:
   next difficulty = tip difficulty jika tip non-genesis.
   Jika tip difficulty 0/genesis:
   next difficulty = InitialDifficulty.

3. Jika belum di retarget boundary:
   next difficulty = tip difficulty.

4. Jika next block berada di retarget boundary:

   * Ambil window terakhir sebanyak RetargetWindow block non-genesis.
   * lastBlock = tip.
   * firstBlock = block pada height tipHeight - RetargetWindow + 1.
   * actualTimespan = lastBlock.Timestamp - firstBlock.Timestamp.
   * expectedTimespan = TargetBlockTimeSeconds * (RetargetWindow - 1).

5. Adjustment sederhana Phase 2.7:

   * Jika actualTimespan <= 0:
     newDifficulty = oldDifficulty + 1
   * Jika actualTimespan < expectedTimespan / 2:
     newDifficulty = oldDifficulty + 1
   * Jika actualTimespan > expectedTimespan * 2:
     newDifficulty = oldDifficulty - 1
   * Selain itu:
     newDifficulty = oldDifficulty

6. Clamp:

   * newDifficulty >= MinDifficulty
   * newDifficulty <= MaxDifficulty

7. Jangan naik/turun lebih dari 1 per retarget window pada Phase ini.

Contoh:

* target 10s
* window 10
* expected approx 90s
* actual 9s => naik +1
* actual 900s => turun -1
* actual 80s => tetap

==================================================
3. Mining harus memakai next difficulty
=======================================

Saat miner membuat block baru:

* Hitung next difficulty dari chain canonical saat ini.
* Set block.Difficulty = next difficulty.
* PoW harus mencari hash sesuai difficulty tersebut.
* Log mining harus menampilkan difficulty yang sama dengan block.

Contoh log:

mining block started target_height=10 difficulty=4 pending_txs=0
mining block started target_height=11 difficulty=5 pending_txs=0

CLI output:

mined block height=11 ... difficulty=5

Jangan lagi memakai constant difficulty langsung di miner.

==================================================
4. Chain validation harus cek expected difficulty
=================================================

Update chain validate.

Untuk setiap block non-genesis:

1. Hitung expected difficulty berdasarkan chain sampai parent block.
2. Bandingkan:
   block.Difficulty == expectedDifficulty
3. Validasi PoW:
   block hash memenuhi block.Difficulty.
4. Jika mismatch:
   chain invalid: invalid difficulty at height X expected Y got Z

Pastikan:

* Existing localnet chain dengan height kecil dan difficulty 4 tetap valid.
* Genesis difficulty boleh 0.
* Block 1 expected difficulty 4.
* Block sebelum retarget window tetap difficulty 4.
* Block setelah retarget boundary bisa berubah.

==================================================
5. Timestamp validation ringan
==============================

Tambahkan timestamp sanity validation:

Rules:

* Block timestamp harus >= parent timestamp.
* Block timestamp tidak boleh lebih jauh dari now + MaxFutureDriftSeconds.
* Untuk localnet, future drift default 15 menit.

Errors:

* invalid block timestamp: before parent
* invalid block timestamp: too far in future

Pastikan validasi ini dipakai pada:

* chain validate,
* block receive P2P,
* sync import,
* reorg branch validation,
* mining local/RPC.

Jangan terlalu ketat agar localnet tidak mudah rusak.

==================================================
6. Realistic cumulative work
============================

Saat ini cumulative work kemungkinan masih:

work = difficulty

Ubah menjadi lebih realistis sesuai model leading hex zero:

work = 16 ^ difficulty

Rules:

* genesis atau difficulty 0:
  work = 1
* difficulty 1:
  work = 16
* difficulty 2:
  work = 256
* difficulty 4:
  work = 65536
* difficulty 8:
  work = 4294967296

Implement helper:

CalculateBlockWork(difficulty uint32) uint64

Untuk menghindari overflow:

* Jika difficulty terlalu tinggi untuk uint64, clamp ke math.MaxUint64.
* Localnet max difficulty 8, jadi aman.
* Testnet/mainnet placeholder harus tetap aman.

Pastikan:

* chain info cumulative work berubah sesuai model baru.
* reorg preview/apply memakai cumulative work baru.
* node compare local work/peer work memakai cumulative work baru.
* peer chain dengan height sama tapi difficulty/work lebih besar bisa menang reorg jika branch valid.

Catatan:

* Ini breaking change untuk angka cumulative work saja, bukan block format.
* Reorg decision menjadi lebih benar untuk variable difficulty.

==================================================
7. Tambahkan command chain difficulty
=====================================

Tambahkan command:

chain difficulty

Local:
go run ./node/cmd/deskachain --datadir ./testdata/node1 chain difficulty

Remote:
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain difficulty

Output:

network: localnet
chain id: 777001
height: 12
tip difficulty: 5
next difficulty: 5
target block time: 10s
retarget window: 10
min difficulty: 1
max difficulty: 8
blocks until retarget: 8
cumulative work: 123456

RPC:
GET /chain/difficulty

Response JSON minimal:
{
"network": "localnet",
"chain_id": 777001,
"height": 12,
"tip_difficulty": 5,
"next_difficulty": 5,
"target_block_time_seconds": 10,
"retarget_window": 10,
"min_difficulty": 1,
"max_difficulty": 8,
"blocks_until_retarget": 8,
"cumulative_work": 123456
}

==================================================
8. Update chain info difficulty fields
======================================

Update `chain info` local dan remote.

Tambahkan field:

* network
* chain id
* protocol version jika sudah ada
* target block time
* retarget window
* min difficulty
* max difficulty
* tip difficulty
* next difficulty
* blocks until retarget
* cumulative work

Jangan hapus field lama.

Contoh:

network: localnet
chain id: 777001
height: 3
tip hash: ...
difficulty: 4
tip difficulty: 4
next difficulty: 4
target block time: 10s
retarget window: 10
blocks until retarget: 7
total supply: 150 IDR
cumulative work: 196609

Catatan:

* Jika masih ada field lama `difficulty`, boleh isi sama dengan next difficulty atau tip difficulty, tapi docs harus jelas.
* Lebih baik:
  difficulty = next difficulty
  tip difficulty = tip
  next difficulty = next

==================================================
9. Tambahkan test helper mine with timestamp
============================================

Agar tests tidak perlu menunggu waktu asli, tambahkan helper internal:

MineBlockWithTimestamp(...)
BuildTestChainWithIntervals(intervalSeconds []int64)
BuildTestChainWithConstantInterval(blocks int, intervalSeconds int64)

Atau helper di test package chain/mining.

Tujuan:

* Simulasi block terlalu cepat.
* Simulasi block terlalu lambat.
* Simulasi block normal.
* Tidak perlu sleep real time.

Opsional dev command:

dev mine-timed --address <addr> --blocks <n> --interval-seconds <n>

Jika terlalu besar, cukup internal test helper dulu.

==================================================
10. P2P block receive harus validasi difficulty
===============================================

Saat menerima block via P2P:

* Validasi expected difficulty.
* Validasi PoW sesuai block difficulty.
* Validasi timestamp.
* Jika invalid:
  reject block: invalid difficulty expected X got Y
* Jangan import block invalid.
* Jangan update height/tip.
* Peer score boleh turun ringan.

P2P sync juga harus sama:

* Jika peer mengirim branch dengan invalid difficulty, sync gagal aman.

==================================================
11. Reorg compatibility
=======================

Reorg preview/apply harus:

* Menghitung cumulative work dengan model baru.
* Memvalidasi difficulty setiap block peer branch.
* Menolak branch jika ada invalid difficulty.
* Menolak branch jika timestamp invalid.
* Tidak mengubah chain lokal jika branch invalid.
* Tetap membutuhkan --yes untuk apply.

Reorg preview output:
local work: <new work>
peer work: <new work>
allowed: true/false

Jika peer work lebih besar karena difficulty lebih tinggi walau height sama, preview boleh allowed true jika branch valid dan reorg depth within limit.

==================================================
12. Network profile integration
===============================

Pastikan difficulty params diambil dari selected network profile.

Command:
--network localnet
--network testnet
--network mainnet

harus memengaruhi:

* network id,
* chain id,
* address validation version,
* difficulty params,
* max future drift.

Jika beberapa area masih hardcode localnet, rapikan minimal pada:

* mining,
* validation,
* chain info,
* p2p handshake,
* reorg.

Jangan refactor besar jika berisiko, tapi jangan tambahkan hardcode baru.

==================================================
13. Tests wajib
===============

Tambahkan/update tests:

1. Initial difficulty:

* Genesis height 0.
* next difficulty == InitialDifficulty 4.

2. Before retarget:

* Mine height 1 sampai 9 dengan normal interval.
* Difficulty tetap 4.

3. At retarget with fast blocks:

* Simulasi window 10 block interval 1 detik.
* Next difficulty naik dari 4 ke 5.
* Tidak naik lebih dari +1.

4. At retarget with slow blocks:

* Simulasi window 10 block interval 60 detik.
* Next difficulty turun dari 4 ke 3.
* Tidak turun di bawah min difficulty.

5. At retarget with normal blocks:

* Simulasi interval sekitar 10 detik.
* Difficulty tetap.

6. Difficulty clamp min:

* Jika difficulty 1 dan blocks lambat, tetap 1.

7. Difficulty clamp max:

* Jika difficulty max dan blocks cepat, tetap max.

8. Chain validate catches invalid difficulty:

* Buat block dengan difficulty salah.
* chain validate gagal:
  expected X got Y

9. Chain validate catches invalid PoW:

* Difficulty benar tapi hash tidak memenuhi.
* chain validate gagal.

10. Timestamp before parent invalid:

* block timestamp < parent timestamp.
* validasi gagal.

11. Timestamp too far future invalid:

* block timestamp > now + drift.
* validasi gagal.

12. CalculateBlockWork:

* diff 0 => 1
* diff 1 => 16
* diff 4 => 65536
* diff 8 => 4294967296

13. Cumulative work:

* genesis + 3 blocks difficulty 4:
  work = 1 + 3*65536 = 196609
* chain info cumulative work matches.

14. Reorg work decision:

* fork A height 3 diff 4.
* fork B height 3 but higher cumulative work due higher difficulty if test helper can build it.
* preview allowed true only if peer work > local work and branch valid.

15. Reorg rejects invalid difficulty branch:

* peer branch has invalid difficulty.
* preview/apply fails.
* local chain unchanged.

16. P2P rejects invalid difficulty block:

* receive block with wrong difficulty.
* reject.
* height unchanged.

17. chain difficulty RPC:

* /chain/difficulty returns fields.
* CLI remote `chain difficulty` prints expected fields.

18. chain info fields:

* local and remote chain info include target block time, retarget window, min/max difficulty, tip/next difficulty, cumulative work.

19. Existing tests still pass:

* send IDR -> IDR.
* mempool stats.
* reorg tx scenarios.
* wallet IDR address generation.

==================================================
14. README / docs update
========================

Update README.md and/or docs:

Current phase:
Phase 2.7 — Difficulty Adjustment & Realistic Cumulative Work

Tambahkan penjelasan:

* DesKaChain tidak lagi bergantung pada fixed difficulty.
* Localnet memakai:
  target block time: 10s
  retarget window: 10 blocks
  initial difficulty: 4
  min difficulty: 1
  max difficulty: 8
* Difficulty naik jika block terlalu cepat.
* Difficulty turun jika block terlalu lambat.
* Adjustment Phase 2.7 konservatif: max naik/turun 1 per retarget.
* Cumulative work memakai `16^difficulty`.
* Reorg memilih branch dengan cumulative work lebih besar, bukan hanya height lebih tinggi.

Tambahkan command:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain difficulty

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8331 chain info

==================================================
15. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/deskachain --datadir ./testdata/diff dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/diff init
go run ./node/cmd/deskachain --datadir ./testdata/diff wallet new

Expected:
IDR...

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/diff node start --rpc :8361 --p2p :9361 --advertise-p2p http://127.0.0.1:9361

Remote difficulty:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 chain difficulty

Expected at height 0:
network: localnet
chain id: 777001
height: 0
tip difficulty: 0
next difficulty: 4
target block time: 10s
retarget window: 10
min difficulty: 1
max difficulty: 8

Mine:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 mine --address <IDR_ADDR> --blocks 3

Then:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 chain difficulty
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 chain validate

Expected:
chain valid
height: 3
tip difficulty: 4
next difficulty: 4
cumulative work: 196609

Mine 12 blocks:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 mine --address <IDR_ADDR> --blocks 12

Then:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 chain difficulty
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8361 chain validate

Expected:
chain valid
difficulty fields shown
difficulty may adjust at retarget boundary based on timestamps

Jangan over-engineer.
Fokus Phase 2.7 hanya:

* consensus difficulty params,
* next difficulty calculation,
* mining uses next difficulty,
* validation checks expected difficulty,
* timestamp sanity,
* realistic cumulative work,
* chain difficulty command,
* P2P/reorg compatibility with variable difficulty.
