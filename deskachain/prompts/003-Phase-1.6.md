Kamu sedang bekerja pada project Go yang sudah ada: DesKaChain.

Status saat ini:
- Phase 1 dan Phase 1.5 sudah selesai.
- go test ./... sudah pass.
- init sudah aman.
- dev reset --yes sudah aman.
- wallet new/list sudah jalan.
- mining sudah jalan.
- chain info sudah akurat.
- chain validate sudah jalan.
- datadir sudah bisa digunakan.

Tujuan patch ini:
Membuat alur transfer antar wallet benar-benar stabil sebelum masuk Phase 2 P2P.

Nama patch:
DesKaChain Phase 1.6 — Transfer & Mempool Hardening

Aturan penting:
- Jangan tambah P2P dulu.
- Jangan tambah smart contract.
- Jangan tambah EVM.
- Jangan rewrite total project.
- Patch incremental saja.
- Pertahankan command lama.
- Semua tetap harus jalan dengan:
  go mod tidy
  go test ./...
  go run ./node/cmd/deskachain

==================================================
1. Perkuat alur send antar wallet
==================================================

Pastikan command berikut benar-benar bekerja:

  go run ./node/cmd/deskachain send --from <addressA> --to <addressB> --amount 10

Perilaku wajib:
- Wallet pengirim harus ditemukan di wallet store lokal.
- Address tujuan harus valid prefix dkc1.
- Amount wajib > 0.
- Amount tidak boleh melebihi confirmed balance dikurangi pending outgoing amount.
- Nonce transaksi harus:
  confirmed nonce pengirim + jumlah pending tx dari pengirim + 1
- Fee default tetap 0 DKC untuk Phase 1.6.
- Transaksi harus ditandatangani.
- Public key harus tersimpan di transaksi.
- Tx ID harus deterministic berdasarkan isi transaksi.
- Setelah dibuat, transaksi masuk mempool.
- Output command harus jelas:

  tx created
  id: <txid>
  from: <from>
  to: <to>
  amount: 10 DKC
  fee: 0 DKC
  nonce: 1
  status: pending

Kalau balance tidak cukup, tampilkan error jelas:

  send failed: insufficient balance

Kalau address tujuan tidak valid:

  send failed: invalid recipient address

Kalau wallet from tidak ditemukan:

  send failed: local wallet not found for sender

==================================================
2. Perkuat command mempool list
==================================================

Pastikan command:

  go run ./node/cmd/deskachain mempool list

Output jika kosong:

  pending tx count: 0

Output jika ada tx:

  pending tx count: 2
  id=<txid> from=<from> to=<to> amount=10 DKC fee=0 DKC nonce=1
  id=<txid> from=<from> to=<to> amount=5 DKC fee=0 DKC nonce=2

Aturan:
- Jangan tampilkan private key.
- Urutkan berdasarkan waktu masuk atau nonce.
- Format amount harus pakai formatter amount yang sudah ada.

==================================================
3. Mining harus memasukkan transaksi pending
==================================================

Saat menjalankan:

  go run ./node/cmd/deskachain mine --address <miner> --blocks 1

Block baru harus:
- Membuat coinbase tx untuk miner.
- Mengambil transaksi valid dari mempool.
- Memvalidasi ulang tx sebelum dimasukkan ke block.
- Menghindari double spend dalam satu block.
- Mengurutkan transaksi berdasarkan sender + nonce atau urutan masuk yang aman.
- Setelah block berhasil ditambahkan, hapus tx yang sudah masuk dari mempool.
- Tx yang invalid jangan dimasukkan ke block.

Output mining harus menampilkan jumlah tx:

  mined block height=4 hash=<hash> txs=2 reward=50 DKC difficulty=4 nonce=<nonce>

Catatan:
- txs termasuk coinbase.
- Jika 1 transfer + 1 coinbase, maka txs=2.

==================================================
4. Balance setelah transfer harus benar
==================================================

Contoh alur:

  wallet A mining 3 block = 150 DKC
  wallet A kirim 10 DKC ke wallet B
  mine 1 block oleh wallet A

Hasil:
- Wallet A:
  150 - 10 + 50 = 190 DKC
- Wallet B:
  10 DKC
- Total supply:
  200 DKC

Fee masih 0, jadi tidak ada fee tambahan.

Pastikan ledger replay menghasilkan angka tersebut.

==================================================
5. Tambahkan command tx get
==================================================

Tambahkan command:

  tx get <txid>

Perilaku:
- Cari tx di chain terlebih dahulu.
- Jika tidak ditemukan, cari di mempool.
- Jika ditemukan di chain, tampilkan:

  id: <txid>
  status: confirmed
  block_height: <height>
  from: <from>
  to: <to>
  amount: <amount>
  fee: <fee>
  nonce: <nonce>
  coinbase: false

- Jika ditemukan di mempool:

  id: <txid>
  status: pending
  from: <from>
  to: <to>
  amount: <amount>
  fee: <fee>
  nonce: <nonce>
  coinbase: false

- Jika tidak ditemukan:

  tx not found

==================================================
6. Tambahkan command address validate
==================================================

Tambahkan command:

  address validate <address>

Output jika valid:

  address valid

Output jika invalid:

  address invalid

Aturan validasi minimal:
- Harus prefix dkc1.
- Panjang sesuai format address project saat ini.
- Karakter hex/hash suffix valid sesuai implementasi saat ini.
- Jangan terlalu over-engineer, cukup untuk Phase 1.6.

==================================================
7. Tambahkan command wallet inspect
==================================================

Tambahkan command:

  wallet inspect --address <address>

Output:
  address: <address>
  exists: true
  confirmed balance: <balance>
  confirmed nonce: <nonce>
  pending outgoing tx: <n>
  pending outgoing amount: <amount>
  pending incoming tx: <n>
  pending incoming amount: <amount>

Jika wallet tidak ada lokal tapi address valid:
  address: <address>
  exists: false
  confirmed balance: <balance>
  confirmed nonce: <nonce>
  pending outgoing tx: <n>
  pending outgoing amount: <amount>
  pending incoming tx: <n>
  pending incoming amount: <amount>

Jangan tampilkan private key.

==================================================
8. Tambahkan RPC tx dan mempool detail
==================================================

Pertahankan endpoint lama.

Tambahkan endpoint:

GET /tx/{txid}

Response jika confirmed:
{
  "id": "...",
  "status": "confirmed",
  "block_height": 4,
  "from": "...",
  "to": "...",
  "amount": "10 DKC",
  "fee": "0 DKC",
  "nonce": 1,
  "coinbase": false
}

Response jika pending:
{
  "id": "...",
  "status": "pending",
  "from": "...",
  "to": "...",
  "amount": "10 DKC",
  "fee": "0 DKC",
  "nonce": 1,
  "coinbase": false
}

Response jika tidak ditemukan:
{
  "error": "tx not found"
}

Tambahkan juga:

GET /address/{address}

Response:
{
  "address": "...",
  "valid": true,
  "confirmed_balance": "190 DKC",
  "confirmed_nonce": 1,
  "pending_outgoing_count": 0,
  "pending_outgoing_amount": "0 DKC",
  "pending_incoming_count": 0,
  "pending_incoming_amount": "0 DKC"
}

==================================================
9. Perkuat chain validate untuk transaksi normal
==================================================

Pastikan chain validate mengecek:
- Signature transaksi normal valid.
- Public key cocok dengan address sender.
- Tx ID sesuai isi transaksi.
- Nonce sender berurutan.
- Balance sender cukup saat replay.
- Coinbase hanya satu per block.
- Coinbase ada di index pertama transaksi block.
- Coinbase amount = reward + total fee.
- Tidak ada tx id duplikat di chain.
- Tidak ada tx normal dengan amount 0.
- Tidak ada tx normal dengan from == to kecuali kalau memang ingin diperbolehkan.
  Untuk Phase 1.6, sebaiknya tolak from == to.

Jika gagal, error harus jelas, contoh:
  chain invalid: block 4 tx 1 invalid signature
  chain invalid: block 4 duplicate tx id
  chain invalid: block 4 sender balance insufficient

==================================================
10. Tambahkan test end-to-end transfer
==================================================

Tambahkan test yang mensimulasikan alur ini di datadir temporary:

1. Init chain.
2. Buat wallet A.
3. Buat wallet B.
4. Mine 3 block ke wallet A.
5. Pastikan balance A = 150 DKC.
6. Send 10 DKC dari A ke B.
7. Pastikan mempool count = 1.
8. Mine 1 block ke wallet A.
9. Pastikan mempool count = 0.
10. Pastikan balance A = 190 DKC.
11. Pastikan balance B = 10 DKC.
12. Pastikan total supply = 200 DKC.
13. Pastikan chain validate pass.
14. Pastikan tx get mengembalikan status confirmed.

Test lain yang wajib:
- Send gagal jika balance kurang.
- Send gagal jika address tujuan invalid.
- Send gagal jika wallet pengirim tidak ditemukan.
- Pending outgoing mencegah overspend.
- Nonce pending bertambah benar:
  kirim 2 tx dari sender yang sama sebelum mining:
    tx1 nonce 1
    tx2 nonce 2
- Mempool clear --yes menghapus pending tx.
- tx get bisa menemukan tx pending.
- tx get bisa menemukan tx confirmed setelah mining.
- address validate valid/invalid.

==================================================
11. Update README
==================================================

Update README.md:

Tambahkan bagian:
- Phase sekarang: 1.6 Transfer & Mempool Hardening.
- Contoh transfer antar wallet:

  go run ./node/cmd/deskachain dev reset --yes
  go run ./node/cmd/deskachain init
  go run ./node/cmd/deskachain wallet new
  go run ./node/cmd/deskachain wallet new
  go run ./node/cmd/deskachain mine --address <walletA> --blocks 3
  go run ./node/cmd/deskachain send --from <walletA> --to <walletB> --amount 10
  go run ./node/cmd/deskachain mempool list
  go run ./node/cmd/deskachain mine --address <walletA> --blocks 1
  go run ./node/cmd/deskachain balance <walletA>
  go run ./node/cmd/deskachain balance <walletB>
  go run ./node/cmd/deskachain chain validate

Tambahkan penjelasan:
- Pending tx belum mengubah confirmed balance sampai block ditambang.
- Miner mendapat reward 50 DKC per block.
- Fee masih 0 DKC di Phase 1.6.
- Wallet storage masih dev-only dan belum aman untuk production.

==================================================
12. Expected final commands
==================================================

Setelah patch selesai, command ini harus berhasil:

  go mod tidy
  go test ./...
  go run ./node/cmd/deskachain dev reset --yes
  go run ./node/cmd/deskachain init
  go run ./node/cmd/deskachain wallet new
  go run ./node/cmd/deskachain wallet new

Lalu manual test:

  go run ./node/cmd/deskachain mine --address <walletA> --blocks 3
  go run ./node/cmd/deskachain send --from <walletA> --to <walletB> --amount 10
  go run ./node/cmd/deskachain mempool list
  go run ./node/cmd/deskachain wallet inspect --address <walletA>
  go run ./node/cmd/deskachain wallet inspect --address <walletB>
  go run ./node/cmd/deskachain mine --address <walletA> --blocks 1
  go run ./node/cmd/deskachain balance <walletA>
  go run ./node/cmd/deskachain balance <walletB>
  go run ./node/cmd/deskachain chain info
  go run ./node/cmd/deskachain chain validate
  go run ./node/cmd/deskachain tx get <txid>

Expected:
- Wallet A balance akhir: 190 DKC
- Wallet B balance akhir: 10 DKC
- Total supply: 200 DKC
- Mempool kosong setelah mining
- Chain valid

Jangan over-engineer.
Fokus patch ini hanya agar transfer antar wallet, mempool, nonce, signature, dan mining transaksi normal benar-benar stabil sebelum Phase 2 P2P.