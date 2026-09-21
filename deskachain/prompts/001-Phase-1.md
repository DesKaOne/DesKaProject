Anda sedang mengerjakan proyek Go baru bernama **IndoChain**.

## Tujuan

Membangun MVP (Minimum Viable Product) Fase 1 dari blockchain kecil yang menggunakan CPU mining.

## Penting

* Jangan tambahkan networking/P2P terlebih dahulu.
* Jangan tambahkan smart contract terlebih dahulu.
* Jangan tambahkan EVM terlebih dahulu.
* Jaga arsitektur tetap bersih dan mudah dikembangkan.
* Gunakan Go Standard Library semaksimal mungkin.
* Proyek harus dapat dikompilasi dan dijalankan menggunakan:

```bash
go run ./node/cmd/indochain
```

* Tambahkan komentar yang jelas pada bagian yang nantinya akan digunakan untuk integrasi komponen Fase 2/Fase 3.

## Nama Proyek

indochain

## Koin

* Nama: IndoChain
* Ticker: dIDR
* Desimal: 8

## Fitur Fase 1

1. Genesis block
2. Struktur block
3. Struktur transaksi
4. Ledger saldo berbasis akun (account-based)
5. Pembuatan wallet
6. Pembuatan address
7. Proses mining lokal
8. Validasi Proof-of-Work
9. Placeholder untuk dynamic difficulty
10. Penyimpanan persisten menggunakan BoltDB atau BadgerDB
11. Perintah CLI
12. API HTTP JSON-RPC dasar
13. Unit test untuk validasi inti

## Penyimpanan yang Direkomendasikan

Gunakan bbolt:

```go
go.etcd.io/bbolt
```

## Perintah CLI yang Wajib Tersedia

* init
* wallet new
* wallet list
* balance <address>
* send --from <address> --to <address> --amount <amount>
* mine --address <minerAddress> --blocks <n>
* chain info
* chain print
* rpc --addr :8332

## Struktur Direktori

```text
cmd/indochain/main.go

internal/config/
  config.go

internal/crypto/
  keys.go
  address.go
  hash.go

internal/types/
  transaction.go
  block.go

internal/ledger/
  ledger.go

internal/chain/
  blockchain.go
  validation.go
  pow.go
  genesis.go

internal/storage/
  storage.go
  bbolt.go

internal/wallet/
  wallet.go
  store.go

internal/mempool/
  mempool.go

internal/rpc/
  server.go
  handlers.go

internal/cli/
  cli.go

tests atau internal/* tests:
  validation_test.go
  pow_test.go
  ledger_test.go
```

## Desain Inti

### Block

```go
height uint64
previous_hash string
timestamp int64
nonce uint64
difficulty uint32
miner_address string
transactions []Transaction
merkle_root string
hash string
```

### Transaction

```go
id string
from string
to string
amount uint64
fee uint64
nonce uint64
timestamp int64
signature string
public_key string
coinbase bool
```

## Aturan Nilai (Amount)

* dIDR memiliki 8 digit desimal.
* Simpan jumlah secara internal sebagai uint64 dalam satuan terkecil.
* 1 dIDR = 100000000 unit.
* CLI boleh menerima format desimal seperti `"1.5"` dan mengonversinya secara aman ke uint64.
* Jangan gunakan floating point untuk penyimpanan saldo.

## Genesis Block

* Height = 0.
* Previous hash kosong atau 64 karakter nol.
* Timestamp menggunakan konstanta tetap.
* Pesan genesis:

```text
IndoChain Genesis - fair CPU mining starts here
```

* Tidak ada premine secara default.
* Hash genesis block harus deterministik.

## Mining

* Target waktu block: 60 detik (placeholder).
* Difficulty awal: 4 karakter nol heksadesimal di depan hash.
* Proof-of-Work:

```text
hash(header block + nonce)
```

harus diawali sejumlah karakter `"0"` sesuai difficulty.

### Coinbase Reward

Reward awal:

```text
50 dIDR per block
```

### Coinbase Transaction

```text
from: "COINBASE"
to: miner address
amount: block reward
fee: 0
coinbase: true
```

## Difficulty

* Implementasikan sebagai nilai difficulty sederhana di dalam block.
* Untuk Fase 1, difficulty boleh dibuat rendah agar mudah diuji secara lokal.
* Tambahkan fungsi:

```go
CalculateNextDifficulty(chain)
```

tetapi tetap sederhana dan terdokumentasi untuk peningkatan di masa depan.

## Ledger

Gunakan model saldo berbasis akun.

Saldo dihitung dengan me-replay blockchain.

Validasi:

* transaksi non-coinbase harus memiliki signature yang valid
* saldo pengirim >= amount + fee
* amount > 0
* fee boleh 0 pada Fase 1
* coinbase hanya boleh muncul satu kali dalam satu block
* amount coinbase harus sama dengan reward block + total fee
* previous hash block harus sesuai dengan tip blockchain
* height block harus = tip height + 1
* hash PoW harus valid
* hash block harus sesuai hasil perhitungan
* merkle root harus sesuai dengan transaksi

### Nonce Akun

* Implementasikan tracking nonce untuk transaksi.
* Nonce transaksi pengirim harus sama dengan nonce akun saat ini + 1.
* Coinbase tidak menggunakan nonce akun.

## Wallet

* Gunakan ECDSA secp256k1 jika tersedia melalui library yang kompatibel.
* Jika hanya P256 yang mudah digunakan melalui standard library, gunakan P256 untuk Fase 1.
* Isolasikan implementasi kriptografi di `internal/crypto` agar mudah diganti ke secp256k1 di masa depan.
* Simpan wallet secara lokal pada:

```text
data/wallets.json
```

* Private key boleh disimpan hanya untuk kebutuhan pengembangan.

Tambahkan komentar peringatan:

```text
Phase 1 wallet storage is not production safe.
```

## Address

* Dibuat dari hash public key.
* Format:

```text
iND1 + suffix hash/heksadesimal
```

atau placeholder mirip bech32.

* Validasi prefix address.

## Storage

Direktori data default:

```text
./data
```

Simpan:

* blocks
* chain tip
* wallets
* transaksi pending/mempool (jika diperlukan)

Gunakan bbolt untuk data blockchain.

Simpan wallet dalam JSON terpisah agar lebih mudah di-debug.

## Perilaku CLI

### Inisialisasi Blockchain

```bash
go run ./node/cmd/indochain init
```

* Membuat direktori data.
* Membuat genesis block jika blockchain belum ada.
* Menampilkan pesan:

```text
chain initialized
```

### Membuat Wallet

```bash
go run ./node/cmd/indochain wallet new
```

* Membuat wallet baru.
* Menyimpannya.
* Menampilkan address.

### Menampilkan Wallet

```bash
go run ./node/cmd/indochain wallet list
```

* Menampilkan seluruh address yang tersimpan.

### Melihat Saldo

```bash
go run ./node/cmd/indochain balance <address>
```

* Menampilkan saldo terkonfirmasi dalam format desimal dIDR.

### Mengirim Transaksi

```bash
go run ./node/cmd/indochain send --from <address> --to <address> --amount 1.25
```

* Membuat transaksi yang ditandatangani menggunakan wallet lokal.
* Menambahkan transaksi ke mempool lokal.
* Menampilkan TX ID.

### Mining

```bash
go run ./node/cmd/indochain mine --address <minerAddress> --blocks 1
```

* Menambang block.
* Memasukkan transaksi pending dari mempool.
* Menambahkan reward coinbase.
* Menyimpan block.
* Menghapus transaksi yang sudah dimasukkan ke block.
* Menampilkan height dan hash block yang berhasil ditambang.

### Informasi Blockchain

```bash
go run ./node/cmd/indochain chain info
```

Menampilkan:

* height
* tip hash
* difficulty
* total supply
* jumlah transaksi pending

### Menampilkan Seluruh Blockchain

```bash
go run ./node/cmd/indochain chain print
```

* Menampilkan ringkasan seluruh block.

### Menjalankan RPC Server

```bash
go run ./node/cmd/indochain rpc --addr :8332
```

* Menjalankan HTTP JSON API.

## Endpoint RPC

```http
GET /health
GET /chain/info
GET /chain/blocks
GET /balance/{address}
POST /wallet/new
POST /send
POST /mine
```

### Body RPC /send

```json
{
  "from": "...",
  "to": "...",
  "amount": "1.25"
}
```

### Body RPC /mine

```json
{
  "address": "...",
  "blocks": 1
}
```

## Pengujian

Tambahkan unit test untuk:

* hash genesis yang deterministik
* validasi PoW menerima block yang valid
* validasi PoW menolak nonce/hash yang tidak valid
* ledger menolak saldo yang tidak mencukupi
* ledger menerima reward coinbase
* parser amount menangani angka desimal secara aman
* merkle root berubah ketika transaksi berubah

## README

Buat README.md yang berisi:

* Apa itu IndoChain
* Ruang lingkup Fase 1
* Cara menjalankan
* Contoh penggunaan CLI
* Peringatan: blockchain lokal eksperimental
* Roadmap:

```text
Phase 2 P2P
Phase 3 Difficulty + Mining Pool
Phase 4 Explorer + Faucet
Phase 5 Flutter Wallet
Phase 6 Public Testnet
Phase 7 Mainnet
```

## Perintah yang Diharapkan Setelah Implementasi

```bash
go mod tidy
go test ./...
go run ./node/cmd/indochain init
go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain mine --address <address> --blocks 1
go run ./node/cmd/indochain balance <address>
go run ./node/cmd/indochain chain info
```

## Catatan Akhir

Jangan melakukan over-engineering.

Fokus pada implementasi Fase 1 yang stabil, mudah dibaca, dan mudah dikembangkan di masa depan.
