Patch DesKaChain Phase 4.10.2 — Testnet Reorg Depth Config & Deep Reorg Recovery

Context:
Phase 4.10.1 fixed the false-positive `network id mismatch` during fork sync.

Current result:
Fork sync now correctly reports matching network/chain/genesis:

* local network id: idr-testnet-1
* peer network id: idr-testnet-1
* local chain id: 777101
* peer chain id: 777101
* local genesis matches peer genesis

But sync still fails because:

decision: reorg_depth_exceeds_max
reason: reorg depth exceeds max depth

Observed case:

* Local height: 302
* Peer height: 308
* Common ancestor height: 276
* Reorg depth: 26
* Peer cumulative work is higher:

  * local cumulative work: 4062052353
  * peer cumulative work: 4194172929

Goal:
Allow safe testnet recovery from deeper forks when peer has higher cumulative work, while keeping reorg safety bounded and configurable.

Do not change:

* testnet genesis
* localnet genesis
* PoW consensus
* block format
* tx format
* difficulty algorithm
* public RPC safety
* wallet/admin exposure rules
* monetary assumptions

Requirements:

1. Make max reorg depth configurable.

Add config/CLI/env support:

* CLI flag:
  --max-reorg-depth <int>
* env:
  IDR_MAX_REORG_DEPTH=<int>

Suggested defaults:

* localnet: 64
* testnet: 128
* mainnet: not available / future default can be stricter

If the code already has a constant, replace it with profile/config-driven value.

2. Include max reorg depth in startup logs.

Example:

max reorg depth: 128

3. Include max reorg depth in sync error details.

When reorg is rejected, return/log:

* reorg_depth
* max_reorg_depth
* common_ancestor_height
* common_ancestor_hash
* local_height
* peer_height
* local_cumulative_work
* peer_cumulative_work
* decision: reorg_depth_exceeds_max

4. Allow higher-work reorg when depth <= max_reorg_depth.

Behavior:

* If peer cumulative_work > local cumulative_work
* network_id, chain_id, genesis all match
* common ancestor exists
* reorg_depth <= max_reorg_depth
  Then:
* import peer branch
* reorg to peer chain
* chain validate must pass
* mempool handling must stay safe
* log decision: reorg_apply_higher_work

5. Preserve safety.

Still reject:

* wrong network_id
* wrong chain_id
* wrong genesis_hash
* invalid PoW
* invalid difficulty
* invalid timestamp
* invalid block sequence
* reorg_depth > max_reorg_depth
* same-work fork unless explicit policy says no reorg

6. Public RPC behavior.

Keep:

* public RPC read-only diagnostics allowed
* public RPC `peer sync` write/action disabled
* private/local RPC `peer sync` allowed

7. Tests.

Add/update tests:

* testnet higher-work fork with depth 26 and max_reorg_depth 128 must reorg successfully.
* same fork with max_reorg_depth 10 must reject with `reorg_depth_exceeds_max`.
* rejection message includes reorg_depth and max_reorg_depth.
* wrong network still rejected.
* wrong genesis still rejected.
* same-work fork does not reorg and does not say network mismatch.
* chain validate passes after deep reorg.
* public RPC peer sync still disabled.
* private/local RPC peer sync allowed.
* startup config parses `--max-reorg-depth`.
* env `IDR_MAX_REORG_DEPTH` works if env config is supported.

8. Manual validation after patch.

Start VPS:
./deskachain --datadir ./data/testnet --network testnet node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.86.152.39:10311 
--public-rpc 
--enable-miner-rpc 
--max-reorg-depth 128

Start Mini PC:
./deskachain --datadir ./data/testnet --network testnet node start 
--rpc 127.0.0.1:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--enable-miner-rpc 
--seed-peer http://100.86.152.39:10311 
--max-reorg-depth 128

Run:
./deskachain --rpc-url http://127.0.0.1:9311 peer sync http://100.86.152.39:10311

Expected:

* sync succeeds
* reorg depth 26 accepted
* Mini PC height/tip matches VPS
* chain validate passes

Windows:
.\deskachain.exe --datadir .\data\testnet --network testnet node start `    --rpc 127.0.0.1:9312`
--p2p 0.0.0.0:10312 `    --advertise-p2p http://100.83.159.107:10312`
--enable-miner-rpc `    --seed-peer http://100.86.152.39:10311`
--seed-peer http://100.101.251.7:10311 `
--max-reorg-depth 128

Then:
.\deskachain.exe --rpc-url http://127.0.0.1:9312 peer sync http://100.86.152.39:10311

Expected:

* Windows height/tip matches VPS
* Mini PC height/tip matches VPS
* all chain validate pass

9. Run tests:

go work sync
go test ./node/...
go test -count=1 ./node/...

go test ./node/internal/p2p -run "Reorg|Fork|Sync|MaxReorg|Depth|HigherWork|CommonAncestor|NetworkMismatch|GenesisMismatch" -count=1 -v

go test ./node/internal/rpc -run "PeerSync|Public|Admin|Reorg|Fork|Safety" -count=1 -v

go test ./node/internal/cli -run "Reorg|MaxReorg|Config|Seed|Peer|Public" -count=1 -v

go test ./node/internal/config -run "Profile|Testnet|Reorg|Config" -count=1 -v

Done criteria:

* higher-work fork with depth 26 can reorg on testnet with max_reorg_depth >= 26
* depth limit still blocks too-deep reorg
* all nodes converge to same height/tip
* chain validate passes everywhere
* public RPC safety remains intact
