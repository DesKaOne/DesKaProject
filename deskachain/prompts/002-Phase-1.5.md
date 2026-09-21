Anda sedang mengerjakan proyek Go yang sudah ada: **IndoChain**.

## Status Saat Ini

* MVP Fase 1 sudah berhasil dikompilasi.
* `go test ./...` berhasil dijalankan.
* `init` berfungsi.
* `wallet new` berfungsi.
* `mine` berfungsi.
* `balance` berfungsi.
* `chain info` berfungsi.
* `chain print` berfungsi.

## Tujuan

Mengimplementasikan **Patch Stabilisasi Fase 1.5** sebelum menambahkan fitur P2P.

## Penting

* Jangan tambahkan P2P terlebih dahulu.
* Jangan tambahkan smart contract.
* Jangan tambahkan EVM.
* Jangan menulis ulang seluruh proyek.
* Pertahankan seluruh command dan perilaku yang sudah ada.
* Lakukan patch secara bertahap (incremental).
* Pastikan proyek tetap dapat dijalankan dengan:

```bash
go test ./...
go run ./node/cmd/indochain
```

## Sasaran Utama

1. Menambahkan pengelolaan direktori data yang aman.
2. Menambahkan command reset untuk pengembangan.
3. Menambahkan command validasi blockchain.
4. Memperbaiki output mining.
5. Meningkatkan akurasi informasi blockchain.
6. Meningkatkan keamanan wallet untuk kebutuhan pengembangan.
7. Memperbaiki alur transaksi dan mempool.
8. Menambahkan test yang lebih lengkap.
9. Memperbarui README.

---

# 1. Flag Direktori Data

Tambahkan flag CLI global:

```text
--datadir <path>
```

Default:

```text
./data
```

Contoh:

```bash
go run ./node/cmd/indochain --datadir ./data init
go run ./node/cmd/indochain --datadir ./testdata/dev1 init
go run ./node/cmd/indochain --datadir ./testdata/dev1 wallet new
go run ./node/cmd/indochain --datadir ./testdata/dev1 mine --address <addr> --blocks 3
```

Aturan:

* Seluruh penyimpanan blockchain harus menggunakan `datadir`.
* Penyimpanan wallet harus menggunakan `datadir`.
* Penyimpanan mempool (jika persisten) harus menggunakan `datadir`.
* Perilaku default tetap kompatibel dengan `./data`.

Alasan:

Mempermudah pengujian beberapa blockchain lokal secara bersamaan.

---

# 2. Tambahkan Command Reset untuk Development

Tambahkan command:

```text
dev reset
```

Perilaku:

* Menghapus database blockchain lokal dan file wallet development pada `datadir` yang dipilih.
* Wajib menggunakan flag konfirmasi:

```bash
go run ./node/cmd/indochain dev reset --yes
```

Tanpa `--yes`, tampilkan peringatan dan batalkan proses.

Contoh:

```text
This will delete local IndoChain dev data under ./data.
Re-run with --yes to confirm.
```

Jika dikonfirmasi:

```text
deleted ./data
dev data reset complete
```

Persyaratan:

* Hanya untuk kebutuhan development.
* Jangan pernah menghapus direktori di luar `datadir`.
* Tolak nilai berbahaya seperti:

```text
/
C:\
.
..
(empty string)
```

* Gunakan `filepath.Clean`.
* Tambahkan validasi keamanan.

Jika direktori tidak ada:

```text
datadir does not exist, nothing to reset
```

---

# 3. Tambahkan Command Validasi Blockchain

Tambahkan command:

```text
chain validate
```

Perilaku:

Melakukan replay blockchain dari genesis hingga tip dan memvalidasi:

* hash genesis deterministik
* previous_hash setiap block sesuai
* kenaikan height valid
* hash block sesuai hasil perhitungan
* PoW valid untuk seluruh block non-genesis
* merkle root sesuai transaksi
* aturan coinbase terpenuhi
* total supply sesuai total reward block
* saldo akun tidak pernah negatif
* signature transaksi non-coinbase valid
* nonce valid

Contoh:

```bash
go run ./node/cmd/indochain chain validate
```

Jika sukses:

```text
chain valid
height: 4
blocks: 5
total supply: 200 dIDR
```

Jika gagal:

```text
chain invalid: block 3 previous_hash mismatch
```

Command harus mengembalikan exit code non-zero ketika validasi gagal.

---

# 4. Perbaiki Output Mining

Output saat ini:

```text
mined block 2 <hash>
```

Ubah menjadi:

```text
mined block height=2 hash=<hash> txs=1 reward=50 dIDR difficulty=4 nonce=<nonce>
```

Jika menambang beberapa block, tampilkan informasi setiap block.

Setelah selesai:

```text
mining complete
mined blocks: 3
new height: 4
miner balance: 150 dIDR
```

Saldo miner harus dihitung setelah seluruh proses mining selesai.

---

# 5. Tingkatkan Informasi Blockchain

Tetap pertahankan:

```text
height
tip hash
difficulty
total supply
pending tx count
```

Tambahkan:

```text
blocks
coinbase blocks
total transactions
circulating supply
datadir
```

Definisi:

* height = tinggi tip saat ini
* blocks = jumlah block termasuk genesis
* coinbase blocks = jumlah block non-genesis yang memiliki reward coinbase
* total supply dan circulating supply boleh sama pada Fase 1
* total transactions mencakup transaksi coinbase dan transaksi biasa
* pending tx count berasal dari mempool

---

# 6. Tambahkan Wallet Export dan Peringatan Keamanan

Tambahkan command:

```text
wallet export --address <address>
```

Perilaku:

* Mengekspor private key wallet development lokal.
* Wajib menggunakan flag:

```text
--show-private-key
```

Tanpa flag:

```text
Refusing to print private key without --show-private-key.
Phase 1 wallet storage is for development only.
```

Dengan flag:

```text
address: <addr>
private_key: <hex>
```

Perbaiki juga:

```text
wallet list
```

Format:

```text
address=<addr>
```

Jangan pernah menampilkan private key pada `wallet list`.

---

# 7. Perbaiki Workflow Send dan Mempool

Pastikan command:

```bash
send --from <address> --to <address> --amount 1.25
```

Melakukan seluruh proses berikut:

* Mencari wallet lokal berdasarkan alamat pengirim.
* Memeriksa saldo terkonfirmasi sebelum masuk mempool.
* Memeriksa transaksi pending keluar untuk mencegah overspending.
* Menghitung nonce berikutnya:

```text
confirmed account nonce
+ pending tx count from sender
+ 1
```

* Menandatangani transaksi.
* Menambahkan transaksi ke mempool.

Output:

```text
tx created
id: <txid>
from: <from>
to: <to>
amount: 1.25 dIDR
fee: 0 dIDR
nonce: <nonce>
status: pending
```

Tambahkan command:

```text
mempool list
```

Output:

```text
pending tx count: N
id=<id> from=<from> to=<to> amount=<amount> fee=<fee> nonce=<nonce>
```

Tambahkan command:

```text
mempool clear --yes
```

Perilaku:

* Menghapus seluruh transaksi pending lokal.
* Wajib menggunakan flag `--yes`.

---

# 8. Perjelas Perilaku Init

Saat ini:

```text
chain initialized
```

Perbaiki menjadi:

Jika blockchain belum ada:

```text
chain initialized
genesis hash: <hash>
datadir: <datadir>
```

Jika blockchain sudah ada:

```text
chain already initialized
height: <height>
tip hash: <hash>
datadir: <datadir>
```

Hal ini membantu menjelaskan mengapa proses mining dapat dimulai dari height yang lebih tinggi.

---

# 9. Perkuat Parser dan Formatter Amount

Parser harus:

* menerima `"1"`
* menerima `"1.0"`
* menerima `"1.00000000"`
* menerima `"0.00000001"`
* menolak nilai negatif
* menolak lebih dari 8 digit desimal
* menolak string tidak valid
* menolak string kosong
* menolak pembulatan floating-point
* hanya menggunakan operasi string dan integer

Formatter harus konsisten.

Contoh:

```text
100000000   => 1 dIDR
15000000000 => 150 dIDR
1           => 0.00000001 dIDR
```

Rekomendasi:

* Hilangkan nol di belakang untuk keterbacaan CLI.
* Tetap simpan presisi penuh secara internal.

---

# 10. Tambahkan Test

### Data Directory

* custom datadir digunakan dengan benar

### Init

* genesis hanya dibuat sekali
* init kedua tidak membuat genesis baru

### Mining

* mining N block menaikkan height sebanyak N
* miner menerima saldo yang sesuai
* total supply sesuai jumlah reward block

### Chain Validate

* blockchain valid lolos validasi
* previous hash yang dimodifikasi gagal
* merkle root yang dimodifikasi gagal (jika mudah diuji)
* PoW tidak valid gagal

### Amount

* kasus parse valid
* kasus parse tidak valid
* parse unit terkecil
* kasus formatting

### Mempool

* transaksi dari wallet yang memiliki saldo masuk ke mempool
* saldo tidak cukup ditolak
* transaksi pending mencegah overspending
* nonce bertambah sesuai transaksi pending

### Wallet

* wallet list tidak membocorkan private key
* wallet export membutuhkan flag `--show-private-key`

---

# 11. Peningkatan RPC

Pertahankan endpoint yang sudah ada.

Tambahkan:

```http
GET /mempool
POST /mempool/clear
GET /chain/validate
```

Response valid:

```json
{
  "valid": true,
  "height": 4,
  "blocks": 5,
  "total_supply": "200 dIDR"
}
```

Response tidak valid:

```json
{
  "valid": false,
  "error": "block 3 previous_hash mismatch"
}
```

---

# 12. Pembaruan README

Tambahkan:

## Current Phase

```text
1.5
```

## Warning

```text
IndoChain is experimental local blockchain software.
Do not use Phase 1.5 wallets for real funds.
```

## Daftar Command

* init
* dev reset --yes
* wallet new
* wallet list
* wallet export
* balance
* send
* mempool list
* mempool clear --yes
* mine
* chain info
* chain print
* chain validate
* rpc

## Contoh Alur Penggunaan

```bash
go run ./node/cmd/indochain dev reset --yes
go run ./node/cmd/indochain init
go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain mine --address <addr> --blocks 3
go run ./node/cmd/indochain balance <addr>
go run ./node/cmd/indochain chain info
go run ./node/cmd/indochain chain validate
```

Jelaskan juga bahwa:

```text
Total supply dapat lebih besar daripada saldo satu wallet karena block sebelumnya mungkin ditambang oleh alamat miner yang berbeda.
```

---

# Validasi Akhir yang Diharapkan

Setelah patch selesai, seluruh command berikut harus berfungsi:

```bash
go mod tidy
go test ./...

go run ./node/cmd/indochain dev reset --yes

go run ./node/cmd/indochain init
go run ./node/cmd/indochain init

go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain wallet list

go run ./node/cmd/indochain mine --address <addr> --blocks 3

go run ./node/cmd/indochain balance <addr>

go run ./node/cmd/indochain chain info
go run ./node/cmd/indochain chain validate
go run ./node/cmd/indochain chain print
```

Pengujian custom datadir:

```bash
go run ./node/cmd/indochain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/node1 init
go run ./node/cmd/indochain --datadir ./testdata/node1 wallet new
go run ./node/cmd/indochain --datadir ./testdata/node1 chain info
```

Jangan melakukan over-engineering.

Fokuskan Fase 1.5 pada stabilitas dan kesiapan sebelum memasuki Fase 2 (P2P).
