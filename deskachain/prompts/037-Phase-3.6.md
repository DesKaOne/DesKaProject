Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.5 sudah valid full.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* Phase 3.4 Dev Faucet & Testnet Funding Flow valid full.
* Phase 3.4.1 Faucet -> Stake 1000 -> Service Eligible E2E valid full.
* Phase 3.5 Public Testnet Node Packaging & Operator Runbook valid full.
* Public testnet node mode valid.
* Public RPC safety valid.
* Wallet/admin RPC disabled in public mode.
* Miner RPC explicit enable/disable valid.
* Faucet disabled by default in public mode.
* Service RPC explicit enable/disable valid.
* Datadir lock safety valid.
* Separate miner wallet flow valid.
* Chain info circulating supply semantics fixed and tested.
* Public node mining sanity valid.
* Staking remains collateral-only.
* Service points remain simulation-only.
* PoW remains the only block-production consensus.

Patch name:
DesKaChain Phase 3.6 — Public Testnet Bootstrap Registry & Seed Peer Config

Goal:
Make it easy for new testnet nodes to discover and connect to official/known seed peers without requiring manual `--bootnode` every time.

This phase should add:

1. built-in seed peer registry per network profile,
2. optional external seed peer config file,
3. deterministic peer normalization/deduplication,
4. startup bootstrap from built-in seeds + config seeds + CLI bootnodes,
5. safe public testnet docs,
6. tests for seed peer loading and mismatch safety.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not implement DNS seed crawler unless very small/simple.
* Do not implement peer gossip beyond current design unless already supported.
* Do not implement peer reputation overhaul.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not implement validator/PoS/slashing.
* Do not make service points spendable.
* Do not expose wallet/admin RPC publicly.
* Do not bypass network id/genesis validation.
* Do not hardcode private/local-only addresses as public production seeds except examples.

==================================================

1. Network profile seed peers
   ==================================================

Add seed peer support to network profiles.

Example conceptual fields:

* Profile.SeedPeers []string
* localnet seed peers default: empty
* testnet seed peers default: can include placeholder/example only if documented, or actual known seed URLs if available
* mainnet: empty/not implemented

Important:

* Do not break existing profile fields.
* Do not change chain_id/network_id/genesis.
* Seed peers are only discovery bootstrap hints.
* A bad seed must not corrupt chain or bypass validation.

Expected:

* `config.Localnet().SeedPeers` exists and is empty by default.
* `config.Testnet().SeedPeers` exists.
* If no official public seed exists yet, keep default empty and support config/env/CLI seeds.
* If adding example seed, use clearly documented placeholder such as:
  `http://127.0.0.1:9811` only in docs/tests, not as global public default.

Add tests:

* TestNetworkProfilesHaveSeedPeerField
* TestLocalnetHasNoDefaultPublicSeeds
* TestTestnetSeedPeersDoNotChangeGenesisOrNetworkID

==================================================
2. CLI/config seed peer inputs
==============================

Support multiple ways to provide seeds:

A. Existing:
--bootnode [http://host:port](http://host:port)

B. New optional repeated flag or comma-separated flag:
--seed-peer [http://host1:port](http://host1:port)
--seed-peer [http://host2:port](http://host2:port)

or:
--seed-peers [http://host1:port,http://host2:port](http://host1:port,http://host2:port)

Choose the style consistent with current CLI.

C. Optional config/env:
DKC_SEED_PEERS=[http://host1:port,http://host2:port](http://host1:port,http://host2:port)

D. Optional file:
--seed-file ./config/testnet-seeds.txt

Seed file format:

* one peer URL per line
* blank lines ignored
* `# comments` ignored
* supports `http://host:port`
* optional trailing slash normalized
* invalid URLs rejected with clear error or skipped with warning depending current style

Do not require seed file for normal operation.

Add tests:

* parse comma-separated seed peers.
* parse repeated seed peers if implemented.
* seed file ignores comments/blank lines.
* invalid seed URL returns clear error.
* duplicate seed URLs deduped after normalization.
* CLI bootnode + seed peer + profile seeds deduped.

==================================================
3. Startup bootstrap behavior
=============================

On node start:

* collect seed peers from:

  1. profile built-in seeds,
  2. env/config seeds,
  3. seed file,
  4. CLI --bootnode / --seed-peer flags.
* normalize URLs.
* dedupe.
* do not add self advertised P2P URL as peer.
* persist valid seed peers to peer store.
* attempt peer check/sync using existing safe path.
* network id and genesis mismatch must still reject.
* offline seed should not prevent node startup.
* log clear summary.

Startup summary should include:

* bootnodes count
* seed peers count
* persisted peers count
* skipped self peer count
* failed seed count if checked
* network id
* genesis hash

Example log lines:
seed peers loaded count=2 source=profile/config/cli
seed peer normalized url=http://example.org:9811
seed peer skipped reason=self url=http://127.0.0.1:9811
seed peer offline url=http://...
seed peer accepted url=http://...

Add tests:

* node start stores seed peers.
* self peer skipped.
* offline seed does not fail startup.
* wrong network seed rejected/penalized/skipped.
* wrong genesis seed rejected/penalized/skipped.
* restart keeps persisted good seeds.

==================================================
4. Peer store metadata
======================

If peer store already supports metadata, add seed metadata if simple.

Possible fields:

* source: manual / seed / bootnode / discovered
* first_seen
* last_seen
* last_error
* network_id
* genesis_hash
* score/status

Do not over-engineer.
If metadata is already present, extend carefully with backward compatibility.

Add tests:

* legacy peer store still loads.
* seed peer source persisted.
* duplicate URL does not create duplicate peer.
* remove peer still works.

==================================================
5. Public testnet operator docs
===============================

Update docs/Operator.md or create it if missing.

Add section:
“Joining the public testnet with seed peers”

Include:

* one-node quickstart.
* two-node local bootstrap example.
* public node with built-in/config seed peers.
* seed file example.
* Windows PowerShell examples.
* Linux/systemd examples.
* firewall ports.

Example seed file:

# DesKaChain testnet seed peers

http://seed1.example.org:9811
http://seed2.example.org:9811

Example command:

deskachain --datadir ./data/testnet --network testnet init

deskachain --datadir ./data/testnet node start 
--rpc :8811 
--p2p :9811 
--advertise-p2p http://<PUBLIC_HOST>:9811 
--public-rpc 
--enable-miner-rpc 
--seed-file ./config/testnet-seeds.txt

Windows go run example:

go run ./node/cmd/deskachain --datadir ./testdata/node_b node start --rpc :8812 --p2p :9812 --advertise-p2p http://127.0.0.1:9812 --public-rpc --enable-miner-rpc --seed-peer http://127.0.0.1:9811

Add warnings:

* Testnet DKC has no monetary value.
* Do not expose wallet/admin RPC publicly.
* Bootnodes/seeds are not trusted authorities.
* Every peer must still pass network id/genesis validation.
* A seed can be offline; node should still start.
* Back up wallet files before deleting datadir.

==================================================
6. README and API docs
======================

Update README.md and README-ID.md:

* Add Phase 3.6 summary.
* Add link to Operator docs.
* Add short seed peer example.
* Keep wording safe: public testnet preparation, not mainnet launch.

Update docs/API.md:

* clarify public node endpoints.
* clarify P2P bootstrap endpoints if documented.
* mention seed peers are startup config, not API endpoint unless implemented.

Update docs/Testnet.md:

* Add seed peer bootstrap flow.
* Add peer validation safety notes.

==================================================
7. Optional command: peer seeds
===============================

Optional, only if small:
Add CLI command:

peer seeds

It should print:

* built-in seed peers for active network.
* configured seed peers if available.
* normalized output.

Do not implement if it complicates architecture.

If implemented, tests:

* `peer seeds` on localnet shows none.
* `peer seeds` on testnet shows configured/listed seeds.
* respects datadir profile.

==================================================
8. Manual validation
====================

After patch:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/config -run "Seed|Profile|Testnet|Localnet" -count=1 -v
go test ./node/internal/p2p -run "Seed|Bootnode|Peer|Normalize|NetworkMismatch|GenesisMismatch|Restart" -count=1 -v
go test ./node/internal/cli -run "Seed|Bootnode|Operator|Peer" -count=1 -v
go test ./node/internal/rpc -run "Peer|Public|NetworkMismatch|GenesisMismatch" -count=1 -v

Manual local two-node seed flow:

Clean:

go run ./node/cmd/deskachain --datadir ./testdata/seed_a dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/seed_b dev reset --yes

Init:

go run ./node/cmd/deskachain --datadir ./testdata/seed_a --network testnet init
go run ./node/cmd/deskachain --datadir ./testdata/seed_b --network testnet init

Start seed node A:

go run ./node/cmd/deskachain --datadir ./testdata/seed_a node start --rpc :9111 --p2p :10111 --advertise-p2p http://127.0.0.1:10111 --public-rpc --enable-miner-rpc

Start node B using seed peer, not bootnode:

go run ./node/cmd/deskachain --datadir ./testdata/seed_b node start --rpc :9112 --p2p :10112 --advertise-p2p http://127.0.0.1:10112 --public-rpc --enable-miner-rpc --seed-peer http://127.0.0.1:10111/

Expected:

* B normalizes seed URL.
* B stores peer.
* B does not fail startup.
* peer list shows A.
* peer check A OK.

Commands:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 peer list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 peer check http://127.0.0.1:10111
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 peer sync http://127.0.0.1:10111

Mine A, sync B:

Create miner wallet:

go run ./node/cmd/deskachain --datadir ./testdata/seed_miner --network testnet init
go run ./node/cmd/deskachain --datadir ./testdata/seed_miner wallet new

Mine on A:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:9111 --address <MINER_ADDR> --threads 2 --once

Sync B:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 peer sync http://127.0.0.1:10111
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:9112 chain validate

Expected:

* B imports block.
* B height matches A.
* chain valid.

Seed file manual:

Create file:
./testdata/testnet-seeds.txt

Contents:

# local seed

http://127.0.0.1:10111/

Start B with:
--seed-file ./testdata/testnet-seeds.txt

Expected:

* same behavior as --seed-peer.
* comments ignored.
* trailing slash normalized.

Wrong network manual:

* Start localnet node C.
* Try to seed testnet node B with localnet C.
* Expected: rejected network id mismatch; B keeps running.

==================================================
9. Done criteria
================

Phase 3.6 valid if:

* seed peer config works from CLI/env/file or chosen supported sources.
* seed peers normalize and dedupe.
* self peer skipped.
* offline/bad seeds do not crash node startup.
* wrong network/genesis seeds rejected.
* persisted seed peers survive restart.
* docs explain public testnet bootstrap.
* public node safety from Phase 3.5 remains intact.
* all tests pass.
* manual two-node seed flow works.
