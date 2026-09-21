Kamu sedang bekerja pada project Go monorepo IndoChain.

Status saat ini:

* Phase 1 sampai Phase 4.3 sudah valid full.
* Phase 4.3 Public Testnet Explorer API & Read-Only Indexer Prep valid full.
* Explorer API read-only sudah berjalan:

  * /explorer/status
  * /explorer/blocks
  * /explorer/blocks/{height}
  * /explorer/block/{hash}
  * /explorer/tx/{txid}
  * /explorer/address/{address}
  * /explorer/address/{address}/txs
  * explorer stake/service summary
* Manual curl explorer di Mini PC sudah lolos.
* Public RPC safety tetap aman:

  * public_rpc=true
  * wallet_rpc=false
  * admin_rpc=false
  * faucet_rpc=false by default
  * service_rpc=false by default
* Real multi-host testnet sudah valid.
* Faucet -> stake -> service eligible multi-host sudah valid.
* Testnet dIDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
IndoChain Phase 4.4 — Public Testnet Explorer Web UI MVP

Goal:
Add a lightweight read-only web explorer UI for public testnet, backed by the Phase 4.3 explorer API.

This phase should provide a simple browser UI for:

* chain dashboard,
* recent blocks,
* block detail,
* transaction detail,
* address summary,
* address transaction history,
* stake summary,
* service node summary,
* search by height/hash/txid/address,
  without adding any wallet/admin/write functionality.

This is a minimal explorer UI MVP, not a full production explorer.

Non-goals:

* Do not launch mainnet.
* Do not change consensus.
* Do not change testnet genesis.
* Do not change block/tx format.
* Do not implement wallet web UI.
* Do not expose private keys.
* Do not add sign/send/stake/faucet write actions to UI.
* Do not add staking APY/reward claims.
* Do not make service points spendable.
* Do not promise price/profit/rewards.
* Do not add external analytics/tracking.
* Do not depend on external CDN if avoidable.
* Do not require Node.js/npm unless absolutely necessary.

==================================================

1. UI serving path
   ==================================================

Add static explorer UI served by the node.

Recommended route:

GET /explorer-ui

or:

GET /ui

Avoid breaking existing API routes under:

/explorer/status
/explorer/blocks
/explorer/tx/{txid}
etc.

If using `/explorer`, make sure API endpoints keep working.
Preferred safe option:

/explorer-ui
/explorer-ui/
/explorer-ui/assets/*

Optional redirect:

/explorer-ui -> /explorer-ui/

The UI should call the existing read-only explorer API on the same host by default.

Example:

* UI loaded from `http://127.0.0.1:9311/explorer-ui/`
* JS fetches `/explorer/status`
* JS fetches `/explorer/blocks?limit=20`

Do not expose wallet/admin endpoints.

==================================================
2. Frontend tech
================

Keep it simple.

Preferred:

* plain HTML
* plain CSS
* plain JavaScript
* no build step
* no npm dependency
* no external CDN
* no tracking
* no analytics

Suggested files:

node/web/explorer/index.html
node/web/explorer/app.js
node/web/explorer/styles.css

or equivalent source layout.

Embed via Go `embed` if appropriate:

//go:embed web/explorer/*
embed.FS

or serve from filesystem if existing architecture prefers it.

Release archive should include UI automatically if embedded.
If not embedded, package scripts must include static UI files.

==================================================
3. UI pages / views
===================

Implement MVP as either:

* single-page app with hash routing, or
* simple multi-page static UI.

Required views:

A. Dashboard
Path example:
/explorer-ui/
/explorer-ui/#/

Shows:

* network
* network_id
* chain_id
* height
* tip hash
* difficulty
* next difficulty
* total supply
* circulating supply
* pending tx count
* peer count
* public RPC flags
* mainnet available false
* testnet warning

Must show visible warning:
“Testnet dIDR has no monetary value.”
“Mainnet is not available.”
“Explorer is read-only.”

B. Blocks
Path example:
/explorer-ui/#/blocks

Shows:

* recent block table
* height
* hash
* timestamp
* tx count
* miner address
* reward
* difficulty

Controls:

* limit selector or simple pagination
* next/previous page using limit/offset

C. Block detail
Path example:
/explorer-ui/#/block/123
/explorer-ui/#/block/<hash>

Shows:

* height
* hash
* previous hash
* timestamp
* difficulty
* nonce
* tx count
* transactions table

D. Transaction detail
Path example:
/explorer-ui/#/tx/<txid>

Shows:

* txid
* status
* type
* block height/hash
* confirmations
* from/to
* amount
* fee
* involved addresses

E. Address detail
Path example:
/explorer-ui/#/address/<address>

Shows:

* address
* confirmed balance
* mature balance
* immature balance
* spendable balance
* active stake
* unlocking stake
* tx count
* first seen
* last seen
* service collateral eligibility
* service points warning if shown

Also shows recent tx history:

* txid
* type
* block height
* amount delta
* confirmations

F. Stake view
Path example:
/explorer-ui/#/stakes

Shows read-only stake records:

* stake id
* owner
* amount
* status
* lock height
* unlock height
* release height if available

G. Service view
Path example:
/explorer-ui/#/services

Shows:

* owner address
* endpoint
* score
* points
* required stake
* active stake
* eligible/not eligible
* simulation_only flag

Must visibly say:
“Service points are simulation-only and are not spendable dIDR.”

==================================================
4. Search
=========

Add search box in header.

Search should accept:

* block height integer,
* block hash,
* txid,
* dIDR address.

Behavior:

* If numeric, open block detail by height.
* If dIDR address, open address page.
* If hash-like string, try:

  1. tx detail,
  2. block detail,
     or use `/explorer/search?q=` if implemented.
* Show clear not found message.
* Show clear invalid input message.

No write actions.

==================================================
5. API client behavior
======================

Implement small JS API client.

Requirements:

* handles JSON fetch errors,
* handles HTTP 400/404 gracefully,
* shows loading state,
* shows error panel,
* avoids crashing on missing optional fields,
* formats timestamps,
* formats long hashes/address with copy button or shortened display,
* keeps full value accessible.

No secrets.
No local storage for private data.
Optional local storage only for UI preference like API base, but not necessary.

API base:

* default same origin.
* optional query param:
  `/explorer-ui/?api=http://127.0.0.1:9311`
  only if CORS is already handled or not needed.
* If adding configurable API base is too much, same-origin only is fine.

==================================================
6. Read-only safety
===================

UI must not include buttons/forms for:

* wallet new,
* wallet export,
* send tx,
* stake lock,
* stake unlock,
* faucet request,
* service register,
* service heartbeat,
* miner start,
* admin reset,
* dev reset,
* private key import/export.

Only read-only navigation and fetch.

Tests:

* explorer UI routes are GET-only.
* POST to UI/API read-only paths is rejected/method not allowed.
* wallet/admin endpoints remain disabled in public RPC mode.
* explorer UI is available in public RPC mode if intended.
* explorer UI does not require wallet/admin RPC.

==================================================
7. Styling
==========

Keep UI clean and lightweight.

Requirements:

* responsive layout for desktop/mobile.
* readable tables.
* dark-friendly or simple neutral theme.
* no external font/CDN required.
* header/nav:

  * Dashboard
  * Blocks
  * Stakes
  * Services
  * Search

Footer or banner:

* Testnet only.
* Testnet dIDR has no monetary value.
* Mainnet not available.
* Read-only explorer.

Do not include investment/profit wording.

==================================================
8. Backend static serving tests
===============================

Add tests in RPC or CLI package.

Tests should verify:

* GET /explorer-ui or /explorer-ui/ returns 200.
* HTML contains app root/title.
* static JS/CSS returns 200.
* explorer API still works.
* public RPC mode serves explorer UI.
* wallet/admin endpoints remain disabled.
* unknown static asset returns 404.
* POST to static UI route is rejected or safe.

If the project uses Go embed:

* test embedded files exist.

==================================================
9. Explorer UI smoke tests
==========================

Optional simple tests:

* index.html references app.js and styles.css.
* app.js contains API endpoint references.
* no obvious write endpoint strings in app.js:

  * `/wallet/new`
  * `/wallet/export`
  * `/send`
  * `/stake/lock`
  * `/stake/unlock`
  * `/faucet/request`
  * `/service/register`
  * `/admin`
  * `/dev/reset`

This is a safety smoke test, not a full frontend test.

==================================================
10. Docs
========

Update:

docs/Explorer.md
docs/DeployTestnet.md
docs/MultiHostTestnet.md
README.md
README-ID.md

Add:

* Explorer Web UI MVP section.
* How to open UI:

  * `http://127.0.0.1:9311/explorer-ui/`
  * Tailscale/LAN example:
    `http://100.101.251.7:9311/explorer-ui/`
* Explain UI is read-only.
* Explain no wallet/admin/write actions.
* Explain API is still available under `/explorer/*`.
* Explain service points are simulation-only.
* Explain testnet dIDR has no monetary value.
* Explain mainnet not available.

==================================================
11. Release/package
===================

If static UI is embedded into binary:

* no package change needed except docs.

If static UI is served from filesystem:

* update build/package scripts to include UI files in release archives.

Ensure release archive includes:

* binaries,
* docs/Explorer.md,
* UI static files if not embedded.

Ensure release archive excludes:

* datadirs,
* wallets,
* private keys,
* faucet state,
* service state,
* testdata,
* .env with secrets.

==================================================
12. Tests to run
================

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/rpc -run "Explorer|Public|Health|UI|Static|Wallet|Admin" -count=1 -v

go test ./node/internal/cli -run "Explorer|Public|Release|UI|Wallet|Admin" -count=1 -v

If a new internal explorer UI package exists:

go test ./node/internal/explorerui -count=1 -v

Existing explorer tests must still pass:

go test ./node/internal/rpc -run "ExplorerStatus|ExplorerBlock|ExplorerTx|ExplorerAddress|ExplorerStake|ExplorerService" -count=1 -v

Build/release validation:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.4-testnet -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.4-testnet -SkipTests

Optional Bash:

bash ./scripts/build.sh v0.4.4-testnet --skip-tests
bash ./scripts/package.sh v0.4.4-testnet --skip-tests

==================================================
13. Manual validation
=====================

Use a node with existing testnet data and explorer API.

Start node:

./indochain --datadir ./data/seed node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc

Open browser:

http://127.0.0.1:9311/explorer-ui/

or remote/Tailscale:

http://100.101.251.7:9311/explorer-ui/

Manual checks:

* dashboard loads.
* network shows testnet.
* height/tip matches `/explorer/status`.
* block table loads recent blocks.
* block detail opens.
* tx detail opens.
* address page opens.
* address tx history loads.
* stakes page loads.
* services page loads.
* search works for:

  * block height,
  * block hash,
  * txid,
  * dIDR address.
* invalid search shows friendly error.
* mobile width still readable.

Safety checks:

* no wallet/admin/write buttons.
* public RPC wallet new remains rejected.
* explorer API still works with curl.
* `/health` still works.

==================================================
14. Done criteria
=================

Phase 4.4 valid if:

* all tests pass.
* Explorer Web UI is served by node.
* dashboard works.
* blocks page works.
* block detail works.
* tx detail works.
* address page works.
* address tx history works.
* stake read-only page works.
* service read-only page works.
* search works.
* invalid/not-found states are handled.
* UI is read-only.
* no wallet/admin/write actions in UI.
* public RPC safety remains intact.
* docs updated.
* release build/package still works.
* manual browser validation passes.
