Patch DesKaChain Phase 4.10.1 — Reorg Sync Fork False-Positive Fix

Context:
Public testnet multi-miner test produced a fork.

Observed nodes:

* VPS:

  * network_id: idr-testnet-1
  * chain_id: 777101
  * genesis: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
  * height: 308
  * cumulative_work: 4194172929
  * tip: 000000c2770566641d1dbe49a59dada58b9d6ba528885f6944be2056bb702a8f

* Windows/Mini PC:

  * network_id: idr-testnet-1
  * chain_id: 777101
  * genesis: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
  * height: 300
  * cumulative_work: 4059955201
  * tip: 000000ac394563bcec4e78d26235b46daa5e20bcce8a76c8a8672e1d2698a39c

Problem:
When Windows/Mini PC try to sync from VPS, sync fails with:

sync failed: fork detected: peer rejected: network id mismatch

But this is wrong because `/p2p/handshake` and `/p2p/status` show the same:

* network_id
* chain_id
* genesis_hash

So the fork/reorg path is incorrectly returning or wrapping `network id mismatch`.

Goal:
Fix fork/reorg sync so that a peer with the same network_id, chain_id, genesis_hash, and higher cumulative_work can be used for reorg/import.

Do not change:

* testnet genesis
* localnet genesis
* consensus rules
* block format
* tx format
* PoW algorithm
* difficulty algorithm
* public RPC safety
* wallet/admin RPC exposure rules

Required behavior:

1. Before sync/reorg, validate peer identity using handshake/status:

   * network_id must match
   * chain_id must match
   * genesis_hash must match
   * protocol version must be compatible

2. If these match, do not return `network id mismatch`.

3. If local and peer tips differ:

   * If peer cumulative_work is higher, find common ancestor and reorg/import peer branch.
   * If peer cumulative_work is equal and tip differs, report fork tie / same work fork, not network mismatch.
   * If peer cumulative_work is lower, keep local chain and report local ahead.

4. Preserve safety:

   * wrong network_id must still reject
   * wrong chain_id must still reject
   * wrong genesis_hash must still reject
   * invalid peer blocks must still reject
   * invalid PoW/difficulty/timestamps must still reject
   * chain validate must pass after reorg

5. Improve error/log detail:
   When fork/reorg sync fails, log/return:

   * local_network_id
   * peer_network_id
   * local_chain_id
   * peer_chain_id
   * local_genesis
   * peer_genesis
   * local_height
   * peer_height
   * local_tip
   * peer_tip
   * local_cumulative_work
   * peer_cumulative_work
   * common_ancestor_height
   * common_ancestor_hash
   * decision
   * exact reject reason

6. Public RPC behavior:

   * public RPC must still reject write/action endpoint `peer sync`.
   * private/local RPC must allow `peer sync`.

Add/adjust tests:

* same network/genesis fork with peer higher cumulative_work must reorg successfully.
* same network/genesis fork with same cumulative_work and different tip must not return network mismatch.
* same network/genesis fork with peer lower cumulative_work should keep local chain.
* wrong network_id still rejected.
* wrong chain_id still rejected.
* wrong genesis_hash still rejected.
* public RPC `peer sync` still disabled.
* private/local RPC `peer sync` allowed.
* chain validate passes after reorg.
* error message must not say `network id mismatch` when local and peer network_id are equal.
* reproduce case:
  local height 300 / lower cumulative work
  peer height 308 / higher cumulative work
  same network_id, chain_id, genesis
  sync must reorg/import instead of false network mismatch.

Also fix existing failing test:
`TestMinerSubmitInvalidPoWStaleAndWrongDifficulty`

Current failure:

* test expects stale template
* actual response is duplicate block

Patch test logic:

* duplicate block and stale template should be separate test cases.
* duplicate test:

  1. mine valid block from template
  2. submit accepted
  3. submit same block again
  4. expect duplicate=true and reason duplicate block
* stale template test:

  1. get old template A
  2. advance chain using a different valid block/template
  3. submit a different candidate based on old template A
  4. expect stale/parent mismatch/current tip mismatch, not duplicate

Run:
go work sync
go test ./node/...
go test -count=1 ./node/...
go test ./node/internal/p2p -run "Reorg|Fork|Sync|NetworkMismatch|GenesisMismatch|Cumulative|CommonAncestor|HigherWork|SameWork" -count=1 -v
go test ./node/internal/rpc -run "PeerSync|Public|Admin|Reorg|Fork|Safety|Miner|Submit|Stale|Duplicate|Mining" -count=1 -v
go test ./node/internal/chain -run "Difficulty|Retarget|Cumulative|Validate|Work|Reorg" -count=1 -v

Manual validation after patch:

1. Start VPS branch at higher height/work.
2. Start Windows/Mini PC on lower fork.
3. Run private/local RPC:
   deskachain peer sync http://100.86.152.39:10311
4. Expected:

   * sync succeeds
   * Windows/Mini PC reorg/import to VPS branch
   * all nodes same height/tip
   * chain validate passes everywhere
5. Re-enable public RPC after repair and confirm:

   * wallet_rpc false
   * admin_rpc false
   * peer diagnostics read-only allowed
   * peer sync write action rejected in public mode

Done criteria:

* false network mismatch fixed
* higher cumulative work reorg works
* same-work fork does not produce wrong error
* tests pass
* manual three-node convergence passes
