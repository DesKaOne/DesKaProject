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

* Phase 1 sampai Phase 2.7 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... pass.
* Address migration sudah selesai:

  * wallet baru menghasilkan address `IDR...`
  * format address final: `IDR` + Base58Check
  * private key raw 32-byte hex
  * remote wallet new valid
* Difficulty adjustment sudah valid:

  * localnet target block time 10s
  * retarget window 10
  * initial difficulty 4
  * mining memakai next difficulty
  * chain validate cek expected difficulty
  * cumulative work memakai `16^difficulty`
  * chain difficulty command valid
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
  * dynamic difficulty,
  * realistic cumulative work.

Nama patch:
DesKaChain Phase 2.8 — Coinbase Maturity & Mature Balance

Tujuan:
Menambahkan aturan coinbase maturity supaya reward mining tidak langsung bisa dibelanjakan sebelum mencapai jumlah konfirmasi tertentu.

Fitur utama:

* consensus param coinbase maturity,
* mature balance,
* immature balance,
* spendable balance,
* pending outgoing,
* validation mencegah spend immature coinbase,
* wallet/balance output lebih jelas,
* reorg compatible dengan mature/immature balance,
* mining tetap menerima reward, tapi reward baru belum spendable.

Aturan penting:

* Jangan ubah address format `IDR...`.
* Jangan rollback Base58Check.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah cumulative work formula.
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

1. Tambahkan coinbase maturity ke network profile
   ==================================================

Tambahkan consensus param ke NetworkProfile atau Difficulty/Consensus params.

Minimal:

type ConsensusParams struct {
CoinbaseMaturity uint64
...
}

Default:

localnet:
CoinbaseMaturity: 10

testnet:
CoinbaseMaturity: 100

mainnet placeholder:
CoinbaseMaturity: 100

Catatan:

* Mainnet masih placeholder.
* Localnet 10 dipilih agar mudah dites.
* Semua ledger/balance/validation harus mengambil param dari selected network profile.
* Jangan hardcode maturity di banyak tempat.

==================================================
2. Definisi maturity
====================

Coinbase transaction pada block height H dianggap mature jika:

currentTipHeight >= H + CoinbaseMaturity

Contoh localnet maturity 10:

* Coinbase di height 1 mature saat tip height >= 11.
* Coinbase di height 2 mature saat tip height >= 12.
* Coinbase di height 10 mature saat tip height >= 20.

Genesis tidak dianggap coinbase jika tidak punya tx coinbase.

Normal transaction output / balance hasil transfer:

* langsung mature/spendable setelah confirmed, kecuali project punya aturan lain.
* Phase ini hanya membatasi coinbase mining reward.

==================================================
3. Balance model baru
=====================

Tambahkan balance result yang membedakan:

* confirmed_balance
* mature_balance
* immature_balance
* spendable_balance
* pending_outgoing
* pending_incoming jika mudah
* total_balance

Definisi minimal:

confirmed_balance:
Semua saldo confirmed di canonical chain, termasuk mature + immature coinbase.

mature_balance:
Saldo yang boleh dibelanjakan:

* confirmed normal tx received,
* coinbase yang sudah mature,
* dikurangi outgoing confirmed.

immature_balance:
Coinbase reward confirmed tetapi belum mature.

spendable_balance:
Mature balance dikurangi pending outgoing di mempool.

pending_outgoing:
Total amount + fee dari tx pending yang berasal dari address tersebut.

pending_incoming:
Total amount tx pending yang masuk ke address tersebut.
Jika terlalu besar, boleh TODO, tapi lebih baik ada.

total_balance:
confirmed_balance + pending_incoming jika existing UX butuh.
Atau tetap confirmed_balance, tapi docs harus jelas.

Yang paling penting:

* send harus memakai spendable_balance, bukan confirmed_balance.
* balance command harus menampilkan mature/immature agar user tidak bingung.

==================================================
4. Update ledger calculation
============================

Update ledger builder / balance scanner supaya bisa menghitung maturity berdasarkan current tip height.

Jika ledger saat ini hanya menyimpan account balance total:

* Tambahkan function baru yang scan canonical blocks dan menghasilkan maturity-aware balances.
* Jangan hapus function lama kalau masih dipakai, tapi arahkan wallet/balance/send validation ke function baru.

Fungsi yang disarankan:

GetBalance(address string) Amount
GetBalanceDetails(address string, currentHeight uint64, params ConsensusParams) BalanceDetails

BalanceDetails:
Address string
Confirmed Amount
Mature Amount
Immature Amount
Spendable Amount
PendingOutgoing Amount
PendingIncoming Amount
CoinbaseMaturity uint64
CurrentHeight uint64

Pastikan:

* Reorg menggunakan canonical chain terbaru.
* Orphan block coinbase tidak dihitung.
* Mempool pending outgoing dikurangi dari spendable.
* Tidak boleh negative amount. Clamp ke zero jika perlu, tapi logic harus benar.

==================================================
5. Update send validation
=========================

Saat membuat tx:

Sebelumnya mungkin:
if balance < amount+fee => insufficient funds

Ubah menjadi:
if spendable_balance < amount+fee => insufficient mature funds

Error harus jelas:
insufficient mature balance: spendable X IDR, required Y IDR, immature Z IDR

Contoh:

* Miner baru mine 3 block localnet.
* confirmed balance 150 IDR.
* maturity 10.
* mature balance 0.
* immature balance 150.
* send 10 harus gagal:
  insufficient mature balance

Setelah mine sampai height 11:

* reward height 1 mature 50 IDR.
* send 10 harus berhasil.

==================================================
6. Mining behavior
==================

Mining tetap:

* membuat coinbase tx,
* menambah confirmed supply,
* menampilkan miner balance.

Tapi output mining harus lebih jelas.

Setelah mining:

miner confirmed balance: 150 IDR
miner mature balance: 0 IDR
miner immature balance: 150 IDR
miner spendable balance: 0 IDR

Jika ingin tetap mempertahankan output lama:
miner balance: 150 IDR

Tambahkan field tambahan di bawahnya:
mature: 0 IDR
immature: 150 IDR
spendable: 0 IDR

Jangan membuat mining gagal hanya karena reward belum mature.

==================================================
7. Update balance command
=========================

Command:

balance <address>

Local dan remote harus menampilkan:

address: IDR...
confirmed balance: 150 IDR
mature balance: 0 IDR
immature balance: 150 IDR
spendable balance: 0 IDR
pending outgoing: 0 IDR
pending incoming: 0 IDR
coinbase maturity: 10
current height: 3

Jika address punya mature coinbase:

confirmed balance: 550 IDR
mature balance: 50 IDR
immature balance: 500 IDR
spendable balance: 50 IDR

Jangan hapus output lama jika test lama bergantung pada string tertentu, tapi tambahkan field baru.

==================================================
8. Update wallet list / inspect jika ada
========================================

Jika command wallet list menampilkan balance:

* Tambahkan mature/spendable balance jika mudah.
* Jika terlalu besar, cukup balance command dulu.

Wallet inspect:

* Tampilkan address format tetap:
  base58check
  IDR
  secp256k1
* Tidak perlu balance detail jika belum ada.

==================================================
9. Update RPC
=============

Update endpoint balance RPC.

Jika endpoint lama:
GET /balance/{address}
atau:
GET /wallet/balance?address=...

Tambahkan response fields:

{
"address": "IDR...",
"confirmed_balance": "150",
"mature_balance": "0",
"immature_balance": "150",
"spendable_balance": "0",
"pending_outgoing": "0",
"pending_incoming": "0",
"coinbase_maturity": 10,
"current_height": 3
}

Gunakan format amount existing project.
Jangan breaking response lama jika ada field `balance`; boleh isi `balance` = confirmed_balance untuk backward compatibility.

Update send RPC:

* harus memakai spendable balance.
* return error clear jika insufficient mature balance.

==================================================
10. Update chain info
=====================

Tambahkan field:

coinbase maturity: 10

di `chain info` dan RPC chain info.

Jangan hapus field lama.

Contoh:

network: localnet
chain id: 777001
height: 3
coinbase maturity: 10
total supply: 150 IDR
circulating supply: 0 IDR

==================================================
11. Circulating supply
======================

Update circulating supply definition.

Sebelumnya mungkin sama dengan total supply.

Setelah maturity:

* total supply = semua coinbase reward confirmed di canonical chain.
* circulating supply = semua coinbase reward yang sudah mature.

Localnet example:

* height 3, reward 50, maturity 10:
  total supply = 150
  circulating supply = 0
* height 11:
  total supply = 550
  circulating supply = 50
* height 15:
  total supply = 750
  circulating supply = 250
  karena reward height 1-5 mature jika current height >= H+10.

Tambahkan tests untuk ini.

==================================================
12. Mempool pending outgoing
============================

Jika address punya mature spendable 50 lalu membuat tx 10:

* sebelum tx mined:
  mature balance: 50
  pending outgoing: 10
  spendable balance: 40
* tx kedua 45 harus gagal karena spendable tinggal 40.
* tx kedua 40 boleh jika nonce policy mendukung multiple pending tx.
  Jika nonce policy belum mendukung multiple pending tx, jangan ubah besar.
  Minimal pending outgoing mencegah double spend pending.

Pastikan:

* mempool duplicate guard tetap jalan.
* reorg requeue tidak merusak pending outgoing.
* setelah tx mined, pending outgoing hilang dari mempool dan mature/confirmed berubah sesuai ledger.

==================================================
13. Reorg compatibility
=======================

Reorg harus memengaruhi maturity sesuai canonical chain baru.

Rules:

* Jika reorg mengganti branch, mature/immature dihitung ulang dari canonical chain.
* Coinbase dari orphaned blocks tidak dihitung.
* Normal tx dari orphaned blocks yang requeue ke mempool menjadi pending, bukan confirmed.
* Pending outgoing dari requeued tx harus mengurangi spendable.
* Jika tx requeued sekarang tidak lagi punya mature funds karena maturity berubah, drop invalid.
* Chain validate harus tetap pass.

Update reorg tx scenarios jika perlu.

==================================================
14. Chain validation
====================

Chain validation harus memastikan block tx tidak membelanjakan immature coinbase.

Saat validate block pada height H:

* Ledger state sebelum applying block H harus tahu mature balance pada parent height H-1.
* Normal tx dalam block H hanya boleh spend mature/spendable confirmed funds.
* Coinbase di block H tidak boleh dipakai di block yang sama.
* Coinbase di height K hanya boleh dipakai jika parent tip height before tx/block >= K + maturity.

Jika existing validation apply sequentially:

* Tambahkan maturity-aware spend check saat applying normal tx.

Error:
invalid transaction at height X: spends immature coinbase

==================================================
15. Coinbase maturity and nonce
===============================

Pastikan nonce tetap benar:

* Failed send karena immature balance tidak menaikkan nonce.
* Pending tx tetap memakai nonce policy existing.
* Reorg yang drop tx harus membuat nonce canonical kembali benar sesuai chain + mempool.

Jangan ubah nonce logic besar kecuali perlu.

==================================================
16. Dev/test commands
=====================

Update existing dev simulations:

* dev fork-sim
* dev reorg-tx-sim

Karena coinbase maturity localnet 10, test yang dulu langsung send setelah mining 2-3 blocks akan gagal.

Solusi:

* Tambahkan helper mine funding blocks sampai coinbase mature.
* Untuk scenario yang butuh spendable balance:
  mine CoinbaseMaturity + N blocks dulu.
* Jangan disable maturity di dev tests kecuali explicit test profile.
* Jika perlu, tambahkan local test profile dengan maturity rendah hanya untuk unit test, tapi default localnet tetap 10.

Output dev sim harus tetap jelas:
funding blocks mined: 11
spendable balance: 50 IDR

==================================================
17. Tests wajib
===============

Tambahkan/update tests:

1. Coinbase immature after mining:

* localnet maturity 10.
* mine 3 blocks.
* confirmed balance 150.
* mature balance 0.
* immature balance 150.
* spendable balance 0.
* circulating supply 0.

2. Coinbase matures after maturity:

* mine 11 blocks.
* reward height 1 mature.
* confirmed balance 550.
* mature balance 50.
* immature balance 500.
* spendable balance 50.
* circulating supply 50.

3. Multiple mature rewards:

* mine 15 blocks.
* mature reward height 1-5.
* mature balance 250.
* immature balance 500.
* total confirmed 750.
* circulating supply 250.

4. Send immature rejected:

* mine 3 blocks.
* try send 10.
* fail insufficient mature balance.
* mempool count remains 0.
* nonce unchanged.

5. Send after maturity success:

* mine 11 blocks.
* send 10.
* tx pending.
* pending outgoing 10.
* spendable 40.

6. Pending outgoing reduces spendable:

* mature balance 50.
* create pending tx 10.
* spendable 40.
* second tx requiring 45 fails.

7. Mine pending tx:

* after mining tx:
  sender confirmed/mature balance decreases correctly.
  receiver confirmed/mature balance increases correctly.
  pending outgoing 0.
  normal transactions count increases.

8. Receiver normal tx is mature immediately:

* sender sends 10 to receiver.
* tx mined.
* receiver mature/spendable balance 10.

9. Chain validate rejects immature spend:

* construct chain where tx spends coinbase before maturity.
* validation fails.

10. Chain validate accepts mature spend:

* construct chain where tx spends matured coinbase.
* validation passes.

11. Circulating supply:

* height 0 -> 0.
* height 3 -> 0.
* height 11 -> 50.
* height 15 -> 250.

12. Reorg recalculates maturity:

* branch A and B have different heights.
* after reorg, mature/immature follows new canonical height/blocks.
* orphaned coinbase not counted.

13. Reorg requeue with maturity:

* orphan tx requeued only if sender still has mature spendable funds.
* if not, dropped invalid.

14. Mempool pending outgoing after reorg:

* requeued pending tx appears in pending outgoing.
* spendable reduced.

15. Balance RPC:

* returns confirmed/mature/immature/spendable fields.

16. CLI balance:

* prints mature/immature/spendable.

17. Chain info:

* includes coinbase maturity.
* circulating supply uses mature supply.

18. Existing tests still pass:

* address tests.
* difficulty tests.
* reorg tests.
* mempool tests.
* P2P tests.

==================================================
18. README / docs update
========================

Update README.md and/or docs:

Current phase:
Phase 2.8 — Coinbase Maturity & Mature Balance

Tambahkan penjelasan:

* Mining reward tidak langsung spendable.
* Localnet coinbase maturity = 10 blocks.
* Testnet/mainnet placeholder = 100 blocks.
* Confirmed balance berbeda dengan mature/spendable balance.
* Send memakai spendable balance.
* Circulating supply dihitung dari mature coinbase rewards.
* Tujuan maturity:
  mencegah reorg abuse,
  mencegah reward baru langsung dipakai,
  membuat chain lebih aman sebelum public testnet.

Tambahkan contoh:

Mine 3 blocks:
confirmed: 150 IDR
mature: 0 IDR
immature: 150 IDR
spendable: 0 IDR

Mine 11 blocks:
confirmed: 550 IDR
mature: 50 IDR
immature: 500 IDR
spendable: 50 IDR

==================================================
19. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/deskachain --datadir ./testdata/maturity dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/maturity init
go run ./node/cmd/deskachain --datadir ./testdata/maturity wallet new

Expected:
IDR...

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/maturity node start --rpc :8391 --p2p :9391 --advertise-p2p http://127.0.0.1:9391

Mine 3:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 mine --address <ADDR_A> --blocks 3

Balance:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 balance <ADDR_A>

Expected:
confirmed balance: 150 IDR
mature balance: 0 IDR
immature balance: 150 IDR
spendable balance: 0 IDR

Try send:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 wallet new
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 send --from <ADDR_A> --to <ADDR_B> --amount 10

Expected:
error: insufficient mature balance

Mine until 11:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 mine --address <ADDR_A> --blocks 8

Balance:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 balance <ADDR_A>

Expected:
confirmed balance: 550 IDR
mature balance: 50 IDR
immature balance: 500 IDR
spendable balance: 50 IDR

Send after maturity:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 send --from <ADDR_A> --to <ADDR_B> --amount 10

Expected:
tx created
status: pending

Mempool:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 mempool list --detail

Balance sender before mining tx:
mature balance: 50 IDR
pending outgoing: 10 IDR
spendable balance: 40 IDR

Mine tx:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 mine --address <ADDR_A> --blocks 1

Balance receiver:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 balance <ADDR_B>

Expected:
confirmed balance: 10 IDR
mature balance: 10 IDR
spendable balance: 10 IDR

Chain info:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 chain info

Expected:
coinbase maturity: 10
total supply: 600 IDR
circulating supply: 100 IDR or sesuai maturity rule after height 12:
mature coinbase height 1-2 = 100 IDR
normal transactions: 1

Validate:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8391 chain validate

Expected:
chain valid

Jangan over-engineer.
Fokus Phase 2.8 hanya:

* coinbase maturity,
* mature/immature/spendable balance,
* send validation using mature balance,
* circulating supply based on mature rewards,
* reorg compatibility,
* balance/chain info/RPC docs/tests.
