Kamu sedang bekerja pada project Go monorepo IndoChain.

Status saat ini:

* Phase 1 sampai Phase 4.0 sudah valid full.
* Phase 4.0 Public Testnet Multi-Host Deployment valid full.
* Multi-host real test sudah berhasil:

  * Host A Mini PC Linux sebagai seed node.
  * Host B Windows sebagai peer node.
  * Block mined dan tersinkron ke dua host.
  * Host A dan Host B punya height sama.
  * Host A dan Host B punya tip hash sama.
  * Chain validate pass di dua host.
* CI dan release artifact workflow hijau.
* Release binary Windows/Linux valid.
* Public testnet genesis candidate stable:

  * network: testnet
  * network_id: ind-testnet-1
  * chain_id: 777101
  * genesis hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* Testnet dIDR has no monetary value.
* Mainnet does not exist yet.
* PoW remains the only block-production consensus.
* Staking remains collateral-only.
* Service points remain simulation-only.

Patch name:
IndoChain Phase 4.1 — Public Testnet Long-Running Stability & Restart Recovery

Goal:
Harden public testnet nodes for long-running operation across real hosts.

This phase should validate and improve:

* graceful shutdown,
* restart recovery,
* datadir lock safety,
* peer persistence,
* background peer reconnect,
* missed block recovery,
* periodic sync,
* systemd restart behavior,
* health/status observability,
* long-running mining/sync stability.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change consensus.
* Do not add staking rewards/APY.
* Do not add slashing/PoS.
* Do not make service points spendable.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not promise mining profit/rewards.
* Do not expose wallet/admin RPC publicly.
* Do not require centralized seed trust.

==================================================

1. Long-running node stability
   ==================================================

Review node runtime behavior for long-running public testnet use.

Ensure:

* node can run for extended time without panics,
* peer store persists across restart,
* service node store persists across restart,
* mempool persists safely where already supported,
* datadir lock is acquired on start and released on shutdown,
* second node process using same datadir is rejected clearly,
* Ctrl+C graceful shutdown works,
* systemd SIGINT graceful shutdown works,
* startup after unclean shutdown still validates chain.

Add or improve tests:

* datadir lock rejects duplicate process.
* lock released after graceful shutdown.
* restart keeps chain height/tip.
* restart keeps peer store.
* restart keeps mempool if this is expected behavior.
* chain validate passes after restart.

Do not change consensus or block format.

==================================================
2. Periodic peer reconnect and sync
===================================

If not already implemented, add a conservative background peer maintenance loop.

Purpose:

* if peer was offline during block broadcast,
* if node started before seed peer was reachable,
* if network temporarily failed,
* node should eventually reconnect and sync.

Behavior:

* periodically check known peers,
* skip self peers,
* throttle failed peers,
* avoid hammering offline nodes,
* sync from healthy peers when local node is behind,
* do not block mining on slow/offline peers,
* do not mark chain synced if peer has wrong network/genesis,
* do not panic on timeout/refused peer.

Config defaults:

* peer check interval: reasonable, e.g. 30s or 60s.
* peer sync interval: reasonable, e.g. 30s or 60s.
* peer timeout: reasonable, e.g. 3s to 10s.
* max sync peers per cycle: conservative.

Add flags/env only if project already has config style:

* `--peer-sync-interval`
* `--peer-check-interval`
* `IND_PEER_SYNC_INTERVAL`
* `IND_PEER_CHECK_INTERVAL`

If adding flags is too much, use internal defaults and document them.

Tests:

* node B starts while seed A is offline, later A starts, B eventually marks peer active.
* A mines while B is offline, B restarts and catches up.
* A mines while B is online but broadcast fails, B later catches up via periodic sync.
* wrong network peer never syncs.
* wrong genesis peer never syncs.
* slow peer does not block mining.
* peer failure is logged and throttled.

==================================================
3. Missed block recovery
========================

Add or verify missed block recovery.

Scenario:

1. Node A and Node B are connected.
2. Stop Node B.
3. Mine several blocks on Node A.
4. Restart Node B.
5. Node B should sync missing blocks from Node A.
6. Node B chain info should match Node A.

Expected:

* same height,
* same tip hash,
* same total supply,
* chain valid on both nodes.

Add automated test if feasible in p2p/rpc package:

* create two nodes,
* connect peer,
* mine/import blocks on A,
* simulate B offline,
* restart B with persisted peer,
* run sync,
* assert B catches up.

==================================================
4. Systemd restart recovery
===========================

Update or add docs:

docs/Systemd.md
docs/MultiHostTestnet.md
docs/DeployTestnet.md

Systemd unit should support:

* restart on crash,
* graceful stop using SIGINT,
* TimeoutStopSec 30,
* LimitNOFILE 65535,
* logs via journalctl,
* clean env file,
* no private keys in env.

Example commands:

sudo systemctl daemon-reload
sudo systemctl enable --now indochain-testnet
sudo systemctl status indochain-testnet
journalctl -u indochain-testnet -f

Restart test:

sudo systemctl restart indochain-testnet
./indochain --rpc-url http://127.0.0.1:9311 chain info
./indochain --rpc-url http://127.0.0.1:9311 chain validate

Failure recovery:

* kill process,
* systemd restarts it,
* chain remains valid,
* datadir lock does not remain stuck.

Do not add systemd behavior that exposes wallet/admin RPC.

==================================================
5. Health and status observability
==================================

Improve `/health` or node startup summary if needed.

Health should include:

* network,
* network_id,
* chain_id,
* genesis hash,
* height,
* tip hash,
* public_rpc,
* wallet_rpc,
* miner_rpc,
* admin_rpc,
* service_rpc,
* faucet_rpc,
* peer count if available,
* uptime if easy,
* datadir if already exposed in existing health style.

Do not expose secrets.

Optional CLI commands:

* `indochain node status`
* or improve existing `chain info`, `peer list`, and `/health` docs.

Tests:

* health includes network/genesis/rpc mode.
* public health does not expose wallet/private data.

==================================================
6. Long-run manual test checklist
=================================

Add doc:

docs/LongRunTestnet.md

Include checklist:

Host A: Mini PC seed node.
Host B: Windows/Linux peer node.

Test A — baseline:

1. Start Host A.
2. Start Host B with seed-peer Host A.
3. Confirm peer list/check.
4. Mine 1 block.
5. Confirm both nodes same height/tip.
6. Chain validate both nodes.

Test B — peer offline catch-up:

1. Stop Host B.
2. Mine 3 to 10 blocks on Host A.
3. Restart Host B.
4. Wait for peer sync.
5. Confirm Host B catches up.
6. Confirm both nodes same height/tip.
7. Chain validate both nodes.

Test C — seed restart:

1. Stop Host A.
2. Keep Host B running.
3. Restart Host A.
4. Confirm peers reconnect.
5. Mine 1 block on either host.
6. Confirm both nodes sync.

Test D — systemd restart:

1. Run Host A under systemd.
2. Mine at least 1 block.
3. `sudo systemctl restart indochain-testnet`
4. Confirm height/tip persists.
5. Confirm chain validate.

Test E — duplicate datadir lock:

1. Start Host A normally.
2. Attempt second node with same datadir.
3. Expected: fails clearly due datadir lock.

==================================================
7. CLI diagnostics for recovery
===============================

Document recovery commands:

Peer state:

./indochain --rpc-url http://127.0.0.1:9311 peer list
./indochain --rpc-url http://127.0.0.1:9311 peer check http://<peer>:<p2p>
./indochain --rpc-url http://127.0.0.1:9311 peer sync http://<peer>:<p2p>

Chain state:

./indochain --rpc-url http://127.0.0.1:9311 chain info
./indochain --rpc-url http://127.0.0.1:9311 chain validate

Logs:

journalctl -u indochain-testnet -f
journalctl -u indochain-testnet --since "30 minutes ago"

Expected errors should be documented:

* connection refused,
* timeout,
* wrong network id,
* wrong genesis hash,
* stale template,
* datadir locked.

==================================================
8. Release artifact impact
==========================

Update release docs if new flags/env are added.

Ensure:

* build scripts still work,
* package scripts still work,
* release archive still excludes runtime/private files,
* CI release artifacts still pass.

Run:

powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.1-testnet -SkipTests
powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.1-testnet -SkipTests

If using Bash:

bash ./scripts/build.sh v0.4.1-testnet --skip-tests
bash ./scripts/package.sh v0.4.1-testnet --skip-tests

==================================================
9. Required tests
=================

Run:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/p2p -run "Peer|Sync|Restart|Reconnect|Offline|Throttle|NetworkMismatch|GenesisMismatch" -count=1 -v

go test ./node/internal/rpc -run "Health|ChainInfo|Peer|Public|Miner|Restart" -count=1 -v

go test ./node/internal/cli -run "Peer|Sync|Restart|Lock|Datadir|Systemd|LongRun|Public" -count=1 -v

go test ./node/internal/config -run "Peer|Sync|Interval|Env|Profile|Testnet" -count=1 -v

If specific test names do not exist, keep full package tests passing and add relevant new tests where code changed.

==================================================
10. Manual validation
=====================

Manual multi-host long-run validation:

Host A Mini PC:

./indochain --datadir ./data/seed node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://<HOST_A_REACHABLE_IP>:10311 
--public-rpc 
--enable-miner-rpc

Host B Windows/Linux:

./indochain --datadir ./data/node_b node start 
--rpc 0.0.0.0:9312 
--p2p 0.0.0.0:10312 
--advertise-p2p http://<HOST_B_REACHABLE_IP>:10312 
--public-rpc 
--enable-miner-rpc 
--seed-peer http://<HOST_A_REACHABLE_IP>:10311/

Mine blocks:

./indominer --rpc-url http://127.0.0.1:9311 --address <ADDR_A> --threads 2 --max-blocks 3

or:

./indominer --rpc-url http://127.0.0.1:9312 --address <ADDR_B> --threads 2 --max-blocks 3

Check both:

./indochain --rpc-url http://127.0.0.1:9311 chain info
./indochain --rpc-url http://127.0.0.1:9312 chain info

./indochain --rpc-url http://127.0.0.1:9311 chain validate
./indochain --rpc-url http://127.0.0.1:9312 chain validate

Expected:

* same height,
* same tip hash,
* chain valid.

Offline catch-up:

1. Stop Host B.
2. Mine 3 blocks on Host A.
3. Restart Host B.
4. Wait for sync.
5. Confirm Host B catches up to Host A.

Systemd:

1. Run Host A as systemd.
2. Restart service.
3. Confirm height/tip persists.
4. Confirm chain validate.

==================================================
11. Done criteria
=================

Phase 4.1 valid if:

* all tests pass.
* peer persistence after restart is tested.
* offline catch-up works.
* missed block recovery works.
* slow/offline peer does not block mining.
* datadir lock behavior is safe.
* graceful shutdown and restart preserve chain validity.
* systemd restart docs are updated.
* health/status docs are updated.
* at least one real multi-host long-run/restart test passes.
* both nodes end with same height and same tip hash after restart/offline recovery.
* public safety remains intact.
