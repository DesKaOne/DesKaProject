Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.3.2 sudah valid.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* localnet/testnet profile sudah terpisah.
* testnet genesis stable:
  db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* testnet:

  * network: testnet
  * network_id: idr-testnet-1
  * chain_id: 777101
  * coinbase maturity: 100
  * min stake amount: 100 IDR
  * min service stake: 1000 IDR
  * unbonding period: 100 blocks
* Multi-node controlled local testnet sudah valid:

  * bootnode persistence
  * peer dedupe/normalize
  * peer sync after restart
  * peer offline/recovery
  * network mismatch rejected
  * invalid block penalty
* Miner testnet valid.
* Service node testnet valid.
* Staking tetap collateral-only.
* Service points tetap simulation-only.
* PoW tetap satu-satunya block production consensus.

Nama patch:
DesKaChain Phase 3.4 — Dev Faucet & Testnet Funding Flow

Tujuan:
Menambahkan faucet khusus dev/testnet agar wallet bisa menerima IDR testnet untuk testing transfer, staking collateral, dan service node eligibility tanpa harus mining manual terlalu lama.

Fokus:

* faucet testnet/dev-only,
* faucet wallet/address funding,
* rate limit dasar,
* faucet balance/source account,
* faucet tx masuk mempool/block biasa,
* faucet tidak aktif di localnet/mainnet kecuali explicit dev flag,
* faucet tidak mencetak supply diam-diam,
* faucet docs/runbook,
* tests.

Non-goals:

* Jangan implement public faucet production.
* Jangan implement CAPTCHA.
* Jangan implement web UI.
* Jangan implement explorer.
* Jangan implement mainnet faucet.
* Jangan implement real-money reward.
* Jangan implement PoS.
* Jangan implement staking reward.
* Jangan implement slashing.
* Jangan ubah address format IDR.
* Jangan ubah private key format.
* Jangan ubah localnet genesis.
* Jangan ubah testnet genesis kecuali benar-benar perlu.
* Jangan bypass mempool/chain validation.
* Jangan membuat faucet credit yang spendable tanpa tx/block.

==================================================

1. Desain faucet
   ==================================================

Faucet harus bekerja sebagai fitur testnet/dev.

Model yang direkomendasikan:

* Faucet punya wallet/address sumber.
* Faucet mengirim transaksi normal dari faucet address ke recipient.
* Tx masuk mempool.
* Tx harus di-mine agar confirmed.
* Tidak boleh langsung menambah balance tanpa block.
* Tidak boleh mengubah total supply kecuali melalui coinbase normal.
* Faucet balance berasal dari:
  A. faucet wallet yang sudah mining coinbase matang, atau
  B. dev-only faucet pre-funded account pada testnet genesis jika project memilih cara ini.

Rekomendasi untuk Phase 3.4:

* Jangan ubah testnet genesis dulu kalau tidak perlu.
* Gunakan faucet wallet yang bisa dibuat dan didanai via mining.
* Tambahkan CLI/helper untuk faucet supaya flow test mudah.

Jika ingin pre-funded faucet genesis:

* harus testnet-only.
* harus terdokumentasi jelas.
* harus masuk total supply genesis dengan benar.
* tidak boleh memengaruhi localnet.
* tambahkan test total supply genesis.
* Namun untuk fase ini lebih aman pakai faucet wallet biasa.

==================================================
2. Faucet configuration
=======================

Tambahkan config faucet ke testnet profile atau runtime node config.

Field minimal:

* faucet_enabled bool
* faucet_address string
* faucet_amount amount.Amount
* faucet_daily_limit_per_address amount.Amount atau request count
* faucet_min_interval_seconds int
* faucet_max_amount_per_request amount.Amount
* faucet_require_testnet bool

Default:

* localnet: faucet disabled
* testnet: faucet disabled by default, enabled only with flag
* public_rpc: faucet disabled by default unless explicit `--enable-faucet-rpc`

Flags node start:

* --enable-faucet-rpc
* --faucet-address <IDR_ADDR>
* --faucet-amount <IDR_AMOUNT>
* --faucet-min-interval <duration>
* --faucet-max-per-address <IDR_AMOUNT> optional

Rules:

* Faucet cannot start enabled without valid faucet address.
* Faucet address must be valid for active network.
* Faucet amount must be > 0.
* Faucet amount must not exceed configured max.
* Faucet RPC should refuse non-testnet unless explicit dev override exists.
* Do not expose faucet automatically on public RPC unless explicitly enabled.

==================================================
3. Faucet RPC endpoints
=======================

Tambahkan endpoint RPC:

GET /faucet/info

Response:
{
"enabled": true,
"network": "testnet",
"network_id": "idr-testnet-1",
"chain_id": 777101,
"faucet_address": "...",
"amount": "100",
"min_interval_seconds": 3600,
"mempool_pending": N,
"note": "testnet faucet only; testnet IDR has no monetary value"
}

POST /faucet/request

Request:
{
"address": "<IDR_ADDR>"
}

Optional:
{
"address": "<IDR_ADDR>",
"amount": "100"
}

Response on success:
{
"tx_id": "...",
"from": "<FAUCET_ADDR>",
"to": "<RECIPIENT_ADDR>",
"amount": "100",
"status": "pending",
"note": "mine a block to confirm faucet transaction"
}

Error cases:

* faucet disabled.
* not testnet.
* invalid address.
* invalid amount.
* faucet address not configured.
* faucet has insufficient mature balance.
* recipient rate limited.
* pending faucet tx already exists for address.
* public RPC faucet disabled.

Endpoint behavior:

* Faucet request creates normal signed tx if wallet/private key available.
* If current wallet storage cannot sign from arbitrary faucet address, add a clear faucet wallet mechanism.
* Do not create unsigned spendable tx.
* Do not bypass normal tx signature checks.

==================================================
4. Faucet CLI commands
======================

Tambahkan CLI commands:

1. faucet info

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 faucet info

Output:

* enabled
* network
* chain id
* faucet address
* amount
* min interval
* note testnet only

2. faucet request

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 faucet request --address <IDR_ADDR>

Output:

* faucet tx created
* tx id
* from
* to
* amount
* status pending
* note mine a block to confirm

3. faucet fund helper optional local command

If needed, add dev helper:

go run ./node/cmd/deskachain --datadir <DIR> faucet set-address --address <IDR_ADDR>

or use node start flag only.

Do not over-engineer wallet management.

==================================================
5. Faucet state / rate limit
============================

Tambahkan faucet state file di datadir:

faucet_state.json

Isi minimal:

* requests by address
* last_request_time
* total_amount_requested
* pending tx ids

Rules:

* file disimpan di datadir.
* ignore di .gitignore:
  faucet_state.json
  **/faucet_state.json
* corrupt file handled gracefully:

  * error jelas, or
  * backup and reset.
* Rate limit per recipient address.
* Pending faucet tx to same address should block duplicate request until mined or expired.

Suggested defaults for dev:

* faucet amount: 100 IDR
* min interval: 1 minute for local controlled testing
* max per address per day: 1000 IDR
* For docs, explain dev defaults are not production-safe.

==================================================
6. Faucet balance rules
=======================

Faucet must use mature balance.

Rules:

* If faucet address has confirmed but immature coinbase, request fails:
  insufficient mature faucet balance
* If faucet has pending outgoing tx, spendable reduced.
* Faucet cannot overspend.
* Faucet tx fee behavior follows existing tx rules.
* Faucet tx uses normal nonce/pending logic.

Testnet coinbase maturity is 100, so manual faucet funding may require mining enough blocks.

For easier dev flow, add docs command:

idrminer --max-blocks 105

Then faucet wallet has mature balance.

Do not reduce testnet maturity just for faucet.

==================================================
7. Security / safety
====================

Even though this is dev faucet:

* Do not enable by default on public RPC.
* Do not enable on mainnet.
* Do not expose private key in RPC responses/logs.
* Do not log seed/private key.
* Do not store private key in faucet_state.json.
* Do not create faucet auto-mint endpoint.
* Do not allow arbitrary `from` address.
* Do not allow request amount above max.
* Rate limit by address.
* Optionally rate limit by IP if RPC layer supports it.

==================================================
8. Tests required
=================

Tambahkan tests di internal/rpc, internal/cli, internal/ledger/mempool jika perlu.

Required tests:

1. TestFaucetInfoDisabledByDefault

* localnet/testnet faucet disabled unless flag/config enabled.

2. TestFaucetInfoTestnetEnabled

* testnet node with faucet enabled returns:

  * enabled true
  * network testnet
  * network_id idr-testnet-1
  * chain_id 777101
  * faucet_address
  * amount

3. TestFaucetRejectsLocalnetByDefault

* localnet faucet request rejected unless explicit dev override exists.

4. TestFaucetRejectsPublicRPCWithoutExplicitEnable

* public RPC does not expose faucet by default.

5. TestFaucetRequestInvalidAddress

* invalid address rejected.

6. TestFaucetRequestCreatesPendingTx

* faucet has mature balance.
* request creates tx.
* tx goes to mempool.
* recipient pending incoming increases.
* tx mined normally.
* recipient confirmed balance increases after block.

7. TestFaucetRequiresMatureBalance

* faucet has only immature coinbase.
* request rejected.

8. TestFaucetCannotOverspend

* faucet spendable < faucet amount.
* request rejected.

9. TestFaucetRateLimitByAddress

* first request ok.
* second request before min interval rejected.

10. TestFaucetPendingDuplicateRejected

* if recipient has pending faucet tx, duplicate request rejected.

11. TestFaucetStatePersists

* request updates faucet_state.json.
* reload node/state.
* rate limit still applies.

12. TestFaucetStateCorruptHandled

* corrupt faucet_state.json.
* node returns clear error or backups/reset safely.

13. TestFaucetDoesNotChangeTotalSupplyDirectly

* before faucet tx total supply X.
* faucet request pending tx does not change supply.
* after mined block, total supply increases only by coinbase reward.
* transfer from faucet to recipient does not mint.

14. TestFaucetCLIInfo

* CLI displays faucet info.

15. TestFaucetCLIRequest

* CLI request output contains tx id, amount, status pending.

16. Existing tests pass:
    go test ./node/...
    go test -count=1 ./node/...

==================================================
9. Manual validation
====================

After patch:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target:

go test ./node/internal/rpc -run Faucet -v
go test ./node/internal/cli -run Faucet -v
go test ./node/internal/mempool -run Faucet -v
go test ./node/internal/ledger -run Faucet -v

Manual testnet flow:

Clean:

go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn dev reset --yes

Init testnet:

go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn --network testnet init

Create faucet wallet:

go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn wallet new

Save as:
<FAUCET_ADDR>

Create recipient wallet:

go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn wallet new

Save as:
<RECIPIENT_ADDR>

Start testnet node with faucet enabled:

go run ./node/cmd/deskachain --datadir ./testdata/faucet_tn node start --rpc :8811 --p2p :9811 --advertise-p2p http://127.0.0.1:9811 --enable-faucet-rpc --faucet-address <FAUCET_ADDR> --faucet-amount 100 --faucet-min-interval 1m

Fund faucet by mining enough blocks:

go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --max-blocks 105

Check faucet balance:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <FAUCET_ADDR>

Expected:

* mature balance > 0
* spendable enough

Faucet info:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 faucet info

Expected:

* enabled true
* network testnet
* chain id 777101
* faucet amount 100 IDR

Request faucet:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 faucet request --address <RECIPIENT_ADDR>

Expected:

* faucet tx created
* status pending
* amount 100 IDR

Check recipient before mine:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <RECIPIENT_ADDR>

Expected:

* pending incoming 100 IDR or mempool pending shown.

Mine one block:

go run ./node/cmd/idrminer --rpc-url http://127.0.0.1:8811 --address <FAUCET_ADDR> --threads 4 --once

Check recipient after mine:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 balance <RECIPIENT_ADDR>

Expected:

* confirmed balance 100 IDR
* spendable 100 IDR if normal tx maturity does not apply.
* if project applies rules differently, document it.

Rate limit check:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8811 faucet request --address <RECIPIENT_ADDR>

Expected:

* error rate limited.

Localnet reject check:

go run ./node/cmd/deskachain --datadir ./testdata/faucet_ln dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/faucet_ln --network localnet init
go run ./node/cmd/deskachain --datadir ./testdata/faucet_ln node start --rpc :8821 --p2p :9821 --advertise-p2p http://127.0.0.1:9821 --enable-faucet-rpc --faucet-address <ANY_IDR_ADDR>

Then:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8821 faucet info

Expected:

* faucet disabled or error: faucet is testnet-only.

==================================================
10. Docs update
===============

Update/create:

docs/Faucet.md

Content:

* Dev/testnet faucet only.
* Testnet IDR has no monetary value.
* Faucet does not mint silently.
* Faucet uses normal transactions.
* Faucet tx must be mined.
* Faucet requires mature balance.
* How to fund faucet wallet.
* How to request faucet funds.
* How to mine confirmation.
* How to use faucet funds for staking/service collateral.
* Security notes.

Update:

* docs/Testnet.md
* docs/Architecture.md
* README.md
* README-ID.md
* Roadmap.md

Roadmap:

* Current phase: Phase 3.4 — Dev Faucet & Testnet Funding Flow
* Next possible:

  * Phase 3.4.1 — Faucet Stake/Service Testnet Scenario
  * Phase 3.5 — Read-only Explorer API
  * Phase 3.6 — Public Testnet Packaging

==================================================
11. Done criteria
=================

Phase 3.4 valid if:

* faucet disabled by default.
* faucet only enabled explicitly.
* faucet testnet info works.
* faucet request creates normal pending tx.
* faucet requires mature spendable balance.
* faucet cannot overspend.
* faucet rate limit works.
* faucet state persists.
* faucet tx mined normally.
* total supply not mutated directly.
* docs exist.
* all tests pass.
