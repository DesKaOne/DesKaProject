Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur project:

* node/

  * cmd/deskachain/
  * cmd/idrminer/
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

* Phase 1 sampai Phase 2.9.1 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... pass.
* Address final sudah aktif:

  * wallet baru menghasilkan address `IDR...`
  * format address: `IDR` + Base58Check
  * private key raw 32-byte hex
* Dynamic difficulty sudah aktif:

  * localnet target block time 10s
  * retarget window 10
  * next difficulty calculation valid
  * cumulative work memakai `16^difficulty`
* Coinbase maturity sudah aktif:

  * localnet maturity 10 blocks
  * mining reward belum langsung spendable
  * balance punya confirmed/mature/immature/spendable
  * send memakai spendable balance
  * circulating supply berdasarkan mature coinbase
* Standalone miner sudah aktif:

  * node punya `/miner/template`
  * node punya `/miner/submit`
  * `idrminer` bisa mine block via RPC
  * submit block sudah tidak deadlock
  * standalone miner bisa include mempool tx
  * chain validate pass

Nama patch:
DesKaChain Phase 2.10 — Node Public Hardening

Tujuan:
Memperkuat node sebelum masuk fase public testnet preparation.

Phase ini fokus pada:

* HTTP server hardening,
* RPC/P2P request size limits,
* basic rate limiting,
* max peers,
* peer score/ban improvement,
* graceful shutdown,
* config file/env foundation,
* safer public RPC mode,
* better operational logs,
* no consensus feature change.

Aturan penting:

* Jangan ubah address format `IDR...`.
* Jangan rollback Base58Check.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah cumulative work formula.
* Jangan ubah coinbase maturity.
* Jangan ubah standalone miner protocol secara breaking.
* Jangan implement GPU miner.
* Jangan implement mining pool.
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

1. HTTP server hardening
   ==================================================

Audit semua HTTP server:

* RPC server
* P2P server

Pastikan tidak lagi memakai raw:

http.ListenAndServe(addr, mux)

Gunakan:

http.Server{
Addr: addr,
Handler: mux,
ReadHeaderTimeout: ...,
ReadTimeout: ...,
WriteTimeout: ...,
IdleTimeout: ...,
MaxHeaderBytes: ...,
}

Default localnet:

RPC:

* ReadHeaderTimeout: 5s
* ReadTimeout: 15s
* WriteTimeout: 60s
* IdleTimeout: 60s
* MaxHeaderBytes: 1 MB

P2P:

* ReadHeaderTimeout: 5s
* ReadTimeout: 15s
* WriteTimeout: 30s
* IdleTimeout: 60s
* MaxHeaderBytes: 1 MB

Catatan:

* Jangan bikin timeout terlalu pendek sampai `/miner/submit` atau `mine` remote jadi timeout.
* Jika ada endpoint long-running, arahkan ke job/polling atau pastikan handler response cepat.
* Standalone miner `/miner/template` dan `/miner/submit` harus tetap bekerja.

==================================================
2. Request body size limits
===========================

Pastikan endpoint JSON punya body limit.

Tambahkan helper:

withMaxBodyBytes(next http.HandlerFunc, maxBytes int64)

Atau helper sejenis yang konsisten.

Default limits:

* Generic JSON RPC body: 1 MB
* Transaction submit: 1 MB
* Mempool tx receive: 1 MB
* Miner template: no large body / 128 KB
* Miner submit block: 8 MB
* P2P tx receive: 1 MB
* P2P block receive: 8 MB
* P2P headers/locator/common ancestor: 1 MB
* Peer add/connect: 128 KB

Jika body terlalu besar:

* return HTTP 413
* JSON error jelas:
  {
  "ok": false,
  "error": "request body too large"
  }

Jangan panic.

==================================================
3. Basic RPC rate limit
=======================

Tambahkan basic in-memory rate limiter per remote IP.

Tujuan:

* melindungi public RPC dari spam ringan.
* belum perlu distributed limiter.

Config default localnet:

* RPC requests per minute per IP: 300
* Miner template requests per minute per IP: 120
* Miner submit requests per minute per IP: 120
* Wallet management endpoints per minute per IP: 30
* P2P requests per minute per IP: 600

Implementation:

* token bucket sederhana atau fixed window.
* key berdasarkan remote IP.
* bersihkan entry lama supaya memory tidak bocor.
* localhost tetap kena limit, tapi limit cukup tinggi agar test tidak terganggu.
* Tambahkan option untuk disable rate limit di local dev jika perlu.

Response:

* HTTP 429
* JSON:
  {
  "ok": false,
  "error": "rate limit exceeded"
  }

Jangan over-engineer.

==================================================
4. Public RPC safety mode
=========================

Tambahkan config flag untuk membedakan node admin/local dan public RPC.

Flags:

* `--public-rpc` bool, default false.
* `--enable-wallet-rpc` bool, default true untuk local mode, false jika public-rpc true.
* `--enable-miner-rpc` bool, default true.
* `--enable-admin-rpc` bool, default true untuk local mode, false jika public-rpc true.

Rules:

* Jika `--public-rpc=true`:

  * wallet management RPC harus disabled by default:

    * wallet new
    * wallet import
    * wallet export
    * wallet list jika dianggap sensitif
  * admin/dev endpoints harus disabled:

    * dev reset
    * debug locks
    * unsafe debug endpoints
  * read-only endpoints tetap boleh:

    * chain info
    * chain difficulty
    * balance
    * tx get
    * block get jika ada
    * mempool list optional, boleh tetap enabled
  * miner endpoints boleh tetap enabled jika `--enable-miner-rpc=true`.

Error:
{
"ok": false,
"error": "endpoint disabled in public RPC mode"
}

CLI behavior:

* Jika remote wallet new dipanggil ke public RPC:

  * tampilkan error jelas.
* README warning:

  * jangan expose wallet RPC ke internet.

==================================================
5. Basic CORS config
====================

Tambahkan CORS middleware/config minimal jika belum ada.

Defaults:

* localnet dev:

  * allowed origins: localhost only or `*` if existing behavior needs it.
* public-rpc:

  * default allowed origins: empty or explicit config.
  * jangan asal allow credentials dengan wildcard.

Flags/env:

* `--cors-origins`
* env `IDR_CORS_ORIGINS`

Format:

* comma-separated origins.
* contoh:
  `http://localhost:3000,http://127.0.0.1:3000`

Jika terlalu besar untuk Phase ini, minimal tambahkan TODO dan jangan merusak existing RPC.

==================================================
6. Max peers and peer store hygiene
===================================

Tambahkan config:

* max peers default localnet: 32
* testnet placeholder: 128
* mainnet placeholder: 256

Rules:

* Peer add harus menolak jika max peers tercapai.
* Jangan simpan duplicate peer.
* Normalize peer URL.
* Reject invalid URL.
* Reject localhost/private peers jika network public mode dan config tidak mengizinkan private peers.
* Localnet tetap boleh private/localhost.

Peer list output tambahkan:

* peer count
* max peers
* banned count jika ada

==================================================
7. Peer score and temporary ban
===============================

Peer score sudah ada dari phase sebelumnya. Perkuat ringan.

Tambahkan:

* peer score min/max clamp.
* temporary ban jika score terlalu buruk.
* ban duration default: 10 minutes localnet/testnet.
* reject request from banned peer IP or peer URL jika bisa.
* peer status menampilkan:

  * score,
  * banned true/false,
  * ban until,
  * last error,
  * last seen.

Score penalty events:

* invalid block
* invalid tx
* wrong network id
* wrong chain id
* invalid handshake
* repeated timeout
* body too large
* malformed JSON

Score reward:

* successful handshake
* successful status response
* valid block/tx receive
* successful sync

Jangan membuat peer ban terlalu agresif sampai local testing susah.

==================================================
8. Network guard hardening
==========================

Pastikan semua P2P endpoint memvalidasi:

* network_id
* chain_id
* protocol_version jika tersedia
* genesis hash jika sudah exposed
* address/profile mismatch jika relevan

Jika peer network mismatch:

* reject jelas.
* score penalty.
* jangan import block/tx.

P2P handshake response harus mencantumkan:

* node_id
* network
* network_id
* chain_id
* protocol_version
* p2p_protocol_version
* genesis_hash
* height
* tip_hash
* cumulative_work
* public_rpc mode jika relevan

==================================================
9. Graceful shutdown
====================

Tambahkan graceful shutdown pada `node start`.

Saat Ctrl+C / SIGINT / SIGTERM:

* stop accepting new RPC/P2P request.
* shutdown HTTP servers dengan timeout, contoh 10s.
* stop sync loops.
* stop mining job jika built-in mining job sedang jalan.
* flush mempool to disk.
* flush peer store.
* release datadir lock.
* log:

  * shutdown started
  * rpc server stopped
  * p2p server stopped
  * mempool saved
  * datadir lock released
  * shutdown complete

Pastikan:

* setelah Ctrl+C, node bisa start lagi tanpa lock stale.
* jika shutdown abnormal tetap lock cleanup existing tidak rusak.

==================================================
10. Config file foundation
==========================

Tambahkan optional config file support untuk node.

Flag:
--config ./deska.yaml

Format boleh JSON/YAML/TOML. Pilih yang paling mudah dan dependency ringan.
Jika ingin tanpa dependency baru, gunakan JSON.

Minimal config JSON:

{
"network": "localnet",
"rpc": {
"addr": ":8331",
"public": false,
"enable_wallet": true,
"enable_miner": true,
"rate_limit_per_minute": 300,
"cors_origins": ["http://localhost:3000"]
},
"p2p": {
"addr": ":9331",
"advertise": "http://127.0.0.1:9331",
"max_peers": 32,
"allow_private_peers": true
},
"limits": {
"max_json_body_bytes": 1048576,
"max_block_body_bytes": 8388608
}
}

Rules:

* CLI flags override config file.
* Env override optional, not required.
* Existing CLI flags tetap bekerja tanpa config file.
* Jangan membuat config file wajib.
* Tambahkan `node config example` command jika mudah.
* Jika terlalu besar, buat docs/example config saja dan parsing minimal.

==================================================
11. Better operational logs
===========================

Tambahkan log yang berguna tapi tidak spam.

Node start:

* network
* chain id
* protocol version
* datadir
* rpc addr
* p2p addr
* public rpc true/false
* wallet rpc enabled/disabled
* miner rpc enabled/disabled
* max peers
* rate limit enabled/disabled
* body limits

Request error logs:

* invalid JSON
* body too large
* rate limited
* endpoint disabled
* peer banned
* invalid block/tx reason

Jangan log private key.
Jangan log full request body.

==================================================
12. Health endpoint
===================

Tambahkan atau rapikan endpoint:

GET /health

Response:

{
"ok": true,
"network": "localnet",
"chain_id": 777001,
"height": 12,
"tip_hash": "...",
"peers": 0,
"mempool": 0,
"public_rpc": false,
"uptime_seconds": 123
}

Harus cepat dan tidak mengambil lock berat.

==================================================
13. Readiness endpoint optional
===============================

Tambahkan jika mudah:

GET /ready

Response:

* ok true jika:

  * chain loaded,
  * storage reachable,
  * genesis exists,
  * node not shutting down.

Jika tidak mudah, boleh TODO.

==================================================
14. Metrics endpoint optional lightweight
=========================================

Tambahkan simple JSON metrics jika mudah:

GET /metrics

Bukan Prometheus dulu, cukup JSON:
{
"height": 12,
"peers": 2,
"mempool": 1,
"total_supply": "600",
"requests_total": 123,
"rate_limited_total": 2,
"blocks_received": 10,
"tx_received": 3
}

Jika terlalu besar, skip dan TODO.

==================================================
15. Miner RPC compatibility
===========================

Pastikan Phase 2.9 tetap valid:

* `/miner/template` tetap cepat.
* `/miner/submit` tetap cepat.
* idrminer --once tetap accepted.
* idrminer --max-blocks tetap jalan.
* body limit submit tidak menolak block normal.
* rate limit tidak memblok miner normal.
* public RPC mode tetap bisa enable miner endpoint jika flag enable.

Manual miner test wajib tetap pass.

==================================================
16. Existing command compatibility
==================================

Command lama wajib tetap bekerja:

* init
* wallet new local
* wallet new remote local mode
* balance
* send
* mempool list
* mine built-in
* node start
* peer list/add/check/status/sync
* chain info
* chain difficulty
* chain validate
* fork/reorg dev sims
* idrminer

Jangan breaking CLI output besar-besaran, tapi boleh tambah field.

==================================================
17. Tests wajib
===============

Tambahkan/update tests:

1. HTTP server timeout config:

* RPC server has ReadHeaderTimeout > 0.
* P2P server has ReadHeaderTimeout > 0.
* MaxHeaderBytes > 0.

2. Body limit generic:

* send oversized JSON to a limited endpoint.
* expect HTTP 413.

3. Body limit miner submit:

* oversized block submit rejected 413.
* normal block submit accepted.

4. Rate limit:

* configure very low limit.
* send more requests than limit.
* expect HTTP 429.
* after window/refill, request allowed again if testable.

5. Public RPC disables wallet:

* start handler config public_rpc true.
* POST /wallet/new returns endpoint disabled.
* chain info still works.

6. Public RPC miner allowed:

* public_rpc true + enable_miner true.
* /miner/template works.
* /miner/submit normal block works.

7. Public RPC miner disabled:

* public_rpc true + enable_miner false.
* /miner/template returns endpoint disabled.

8. CORS:

* allowed origin gets CORS header if configured.
* disallowed origin does not get allowed header if implemented.

9. Max peers:

* set max peers = 1.
* add first peer success.
* add second peer fails max peers reached.

10. Peer duplicate:

* add same peer twice.
* count remains 1.

11. Invalid peer URL:

* rejected.

12. Peer ban:

* apply repeated penalties.
* peer marked banned.
* banned peer status shows ban until.
* after ban duration if testable, unbanned.

13. Network guard:

* handshake wrong network/chain rejected.
* score penalty.

14. Graceful shutdown:

* start node in test if infrastructure exists.
* call shutdown.
* servers stop.
* lock released.
* node can start again with same datadir.
* If full test hard, at least unit test shutdown manager.

15. Health endpoint:

* /health returns ok true and height.
* endpoint fast.

16. Config file:

* load config file.
* flags override config.
* invalid config returns clear error.

17. Miner regression:

* idrminer --once equivalent integration or RPC submit test.
* accepted block.
* chain validate pass.

18. Existing tests still pass:

* address
* crypto
* ledger
* mempool
* p2p
* rpc
* cpuminer
* wallet
* chain
* cli

==================================================
18. README / docs update
========================

Update README.md and/or docs.

Current phase:
Phase 2.10 — Node Public Hardening

Tambahkan dokumentasi:

1. Local node:
   go run ./node/cmd/deskachain --datadir ./testdata/node node start --rpc :8331 --p2p :9331 --advertise-p2p http://127.0.0.1:9331

2. Public RPC mode:
   go run ./node/cmd/deskachain --datadir ./testdata/node node start --rpc :8331 --p2p :9331 --public-rpc --enable-wallet-rpc=false

3. Warning:

* Jangan expose wallet RPC ke internet.
* Public RPC sebaiknya hanya read-only + miner submit jika memang diperlukan.
* Wallet management harus local/admin only.
* Public node butuh firewall/reverse proxy/rate limit tambahan di production.

4. Health:
   curl http://127.0.0.1:8331/health

5. Miner tetap:
   go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:8331 --address <IDR_ADDR> --threads 4 --once

6. Config file example:
   docs/config.example.json

==================================================
19. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/deskachain --datadir ./testdata/harden dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/harden init
go run ./node/cmd/deskachain --datadir ./testdata/harden wallet new

Start local/admin node:

go run ./node/cmd/deskachain --datadir ./testdata/harden node start --rpc :8411 --p2p :9411 --advertise-p2p http://127.0.0.1:9411

Check:

Invoke-RestMethod http://127.0.0.1:8411/health

Expected:
ok true
network localnet
height 0

Remote wallet local mode:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8411 wallet new

Expected:
IDR...

Miner once:

go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:8411 --address <IDR_ADDR> --threads 4 --once

Expected:
submit accepted

Chain:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8411 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8411 chain validate

Expected:
chain valid

Public RPC mode test:

go run ./node/cmd/deskachain --datadir ./testdata/harden_pub node start --rpc :8421 --p2p :9421 --advertise-p2p http://127.0.0.1:9421 --public-rpc

Then:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8421 wallet new

Expected:
error: endpoint disabled in public RPC mode

Read-only:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8421 chain info

Expected:
works

Health:

Invoke-RestMethod http://127.0.0.1:8421/health

Expected:
ok true
public_rpc true

Ctrl+C shutdown:
Expected logs:
shutdown started
rpc server stopped
p2p server stopped
datadir lock released
shutdown complete

Then start same datadir again:
Expected:
no stale lock error

Jangan over-engineer.
Fokus Phase 2.10 hanya:

* server timeout,
* body limit,
* basic rate limit,
* public RPC safety,
* peer limits/ban,
* graceful shutdown,
* health endpoint,
* config foundation,
* miner regression compatibility.
