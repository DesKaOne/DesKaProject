Kamu sedang bekerja pada project Go monorepo IndoChain.

Struktur project:

* node/

  * cmd/indochain/
  * cmd/indominer/
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

* Phase 1 sampai Phase 2.10 sudah selesai dan valid.
* go work sync berhasil.
* go test -count=1 ./node/... pass.
* Node bisa restart ulang tanpa stale lock.
* Address final sudah aktif:

  * wallet baru menghasilkan address `iND...`
  * format address: `iND` + Base58Check
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
  * `indominer` bisa mine block via RPC
  * standalone miner bisa include mempool tx
* Node hardening sudah aktif:

  * `/health`
  * public RPC mode
  * wallet RPC disabled saat public mode
  * graceful shutdown
  * HTTP timeout/body limit/rate limit foundation
  * peer max/public safety foundation

Nama patch:
IndoChain Phase 3.0 — Bandwidth Mining Research & Reward Simulation

Tujuan:
Membuat fondasi awal untuk konsep bandwidth/service-node contribution, tetapi hanya sebagai research dan reward simulation layer.

Prinsip penting:

* Bandwidth mining TIDAK menjadi consensus block mining pada phase ini.
* PoW tetap satu-satunya pembuat block canonical.
* Service node reward belum masuk coinbase.
* Tidak ada real payout.
* Tidak ada mainnet reward.
* Tidak ada janji profit.
* Tidak ada klaim “nyalakan internet langsung dapat uang”.
* Semua reward pada phase ini hanya simulasi/testnet accounting.
* Fokus utama: model data, scoring, anti-abuse awal, verifier simulation, dan CLI/RPC untuk melihat score.

Aturan penting:

* Jangan ubah address format `iND...`.
* Jangan ubah private key format.
* Jangan ubah difficulty adjustment.
* Jangan ubah cumulative work formula.
* Jangan ubah coinbase maturity.
* Jangan ubah PoW consensus.
* Jangan memasukkan bandwidth reward ke block reward consensus.
* Jangan implement public exit proxy.
* Jangan implement VPN/proxy relay publik.
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

1. Konsep Phase 3.0
   ==================================================

Tambahkan module baru untuk service node / bandwidth reward simulation.

Nama package disarankan:

* internal/service
  atau
* internal/bandwidth
  atau
* internal/servicenode

Pilih nama yang rapi. Rekomendasi:
internal/servicenode

Phase ini hanya menyimpan dan menghitung:

* service node registration,
* heartbeat,
* uptime sample,
* latency sample,
* bandwidth sample,
* verification challenge simulation,
* service score,
* simulated reward points,
* anti-abuse flags.

Reward hasil simulasi tidak boleh langsung menjadi dIDR spendable balance.

Gunakan istilah:

* service points
* simulated reward
* testnet service score

Jangan gunakan istilah:

* guaranteed income
* passive income
* real payout
* mainnet reward guaranteed

==================================================
2. Service node identity
========================

Service node harus terikat ke address iND.

Registration minimal:

* service_node_id
* owner_address
* advertised_endpoint optional
* mode:

  * local
  * test
  * disabled
* created_at
* last_seen_at
* status:

  * registered
  * active
  * inactive
  * banned
* metadata:

  * client_version
  * platform
  * user_agent

Command:

service register --address <IND_ADDR> --endpoint <URL>

Remote:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8411 service register --address <IND_ADDR> --endpoint http://127.0.0.1:9501

Rules:

* Validate iND address.
* Endpoint optional for local simulation.
* Duplicate registration for same address updates existing record, not duplicate.
* Do not require private key yet.
* Add TODO for future signature-based registration.

RPC endpoint:
POST /service/register

Response:
{
"ok": true,
"service_node_id": "...",
"owner_address": "iND...",
"status": "registered"
}

==================================================
3. Heartbeat
============

Tambahkan heartbeat untuk service node.

Command:

service heartbeat --address <IND_ADDR>

RPC:
POST /service/heartbeat

Request:
{
"address": "iND...",
"endpoint": "http://127.0.0.1:9501",
"client_version": "dev",
"platform": "windows"
}

Behavior:

* Update last_seen_at.
* Mark active.
* Record uptime sample.
* Do not create coin reward.
* Return current score summary.

Response:
{
"ok": true,
"status": "active",
"uptime_score": 10,
"latency_score": 0,
"bandwidth_score": 0,
"service_score": 10
}

==================================================
4. Verification challenge simulation
====================================

Tambahkan challenge simulation.

Phase ini tidak perlu benar-benar mengukur bandwidth internet publik.
Cukup buat model challenge yang bisa dites local.

Commands:

service challenge create --address <IND_ADDR>
service challenge submit --challenge-id <ID> --latency-ms 50 --bytes-up 1000000 --bytes-down 2000000

RPC:
POST /service/challenge/create
POST /service/challenge/submit

Challenge fields:

* challenge_id
* address
* issued_at
* expires_at
* nonce
* expected_mode
* status:

  * pending
  * passed
  * failed
  * expired

Submit fields:

* latency_ms
* bytes_up
* bytes_down
* success true/false

Rules:

* Expired challenge rejected.
* Negative latency/bytes rejected.
* Too large unrealistic values flagged.
* Challenge can only be submitted once.
* Challenge result creates measurement sample.
* No dIDR balance mutation.

==================================================
5. Scoring model
================

Implement simple scoring model.

Score components:

* uptime_score: 0-100
* latency_score: 0-100
* bandwidth_score: 0-100
* reliability_score: 0-100
* abuse_penalty: 0-100
* final service_score: 0-100

Suggested formula:
raw_score =
uptime_score * 0.30 +
latency_score * 0.20 +
bandwidth_score * 0.30 +
reliability_score * 0.20

service_score = clamp(raw_score - abuse_penalty, 0, 100)

Do not use floating chaos if project prefers integer.
Integer basis points allowed.

Latency score:

* <= 50ms: 100
* <= 100ms: 80
* <= 250ms: 60
* <= 500ms: 30
* > 500ms: 10

Bandwidth score:

* bytes_down + bytes_up per sample.
* Keep simple tier:

  > = 100 MB: 100
  > = 50 MB: 80
  > = 10 MB: 50
  > = 1 MB: 20
  > else: 5

Uptime score:

* based on recent heartbeats.
* Phase 3.0 simple:
  active heartbeat within last 5 minutes => 100
  last 30 minutes => 50
  older => 0

Reliability:

* passed challenges / total challenges.
* no challenge yet => 0 or neutral 50. Choose one and document.
* Recommended for phase 3.0:
  no challenges => 0, because unverified.

Abuse penalty:

* suspicious repeated identical samples,
* impossible bandwidth,
* too many failed challenges,
* too many heartbeats too quickly,
* endpoint changes too often.

Keep penalty simple.

==================================================
6. Simulated reward points
==========================

Tambahkan reward simulation, bukan dIDR balance.

Fields:

* epoch
* address
* service_score
* simulated_points
* reason
* created_at

Epoch can be simple:

* local epoch by block height window or wall-clock date.
* Recommended Phase 3.0:
  epoch = current chain height / 100
  or
  epoch = YYYY-MM-DD
  Pick one and document.

Reward formula simple:
simulated_points = service_score * reward_weight

Example:

* reward_weight default 10
* score 80 => 800 points

Important:

* simulated_points are not dIDR.
* simulated_points are not spendable.
* simulated_points do not affect chain consensus.
* simulated_points are for leaderboard/research only.

Command:

service score --address <IND_ADDR>
service rewards --address <IND_ADDR>

Output:
address: iND...
service score: 80
simulated points: 800
note: service points are simulation only and are not spendable dIDR

RPC:
GET /service/score?address=<IND_ADDR>
GET /service/rewards?address=<IND_ADDR>

==================================================
7. Storage
==========

Persist service node data under datadir.

Files:

* service_nodes.json
* service_challenges.json
* service_rewards.json

Use atomic write helper if available.
Add mutex for runtime safety.
Do not corrupt file on crash.
Missing files => empty state.

If project uses bbolt for chain only, JSON is fine for Phase 3.0.
Do not mix with consensus DB yet.

==================================================
8. Public RPC safety
====================

Service endpoints in public RPC mode:

* Read-only service score/rewards can be enabled.
* Register/heartbeat/challenge submit can be enabled only if `--enable-service-rpc=true`.

Add flag:
--enable-service-rpc bool

Defaults:

* local/admin mode: true
* public-rpc mode: false by default

Reason:

* public testnet service verifier should be controlled.
* avoid open spam endpoints by default.

If endpoint disabled:
{
"ok": false,
"error": "service RPC disabled"
}

==================================================
9. Rate limit and body limit
============================

Use Phase 2.10 rate limiter/body limits.

Service endpoint limits:

* register: 128 KB
* heartbeat: 128 KB
* challenge create: 128 KB
* challenge submit: 128 KB
* score/rewards read: no body

Rate limit:

* register: 30/min per IP
* heartbeat: 120/min per IP
* challenge create: 60/min per IP
* challenge submit: 60/min per IP
* read score/rewards: general RPC rate limit

==================================================
10. CLI commands
================

Add top-level command group:

service

Commands:

1. Register:
   service register --address <IND_ADDR> --endpoint <URL>

2. Heartbeat:
   service heartbeat --address <IND_ADDR> --endpoint <URL>

3. Challenge create:
   service challenge create --address <IND_ADDR>

4. Challenge submit:
   service challenge submit --challenge-id <ID> --latency-ms 50 --bytes-up 1000000 --bytes-down 2000000 --success true

5. Score:
   service score --address <IND_ADDR>

6. Rewards:
   service rewards --address <IND_ADDR>

7. List:
   service list

Local and remote mode:

* All commands should work with `--rpc-url` when node running.
* Local mode can read/write datadir directly if node not running.
* If datadir locked, suggest valid remote command.

==================================================
11. Node startup logs
=====================

Node start should log:

* service rpc enabled/disabled
* service store path
* registered service node count

Example:
service rpc: true
service store: testdata/node/service_nodes.json
service nodes: 0

Do not log sensitive private keys.

==================================================
12. Health endpoint addition
============================

Update `/health` response to include optional service summary:

{
"service_nodes": 0,
"service_rpc": true
}

Keep health fast.

==================================================
13. No consensus mutation
=========================

This is critical.

Service node score/reward simulation must NOT:

* modify total supply,
* modify circulating supply,
* add dIDR balance,
* create block tx,
* affect PoW difficulty,
* affect cumulative work,
* affect coinbase reward,
* affect chain validation.

Add tests to ensure:

* total supply unchanged after service reward simulation.
* chain info unchanged except unrelated height.
* balance unchanged after service score/reward.

==================================================
14. Anti-abuse initial flags
============================

Add simple abuse flags.

Potential flags:

* heartbeat_spam
* impossible_bandwidth
* repeated_identical_samples
* challenge_failed
* challenge_expired
* endpoint_changed_too_often

Store:

* flags []string
* abuse_penalty
* last_abuse_reason

Do not permanently ban in Phase 3.0 unless simple status banned is needed for tests.
Prefer:

* status active/inactive
* score penalty

==================================================
15. Docs
========

Create/update:

docs/ServiceNode.md

Content:

* What is service node simulation.
* It is not consensus mining.
* It is not spendable dIDR.
* It is not real payout.
* PoW still creates blocks.
* Service points are research/testnet-only.
* Why anti-abuse is needed.
* Future phases:

  * real verifier,
  * signed registration,
  * mobile safe mode,
  * staking/collateral,
  * capped reward pool,
  * public testnet leaderboard.

Update README/Roadmap:

* Current phase:
  Phase 3.0 — Bandwidth Mining Research & Reward Simulation
* Mention:
  Service node layer introduced as simulation only.

==================================================
16. Tests wajib
===============

Add/update tests:

1. Service register valid:

* dIDR address accepted.
* service_node_id created.
* status registered.

2. Service register invalid address:

* rejected.

3. Duplicate register:

* same address updates existing.
* count remains 1.

4. Heartbeat:

* updates last_seen_at.
* status active.
* uptime score > 0.

5. Challenge create:

* creates pending challenge.
* challenge id non-empty.
* expires_at > issued_at.

6. Challenge submit valid:

* pending challenge becomes passed.
* sample recorded.
* latency/bandwidth score updated.

7. Challenge submit expired:

* rejected or marked expired.

8. Challenge submit duplicate:

* second submit rejected.

9. Negative values:

* negative latency/bytes rejected.

10. Impossible bandwidth:

* flagged with impossible_bandwidth.
* abuse penalty > 0.

11. Score formula:

* known inputs produce expected score.
* clamp 0-100.

12. Simulated rewards:

* score creates points.
* points are not dIDR.
* reward list returns points.

13. No consensus mutation:

* before service reward:
  total supply X
  balance address Y
* after service reward:
  total supply still X
  balance still Y
  chain validate still pass.

14. Storage persistence:

* register node.
* save.
* reload store.
* node still exists.

15. Atomic write:

* service store write leaves valid JSON.
* missing file loads empty.

16. Public RPC disabled:

* public-rpc with service disabled rejects register/heartbeat.
* score read behavior according to config.

17. Public RPC enabled:

* with enable-service-rpc true, register/heartbeat works.

18. Rate/body limit:

* oversized service request rejected.
* spam can be rate limited if test infra supports.

19. CLI remote service register:

* command with --rpc-url works.
* prints service node id/status.

20. Health:

* includes service_nodes and service_rpc.

21. Existing tests still pass:

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
17. Expected final commands
===========================

After patch:

go work sync
go mod tidy
go test ./node/...

Manual basic:

go run ./node/cmd/indochain --datadir ./testdata/service dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/service init
go run ./node/cmd/indochain --datadir ./testdata/service wallet new

Start node:

go run ./node/cmd/indochain --datadir ./testdata/service node start --rpc :8431 --p2p :9431 --advertise-p2p http://127.0.0.1:9431

Health:

Invoke-RestMethod http://127.0.0.1:8431/health

Expected:
ok true
service_rpc true
service_nodes 0

Register:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service register --address <IND_ADDR> --endpoint http://127.0.0.1:9501

Expected:
service node registered
service node id: ...

Heartbeat:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service heartbeat --address <IND_ADDR> --endpoint http://127.0.0.1:9501

Expected:
status: active
service score shown

Challenge:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service challenge create --address <IND_ADDR>

Expected:
challenge id: ...

Submit:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service challenge submit --challenge-id <ID> --latency-ms 50 --bytes-up 10000000 --bytes-down 50000000 --success true

Score:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service score --address <IND_ADDR>

Expected:
service score: > 0
simulated points: maybe > 0
note: simulation only, not spendable dIDR

Rewards:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 service rewards --address <IND_ADDR>

Expected:
service points listed
not dIDR

Consensus check:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 balance <IND_ADDR>
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8431 chain validate

Expected:
total supply unchanged by service rewards
dIDR balance unchanged by service rewards
chain valid

Public RPC service disabled:

go run ./node/cmd/indochain --datadir ./testdata/service_pub node start --rpc :8441 --p2p :9441 --advertise-p2p http://127.0.0.1:9441 --public-rpc

Then:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8441 service register --address <IND_ADDR> --endpoint http://127.0.0.1:9501

Expected:
error: service RPC disabled

Jangan over-engineer.
Fokus Phase 3.0 hanya:

* service node research model,
* heartbeat,
* challenge simulation,
* scoring,
* simulated reward points,
* storage,
* RPC/CLI,
* no consensus mutation,
* docs and tests.
