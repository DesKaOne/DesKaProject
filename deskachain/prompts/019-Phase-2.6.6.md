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

* Phase 1 sampai Phase 2.6.5 sudah selesai.
* go work sync berhasil.
* go test ./node/... harus pass.
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
  * mempool atomic write/duplicate guard/basic safety cleanup.

Nama patch:
DesKaChain Phase 2.6.6 — Protocol Spec Freeze & DKC Base58 Address Migration

Tujuan:
Memfinalkan fondasi protokol sebelum project makin besar, terutama:

* format address final,
* format private key,
* crypto key/signature direction,
* network profile foundation,
* protocol versioning,
* backward compatibility lokal untuk address lama `dkc1...`.

Setelah phase ini, wallet baru harus memakai address format baru:
DKC + Base58Check(...)

Aturan penting:

* Jangan implement difficulty adjustment.
* Jangan implement coinbase maturity.
* Jangan implement standalone miner.
* Jangan implement staking.
* Jangan implement PoS.
* Jangan implement bandwidth mining.
* Jangan implement explorer.
* Jangan rewrite besar.
* Jangan mengubah fitur reorg besar.
* Jangan menghapus command lama.
* Patch incremental, tapi address/crypto boleh dimigrasi dengan rapi.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...

==================================================

1. Final address format
   ==================================================

Ganti format address baru dari format dev lama:

dkc1 + sha256(publicKeyHex)[:40]

menjadi format final:

DKC + Base58Check(payload)

Format:

* Human prefix literal: `DKC`
* Encoding body: Bitcoin-style Base58 alphabet
* Payload sebelum checksum:
  version byte      1 byte
  public key hash   20 bytes
* Checksum:
  first 4 bytes of double SHA256(payload)
* Base58Check body:
  base58(payload + checksum)
* Final address:
  "DKC" + base58(payload + checksum)

Contoh bentuk:
DKCxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

Catatan:

* Jangan mengurangi ukuran public key hash hanya demi membuat panjang address sama persis dengan Bitcoin/Dogecoin.
* Panjang address boleh sedikit berbeda, yang penting aman dan konsisten.
* Address harus case-sensitive.
* Prefix `DKC` harus uppercase.
* Lowercase `dkc...` untuk format baru harus invalid, kecuali legacy `dkc1...` dev address dalam mode compatibility.

Tambahkan package/helper address, misalnya:
internal/address
atau:
internal/crypto/address.go

Fungsi minimal:
EncodeAddress(pubKeyBytes []byte, profile NetworkProfile) (string, error)
DecodeAddress(addr string, profile NetworkProfile) (AddressPayload, error)
ValidateAddress(addr string, profile NetworkProfile) bool
IsLegacyDevAddress(addr string) bool

AddressPayload minimal:
Version byte
PubKeyHash []byte

==================================================
2. Public key hash
==================

Untuk address baru, gunakan:

HASH160 = RIPEMD160(SHA256(compressed_public_key))

Jika belum ada dependency RIPEMD160, tambahkan:
golang.org/x/crypto/ripemd160

Jangan gunakan hex string public key sebagai input hash jika sudah ada raw compressed public key bytes.

Expected:

* pubkey compressed bytes -> SHA256 -> RIPEMD160 -> 20-byte pubkey hash.

==================================================
3. Base58Check implementation
=============================

Tambahkan Base58 encode/decode internal.

Boleh implement sendiri di internal package.
Jangan pakai library tidak jelas.

Alphabet:
123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz

Fungsi:
Base58Encode([]byte) string
Base58Decode(string) ([]byte, error)
Base58CheckEncode(payload []byte) string
Base58CheckDecode(encoded string) ([]byte, error)

Validasi:

* decode gagal jika karakter invalid.
* decode gagal jika checksum mismatch.
* decode gagal jika payload terlalu pendek.
* encode/decode harus preserve leading zero bytes.

Tests wajib untuk Base58:

* roundtrip payload.
* invalid char rejected.
* checksum mismatch rejected.
* leading zero roundtrip.

==================================================
4. Crypto key curve decision
============================

Saat ini project kemungkinan memakai ECDSA P-256 dari Go standard library.

Untuk protocol freeze, migrasi ke secp256k1 agar lebih crypto-native dan lebih dekat dengan ekosistem Bitcoin/Dogecoin/Ethereum.

Gunakan library Go yang stabil, misalnya:
github.com/btcsuite/btcd/btcec/v2
github.com/btcsuite/btcd/btcec/v2/ecdsa

Atau library secp256k1 lain yang mature dan umum dipakai.
Pilih satu dependency yang stabil dan rapi.

Target:

* Private key tetap raw 32-byte hex.
* Public key gunakan compressed public key 33 bytes untuk address.
* Signing pakai secp256k1 ECDSA.
* Verify pakai secp256k1 ECDSA.
* Signature format harus konsisten, misalnya DER atau compact, sesuai existing code yang paling mudah.
* Jangan hash private key saat export/import.
* Private key export/import = raw private key 32 bytes encoded hex 64 chars.

Jika existing code menyebut “SHA256 private key”, luruskan di docs:
private key is a raw 32-byte scalar encoded as hex.

Fungsi minimal:
GeneratePrivateKey() (*PrivateKey, error)
PrivateKeyFromHex(hex string) (*PrivateKey, error)
PrivateKeyToHex(priv) string
PublicKeyBytes(priv) []byte
AddressFromPrivateKey(priv, profile) string
Sign(hash/message) []byte
Verify(pubKeyBytes, hash/message, signature) bool

Pastikan semua transaksi baru memakai key/signature secp256k1.

==================================================
5. Backward compatibility untuk legacy dev address
==================================================

Address lama:
dkc1...

harus dianggap legacy dev address.

Aturan:

* wallet baru harus selalu generate address baru `DKC...`.
* address validation default menerima address baru.
* legacy `dkc1...` boleh diterima hanya untuk localnet/dev compatibility jika existing tests/data masih butuh.
* Jangan jadikan legacy address sebagai format utama.
* Tambahkan warning di wallet inspect/export jika wallet masih memakai legacy address.
* README harus menjelaskan:
  `dkc1...` adalah legacy dev address dan tidak untuk public testnet/mainnet.

Jika terlalu rumit migrasi otomatis, cukup:

* wallet load tetap bisa membaca wallet lama.
* wallet new menghasilkan wallet baru.
* command balance bisa membaca legacy address localnet.
* command send ke legacy address boleh localnet saja, tapi docs warning.
* mining ke legacy address boleh hanya jika wallet lama ada dan localnet compatibility aktif.

==================================================
6. Network profile foundation
=============================

Tambahkan network profile terpusat.

Misalnya di:
internal/config/network.go

NetworkProfile minimal:
Name string
ChainID uint64
NetworkID string
AddressPrefix string
AddressVersion byte
LegacyAddressAllowed bool
DefaultRPCPort int
DefaultP2PPort int
ProtocolVersion uint32
RPCAPIVersion string
P2PProtocolVersion string
BlockVersion uint32
TxVersion uint32

Profiles minimal:

1. localnet
   Name: localnet
   ChainID: 777001
   NetworkID: dkc-local-1
   AddressPrefix: DKC
   AddressVersion: 0x1E
   LegacyAddressAllowed: true
   DefaultRPCPort: 8331
   DefaultP2PPort: 9331

2. testnet
   Name: testnet
   ChainID: 777002
   NetworkID: dkc-test-1
   AddressPrefix: DKC
   AddressVersion: 0x1F
   LegacyAddressAllowed: false
   DefaultRPCPort: 18331
   DefaultP2PPort: 19331

3. mainnet
   Name: mainnet
   ChainID: 777000
   NetworkID: dkc-main-1
   AddressPrefix: DKC
   AddressVersion: 0x20
   LegacyAddressAllowed: false
   DefaultRPCPort: 8333
   DefaultP2PPort: 9333

Catatan:

* Untuk Phase ini boleh semua tetap memakai prefix visual `DKC`.
* Perbedaan network minimal ada di address version dan network id.
* Later phase bisa mempertimbangkan prefix visual berbeda untuk testnet/localnet jika perlu.
* Jangan biarkan config tersebar hardcode `config.Localnet()` di banyak tempat tanpa alasan.
* Existing flag:
  --network localnet
  harus memakai profile ini.

==================================================
7. Protocol versioning
======================

Tambahkan constants/profile fields:

* BlockVersion
* TxVersion
* P2PProtocolVersion
* RPCAPIVersion
* ProtocolVersion

Jika struct Block/Tx sudah punya version field:

* pakai profile default.
  Jika belum:
* tambahkan field Version secara backward-compatible jika aman.
* Jika terlalu invasif untuk storage lama, tambahkan TODO dan minimal expose protocol versions di node status/handshake.

Target minimal Phase ini:

* /p2p/handshake menampilkan:
  network_id
  chain_id
  protocol_version
  p2p_protocol_version
* /node/status atau /chain/info menampilkan:
  network
  chain_id
  protocol_version
  rpc_api_version
* README mencatat protocol version awal.

Jangan memaksakan storage migration besar jika berisiko.

==================================================
8. Update wallet commands
=========================

Update command:

wallet new

Expected:

* output address baru `DKC...`
* private key tetap bisa diexport hex 64 chars jika command export ada.
* wallet file menyimpan cukup data untuk recover address.
* public key compressed bytes atau hex disimpan jika existing wallet model butuh.

Update:
wallet inspect
wallet list
wallet export
wallet import

Expected:

* address baru valid.
* import private key hex menghasilkan address yang sama.
* legacy wallet diberi label/warning jika ada.

Output contoh:
address: DKC...
format: base58check
network: localnet
key curve: secp256k1

Jangan menampilkan private key kecuali command export memang eksplisit.

==================================================
9. Update send/balance/mining validation
========================================

Semua command yang menerima address harus memakai ValidateAddress:

* balance <address>
* send --from/--to
* mine --address
* mempool detail display
* reorg tx sim
* dev fork-sim
* dev reorg-tx-sim

Rules:

* New address DKC Base58Check valid.
* Wrong checksum invalid.
* Wrong version invalid untuk selected network.
* Wrong prefix invalid.
* Legacy dkc1 accepted only if profile.LegacyAddressAllowed == true.
* Invalid address error harus jelas:
  invalid address: checksum mismatch
  invalid address: wrong network version
  invalid address: unsupported legacy address

==================================================
10. Update transactions and signatures
======================================

Pastikan transaction signing dan verification tetap benar setelah key migration.

Jika tx menyimpan public key:

* simpan compressed public key hex.
* verify address derived from public key matches tx.From.
* verify signature secp256k1.

Jika existing tx format tidak menyimpan public key:

* jangan rewrite besar.
* lakukan perubahan minimal agar send/mine/validate tetap benar.
* Tambahkan TODO jika ada kelemahan existing design.

Tests:

* tx signed with private key verifies.
* tampered tx fails verify.
* tx from address mismatch fails.
* imported private key can sign tx from derived address.

==================================================
11. Update reorg/dev simulation
===============================

Commands yang membuat wallet/address otomatis harus memakai format baru:

* dev fork-sim
* dev reorg-tx-sim
* tests reorg
* tests mempool
* tests mining

Expected output:
miner: DKC...
user1: DKC...
user2: DKC...

Jangan biarkan test helper generate `dkc1...` lagi kecuali explicit legacy test.

==================================================
12. Docs update
===============

Update:

* README.md
* Roadmap.md
* docs/Whitepaper.md jika ada
* docs/Wallet.md jika ada
* docs/Protocol.md buat baru jika belum ada

Tambahkan section:

Address format:

* New format:
  DKC + Base58Check(version + pubkey_hash + checksum)
* PubKeyHash:
  RIPEMD160(SHA256(compressed secp256k1 public key))
* Private key:
  raw 32-byte hex
* Legacy:
  dkc1... is localnet/dev legacy only

Network profiles:

* localnet
* testnet
* mainnet

Protocol versions:

* block version
* tx version
* p2p protocol version
* rpc api version

Catatan publik:

* Testnet DKC has no monetary value.
* Mainnet claim, if implemented, will be capped and time-limited.
* No price promise.
* No APY promise.

==================================================
13. Tests wajib
===============

Tambahkan/update tests:

1. Base58 roundtrip:

* payload encode/decode sukses.

2. Base58 invalid char:

* decode invalid char gagal.

3. Base58 checksum mismatch:

* address body dimodifikasi lalu decode gagal.

4. Address encode:

* generated address starts with DKC.
* address validates.

5. Address wrong prefix:

* lowercase/new wrong prefix invalid.

6. Address wrong checksum:

* invalid.

7. Address wrong network version:

* address localnet tidak valid jika profile testnet, kecuali desain membolehkan explicit cross-network false.

8. Legacy address:

* `dkc1...` valid hanya di localnet if LegacyAddressAllowed true.
* invalid di testnet/mainnet.

9. Private key hex:

* generated private key export length 64 hex chars.
* import private key returns same public key/address.
* invalid length rejected.
* invalid scalar rejected.

10. secp256k1 sign verify:

* sign message/tx hash.
* verify success.
* tampered message fails.
* wrong pubkey fails.

11. Wallet new:

* generated wallet address starts DKC.
* wallet inspect shows base58check/secp256k1.

12. Wallet import/export:

* export private key.
* import into fresh wallet.
* address same.

13. Send tx:

* send from DKC address to DKC address.
* signature validates.
* balance changes after mining.

14. Mining address validation:

* mine to DKC address success.
* mine to invalid checksum address fails.

15. RPC address validation:

* remote send/balance/mine reject invalid address.

16. Dev fork-sim:

* generated miner addresses start DKC.

17. Reorg tx sim:

* miner/user addresses start DKC.
* scenarios still pass.

18. P2P handshake:

* includes network_id, chain_id, protocol version.

19. Node status:

* includes network/profile/protocol version.

20. Existing Phase 2.6.5 stats tests still pass:

* chain info stats correct.

==================================================
14. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual wallet test:

go run ./node/cmd/deskachain --datadir ./testdata/address dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/address init
go run ./node/cmd/deskachain --datadir ./testdata/address wallet new

Expected:
DKC...

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/address node start --rpc :8381 --p2p :9381 --advertise-p2p http://127.0.0.1:9381

Mine:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8381 mine --address <DKC_ADDRESS> --blocks 3

Expected:
mined block height=1 ...
miner balance: 150 DKC

Chain info:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8381 chain info

Expected:
height: 3
total supply: 150 DKC
network: localnet
chain id: 777001
protocol version: ...

Send test:

go run ./node/cmd/deskachain --datadir ./testdata/address wallet new
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8381 send --from <ADDR_A> --to <ADDR_B> --amount 10

Then mine 1 block:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8381 mine --address <ADDR_A> --blocks 1

Expected:
balance B: 10 DKC
normal transactions: 1

Dev reorg tx sim:

go run ./node/cmd/deskachain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario requeue-valid

Expected output addresses:
miner common: DKC...
user1: DKC...
user2: DKC...

Do not over-engineer.
Focus Phase 2.6.6 only on:

* DKC Base58Check address,
* secp256k1 key/signature migration,
* raw 32-byte private key hex,
* network profile foundation,
* protocol version metadata,
* validation updates,
* docs/tests.
