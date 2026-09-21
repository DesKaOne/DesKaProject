# IndoChain

IndoChain adalah blockchain kecil eksperimental berbasis CPU mining yang ditulis dengan Go.

Fase saat ini: Phase 4.12 - Peer Discovery & Auto Bootstrap.

Peringatan: IndoChain adalah software blockchain lokal yang masih eksperimental. Jangan gunakan wallet Phase 1.5 untuk dana sungguhan. Private key wallet disimpan secara lokal hanya untuk kenyamanan pengembangan.

Detail coin:

- Nama: IndoChain
- Ticker: dIDR
- Decimals: 8
- Unit terkecil: 1 dIDR = 100000000 unit

## Scope

Sudah termasuk:

- Genesis block deterministik
- Ledger berbasis account dengan tracking nonce
- Wallet lokal dan pembuatan address
- Mining proof-of-work lokal
- Penyimpanan chain dengan bbolt
- File wallet JSON dan mempool JSON untuk development lokal
- Perintah CLI
- JSON HTTP API dasar
- Perintah validasi chain
- Data directory per node dengan `--datadir`
- Alur transfer lokal dan mempool yang diperkuat
- Perintah lookup transaction dan inspect wallet
- Server HTTP P2P lokal untuk block sync, tx broadcast, dan block broadcast
- Runtime lock datadir untuk node yang sedang berjalan
- Kontrol CLI jarak jauh dengan `--rpc-url`
- `node_id` persisten per datadir
- P2P handshake dan guard network/genesis
- Header-first sync sebelum full block import
- Bounded peer score, fork detection, P2P advertise URL, dan two-way peer connect
- Runtime status cache, debug latency P2P, dan guarded auto-sync loop
- State mining job RPC sinkron, PoW yang bisa dibatalkan, dan timeout block broadcast yang dibatasi
- Statistik `chain info` canonical, atomic write mempool, guard duplicate tx mempool, dan fondasi timeout/body limit HTTP
- Format address final `iND...` Base58Check, key/signature secp256k1, network profile, dan metadata protocol version awal
- Dynamic difficulty localnet, validasi timestamp ringan, `chain difficulty`, dan cumulative work berbasis `16^difficulty`
- Coinbase maturity, mature/immature/spendable balance, dan circulating supply berbasis reward yang sudah mature
- Standalone CPU miner CLI (`indominer`) memakai RPC block template dan submit validation dari node
- Public RPC safety mode, rate limiting sederhana per IP, CORS allowlist, endpoint health/readiness, graceful shutdown, dan fondasi config file JSON
- Layer riset service node / kontribusi bandwidth dengan registration, heartbeat, challenge simulation lokal, scoring, dan simulated service points
- Standalone `indoservice` agent untuk safe-mode service-node simulation cycle
- Modul staking collateral untuk mengunci dIDR mature dan eligibility service node
- RPC faucet dev/testnet untuk transaksi funding normal yang signed dari wallet faucet matang
- Bootstrap seed peer public-testnet melalui flag, config, environment, dan seed file
- Backfill upstream peer public-testnet agar miner dapat push block ke head/explorer node tanpa mempercayai node tersebut sebagai otoritas consensus
- Guard isolated mining dan faucet/write di testnet dengan flag override eksplisit untuk pengujian lokal
- Script build dan packaging release untuk `indochain`, `indominer`, dan `indoservice`
- Workflow GitHub Actions CI dan artifact release untuk binary public-testnet
- Kandidat genesis public-testnet, profile deployment seed node, dan runbook deployment VPS
- Runbook deployment multi-host LAN, Tailscale, dan VPS public-testnet
- Runbook long-running public-testnet untuk restart recovery, offline catch-up, duplicate lock, dan systemd
- Runbook public-testnet terkontrol untuk faucet, staking collateral, dan service-node E2E multi-host
- Explorer UI/API read-only untuk data chain, block, transaction, address, stake, dan simulasi service lokal public-testnet
- Reputasi peer ringan, pemilihan outbound peer berbasis score, recovery cooldown, dan diagnostik RPC read-only `/p2p/peers` serta `/p2p/reputation`
- Maintenance automatic peer discovery/bootstrap dengan peer gossip, pruning TTL, retry backoff, limit subnet, dan diagnostik RPC read-only `/p2p/discovery`, `/p2p/bootstrap`, serta `/p2p/known-peers`

Belum termasuk:

- Smart contract
- EVM
- Keamanan wallet production
- Payout service node nyata, public proxy/VPN relay, staking, PoS, GPU mining, atau mining pool

## Dasar Protocol

IndoChain Phase 2.6.6 membekukan fondasi address/key/metadata protocol. Phase 2.7 menambahkan dynamic difficulty dan cumulative work yang lebih realistis. Phase 2.8 menambahkan coinbase maturity dan perhitungan mature balance. Phase 2.9 menambahkan standalone CPU miner yang menambang RPC block template tanpa private key wallet.

Phase 2.10 memperkuat node untuk persiapan public testnet tanpa mengubah consensus rule.

Phase 3.0 menambahkan riset kontribusi service node sebagai simulasi saja. PoW tetap satu-satunya pembuat canonical block; service points bukan dIDR, tidak spendable, dan tidak memengaruhi supply, difficulty, cumulative work, coinbase reward, balance, atau chain validation.

Phase 3.1 menambahkan `indoservice`, standalone safe-mode service node agent yang register ke node RPC, mengirim heartbeat, membuat/submit simulated challenge, mengambil score, dan menyimpan local agent state tanpa membuka proxy, relay, atau public listener.

Phase 3.2 menambahkan staking sebagai collateral saja. Ini bukan PoS, tidak membuat validator, tidak mencetak staking reward, dan tidak memengaruhi produksi block PoW.

Phase 3.2.1 menambahkan regression test staking dan consensus safety check. Staking tetap hanya collateral: bukan PoS, tanpa validator set, tanpa APY, tanpa dIDR staking reward, dan tanpa slashing di fase ini.

Phase 3.2.2 memisahkan plumbing runtime localnet dan testnet sebelum multi-node bootstrap. RPC, P2P, miner template, staking info, dan metadata collateral service sekarang menampilkan active network profile.

Phase 3.2.2.1 memfinalkan cleanup sebelum testnet. File runtime state di-ignore, path dokumen Architecture dibersihkan, jalur safe reorg preview/apply memakai active network profile, dan profile testnet sudah aman untuk pekerjaan bootstrap Phase 3.3.

Phase 3.3 menambahkan genesis testnet deterministik yang berbeda dari localnet, auto-resolve profile dari metadata datadir, flag startup `--bootnode`/`--bootnodes`, dan runbook controlled multi-node testnet di `docs/Testnet.md`.

Phase 3.3.1 memperkuat `peer sync` agar direct sync memvalidasi network ID, chain ID, genesis hash, dan kompatibilitas protocol peer sebelum boleh melaporkan local chain already up to date.

Phase 3.3.2 memperkuat controlled multi-node sync dengan normalisasi URL peer, persistensi bootnode yang ter-dedup setelah restart, field profil/status peer yang lebih lengkap, status offline/rejected dengan `last_error`, dan output sync yang menampilkan imported block count, height, serta tip hash.

Phase 3.4 menambahkan flow faucet dev/testnet. Faucet disabled secara default, hanya testnet saat diaktifkan, memakai wallet yang dikonfigurasi sebagai source, membuat transaksi signed normal ke mempool, membutuhkan mining untuk confirmed, dan tidak mencetak supply langsung.

Phase 3.4.1 menambahkan skenario end-to-end collateral service dari faucet: request 1000 dIDR testnet, mine transaksi faucet, lock 1000 dIDR sebagai collateral, mine transaksi stake, jalankan simulasi service, lalu verifikasi service score eligible. Lihat `docs/Faucet.md`, `docs/ServiceNode.md`, `docs/Staking.md`, dan `docs/Testnet.md`.

Phase 3.5 menambahkan packaging operator public-testnet: default public RPC yang aman, miner RPC harus diaktifkan eksplisit, startup/status summary lebih jelas, docs operator, contoh systemd/env, dan regression test circulating supply. Lihat `docs/Operator.md`.

Phase 3.6 menambahkan bootstrap seed peer public-testnet. Node dapat memuat seed peer yang dinormalisasi dari network profile, `--seed-peer`, `--seed-peers`, `IND_SEED_PEERS`, config `p2p.seed_peers`, atau `--seed-file`; seed disimpan di peer store dengan source `seed`, dan validasi peer normal tetap menolak network atau genesis yang salah. Lihat `docs/Operator.md` dan `docs/Testnet.md`.

Phase 3.7 menambahkan packaging build release public-testnet: metadata versi untuk semua binary, script build Windows/Linux, arsip release, checksum, dan quickstart docs. Lihat `docs/Release.md`.

Phase 3.8 menambahkan workflow GitHub Actions CI dan release artifact. CI menjalankan workspace sync dan test Go otomatis; workflow release membangun binary testnet Windows/Linux, menjalankan smoke test Linux, membuat archive, memverifikasi checksum, dan mengunggah artifact tanpa otomatis publish GitHub Release. Lihat `docs/Release.md`.

Phase 3.9 mendokumentasikan kandidat genesis public-testnet dan persiapan deployment seed node. Fase ini menambahkan profile deployment, runbook VPS, checklist preflight, contoh registry seed yang bisa diedit, dan catatan operator faucet/service node. Lihat `docs/TestnetGenesis.md`, `docs/DeployTestnet.md`, dan `docs/Preflight.md`.

Phase 4.0 mendokumentasikan deployment public-testnet multi-host melalui mode LAN, Tailscale, dan VPS. Fase ini memperjelas bind vs advertise address, firewall, diagnostik peer, env systemd seed node, dan validasi propagasi block antar host nyata. Lihat `docs/MultiHostTestnet.md`.

Phase 4.1 mendokumentasikan operasi public-testnet jangka panjang dan restart recovery. Fase ini mencakup graceful shutdown, cek datadir lock, persistensi peer, offline catch-up, perilaku restart systemd, observabilitas health/status, dan checklist long-run dua host. Lihat `docs/Systemd.md` dan `docs/LongRunTestnet.md`.

Phase 4.2 mendokumentasikan flow public-testnet terkontrol untuk faucet, staking collateral, dan simulasi service node lintas host. Transfer faucet adalah transaksi normal, stake collateral tersimpan di chain, service points tetap simulation-only, dan registration/score service adalah state simulasi lokal di node RPC yang dipakai service agent. Lihat `docs/Faucet.md`, `docs/Staking.md`, dan `docs/ServiceNode.md`.

Phase 4.3 menambahkan explorer API read-only di `/explorer/*` untuk ringkasan chain, block, transaction, address, stake record, dan data simulasi service lokal public-testnet.

Phase 4.4 menambahkan explorer web UI read-only yang di-embed di `/explorer-ui/`. UI ini menyediakan dashboard, blocks, detail block, detail transaction, history address, stake record, ringkasan simulasi service node, dan search tanpa aksi wallet/admin/write. Testnet dIDR tidak punya nilai moneter, mainnet belum tersedia, dan service points hanya simulasi. Lihat `docs/Explorer.md`.

Phase 4.5 memperkuat search dan pagination explorer. API read-only sekarang memiliki `/explorer/search?q=<query>`, metadata pagination untuk endpoint list, limit yang di-cap, error JSON explorer yang stabil, serta polish UI untuk search, tombol copy, empty state, dan paging. Lihat `docs/Explorer.md`.

Phase 4.6 menyiapkan `v0.4.6-testnet-rc1` sebagai public testnet release candidate. Fase ini menambahkan release notes, operator checklist, public quickstart, smoke scripts, panduan verifikasi artifact, update checklist seed node, dan safety copy publik final. Lihat `docs/ReleaseNotes-v0.4.6-testnet-rc1.md`, `docs/OperatorChecklist.md`, dan `docs/PublicTestnetQuickstart.md`.

Phase 4.7 menyiapkan RC1 untuk limited external tester dan publikasi GitHub pre-release. Fase ini menambahkan GitHub release checklist, body release siap tempel, tester onboarding guide, template bug report, feedback checklist, announcement draft, dan seed operator publish guide. Lihat `docs/GitHubReleaseChecklist.md`, `docs/GitHubRelease-v0.4.6-testnet-rc1.md`, `docs/TesterOnboarding.md`, `docs/TestnetFeedbackChecklist.md`, `docs/Announcement-v0.4.6-testnet-rc1.md`, dan `docs/SeedOperatorPublish.md`.

Phase 4.8 menambahkan workflow monitoring pasca-rilis dan planning RC2 untuk RC1. Fase ini mencakup script health check, monitoring seed, tracking known issues, issue triage, template feedback summary, snippet respons tester, dan saran label GitHub. Lihat `docs/PostReleaseMonitoring.md`, `docs/SeedMonitoringChecklist.md`, `docs/KnownIssues.md`, `docs/IssueTriage.md`, `docs/FeedbackSummaryTemplate.md`, `docs/RC2Planning.md`, `docs/TesterResponseSnippets.md`, dan `docs/GitHubLabels.md`.

Phase 4.9 memperkuat konektivitas public-testnet dengan repeated `--seed-peer`, bootstrap multi-seed, peer discovery hints, metadata health peer, diagnostik peer read-only, dan opsi peer di health-check script. Lihat `docs/TestnetTopology.md`, `docs/SeedMonitoringChecklist.md`, `docs/PostReleaseMonitoring.md`, dan `docs/PublicTestnetQuickstart.md`.

Phase 4.10 memperkuat stabilitas mining public-testnet tanpa mengubah consensus. `indominer` memiliki retry/backoff bounded, stale job detection, submit timeout control, miner stats, dan submit response yang lebih jelas; node mengekspos metrics read-only `/mining/*` dan CLI `mining status|difficulty|blocks`. Lihat `docs/Mining.md`, `docs/MiningStability.md`, dan `docs/DifficultyObservation.md`.

Phase 4.10.3 menambahkan upstream seed backfill dan guard isolated mining/write. Node miner dapat memakai `--upstream-peer` untuk push/backfill block yang sudah diterima ke VPS head/explorer node, sedangkan `--seed-peer` tetap untuk pull/discovery. Template miner dan write faucet testnet diblokir saat node isolated secara default. Cumulative work PoW tetap menjadi consensus; mainnet belum tersedia.

Phase 4.11 menambahkan reputasi peer ringan yang dipersist di `peers.json`. Sync sukses dan latency rendah menaikkan score peer, failure dan latency tinggi memberi penalti sementara, cooldown yang kedaluwarsa pulih otomatis, dan pemilihan outbound peer memprioritaskan effective reputation yang lebih tinggi tanpa mengubah consensus.

Phase 4.12 menambahkan maintenance automatic peer discovery dan bootstrap. Node secara periodik melakukan gossip daftar known peers, menemukan peer kompatibel, retry peer gagal dengan exponential backoff yang dibatasi, prune peer non-seed yang expired berdasarkan TTL, dan menerapkan limit peer/IP ringan sambil menjaga `peers.json` tetap backward compatible.

Format address:

- Address wallet baru memakai `iND` + payload Base58Check.
- Payload adalah `version byte + HASH160(compressed secp256k1 public key)`.
- Checksum adalah 4 byte pertama dari double SHA256 terhadap payload.
- Public key hash adalah `RIPEMD160(SHA256(compressed_public_key))`.
- Format address dev lama hanya legacy localnet. Wallet baru selalu membuat address Base58Check `iND...`, dan address legacy dev tidak valid untuk public testnet/mainnet.

Key dan signature:

- Export/import private key adalah raw 32-byte scalar hex, 64 karakter hex lowercase.
- Public key memakai compressed secp256k1 public key, 33 byte.
- Signature transaksi memakai secp256k1 ECDSA format DER.

Network profile:

- `localnet`: chain id `777001`, network id `ind-local-1`, address version `0x1E`, RPC `8331`, P2P `9331`, legacy dev address boleh.
- `testnet`: chain id `777101`, network id `ind-testnet-1`, address version `0x1F`, RPC `18331`, P2P `19331`, legacy dev address tidak boleh.
- `mainnet`: chain id `777000`, network id `ind-main-1`, address version `0x20`, RPC `8333`, P2P `9333`, legacy dev address tidak boleh.

Protocol version:

- protocol version: `1`
- block version: `1`
- tx version: `1`
- P2P protocol version: `ind-p2p/1`
- RPC API version: `v1`

Catatan public network: testnet dIDR tidak memiliki nilai ekonomi. Tidak ada janji harga, APY, atau profit. Proses claim mainnet di masa depan, jika dibuat, harus memiliki batas jumlah dan batas waktu.

## Difficulty

IndoChain tidak lagi bergantung pada fixed difficulty. Phase 2.7 menambahkan aturan retarget konservatif untuk eksperimen localnet/testnet.

Parameter localnet:

- initial difficulty: `4`
- min difficulty: `1`
- max difficulty: `8`
- target block time: `10s`
- retarget window: `10` block
- max future timestamp drift: `900s`

Difficulty naik maksimal 1 jika window terakhir terlalu cepat, turun maksimal 1 jika terlalu lambat, dan tetap jika masih dalam rentang normal. Cumulative work memakai `16^difficulty`, sehingga keputusan reorg membandingkan total work, bukan hanya height.

Command berguna:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain difficulty
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain info
```

## Coinbase Maturity

Reward mining langsung confirmed, tapi belum spendable sampai mature. Localnet memakai coinbase maturity `10` block; placeholder testnet dan mainnet memakai `100`.

- `confirmed balance`: semua saldo confirmed di canonical chain.
- `mature balance`: saldo confirmed yang boleh dibelanjakan.
- `immature balance`: reward mining confirmed yang masih terkunci maturity.
- `spendable balance`: mature balance dikurangi pending outgoing di mempool.

Command `send` memakai spendable balance. Miner yang menambang 3 block localnet punya `150 dIDR` confirmed, `0 dIDR` mature, `150 dIDR` immature, dan `0 dIDR` spendable. Pada height 11, reward height 1 sudah mature, sehingga miner punya `550 dIDR` confirmed, `50 dIDR` mature, `500 dIDR` immature, dan `50 dIDR` spendable.

`chain info` menampilkan total supply sebagai semua reward coinbase confirmed, sedangkan circulating supply adalah supply coinbase yang sudah mature. Circulating supply tetap mencakup coin mature yang sedang active/unlocking/released stake karena itu collateral milik owner. Spendable balance adalah field terpisah dan mengecualikan active/unlocking stake plus pending outgoing transaction.

## Standalone CPU Miner

Phase 2.9 menambahkan `indominer`, proses CPU miner terpisah. Miner hanya membutuhkan RPC URL node dan reward address. Miner tidak membaca file wallet, private key, atau datadir node; node yang membuat block template, memvalidasi submitted block, menyimpan block yang diterima, membersihkan transaction mempool yang sudah confirmed, dan broadcast block ke peer.

Start node:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/miner node start --rpc :8401 --p2p :9401 --advertise-p2p http://127.0.0.1:9401
```

Buat reward address melalui node lokal:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8401 wallet new
```

Mine satu block:

```powershell
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4 --once
```

Mine terus-menerus atau build binary:

```powershell
go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8401 --address <IND_ADDR> --threads 4
go build -o indominer ./node/cmd/indominer
```

`indominer` menampilkan job baru, hashrate, block ditemukan, stale template, retry, dan submit accepted. Coinbase maturity tetap berlaku untuk reward standalone miner.

## Public RPC Hardening

Mode lokal/admin tetap menjadi default agar workflow development tidak berubah. Untuk node yang diekspos ke jaringan publik, gunakan `--public-rpc`. Mode ini menonaktifkan wallet RPC, admin/debug RPC, miner RPC, faucet RPC, dan service write RPC secara default, sementara endpoint read-only chain tetap tersedia.

Contoh node lokal biasa:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331
```

Contoh public RPC dengan wallet/admin RPC tetap tertutup:

```powershell
go run ./node/cmd/indochain --network testnet --datadir ./data/testnet node start --public-rpc --rpc :18331 --p2p :19331 --advertise-p2p http://127.0.0.1:19331 --cors-origins https://explorer.example
```

Endpoint operasional:

```powershell
curl http://127.0.0.1:8331/health
curl http://127.0.0.1:8331/ready
```

Config file JSON awal tersedia di `docs/config.example.json`:

```powershell
go run ./node/cmd/indochain node start --config ./docs/config.example.json
```

Jangan expose wallet RPC ke internet kecuali benar-benar memahami risikonya. Public node sebaiknya read-only secara default. Tambahkan `--enable-miner-rpc` hanya ketika node memang harus menerima direct miner template/submit traffic. Dokumentasi operator ada di `docs/Operator.md`.

Bootstrap seed peer untuk node public-testnet:

```powershell
go run ./node/cmd/indochain --datadir ./data/testnet-public --network testnet node start --rpc :8811 --p2p :9811 --advertise-p2p http://<PUBLIC_HOST>:9811 --public-rpc --seed-file ./examples/testnet/testnet-seeds.txt
```

## Release Build

Build binary public-testnet:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.10-testnet-rc1
```

Package archive dan checksum:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.10-testnet-rc1 -SkipTests
```

Nama binary:

- `indochain`
- `indominer`
- `indoservice`

Untuk Linux/macOS:

```sh
sh ./scripts/build.sh v0.4.10-testnet-rc1
SKIP_TESTS=1 sh ./scripts/package.sh v0.4.10-testnet-rc1
```

GitHub Actions:

- `CI` berjalan untuk pull request, push ke `main`/`master`, dan manual dispatch.
- `Release Artifacts` bisa dijalankan manual dengan versi seperti `v0.4.10-testnet-rc1`; push tag `v*` juga membangun artifact.
- Artifact release berisi tiga archive plus `SHA256SUMS.txt`, dan sengaja mengecualikan datadir runtime, wallet, chain DB, peer/mempool store, state faucet/service, private key, `.git`, dan direktori build intermediate.

Dokumentasi release ada di `docs/Release.md`; dokumentasi deployment ada di `docs/DeployTestnet.md`; dokumentasi topologi testnet ada di `docs/TestnetTopology.md`; dokumentasi mining ada di `docs/Mining.md`, `docs/MiningStability.md`, dan `docs/DifficultyObservation.md`; dokumentasi multi-host ada di `docs/MultiHostTestnet.md`; dokumentasi systemd ada di `docs/Systemd.md`; dokumentasi long-run ada di `docs/LongRunTestnet.md`; dokumentasi explorer ada di `docs/Explorer.md`; dokumentasi faucet ada di `docs/Faucet.md`; dokumentasi staking ada di `docs/Staking.md`; dokumentasi service node ada di `docs/ServiceNode.md`; dokumentasi operator ada di `docs/Operator.md`; dokumentasi monitoring pasca-rilis ada di `docs/PostReleaseMonitoring.md`, `docs/SeedMonitoringChecklist.md`, `docs/KnownIssues.md`, `docs/IssueTriage.md`, `docs/FeedbackSummaryTemplate.md`, dan `docs/RC2Planning.md`.

## Simulasi Service Node

Phase 3.0 menambahkan layer service node khusus riset. Layer ini mencatat service node registration, heartbeat uptime, simulated verification challenge, komponen score, anti-abuse flags, dan simulated service points harian. Points ini hanya untuk riset lokal/testnet dan leaderboard; bukan dIDR dan tidak bisa dibelanjakan.

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service register --address <IND_ADDR> --endpoint http://127.0.0.1:9501
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service heartbeat --address <IND_ADDR> --endpoint http://127.0.0.1:9501
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service challenge create --address <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service challenge submit --challenge-id <ID> --latency-ms 50 --bytes-up 10000000 --bytes-down 50000000 --success true
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service score --address <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service rewards --address <IND_ADDR>
```

Dalam public RPC mode, service write endpoint mati secara default. Gunakan `--enable-service-rpc=true` hanya untuk verifier/test setup yang terkontrol.

## Service Node Agent

`indoservice` mengotomasi workflow simulasi service Phase 3.0. Agent hanya membutuhkan node RPC URL dan address dIDR. Agent tidak membaca private key, tidak mine PoW block, tidak membuat dIDR spendable, dan tetap berjalan dalam safe simulation mode.

```powershell
go run ./node/cmd/indochain --datadir ./testdata/service node start --rpc :8431 --p2p :9431 --advertise-p2p http://127.0.0.1:9431
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 wallet new
go run ./node/cmd/indoservice --rpc-url http://127.0.0.1:8431 --address <IND_ADDR> --endpoint http://127.0.0.1:9501 --once
go run ./node/cmd/indoservice --rpc-url http://127.0.0.1:8431 --address <IND_ADDR> --endpoint http://127.0.0.1:9501 --heartbeat-interval 30s --challenge-interval 60s
go build -o indoservice ./node/cmd/indoservice
```

Agent state default berada di `./indoservice-state.json`. Inspect dengan:

```powershell
go run ./node/cmd/indoservice status --state ./indoservice-state.json
```

Jangan aktifkan service RPC secara publik tanpa rate limit dan abuse protection. Service RPC Phase 3.1 ditujukan untuk controlled testnet/verifier simulation.

## Staking Collateral

Staking Phase 3.2 mengunci dIDR mature sebagai collateral service node. Active dan unbonding stake mengurangi spendable balance, tetapi confirmed balance, mature balance, total supply, coinbase reward, difficulty, dan PoW consensus tidak berubah.

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake lock --address <IND_ADDR> --amount 10
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake list --address <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8471 stake unlock --address <IND_ADDR> --stake-id <STAKE_ID>
```

Localnet memakai minimum stake 10 dIDR dan active stake 100 dIDR untuk eligibility simulasi service reward. Tidak ada slashing dan tidak ada APY staking pada Phase 3.2.

Untuk collateral service node testnet, threshold saat ini adalah 1000 dIDR. Runbook faucet-funded tersedia di `docs/Testnet.md`.

## Data Directory

Semua state lokal berada di `./data` secara default. Gunakan `--datadir` sebelum command untuk menjalankan node lokal yang terpisah:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 init
go run ./node/cmd/indochain --datadir ./testdata/node1 wallet new
go run ./node/cmd/indochain --datadir ./testdata/node1 chain info
```

## Clean Run

```powershell
go run ./node/cmd/indochain dev reset --yes
go run ./node/cmd/indochain init
go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain mine --address <addr> --blocks 3
go run ./node/cmd/indochain balance <addr>
go run ./node/cmd/indochain chain info
go run ./node/cmd/indochain chain validate
```

Total supply bisa lebih tinggi dari balance satu wallet karena block yang sudah ditambang sebelumnya bisa saja milik address miner lain.

## Contoh Transfer

Pending transaction tidak mengubah confirmed balance sampai transaction tersebut ditambang ke dalam block. Miner menerima 50 dIDR per block. Fee transaction masih 0 dIDR pada Phase 1.6.

```powershell
go run ./node/cmd/indochain dev reset --yes
go run ./node/cmd/indochain init
go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain mine --address <walletA> --blocks 11
go run ./node/cmd/indochain send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/indochain mempool list
go run ./node/cmd/indochain wallet inspect --address <walletA>
go run ./node/cmd/indochain wallet inspect --address <walletB>
go run ./node/cmd/indochain mine --address <walletA> --blocks 1
go run ./node/cmd/indochain balance <walletA>
go run ./node/cmd/indochain balance <walletB>
go run ./node/cmd/indochain chain validate
go run ./node/cmd/indochain tx get <txid>
```

Dalam alur tersebut, wallet A berakhir dengan 190 dIDR, wallet B berakhir dengan 10 dIDR, dan total supply menjadi 200 dIDR.

## Contoh Local P2P

Phase 2 menggunakan HTTP P2P lokal sederhana. Setiap node harus menggunakan datadir, RPC port, dan P2P port yang berbeda.

Saat `node start` berjalan, jangan jalankan command tulis lokal terhadap datadir yang sama dari process lain. Gunakan `--rpc-url` untuk mengontrol node yang sedang berjalan. IndoChain menulis `<datadir>/node.lock` saat node berjalan untuk mengurangi risiko accidental concurrent writer.

Siapkan dua node lokal di Windows PowerShell:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/node2 dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/node1 init
go run ./node/cmd/indochain --datadir ./testdata/node2 init
go run ./node/cmd/indochain --datadir ./testdata/node1 wallet new
go run ./node/cmd/indochain --datadir ./testdata/node2 wallet new
```

Start node 1 di terminal 1:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331
```

Start node 2 di terminal 2:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331
```

Mining di node 1 dari terminal 3:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 mine --address <walletNode1> --blocks 3
```

Connect peer dan sync node 2:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 chain validate
```

Expected: height dan tip hash node 2 sama dengan node 1, dan chain node 2 valid.

Contoh broadcast tx:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 send --from <walletNode1> --to <walletNode2> --amount 10
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/indochain --datadir ./testdata/node1 mine --address <walletNode1> --blocks 1
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 balance <walletNode2>
```

Alur remote-control yang disarankan untuk Phase 2.3:

```powershell
# Terminal 1
go run ./node/cmd/indochain --datadir ./testdata/node1 node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

# Terminal 2
go run ./node/cmd/indochain --datadir ./testdata/node2 node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331

# Terminal 3
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 peer list
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer list
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 node status
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 3 --timeout 5m
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 send --from <walletA> --to <walletB> --amount 10
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 mempool list
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <walletA> --blocks 1
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 balance <walletB>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain validate
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 chain validate
```

Expected: node 2 menerima tx broadcast, menerima block broadcast, balance wallet B menjadi 10 dIDR, kedua chain valid, dan `node compare` melaporkan `nodes in sync`.

Command debug broadcast Phase 2.3:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p test-broadcast --from <walletA> --to <walletB> --amount 10 --miner <walletA> --peer-rpc http://127.0.0.1:8332
```

Expected output berisi `tx broadcast: success=1 failed=0`, `block broadcast: success=1 failed=0`, `compare: nodes in sync`, dan `p2p broadcast test passed`.

Command latency/debug Phase 2.3.1:

```powershell
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9332/p2p/status

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p ping http://127.0.0.1:9332
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 p2p ping http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p debug
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 p2p debug
```

Expected: `/p2p/status` merespons cepat, `p2p ping` mencetak `result: ok`, dan `p2p debug` menampilkan latency peer plus `last_error`.

Command remote peer Phase 2.3 saat node sedang berjalan:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer sync --peer http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer status
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 node compare --peer http://127.0.0.1:8332
```

Kalau tidak sengaja menjalankan command tulis lokal terhadap datadir yang terkunci, CLI akan mencetak command `--rpc-url` yang sesuai untuk digunakan.

Keterbatasan Phase 2:

- Hanya local HTTP P2P
- Fork dideteksi dan ditolak, tapi belum ada automatic reorg
- Belum ada NAT traversal
- Public peer discovery sudah ditambahkan di Phase 4.9 melalui bounded `/p2p/peers` hints dan command manual `peer discover`
- Automatic background public peer discovery tetap dibatasi/deferred di luar flow manual Phase 4.9
- Belum aman untuk production

Catatan Phase 2.3.1:

- Setiap node menyimpan `<datadir>/node_id` secara persisten.
- `node start --advertise-p2p` menyimpan URL yang harus digunakan node lain untuk memanggil node ini.
- `peer add` menjalankan handshake sebelum menyimpan peer.
- `peer connect` menambahkan peer dan mencoba memperkenalkan node ini kembali melalui `POST /p2p/peer`.
- Peer dengan network ID, chain ID, genesis hash, atau protocol version yang berbeda/incompatible akan ditolak.
- Sync berjalan header-first: header diperiksa continuity-nya sebelum full block di-fetch.
- Peer score dibatasi dari -100 sampai 100. Peer buruk diberi score dan ditandai, tapi tidak di-ban permanen.
- Status RPC/P2P runtime memakai snapshot chain in-memory selama `node start`; validasi tetap membaca storage sebagai source of truth.
- Auto-sync loop memakai guard per peer dan throttle untuk timeout log yang berulang.
- Endpoint status, health, tip, dan handshake harus tetap ringan dan tidak melakukan outbound request ke peer.

Troubleshooting timeout P2P status:

```powershell
Invoke-RestMethod http://127.0.0.1:9331/p2p/health
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
Invoke-RestMethod http://127.0.0.1:9331/p2p/tip
Invoke-RestMethod http://127.0.0.1:9331/p2p/handshake

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p ping http://127.0.0.1:9332
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p debug
```

Kalau `context deadline exceeded` muncul untuk `/p2p/status`, cek dulu command direct `Invoke-RestMethod` di atas. Broadcast tx/block dan sync tidak secara sengaja menahan peer metadata lock saat melakukan outbound request.

Diagnostik mining Phase 2.3.2:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine --address <addr> --blocks 3 --timeout 5m

# Kalau mining terlihat macet, jalankan ini dari terminal lain:
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 mine status
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 debug locks
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 p2p debug
Invoke-RestMethod http://127.0.0.1:9331/p2p/status
```

Catatan mining:

- RPC `/mine` tetap sinkron untuk compatibility, tapi dilacak sebagai mining job.
- Hanya satu mining job yang bisa berjalan per node pada Phase 2.3.2.
- Loop PoW mengecek pembatalan request dan mendukung `--max-nonce`.
- Broadcast block ke peer yang lambat dibatasi per peer dan seharusnya tidak menggantung mining.
- Log node menampilkan mining start, per-block found/committed, broadcast summary, dan completion/failure.

Runtime hygiene Phase 2.3.3:

- Semua file runtime berada dalam scope `--datadir` yang dipilih: `chain.db`, `wallets.json`, `mempool.json`, `peers.json`, `node_id`, dan `node.lock`.
- `dev reset --yes` menghapus seluruh datadir yang dipilih, termasuk `peers.json` yang stale.
- `dev inspect` melaporkan state file runtime, chain height, peer count, dan wallet count tanpa mengambil node lock.
- `node start` mencetak path peer store dan peer source (`peers.json`, `flag`, atau `peers.json + flag`) setelah deduplication.
- `peer clear --yes` hanya menghapus peer; command ini tidak menghapus chain data atau wallet.
- Pada genesis, tip difficulty adalah `0` dan next difficulty adalah `4`. Block pertama yang ditambang dan log mining memakai next difficulty `4`.

Troubleshooting stale peer setelah reset:

```powershell
go run ./node/cmd/indochain --datadir ./testdata/node1 dev inspect
go run ./node/cmd/indochain --datadir ./testdata/node1 peer list --source
go run ./node/cmd/indochain --datadir ./testdata/node1 peer clear --yes
```

Expected setelah `dev reset --yes` dan `init`: `peers: 0`, dengan `peers.json: missing` atau file peer kosong.

Diagnostik fork Phase 2.4:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain locator
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332
```

Fase saat ini: Phase 4.12 - Peer Discovery & Auto Bootstrap.

Reorg adalah proses eksplisit untuk mengganti canonical branch lokal dengan branch peer yang memiliki cumulative work lebih besar. Reorg masih eksperimental untuk localnet/testnet, tidak otomatis secara default, dan `peer sync` normal tetap menolak fork kecuali `--allow-reorg --yes` diberikan. IndoChain memakai cumulative work untuk keputusan reorg, bukan hanya height, dan menolak branch peer dengan work yang sama atau lebih rendah. Preview bersifat dry-run saja; apply membutuhkan `--yes` eksplisit. Reorg preview/apply sekarang memvalidasi handshake, chain ID, network ID, dan replay consensus terhadap active network profile.

Phase 2.6 menambahkan pengecekan recovery mempool untuk reorg dengan transaction normal:

- Block yang dihapus saat reorg menjadi block orphan/disconnected.
- Transaction normal dari block orphan bisa dikembalikan ke mempool hanya jika masih valid terhadap ledger canonical baru.
- Coinbase transaction dari block orphan tidak pernah dikembalikan ke mempool.
- Transaction yang sudah confirmed oleh branch peer yang baru akan dihapus dari pending state, bukan di-requeue.
- Transaction yang conflict, duplicate, atau tidak valid akan di-drop.
- Account balance, nonce, dan total supply mengikuti canonical branch baru saja.

Contoh skenario recovery transaction reorg:

```powershell
go run ./node/cmd/indochain dev reorg-tx-sim --datadir-a ./testdata/reorgTxA --datadir-b ./testdata/reorgTxB --scenario requeue-valid

go run ./node/cmd/indochain --datadir ./testdata/reorgTxA node start --rpc :8351 --p2p :9351 --advertise-p2p http://127.0.0.1:9351
go run ./node/cmd/indochain --datadir ./testdata/reorgTxB node start --rpc :8352 --p2p :9352 --advertise-p2p http://127.0.0.1:9352

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8351 reorg preview --peer http://127.0.0.1:9352
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8351 reorg apply --peer http://127.0.0.1:9352 --yes
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8351 mempool list --detail
```

Expected untuk `requeue-valid`:

```text
requeued transactions: 1
pending tx count: 1
chain valid: true
```

Troubleshooting common ancestor Phase 2.4.1:

Kalau `chain common-ancestor --peer <p2p-url>` melaporkan `common ancestor not found` sementara `fork check` mengatakan node sudah sync, inspect locator persis yang dikirim:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain locator
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 chain common-ancestor --peer http://127.0.0.1:9332 --debug
```

Expected untuk node yang sudah sync:

```text
common ancestor found
height: <tip height>
hash: <tip hash>
```

Simulasi fork lokal Phase 2.4.2:

```powershell
go run ./node/cmd/indochain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3
go run ./node/cmd/indochain --datadir ./testdata/forkA fork inspect --other-datadir ./testdata/forkB
```

Expected:

```text
fork detected
common ancestor height: 0
reorg supported: false
```

Phase 2.4.2 hanya membuktikan bahwa fork nyata terdeteksi dan automatic sync menolaknya dengan aman. Automatic reorg masih dinonaktifkan; safe reorg direncanakan untuk Phase 2.5.

## Commands

Inisialisasi chain:

```powershell
go run ./node/cmd/indochain init
```

Reset data development lokal:

```powershell
go run ./node/cmd/indochain dev reset --yes
```

Inspect file runtime lokal:

```powershell
go run ./node/cmd/indochain dev inspect
```

Buat dan list wallet:

```powershell
go run ./node/cmd/indochain wallet new
go run ./node/cmd/indochain wallet list
```

Saat node sedang berjalan dan datadir terkunci, buat wallet melalui RPC local/admin:

```powershell
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 wallet new
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 wallet list
```

Wallet RPC membuat private key di node. Jangan expose wallet RPC pada public RPC node. Public RPC node sebaiknya menonaktifkan wallet management pada fase hardening berikutnya.

Export private key development lokal:

```powershell
go run ./node/cmd/indochain wallet export --address <addr> --show-private-key
```

Inspect wallet atau address:

```powershell
go run ./node/cmd/indochain wallet inspect --address <addr>
go run ./node/cmd/indochain address validate <addr>
```

Cek balance:

```powershell
go run ./node/cmd/indochain balance <addr>
```

Buat pending transaction:

```powershell
go run ./node/cmd/indochain send --from <fromAddress> --to <toAddress> --amount 1.25
```

Inspect atau clear mempool:

```powershell
go run ./node/cmd/indochain mempool list
go run ./node/cmd/indochain mempool clear --yes
```

Lookup transaction:

```powershell
go run ./node/cmd/indochain tx get <txid>
```

Mining block:

```powershell
go run ./node/cmd/indochain mine --address <addr> --blocks 3
```

Inspect dan validasi chain:

```powershell
go run ./node/cmd/indochain chain info
go run ./node/cmd/indochain chain locator
go run ./node/cmd/indochain chain print
go run ./node/cmd/indochain chain validate
go run ./node/cmd/indochain chain common-ancestor --peer http://127.0.0.1:9332
```

Start HTTP API:

```powershell
go run ./node/cmd/indochain rpc --addr :8332
```

Start node gabungan RPC + P2P:

```powershell
go run ./node/cmd/indochain node start --rpc :8332 --p2p :9332 --advertise-p2p http://127.0.0.1:9332 --peers http://127.0.0.1:9331
```

Tampilkan network identity:

```powershell
go run ./node/cmd/indochain network info
```

Cek status node:

```powershell
go run ./node/cmd/indochain node status
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 node status
```

Kelola peer:

```powershell
go run ./node/cmd/indochain peer list
go run ./node/cmd/indochain peer list --source
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer check http://127.0.0.1:9331
go run ./node/cmd/indochain peer add http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer connect http://127.0.0.1:9331
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer status
go run ./node/cmd/indochain peer remove http://127.0.0.1:9331
go run ./node/cmd/indochain peer clear --yes
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer clear --yes
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8332 peer sync
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8331 fork check --peer http://127.0.0.1:9332
go run ./node/cmd/indochain chain info
```

Menjalankan `peer sync` berulang kali tidak boleh menduplikasi block: height, total supply, dan tip hash harus tetap sama setelah node up to date.

## RPC Endpoints

- `GET /health`
- `GET /ready`
- `GET /network/info`
- `GET /node/id`
- `GET /node/compare?peer=<rpc-url>`
- `GET /node/status`
- `GET /debug/p2p`
- `POST /debug/p2p/ping`
- `GET /debug/locks`
- `GET /peers`
- `GET /p2p/peers`
- `GET /p2p/reputation`
- `GET /p2p/discovery`
- `GET /p2p/bootstrap`
- `GET /p2p/known-peers`
- `POST /peers`
- `POST /peers/connect`
- `POST /peers/clear`
- `POST /peers/sync`
- `POST /peers/status`
- `GET /chain/info`
- `GET /chain/difficulty`
- `GET /chain/locator`
- `GET /chain/blocks`
- `GET /chain/validate`
- `POST /fork/check`
- `GET /faucet/info`
- `POST /faucet/request`
- `GET /miner/template?address=<IND_ADDR>`
- `POST /miner/submit`
- `POST /service/register`
- `POST /service/heartbeat`
- `POST /service/challenge/create`
- `POST /service/challenge/submit`
- `GET /service/score?address=<IND_ADDR>`
- `GET /service/rewards?address=<IND_ADDR>`
- `GET /service/list`
- `GET /stake/info`
- `GET /stake/list`
- `GET /stake/status?id=<STAKE_ID>`
- `POST /stake/lock`
- `POST /stake/unlock`
- `GET /balance/{address}`
  - Mengembalikan `balance` untuk kompatibilitas lama plus `confirmed_balance`, `mature_balance`, `immature_balance`, `spendable_balance`, `pending_outgoing`, `pending_incoming`, `coinbase_maturity`, dan `current_height`.
- `GET /address/{address}`
- `GET /tx/{txid}`
- `GET /mempool`
- `POST /mempool/clear`
- `GET /wallets`
- `POST /wallet/new`
- `POST /send`
- `POST /mine`
- `GET /mine/status`

## P2P Endpoints

- `GET /p2p/handshake`
- `GET /p2p/status`
- `GET /p2p/headers`
- `GET /p2p/locator`
- `GET /p2p/block/{height}`
- `POST /p2p/tx`
- `POST /p2p/block`
- `POST /p2p/peer`
- `POST /p2p/common-ancestor`

P2P endpoints:

- `GET /p2p/health`
- `GET /p2p/handshake`
- `GET /p2p/status`
- `GET /p2p/tip`
- `GET /p2p/headers?from=<height>&limit=<n>`
- `GET /p2p/locator`
- `GET /p2p/block/{height}`
- `GET /p2p/blocks?from=<height>&limit=<n>`
- `POST /p2p/common-ancestor`
- `POST /p2p/tx`
- `POST /p2p/block`

Contoh body send:

```json
{
  "from": "<IND_ADDR>",
  "to": "<IND_ADDR>",
  "amount": "1.25"
}
```

Contoh body mine:

```json
{
  "address": "<IND_ADDR>",
  "blocks": 1
}
```

## Validation

```powershell
go mod tidy
go test ./...
go run ./node/cmd/indochain chain validate
```

## Roadmap

- Phase 2: P2P
- Phase 3: Difficulty + mining pool
- Phase 4: Explorer + faucet
- Phase 5: Flutter wallet
- Phase 6: Public testnet
- Phase 7: Mainnet

## Phase 2.4.3 — Same Height Fork Sync Fix & Chain Info Tx Count

Height yang sama tidak berarti chain yang sama. Node hanya dianggap up to date ketika height dan tip hash sama. Jika dua node memiliki height yang sama tapi tip hash berbeda, IndoChain menganggapnya sebagai fork. Peer sync menolak forked chain sampai Phase 2.5 memperkenalkan safe automatic reorg support.

Contoh fork simulation dan sync rejection:

```bash
go run ./node/cmd/indochain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 3

go run ./node/cmd/indochain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341

go run ./node/cmd/indochain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8341 peer add http://127.0.0.1:9342
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8341 peer sync
```

Expected:

```text
sync failed: fork detected
common ancestor height: 0
automatic reorg: disabled
```

### Workflow safe reorg eksperimental

Buat local chain yang fork:

```sh
go run ./node/cmd/indochain dev fork-sim --datadir-a ./testdata/forkA --datadir-b ./testdata/forkB --blocks-a 3 --blocks-b 5
```

Start kedua node:

```sh
go run ./node/cmd/indochain --datadir ./testdata/forkA node start --rpc :8341 --p2p :9341 --advertise-p2p http://127.0.0.1:9341
go run ./node/cmd/indochain --datadir ./testdata/forkB node start --rpc :8342 --p2p :9342 --advertise-p2p http://127.0.0.1:9342
```

Preview dan apply safe reorg secara eksplisit:

```sh
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8341 reorg preview --peer http://127.0.0.1:9342
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8341 reorg apply --peer http://127.0.0.1:9342 --yes
```

Verifikasi:

```sh
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8341 node compare --peer http://127.0.0.1:8342
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8341 chain validate
```

Expected output berisi `nodes in sync` dan `chain valid`.
