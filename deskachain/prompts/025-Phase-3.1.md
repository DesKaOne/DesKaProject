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
  * testdata/
  * data/

Status saat ini:

* Phase 1 sampai Phase 3.0 sudah selesai dan valid.
* go work sync berhasil.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* Node bisa restart ulang tanpa stale lock.
* Address final sudah aktif:

  * wallet baru menghasilkan address `IDR...`
  * format address: `IDR` + Base58Check
  * private key raw 32-byte hex
* Dynamic difficulty sudah aktif.
* Coinbase maturity sudah aktif.
* Standalone CPU miner sudah aktif:

  * idrminer bisa mine via /miner/template dan /miner/submit.
* Node public hardening sudah aktif:

  * /health
  * public RPC mode
  * wallet RPC disabled saat public mode
  * graceful shutdown
* Phase 3.0 service node simulation sudah aktif:

  * internal/servicenode
  * service register
  * service heartbeat
  * service challenge create
  * service challenge submit
  * service score
  * service rewards
  * simulated points tidak mengubah IDR balance
  * total supply tidak berubah karena service rewards
  * public RPC default service_rpc=false

Nama patch:
DesKaChain Phase 3.1 — Service Node Agent MVP

Tujuan:
Membuat standalone service node agent yang bisa berjalan sebagai proses terpisah dan otomatis melakukan:

* register ke node RPC,
* heartbeat berkala,
* mengambil atau membuat challenge simulasi,
* submit hasil challenge,
* menampilkan score,
* menyimpan local agent state,
* berjalan dengan safe mode,
* shutdown bersih.

Prinsip penting:

* Agent ini BUKAN public exit proxy.
* Agent ini TIDAK membuka relay publik.
* Agent ini TIDAK menjual bandwidth.
* Agent ini TIDAK mengubah consensus.
* Agent ini TIDAK mencetak IDR.
* Agent ini hanya berinteraksi dengan service simulation RPC dari Phase 3.0.
* PoW tetap satu-satunya pembuat block canonical.
* Service points tetap simulasi dan tidak spendable.

Aturan penting:

* Jangan ubah address format IDR.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah coinbase maturity.
* Jangan ubah PoW consensus.
* Jangan memasukkan service reward ke coinbase.
* Jangan implement public proxy/VPN/exit node.
* Jangan implement real bandwidth selling.
* Jangan implement staking.
* Jangan implement PoS.
* Jangan implement GPU miner.
* Jangan implement mining pool.
* Patch incremental.
* Semua command lama harus tetap bekerja.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...

==================================================

1. Tambahkan binary baru idrservice
   ==================================================

Tambahkan binary baru:

node/cmd/idrservice/

Target run:

go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8431 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501

Build:

go build -o idrservice ./node/cmd/idrservice

Flags minimal:

* --rpc-url string, required
* --address string, required
* --endpoint string, optional
* --state string, default ./idrservice-state.json
* --heartbeat-interval duration, default 30s
* --challenge-interval duration, default 60s
* --score-interval duration, default 60s
* --retry-interval duration, default 5s
* --once bool, run one register/heartbeat/challenge cycle then exit
* --safe-mode bool, default true
* --max-bytes-per-challenge int64, default 100000000
* --client-version string, default idrservice/dev
* --platform string, auto detect runtime.GOOS/runtime.GOARCH

Output startup:

DesKaChain Service Node Agent
rpc: http://127.0.0.1:8431
address: IDR...
endpoint: http://127.0.0.1:9501
safe mode: true
heartbeat interval: 30s
challenge interval: 60s

==================================================
2. Agent workflow
=================

Agent workflow:

1. Validate address locally if possible.
2. Load local state.
3. Register service node:
   POST /service/register
4. Send heartbeat:
   POST /service/heartbeat
5. Create challenge:
   POST /service/challenge/create
6. Generate simulated measurement result.
7. Submit challenge:
   POST /service/challenge/submit
8. Fetch score:
   GET /service/score
9. Save local state.
10. Repeat until stopped, unless --once.

For --once:

* register
* heartbeat
* create challenge
* submit challenge
* fetch score
* print summary
* exit 0

==================================================
3. Local agent state
====================

Agent harus menyimpan state lokal di file JSON.

Default:
./idrservice-state.json

Fields:

* address
* service_node_id
* endpoint
* rpc_url
* registered_at
* last_heartbeat_at
* last_challenge_id
* last_challenge_at
* last_submit_at
* last_score
* total_challenges
* successful_challenges
* failed_challenges
* total_simulated_bytes_up
* total_simulated_bytes_down
* client_version
* platform

Gunakan atomic write helper atau implement atomic write sederhana:

* write temp file
* rename

Jika file missing:

* start new state.

Jika file corrupt:

* return error jelas:
  failed to load service agent state: invalid json

Jangan menyimpan private key.

==================================================
4. Simulated measurement generator
==================================

Tambahkan package internal/serviceagent atau internal/idrservice.

Measurement simulator:

* latency_ms
* bytes_up
* bytes_down
* success

Default safe random-ish values:

* latency: 30-120 ms
* bytes_up: 1 MB - 20 MB
* bytes_down: 5 MB - 80 MB
* success: true

Respect safe mode:

* if --safe-mode true:

  * clamp bytes_up/down to --max-bytes-per-challenge
  * do not perform real upload/download
  * do not bind public listener
  * do not open proxy
  * do not connect to random third-party hosts

Phase 3.1 must only submit simulated values to Phase 3.0 RPC.

Add TODO:

* future real verifier should use signed challenge and controlled verifier endpoint.
* future mobile safe mode should check WiFi/charging/battery/temperature.

==================================================
5. Safe mode guards
===================

Safe mode default true.

When safe mode true:

* no public listener,
* no proxy server,
* no relay,
* no external speed test,
* no large traffic generation,
* no private key access,
* only RPC calls to configured DesKaChain node.

If someone passes:
--safe-mode=false

For Phase 3.1:

* allow flag but print warning:
  unsafe mode is not implemented yet; running in safe simulation mode
* Keep behavior safe anyway.
* Do not implement unsafe traffic.

==================================================
6. RPC client
=============

Implement service RPC client methods:

* Register(ctx, req)
* Heartbeat(ctx, req)
* CreateChallenge(ctx, req)
* SubmitChallenge(ctx, req)
* Score(ctx, address)
* Rewards(ctx, address)

Client requirements:

* Timeout per request default 10s.
* Retry on connection refused / temporary network error.
* Do not retry challenge submit blindly if server may have accepted but response lost, unless idempotency is handled.
* If submit duplicate returns error, fetch score and continue.
* Clear errors:
  service RPC disabled
  invalid address
  challenge expired
  rate limit exceeded
  node unavailable

==================================================
7. Agent loop and shutdown
==========================

Agent loop:

* use context.Context
* handle SIGINT/SIGTERM
* on shutdown:

  * stop loop
  * save state
  * print shutdown summary

Logs:
service agent registered id=...
heartbeat ok score=...
challenge created id=...
challenge submitted status=passed score=...
service score=94 points=940
retrying after error: ...
shutdown requested
state saved
service agent stopped

No goroutine leaks.

==================================================
8. Node service endpoints compatibility
=======================================

Do not break Phase 3.0 CLI/RPC.

Existing commands must still work:

* service register
* service heartbeat
* service challenge create
* service challenge submit
* service score
* service rewards
* service list if exists

The new idrservice should use the same RPC endpoints.

==================================================
9. Public RPC mode behavior
===========================

If node is started with:

--public-rpc

service RPC default is disabled.

idrservice should fail clearly:

error: service RPC disabled on target node

If node is started with:

--public-rpc --enable-service-rpc=true

idrservice can work.

Add docs warning:

* Do not enable service RPC publicly without rate limit and abuse protection.
* Service RPC in Phase 3.1 is for controlled testnet/verifier simulation.

==================================================
10. Health integration optional
===============================

No need to change node health heavily.

But if easy, idrservice can check:

GET /health

before starting:

* print network
* chain id
* service_rpc if present
* public_rpc

If health endpoint unreachable:

* continue retrying or error depending --once.

For --once:

* if node unreachable, exit non-zero.

For loop mode:

* retry.

==================================================
11. Agent status command
========================

Add subcommands or flags if easy.

Option A:
Binary supports:

idrservice status --state ./idrservice-state.json

Print:

* address
* service node id
* last heartbeat
* last score
* successful challenges
* failed challenges
* total simulated bytes

Option B:
Keep it simple in Phase 3.1 and only print state at exit.

Recommended:

* implement status if easy.
* otherwise TODO.

==================================================
12. Tests wajib
===============

Tambahkan/update tests:

1. Agent state load missing:

* missing file returns empty state.

2. Agent state save/load:

* save state to temp file.
* reload.
* fields match.

3. Agent state corrupt:

* invalid JSON returns clear error.

4. Measurement simulator safe mode:

* generated latency >= 0.
* bytes_up/down >= 0.
* bytes_up/down <= max bytes.
* success true by default.

5. Safe mode does not create listener:

* no network listener in simulator.
* if hard to test, structure code so simulator has no Listen call.

6. RPC client register:

* httptest server returns register response.
* client parses service_node_id.

7. RPC client heartbeat:

* parses score fields.

8. RPC client challenge create:

* parses challenge id.

9. RPC client challenge submit:

* parses status/score.

10. RPC client score:

* parses service score/simulated points.

11. Agent --once workflow:

* httptest server implementing service endpoints.
* agent runs once.
* calls register, heartbeat, challenge create, submit, score.
* exits success.
* state saved.

12. Agent retry:

* temporary server failure then success if easy.
* agent retries according to retry interval.

13. Service RPC disabled:

* server returns service RPC disabled.
* agent returns clear error.

14. No IDR mutation:

* existing service tests already check no consensus mutation.
* keep them passing.

15. Existing tests still pass:

* address
* crypto
* ledger
* mempool
* p2p
* rpc
* cpuminer
* servicenode
* wallet
* chain
* cli
* idrminer

==================================================
13. Docs update
===============

Create/update:

docs/ServiceAgent.md

Content:

* What idrservice is.
* It is a service node agent for simulation.
* It is not public exit proxy.
* It does not mine PoW blocks.
* It does not earn spendable IDR.
* It submits service simulation data to node RPC.
* Safe mode is default.
* No private key needed.
* Only IDR address required.
* Future:

  * signed service registration,
  * real verifier,
  * mobile safe mode,
  * staking/collateral,
  * capped testnet reward pool,
  * leaderboard.

Update README/Roadmap:

* Current phase:
  Phase 3.1 — Service Node Agent MVP

Add usage:

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/service node start --rpc :8431 --p2p :9431 --advertise-p2p http://127.0.0.1:9431

Create wallet:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8431 wallet new

Run service agent once:

go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8431 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501 --once

Run loop:

go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8431 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501 --heartbeat-interval 30s --challenge-interval 60s

Build:

go build -o idrservice ./node/cmd/idrservice

==================================================
14. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/deskachain --datadir ./testdata/agent dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/agent init
go run ./node/cmd/deskachain --datadir ./testdata/agent wallet new

Start node:

go run ./node/cmd/deskachain --datadir ./testdata/agent node start --rpc :8451 --p2p :9451 --advertise-p2p http://127.0.0.1:9451

Health:

Invoke-RestMethod http://127.0.0.1:8451/health

Expected:
ok true
service_rpc true

Run service agent once:

go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8451 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501 --once

Expected:
DesKaChain Service Node Agent
service agent registered id=...
heartbeat ok
challenge created id=...
challenge submitted status=passed
service score=...
simulated points=...
service points are simulation only and are not spendable IDR
state saved

Check node service score:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8451 service score --address <IDR_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8451 service rewards --address <IDR_ADDR>

Expected:
service score > 0
simulated points > 0
not IDR

Consensus check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8451 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8451 balance <IDR_ADDR>
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8451 chain validate

Expected:
total supply unchanged by service agent
IDR balance unchanged by service agent
chain valid

Public RPC disabled check:

Start public node:

go run ./node/cmd/deskachain --datadir ./testdata/agent_pub node start --rpc :8461 --p2p :9461 --advertise-p2p http://127.0.0.1:9461 --public-rpc

Run agent:

go run ./node/cmd/idrservice --rpc-url http://127.0.0.1:8461 --address <IDR_ADDR> --endpoint http://127.0.0.1:9501 --once

Expected:
error: service RPC disabled

Jangan over-engineer.
Fokus Phase 3.1 hanya:

* standalone idrservice binary,
* agent local state,
* safe-mode simulated measurement,
* auto register/heartbeat/challenge/submit/score,
* clear logs,
* graceful shutdown,
* docs/tests,
* no consensus mutation.
