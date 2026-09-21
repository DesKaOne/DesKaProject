Kamu sedang bekerja pada project Go monorepo IndoChain.

Status:

* Phase 2.6.6 iND Base58 Address Migration sudah diterapkan.
* `wallet new` sekarang menghasilkan address baru dengan format `iND...`.
* Mining ke address `iND...` berhasil.
* Saat node sedang berjalan, datadir lock mencegah command lokal `wallet new` menulis langsung ke datadir.
* Error lock memberi saran:
  `go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 wallet new`
* Tetapi command tersebut gagal:
  `remote mode not supported for this command yet`

Nama patch:
IndoChain Phase 2.6.6.1 — Remote Wallet Command UX Fix

Tujuan:
Memperbaiki UX wallet command saat datadir dikunci oleh running node.

Aturan penting:

* Jangan ubah address format.
* Jangan rollback iND Base58Check.
* Jangan ubah consensus.
* Jangan implement difficulty adjustment.
* Jangan implement coinbase maturity.
* Jangan rewrite besar.
* Semua test harus tetap pass:
  go work sync
  go test ./node/...

Masalah:
Ada inkonsistensi:

* Lock error menyarankan remote wallet command.
* Tetapi remote wallet command belum didukung.

Solusi yang dipilih:
Implement remote wallet command untuk local/admin RPC, atau jika terlalu berisiko, ubah lock hint agar tidak menyarankan command yang belum tersedia.

Prioritas:
Implement remote wallet commands minimal untuk local development.

==================================================

1. Tambahkan RPC endpoint wallet new
   ==================================================

Tambahkan endpoint RPC:

POST /wallet/new

Behavior:

* Generate wallet baru menggunakan crypto/address format aktif.
* Simpan wallet ke wallet store runtime node.
* Return address baru.
* Jangan return private key kecuali ada endpoint/flag eksplisit untuk export.
* Output address harus `iND...`.

Response:
{
"address": "iND...",
"format": "base58check",
"network": "localnet",
"key_curve": "secp256k1"
}

Security note:

* Endpoint ini hanya aman untuk local/admin node.
* Tambahkan TODO untuk auth/admin mode sebelum public RPC.
* Jika project sudah punya bind address detection, boleh reject wallet endpoints jika RPC tidak localhost.
* Minimal tambahkan docs warning:
  Do not expose wallet RPC on public RPC nodes.

==================================================
2. Tambahkan remote CLI support
===============================

Command ini harus bekerja:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 wallet new

Output:
iND...

Jangan print private key.

==================================================
3. Tambahkan wallet list remote jika mudah
==========================================

Jika wallet list sudah ada lokal, tambahkan remote:

wallet list

Remote:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 wallet list

Output:
addresses:

* iND...

Jika terlalu besar, minimal `wallet new` dulu.

==================================================
4. Tambahkan wallet inspect remote jika mudah
=============================================

Jika sudah ada lokal:

wallet inspect <address>

Remote:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 wallet inspect <address>

Output:
address: iND...
format: base58check
network: localnet
key curve: secp256k1

Jangan print private key.

Jika terlalu besar, boleh TODO.

==================================================
5. Update lock error hint
=========================

Jika remote wallet new berhasil, hint lock boleh tetap:

use:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:<port> wallet new

Jika remote wallet new tidak diimplement karena security concern, ubah hint menjadi:

stop the running node before modifying wallet files locally
or use a separate wallet datadir
remote wallet new is not supported yet

Jangan biarkan hint menyarankan command yang gagal.

==================================================
6. Wallet store runtime safety
==============================

Karena wallet new akan dipanggil lewat RPC saat node berjalan:

* Pastikan wallet store punya lock/mutex untuk create/save.
* Save wallet file secara atomic jika sudah ada helper atomic write.
* Duplicate address sangat tidak mungkin, tapi tetap aman.
* Jangan race dengan send/mine yang membaca wallet.

==================================================
7. Tests wajib
==============

Tambahkan/update tests:

1. Remote wallet new handler:

* POST /wallet/new returns address.
* address starts with iND.
* address validates.

2. Remote wallet new persists:

* Setelah wallet new RPC, wallet list/store berisi address tersebut.
* Restart/load wallet store jika test mudah.

3. CLI remote wallet new:

* `wallet new` with rpc-url calls RPC and prints address.
* Does not print private key.

4. Lock hint consistency:

* If datadir locked and command local wallet new attempted, hint command must be valid.
* No hint should suggest unsupported remote command.

5. Wallet store concurrent create:

* Multiple wallet new calls produce valid unique iND addresses.
* No file corruption.

6. Security docs:

* README mentions wallet RPC should not be exposed publicly.

==================================================
8. README update
================

Update README wallet section:

Local wallet:
go run ./node/cmd/indochain --datadir ./testdata/node1 wallet new

When node is running:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 wallet new

Warning:

* Wallet RPC creates private keys on the node.
* Do not expose wallet RPC on public RPC nodes.
* Public RPC nodes should disable wallet management in future hardening phase.

==================================================
9. Expected final commands
==========================

After patch:

go work sync
go test ./node/...

Manual test:

go run ./node/cmd/indochain --datadir ./testdata/address dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/address init
go run ./node/cmd/indochain --datadir ./testdata/address wallet new

Start node:

go run ./node/cmd/indochain --datadir ./testdata/address node start --rpc :8381 --p2p :9381 --advertise-p2p http://127.0.0.1:9381

Remote wallet new:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 wallet new

Expected:
iND...

Then send test:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 send --from <ADDR_A> --to <ADDR_B> --amount 10
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 mempool list --detail
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 mine --address <ADDR_A> --blocks 1
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 balance <ADDR_B>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8381 chain info

Expected:
balance B: 10 dIDR
normal transactions: 1
coinbase transactions: 4
total transactions: 5

Jangan over-engineer.
Fokus patch ini hanya:

* remote wallet new,
* lock hint consistency,
* wallet store runtime safety,
* README warning.
