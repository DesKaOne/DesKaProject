Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.4 sudah valid full.
* Phase 4.4 Public Testnet Explorer Web UI MVP valid full.
* Explorer Web UI sudah berjalan di browser:

  * Dashboard
  * Blocks
  * Block detail
  * Transaction detail
  * Address detail
* Explorer API Phase 4.3 valid full.
* Explorer UI static routes valid.
* Explorer UI embedded files valid.
* Explorer UI public RPC safety valid.
* Explorer UI tidak mengandung write endpoint strings.
* Build/package v0.4.4-testnet valid.
* Public RPC safety tetap aman:

  * wallet_rpc=false
  * admin_rpc=false
  * faucet_rpc=false by default
  * service_rpc=false by default
* Testnet IDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.5 — Explorer UX Polish + API Pagination/Search Hardening

Goal:
Polish the public testnet explorer UI and harden read-only explorer API behavior for public usage.

This phase should improve:

* search behavior,
* pagination,
* empty states,
* loading/error states,
* mobile responsiveness,
* hash/address formatting,
* copy buttons,
* API parameter validation,
* API search endpoint,
* API limit caps,
* not-found handling,
* public read-only safety.

This is still explorer MVP hardening, not a full production explorer.

Non-goals:

* Do not launch mainnet.
* Do not change consensus.
* Do not change testnet genesis.
* Do not change block/tx format.
* Do not add wallet web UI.
* Do not add write actions to explorer UI.
* Do not add send/stake/faucet/service write forms.
* Do not expose private keys.
* Do not add staking APY/reward claims.
* Do not make service points spendable.
* Do not promise price/profit/rewards.
* Do not add external analytics/tracking.
* Do not require npm/Node.js if current UI is plain HTML/CSS/JS.
* Do not add external CDN dependency.

==================================================

1. Explorer search endpoint hardening
   ==================================================

Add or harden:

GET /explorer/search?q=<query>

Search should detect:

* block height integer,
* block hash,
* txid,
* IDR address.

Behavior:

* Numeric query:

  * search block by height.
* IDR address:

  * validate active network.
  * return address result.
* Hash-like query:

  * try block hash.
  * try txid.
  * return one or multiple typed results if ambiguous.
* Empty query:

  * 400 clear error.
* Invalid query:

  * 400 clear error.
* Not found:

  * 404 or 200 with empty results, but be consistent and document it.

Response example:

{
"ok": true,
"query": "...",
"results": [
{
"type": "block",
"label": "Block 126",
"path": "/explorer-ui/#/block/126",
"api_path": "/explorer/blocks/126",
"hash": "..."
}
]
}

No state mutation.

Tests:

* search by height.
* search by block hash.
* search by txid.
* search by address.
* empty query rejected.
* invalid query rejected.
* wrong-network address rejected.
* not found handled.
* public RPC search allowed.

==================================================
2. API pagination hardening
===========================

Review all explorer endpoints with limit/offset:

/explorer/blocks
/explorer/address/{address}/txs
/explorer/stakes
/explorer/services

Rules:

* default limit: 20.
* max limit: 100.
* offset default: 0.
* negative offset rejected.
* negative limit rejected.
* non-integer limit/offset rejected.
* too-large limit is capped or rejected; choose one behavior and document it.
* response includes:

  * limit
  * offset
  * count
  * next_offset if more data can be inferred
  * prev_offset if offset > 0

Example:

{
"ok": true,
"limit": 20,
"offset": 40,
"count": 20,
"next_offset": 60,
"prev_offset": 20,
"blocks": [...]
}

Keep simple scan mode safe. Do not allow unbounded scans by public request.

==================================================
3. Better API error format
==========================

Standardize explorer API errors.

Recommended format:

{
"ok": false,
"error": "invalid_limit",
"message": "limit must be between 1 and 100"
}

Apply to explorer endpoints:

* invalid address.
* wrong network address.
* invalid height.
* block not found.
* tx not found.
* invalid pagination.
* invalid search query.
* unsupported method.

Do not expose internal stack traces or file paths.

Tests:

* all major error responses are JSON.
* status codes are appropriate:

  * 400 invalid input.
  * 404 not found.
  * 405 method not allowed if applicable.
  * 200 OK for successful empty list if that endpoint semantics fits.

==================================================
4. UI search polish
===================

Improve search box in Explorer Web UI.

Requirements:

* Search by block height.
* Search by block hash.
* Search by txid.
* Search by IDR address.
* Enter key submits.
* Search button submits.
* Loading state while searching.
* Not found shows friendly message.
* Invalid input shows friendly message.
* Search result routes to correct view.
* Query stays visible or recent result is clear.

If `/explorer/search` exists, use it.
If not, implement frontend fallback:

* numeric -> block page.
* IDR address -> address page.
* hash -> try tx then block.

No write actions.

==================================================
5. UI pagination polish
=======================

Improve Blocks page:

* Previous/Next buttons.
* Disable Previous at offset 0.
* Disable Next if count < limit.
* Limit selector.
* Show offset/page info.
* Keep URL hash query state if simple:

  * `#/blocks?limit=20&offset=40`
* Or keep internal state if URL parsing is too much.

Improve Address tx history:

* Previous/Next.
* Limit selector.
* Empty state.

Improve Stakes and Services pages:

* Previous/Next if endpoint supports it.
* Empty state if no records.

==================================================
6. UI empty/loading/error states
================================

Every view should have:

* loading state,
* empty state,
* error state.

Views:

* Dashboard.
* Blocks.
* Block detail.
* Tx detail.
* Address detail.
* Stakes.
* Services.

Examples:

* “No blocks found.”
* “No recent transactions for this address.”
* “Stake records not found.”
* “Service records not found.”
* “Block not found.”
* “Transaction not found.”
* “Invalid address for this network.”

Do not show raw JSON unless behind debug toggle or not used.

==================================================
7. Hash/address formatting and copy UX
======================================

Improve rendering:

* truncate long hashes/addresses in tables.
* show full value in title tooltip or expanded detail.
* copy button beside hashes/addresses/txids.
* clicking hash/address navigates to detail.
* copy button should not navigate accidentally.
* show copied feedback briefly.

No clipboard dependency beyond browser Clipboard API with fallback if feasible.

Tests:

* JS source should not contain write endpoint strings.
* Static smoke test should ensure copy helper exists if easy.

==================================================
8. Mobile/responsive polish
===========================

Improve CSS:

* dashboard cards wrap cleanly.
* tables scroll horizontally on small screens.
* header nav wraps.
* search input usable on mobile width.
* footer/banner readable.
* no overflow breaking page layout.

No external CSS framework required.

==================================================
9. Security/read-only hardening
===============================

Explorer UI and API must remain read-only.

Ensure UI does not include:

* wallet new/export/import.
* send tx.
* stake lock/unlock.
* faucet request.
* service register/heartbeat/submit.
* mining controls.
* admin/dev reset.
* private key fields.

Backend tests:

* POST to explorer API endpoints returns 405 or safe error.
* POST to explorer UI route does not mutate state.
* public wallet/admin endpoints remain disabled.
* explorer works without wallet/admin RPC.
* faucet request not exposed through explorer.
* stake lock/unlock not exposed through explorer.
* service register not exposed through explorer.

==================================================
10. Docs update
===============

Update:

docs/Explorer.md
README.md
README-ID.md
docs/DeployTestnet.md
docs/MultiHostTestnet.md

Document:

* Explorer Web UI path:

  * /explorer-ui/
* Explorer API path:

  * /explorer/*
* Search examples:

  * height
  * block hash
  * txid
  * address
* Pagination:

  * limit
  * offset
  * max limit
* Error format.
* Read-only safety.
* Testnet IDR has no monetary value.
* Mainnet not available.
* Service points simulation-only.
* Staking collateral-only.

==================================================
11. Tests
=========

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Explorer|Search|Pagination|Public|Health|UI|Static|Wallet|Admin|Invalid|NotFound" -count=1 -v

go test ./node/internal/cli -run "Explorer|Public|Release|UI|Wallet|Admin" -count=1 -v

go test ./node/internal/chain -run "Block|Tx|Address|Supply|Snapshot" -count=1 -v

go test ./node/internal/servicenode -run "Score|Collateral|Points|Service" -count=1 -v

go test ./node/internal/staking -run "Stake|Status|Total|Unlock" -count=1 -v

Add/verify tests:

* explorer search height/hash/tx/address.
* explorer search invalid/empty/not found.
* pagination limit/offset default.
* pagination max limit.
* invalid limit/offset rejected.
* address tx pagination.
* explorer UI static routes.
* explorer UI no write endpoint strings.
* public RPC safety.

No need to run `go test ./node/internal/explorerui` unless that package exists.

==================================================
12. Manual validation
=====================

Use running public testnet node with explorer UI.

Open:

http://127.0.0.1:9311/explorer-ui/

or Tailscale/LAN:

http://100.101.251.7:9311/explorer-ui/

Manual checks:

* Dashboard loads.
* Blocks page pagination works.
* Limit selector works.
* Block detail opens from block table.
* Tx detail opens from block detail.
* Address detail opens from block/tx pages.
* Address tx pagination works.
* Stakes page loads empty or records.
* Services page loads empty or records.
* Search works for:

  * height
  * block hash
  * txid
  * IDR address
* Invalid search shows friendly error.
* Not found shows friendly error.
* Copy buttons work.
* Mobile/narrow width still usable.
* No wallet/admin/write actions visible.

API manual checks:

curl "http://127.0.0.1:9311/explorer/search?q=126"
curl "http://127.0.0.1:9311/explorer/blocks?limit=5&offset=0"
curl "http://127.0.0.1:9311/explorer/blocks?limit=9999"
curl "http://127.0.0.1:9311/explorer/blocks?limit=-1"
curl "http://127.0.0.1:9311/explorer/address/<ADDR>/txs?limit=1&offset=1"

Safety:
deskachain --rpc-url http://127.0.0.1:9311 wallet new

Expected:

* wallet new rejected in public RPC mode.

==================================================
13. Release validation
======================

Run:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.5-testnet -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.5-testnet -SkipTests

Optional Bash:

bash ./scripts/build.sh v0.4.5-testnet --skip-tests
bash ./scripts/package.sh v0.4.5-testnet --skip-tests

Expected:

* Windows/Linux binaries build.
* release archives created.
* SHA256SUMS.txt created.
* docs included.
* UI still embedded or packaged.
* no runtime/private state included.

==================================================
14. Done criteria
=================

Phase 4.5 valid if:

* all tests pass.
* explorer search endpoint works or frontend fallback is robust.
* search works in UI for height/hash/txid/address.
* API pagination is capped and validated.
* UI pagination works.
* empty/loading/error states are friendly.
* invalid/not-found states are handled.
* copy buttons work.
* mobile layout is usable.
* explorer remains read-only.
* wallet/admin public safety remains intact.
* docs updated.
* release build/package works.
