Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 4.8 sudah valid full.
* Phase 4.6 Public Testnet RC1 valid full.
* Phase 4.7 GitHub Release & External Tester Onboarding valid full.
* Phase 4.8 Post-Release Monitoring & Feedback Loop valid full.
* RC1 version:

  * v0.4.6-testnet-rc1
* Public testnet node sudah jalan.
* Health check lokal sudah pass.
* Health check VPS sudah pass:

  * RPC: http://100.86.152.39:9311
  * network: testnet
  * network_id: idr-testnet-1
  * chain_id: 777101
  * height: 1049
  * public_rpc: true
  * wallet_rpc: false
  * admin_rpc: false
  * explorer_ok: true
* Explorer API/UI read-only sudah valid.
* Public RPC safety valid.
* Testnet IDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
DesKaChain Phase 4.9 — Public Testnet Multi-Seed & Peer Discovery Hardening

Goal:
Harden public testnet peer connectivity by supporting multi-seed bootstrap, better peer discovery, peer health scoring, retry/backoff behavior, and operator-visible peer diagnostics.

This phase should make the network more resilient when:

* one seed is offline,
* a seed is slow,
* a node restarts,
* a peer has stale height,
* a peer has wrong network/genesis,
* a public tester starts from a fresh datadir,
* multiple operator seeds are available.

This is networking/bootstrap hardening.
Do not change consensus or testnet genesis.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change block/tx format.
* Do not change PoW consensus.
* Do not add centralized trusted authority.
* Do not make seed peers trusted for consensus.
* Do not accept blocks without network/genesis validation.
* Do not expose wallet/admin RPC publicly.
* Do not promise price/profit/rewards.
* Do not give testnet IDR monetary value.
* Do not add staking APY.
* Do not make service points spendable.

==================================================

1. Multi-seed bootstrap behavior
   ==================================================

Harden node startup seed bootstrap.

Inputs should support:

* repeated CLI flag:
  --seed-peer http://host1:10311/
  --seed-peer http://host2:10311/
* seed file:
  --seed-file ./config/testnet-seeds.txt
* config/profile default seed list if already supported.
* persisted peers from prior runs.

Rules:

* Normalize peer URLs.
* Deduplicate peer URLs.
* Reject invalid URLs.
* Skip self-advertised URL.
* Track seed source:

  * cli
  * file
  * profile
  * persisted
  * discovered
* Try multiple seeds, not just first seed.
* Continue bootstrap if one seed fails.
* Use timeout per seed.
* Use retry/backoff.
* Do not block node startup forever if seeds are down.
* Log clear result:

  * seed tried
  * seed accepted
  * seed rejected
  * reason
  * network mismatch
  * genesis mismatch
  * offline/timeout
  * imported height
  * already up to date

Consensus safety:

* Always validate network_id.
* Always validate chain_id.
* Always validate genesis hash.
* Never trust seed height/tip without block validation.
* Seeds are discovery/bootstrap helpers only.

==================================================
2. Peer discovery from known peers
==================================

Add or harden peer discovery.

If current P2P status/handshake can expose known peers:

* include known peer list in peer status/handshake response.
* include advertise_p2p URL.
* include network_id, chain_id, genesis_hash, height, tip_hash.
* include only normalized safe peer URLs.

If not already present, add read-only endpoint on P2P side:

GET /p2p/peers

or extend existing status endpoint.

Discovery behavior:

* When connecting to a valid peer, learn its known peers.
* Add discovered peers to peer store.
* Deduplicate.
* Skip self.
* Skip invalid URL.
* Skip wrong network/genesis after check.
* Limit number of discovered peers accepted per peer to avoid spam.
* Do not let discovered peers override consensus.
* Add source metadata `discovered`.

Security:

* Apply max peers per response.
* Apply max stored peers.
* Reject non-http/https schemes if only HTTP P2P is supported.
* Avoid accepting local/private addresses from public peers unless explicitly allowed or already current behavior allows LAN/Tailscale.
* At minimum document address trust rules.

==================================================
3. Peer store metadata hardening
================================

Extend peer metadata if not already available:

Fields:

* url
* source
* first_seen
* last_seen
* last_success
* last_failure
* failure_count
* success_count
* score
* cooldown_until
* last_height
* last_tip_hash
* last_network_id
* last_chain_id
* last_genesis_hash
* last_latency_ms
* status:

  * unknown
  * active
  * offline
  * rejected
  * cooldown

Requirements:

* Preserve backward compatibility with old peer store.
* Migrate old peer store safely.
* Atomic write.
* Corrupt peer store handled gracefully.
* Do not delete chain/wallet data when clearing peers.

==================================================
4. Peer scoring and cooldown
============================

Improve peer scoring if needed.

Score rules:

* Successful check/sync increases score.
* Timeout decreases score.
* Wrong network/genesis heavily penalized or rejected.
* Invalid block penalizes peer.
* Repeated failure enters cooldown.
* Cooldown expires automatically.
* Valid recovery is possible after cooldown.
* Peer store clamps score within safe bounds.

Selection rules:

* Prefer active/high-score peers.
* Randomize peers with similar score to avoid hammering one seed.
* Do not always use the same seed.
* Avoid retrying a peer in cooldown.
* Persist updated score.

==================================================
5. Background peer maintenance loop
===================================

Add or harden optional peer maintenance loop.

Config options:

* --peer-maintenance-interval duration
* --peer-bootstrap-interval duration
* --min-peers int
* --target-peers int
* --max-peers int
* --peer-timeout duration
* --disable-peer-maintenance if needed

Defaults:

* safe for public testnet.
* not too aggressive.
* no high-frequency spam.
* max peer checks bounded.

Behavior:

* Periodically check known peers.
* Discover new peers from active peers.
* Sync from best valid peer if local is behind.
* Keep at least min_peers active if possible.
* Do not spam offline peers.
* Log concise peer maintenance summary:

  * active peers
  * total peers
  * attempted
  * failed
  * discovered
  * synced height if any

If background loop is too much for this phase:

* implement manual CLI peer discovery/check hardening first,
* document background maintenance as next phase.
  But prefer adding minimal bounded maintenance.

==================================================
6. Public read-only peer diagnostics
====================================

Expose safe read-only diagnostics through RPC public mode.

Add or harden endpoints/commands:

GET /peer/status
GET /peer/list
GET /peer/health
GET /peer/seeds

or equivalent existing routes.

Public mode must allow read-only peer diagnostics but must not expose admin write actions.

Response should include:

* active peer count.
* known peer count.
* seed count.
* best peer height.
* local height.
* local tip.
* peers summary:

  * url
  * source
  * status
  * score
  * last_height
  * last_success
  * last_failure
  * failure_count
  * latency_ms
* network_id.
* chain_id.
* genesis_hash.

Do not expose:

* private keys.
* wallet data.
* admin controls.
* local secrets.
* private env.

==================================================
7. CLI peer UX
==============

Improve CLI commands if not already available.

Recommended:

deskachain peer list
deskachain peer list --json
deskachain peer check <url>
deskachain peer sync <url>
deskachain peer add <url>
deskachain peer remove <url>
deskachain peer clear
deskachain peer discover
deskachain peer health
deskachain peer seeds

Behavior:

* `peer list` shows source/status/score/height.
* `peer health` summarizes local peer health.
* `peer discover` tries known active peers and imports their advertised peers.
* `peer seeds` shows configured + persisted seed peers.
* Public RPC mode allows read-only commands.
* Write commands require local/admin-safe mode if current safety model requires it.

Do not break existing commands.

==================================================
8. Health check script upgrade
==============================

Update:

scripts/testnet-health.ps1
scripts/testnet-health.sh

Add optional peer checks:

* `-MinPeers`
* `-ExpectedMinHeight`
* `-CheckPeerList`
* `-CheckSeed <url>`
* `-FailOnZeroPeers`

PowerShell examples:

powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 `    -RpcUrl http://100.86.152.39:9311`
-ExpectedNetwork testnet `    -ExpectedNetworkID idr-testnet-1`
-ExpectedChainID 777101

powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 `    -RpcUrl http://100.86.152.39:9311`
-MinPeers 1 `
-FailOnZeroPeers

Behavior:

* Zero peers is warning by default, not failure.
* If `FailOnZeroPeers` or `MinPeers` is set, fail when condition not met.
* Include peer_count and known_peer_count if API provides it.
* Keep wallet_rpc/admin_rpc false checks.

==================================================
9. Testnet seed config docs
===========================

Update:

config/testnet-seeds.example.txt
examples/testnet/testnet-seeds.txt
docs/SeedOperatorPublish.md
docs/SeedMonitoringChecklist.md
docs/PublicTestnetQuickstart.md
docs/OperatorChecklist.md
docs/PostReleaseMonitoring.md

Add multi-seed examples:

# Public testnet RC1 seeds

# Replace with operator-published seeds.

http://100.86.152.39:10311
http://100.101.251.7:10311

# http://<COMMUNITY_SEED>:10311

Explain:

* Seeds are not trusted authorities.
* Nodes validate network/genesis.
* Multiple seeds improve availability.
* Public seed operator should expose P2P port.
* RPC exposure is optional and must remain public-safe.

==================================================
10. Testnet topology doc
========================

Add:

docs/TestnetTopology.md

Include:

* Recommended topology:

  * VPS public seed.
  * Mini PC Tailscale/LAN seed.
  * Windows peer/miner.
  * optional community seeds.
* Port plan:

  * P2P: 10311
  * RPC: 9311
  * Explorer UI: /explorer-ui/
* Safe exposure:

  * P2P can be public.
  * RPC should be local/trusted unless public_rpc enabled.
  * wallet/admin RPC must not be public.
* Example diagrams as text.
* Example start commands for:

  * seed node,
  * public peer node,
  * miner node,
  * service node.
* Troubleshooting:

  * wrong network.
  * genesis mismatch.
  * peer offline.
  * firewall blocked.
  * peer_count zero.
  * local ahead.
  * stale peer.

==================================================
11. Tests
=========

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/p2p -run "Peer|Seed|Discover|Bootstrap|Score|Cooldown|Sync|NetworkMismatch|GenesisMismatch|Restart" -count=1 -v

go test ./node/internal/rpc -run "Peer|Explorer|Public|Health|UI|Wallet|Admin|Safety" -count=1 -v

go test ./node/internal/cli -run "Peer|Seed|Public|Wallet|Admin|Health" -count=1 -v

go test ./node/internal/config -run "Profile|Testnet|Seed|Genesis" -count=1 -v

Add tests:

* multiple seeds are parsed/deduped.
* invalid seed URL rejected.
* bootstrap continues after first seed fails.
* bootstrap succeeds with second seed.
* wrong network seed rejected.
* wrong genesis seed rejected.
* discovered peers added.
* discovered peer self URL skipped.
* discovered invalid peer skipped.
* peer score changes after success/failure.
* cooldown prevents immediate retry.
* peer recovery after cooldown.
* public read-only peer diagnostics allowed.
* public peer write/admin action rejected.
* peer store backward compatibility.
* corrupt peer store handled.
* health script peer options documented.

==================================================
12. Manual validation
=====================

Use current machines:

Known Tailscale/public nodes:

* VPS:

  * RPC: http://100.86.152.39:9311
  * P2P expected: http://100.86.152.39:10311
* Mini PC:

  * RPC: http://100.101.251.7:9311 if enabled
  * P2P expected: http://100.101.251.7:10311
* Windows:

  * RPC local: http://127.0.0.1:9312
  * P2P expected: http://100.83.159.107:10312

A. Start VPS seed:
./deskachain --datadir ./data/testnet --network testnet node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.86.152.39:10311 
--public-rpc 
--enable-miner-rpc

B. Start Mini PC seed:
./deskachain --datadir ./data/testnet --network testnet node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc 
--seed-peer http://100.86.152.39:10311

C. Start Windows peer with two seeds:
.\deskachain.exe --datadir .\data\testnet --network testnet node start `    --rpc 127.0.0.1:9312`
--p2p 0.0.0.0:10312 `    --advertise-p2p http://100.83.159.107:10312`
--public-rpc `    --enable-miner-rpc`
--seed-peer http://100.86.152.39:10311 `
--seed-peer http://100.101.251.7:10311

D. Validate:
.\deskachain.exe --rpc-url http://127.0.0.1:9312 peer list
.\deskachain.exe --rpc-url http://127.0.0.1:9312 peer health
.\deskachain.exe --rpc-url http://127.0.0.1:9312 chain info
.\deskachain.exe --rpc-url http://127.0.0.1:9312 chain validate

E. Failover test:

* Stop Mini PC seed.
* Restart Windows peer.
* Confirm it can still bootstrap from VPS.
* Start Mini PC seed again.
* Confirm peer discovery/peer health recovers.

F. Wrong network safety:

* Start a localnet node.
* Try adding it as seed to testnet.
* Expected: rejected network/genesis mismatch.

G. Health:
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 `    -RpcUrl http://127.0.0.1:9312`
-CheckPeerList

Optional:
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 `    -RpcUrl http://100.86.152.39:9311`
-FailOnZeroPeers

==================================================
13. Build/package validation
============================

Run:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.9-testnet-rc1 -SkipTests

powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.9-testnet-rc1 -SkipTests

Optional Bash:
bash ./scripts/build.sh v0.4.9-testnet-rc1 --skip-tests
bash ./scripts/package.sh v0.4.9-testnet-rc1 --skip-tests

Expected:

* Windows/Linux binaries build.
* release archives created.
* SHA256SUMS.txt created.
* docs included:

  * TestnetTopology.md
  * SeedMonitoringChecklist.md
  * PostReleaseMonitoring.md
  * KnownIssues.md
  * IssueTriage.md
* scripts included:

  * testnet-health.ps1
  * testnet-health.sh
* no runtime/private state included.

==================================================
14. Done criteria
=================

Phase 4.9 valid if:

* all tests pass.
* multiple seeds are supported and tested.
* bootstrap continues after failed seed.
* valid second seed can sync.
* wrong network/genesis seed rejected.
* peer discovery works or is safely documented if deferred.
* peer metadata/source/status visible.
* peer scoring/cooldown works.
* read-only peer diagnostics work in public RPC mode.
* health script can check peer status.
* docs updated for multi-seed topology.
* manual VPS + Mini PC + Windows multi-seed test passes.
* public wallet/admin safety remains intact.
* build/package works.
