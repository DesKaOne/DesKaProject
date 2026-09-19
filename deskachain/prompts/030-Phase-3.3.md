Kamu sedang bekerja pada project Go monorepo DesKaChain.

Struktur project:

* node/

  * cmd/deskachain/
  * cmd/dkcminer/
  * cmd/dkcservice/
  * internal/

    * address/
    * amount/
    * chain/
    * cli/
    * config/
    * cpuminer/
    * crypto/
    * ledger/
    * mempool/
    * mining/
    * nodestate/
    * p2p/
    * rpc/
    * serviceagent/
    * servicenode/
    * staking/
    * storage/
    * types/
    * wallet/
  * go.mod
  * go.sum
* root repo:

  * go.work
  * README.md
  * README-ID.md
  * Roadmap.md
  * docs/
  * prompts/

Status saat ini:

* Phase 1 sampai Phase 3.2.2.1 sudah valid.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* localnet/testnet profile sudah dipisah.
* testnet chain_id sudah berbeda dari localnet.
* testnet network_id sudah berbeda dari localnet.
* network metadata datadir sudah ada.
* RPC/P2P/miner/stake/service sudah profile-aware.
* P2P reorg sudah tidak hardcode localnet.
* runtime cleanup sudah beres.
* docs cleanup sudah beres.
* Staking tetap collateral-only, bukan PoS.
* Service points tetap simulation-only, bukan spendable DKC.
* PoW tetap satu-satunya block production consensus.

Nama patch:
DesKaChain Phase 3.3 — Testnet Genesis/Profile & Multi-node Bootstrap

Tujuan:
Membuat fondasi testnet multi-node yang bisa dijalankan secara lokal/controlled sebelum public testnet.

Fokus:

* testnet genesis/profile stabil,
* testnet datadir metadata,
* bootstrap peer / bootnode config,
* multi-node local testnet,
* peer connect/sync,
* network mismatch rejection,
* testnet miner flow,
* testnet service node flow controlled,
* docs/runbook testnet.

Non-goals:

* Jangan launch public testnet production.
* Jangan implement faucet publik.
* Jangan implement explorer.
* Jangan implement DNS seed production.
* Jangan implement PoS.
* Jangan implement validator set.
* Jangan implement staking reward.
* Jangan implement slashing.
* Jangan implement mainnet.
* Jangan implement mining pool.
* Jangan implement GPU miner.
* Jangan mengubah address format DKC.
* Jangan mengubah private key format.
* Jangan mengubah localnet genesis hash.
* Jangan rewrite besar.

==================================================

1. Stabilkan Testnet Profile
   ==================================================

Pastikan `config.Testnet()` punya parameter jelas dan stabil.

Expected:

* Network: `testnet`
* NetworkID: `dkc-testnet-1`
* ChainID: `777101`
* ProtocolVersion: sesuai current protocol
* Address prefix: tetap `DKC` untuk fase ini, kecuali project sudah mendukung version network-specific.
* Genesis harus testnet-specific.
* Testnet genesis hash harus berbeda dari localnet genesis hash.
* Testnet consensus params harus profile-specific.
* Testnet staking params harus profile-specific.

Localnet genesis hash jangan berubah:
6e1b3fed63a01109cb665c45c76796b7272b9fec08efcf057b6134a6dc71c22c

Jika testnet genesis belum berbeda:

* Buat testnet genesis berbeda secara deterministic.
* Jangan pakai timestamp random untuk genesis.
* Gunakan fixed timestamp/genesis message/network_id/chain_id agar hash stabil antar mesin.

Tambahkan test:

* localnet genesis hash tetap sama.
* testnet genesis hash stable.
* testnet genesis hash != localnet genesis hash.

==================================================
2. Testnet init metadata
========================

Pastikan:

deskachain --datadir <DIR> --network testnet init

menulis metadata:

{
"network": "testnet",
"network_id": "dkc-testnet-1",
"chain_id": 777101,
"genesis_hash": "..."
}

Rules:

* Start node harus membaca metadata dari datadir.
* Jika datadir sudah testnet, node start harus testnet walaupun flag network omitted.
* Jika datadir testnet dipaksa localnet, harus error mismatch.
* Jika datadir localnet dipaksa testnet, harus error mismatch.
* Datadir lama tanpa metadata fallback localnet dengan behavior backward-compatible.

==================================================
3. Bootstrap peer / bootnode flag
=================================

Tambahkan flag node start:

--bootnode <URL>
--bootnodes <URL1,URL2,...>

atau gunakan nama existing jika project sudah punya konsep peers.

Behavior:

* Saat node start, bootnodes dimasukkan ke peer store.
* Duplicate bootnode tidak menambah duplicate.
* Invalid URL rejected dengan error jelas.
* Localnet boleh localhost/private peer.
* Testnet controlled/local boleh localhost/private peer.
* Public mode nanti bisa punya policy terpisah.

Contoh:

go run ./node/cmd/deskachain --datadir ./testdata/tn2 node start --rpc :8612 --p2p :9612 --advertise-p2p http://127.0.0.1:9612 --bootnode http://127.0.0.1:9611

Expected:

* peer store berisi bootnode.
* node mencoba handshake/sync atau minimal peer check sesuai arsitektur existing.

==================================================
4. Multi-node local testnet
===========================

Target manual:

* Node A testnet start sebagai bootstrap.
* Node B testnet start dengan bootnode Node A.
* Node B connect ke Node A.
* Node A mine block.
* Node B sync sampai height sama.
* chain validate pass di keduanya.

Jika auto-sync belum continuous:

* Sediakan command manual sync:
  peer sync
  atau existing sync command.

Jangan over-engineer auto-sync jika belum siap.
Minimal:

* peer add/check/status/sync bekerja untuk testnet profile.

==================================================
5. P2P handshake profile check
==============================

P2P handshake/status harus mencantumkan:

* network
* network_id
* chain_id
* protocol_version
* genesis_hash
* height
* tip_hash
* cumulative_work

Rules:

* testnet peer hanya menerima testnet peer.
* localnet peer hanya menerima localnet peer.
* wrong chain_id rejected.
* wrong network_id rejected.
* wrong genesis_hash rejected.
* wrong protocol_version rejected jika incompatible.
* error jelas:

  * network mismatch
  * chain id mismatch
  * genesis mismatch
  * protocol mismatch

Tambahkan peer score penalty jika sudah ada scoring.

==================================================
6. Sync behavior testnet
========================

Pastikan sync/reorg path memakai active profile.

Testnet node sync harus:

* mengambil block dari peer testnet,
* validasi block dengan testnet profile,
* validasi cumulative work dengan profile aktif,
* replay ledger dengan testnet consensus params,
* staking validation dengan testnet staking params,
* reject block dari localnet peer.

Localnet behavior jangan rusak.

==================================================
7. Testnet miner flow
=====================

Pastikan miner template testnet mengembalikan:

{
"network": "testnet",
"network_id": "dkc-testnet-1",
"chain_id": 777101
}

Standalone miner harus bisa:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8611 --address <DKC_ADDR> --threads 4 --once

Expected:

* submit accepted.
* chain info tetap network testnet.
* chain validate pass.

Jika testnet difficulty terlalu berat, pakai params testnet yang masih aman untuk dev/local controlled testnet.
Jangan membuat mining manual terlalu lambat.

==================================================
8. Testnet staking/service params
=================================

Stake info pada testnet harus pakai params testnet.

Expected:

* min service stake testnet berbeda dari localnet jika sudah dikonfigurasi.
* unbonding period testnet berbeda dari localnet jika sudah dikonfigurasi.
* service score required stake memakai testnet params.

Test:

* localnet service required stake = 100 DKC.
* testnet service required stake = value dari config.Testnet(), misalnya 1000 DKC jika sudah diset.
* Jangan hardcode localnet di service collateral.

==================================================
9. Controlled service node on testnet
=====================================

Service RPC default public-rpc tetap disabled.

Untuk controlled local testnet, node bisa start dengan service RPC enabled:

--enable-service-rpc=true

atau default local/admin mode true.

Test flow:

* init testnet.
* start node testnet.
* wallet new.
* mine enough for stake if needed.
* stake lock required amount if possible.
* service register.
* dkcservice --once.
* service score shows network/testnet stake requirement.

Tetap:

* service points simulation only.
* no DKC reward.
* no supply mutation.

==================================================
10. Node startup logs
=====================

Node start harus log:

* network
* network_id
* chain_id
* genesis_hash
* protocol_version
* datadir
* rpc
* p2p
* advertise
* bootnodes count
* public_rpc
* wallet_rpc
* miner_rpc
* service_rpc
* max_peers
* height
* tip

Contoh:
node started network=testnet network_id=dkc-testnet-1 chain_id=777101 genesis=...

==================================================
11. Commands / CLI UX
=====================

Tambahkan/rapikan commands jika perlu:

1. network info
   Optional:
   go run ./node/cmd/deskachain --datadir <DIR> network info

Output:

* network
* network_id
* chain_id
* genesis_hash
* protocol_version

2. peer bootstrap/list/status/sync
   Gunakan existing peer commands jika ada.
   Pastikan output jelas:

* peer URL
* network_id
* chain_id
* genesis_hash
* height
* status connected/synced/rejected
* last error

3. node start bootnode flags

* --bootnode
* --bootnodes

==================================================
12. Docs / runbook
==================

Buat/update:

docs/Testnet.md

Isi:

* Phase 3.3 adalah controlled/local testnet bootstrap.
* Belum public production testnet.
* DKC testnet tidak punya nilai uang.
* Staking testnet bukan APY.
* Service points simulation only.
* Cara start Node A bootstrap.
* Cara start Node B dengan bootnode.
* Cara mining testnet.
* Cara sync Node B.
* Cara cek chain validate.
* Cara cek network mismatch.
* Cara run dkcservice controlled.

Update:

* README.md
* README-ID.md
* Roadmap.md
* docs/Architecture.md

Roadmap:

* Current phase: Phase 3.3 — Testnet Genesis/Profile & Multi-node Bootstrap
* Next possible:

  * Phase 3.3.1 — Multi-node Sync Regression Tests
  * Phase 3.4 — Testnet Faucet/Explorer Planning
  * jangan langsung PoS.

==================================================
13. Tests wajib
===============

Tambahkan/update tests:

1. TestTestnetGenesisStable

* testnet genesis hash non-empty.
* stable across repeated calls.
* testnet genesis != localnet genesis.

2. TestLocalnetGenesisUnchanged

* localnet genesis hash tetap:
  6e1b3fed63a01109cb665c45c76796b7272b9fec08efcf057b6134a6dc71c22c

3. TestTestnetInitWritesMetadata

* init testnet.
* metadata network testnet.
* chain_id 777101.
* network_id dkc-testnet-1.
* genesis_hash testnet.

4. TestStartUsesDatadirNetworkMetadata

* init testnet.
* start/open without --network.
* profile resolved testnet.

5. TestNetworkMismatchRejected

* init localnet.
* start/open as testnet rejected.
* init testnet.
* start/open as localnet rejected.

6. TestP2PHandshakeTestnet

* testnet node handshake returns testnet profile.

7. TestP2PRejectsLocalnetPeerFromTestnet

* localnet peer rejected by testnet node.

8. TestP2PRejectsTestnetPeerFromLocalnet

* testnet peer rejected by localnet node.

9. TestBootnodeAddedToPeerStore

* start node with bootnode.
* peer store contains bootnode once.
* duplicate bootnode not duplicated.

10. TestPeerSyncTestnet

* two testnet nodes.
* node A mines block or has taller chain.
* node B syncs.
* node B height equals node A.
* chain validate pass.

11. TestPeerSyncRejectsNetworkMismatch

* localnet source cannot sync into testnet node.

12. TestMinerTemplateTestnetProfile

* /miner/template on testnet returns network testnet, network_id dkc-testnet-1, chain_id 777101.

13. TestDkcMinerTestnetOnce

* optional integration if existing infra supports.
* mine one block on testnet via dkcminer-style RPC.
* chain valid.

14. TestStakeInfoTestnetProfile

* testnet stake info returns testnet staking params.

15. TestServiceCollateralTestnetProfile

* service score required stake uses testnet min service stake.

16. TestHealthIncludesGenesisAndNetworkID

* /health includes network_id and genesis_hash.
* localnet/testnet differ.

17. Existing tests still pass:
    go test ./node/...
    go test -count=1 ./node/...

==================================================
14. Manual validation commands
==============================

After patch:

go work sync
go mod tidy
go test ./node/...
go test -count=1 ./node/...

Clean old data:

go run ./node/cmd/deskachain --datadir ./testdata/tn1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/tn2 dev reset --yes

Init Node A testnet:

go run ./node/cmd/deskachain --datadir ./testdata/tn1 --network testnet init
go run ./node/cmd/deskachain --datadir ./testdata/tn1 wallet new

Start Node A:

go run ./node/cmd/deskachain --datadir ./testdata/tn1 node start --rpc :8611 --p2p :9611 --advertise-p2p http://127.0.0.1:9611

Check Node A:

Invoke-RestMethod http://127.0.0.1:8611/health
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 stake info

Expected:

* network testnet
* network_id dkc-testnet-1
* chain_id 777101
* genesis_hash testnet
* stake params testnet

Init Node B testnet:

go run ./node/cmd/deskachain --datadir ./testdata/tn2 --network testnet init

Start Node B with bootnode:

go run ./node/cmd/deskachain --datadir ./testdata/tn2 node start --rpc :8612 --p2p :9612 --advertise-p2p http://127.0.0.1:9612 --bootnode http://127.0.0.1:9611

Check Node B peers:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer list
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer check http://127.0.0.1:9611

Expected:

* peer accepted.
* network_id dkc-testnet-1.
* chain_id 777101.
* no network mismatch.

Mine Node A:

go run ./node/cmd/dkcminer --rpc-url http://127.0.0.1:8611 --address <NODE_A_DKC_ADDR> --threads 4 --max-blocks 3

Sync Node B:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer sync http://127.0.0.1:9611

Check both:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 chain info
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 chain validate
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 chain validate

Expected:

* Node B height equals Node A.
* both testnet.
* both chain valid.

Network mismatch manual:

go run ./node/cmd/deskachain --datadir ./testdata/ln1 dev reset --yes
go run ./node/cmd/deskachain --datadir ./testdata/ln1 --network localnet init
go run ./node/cmd/deskachain --datadir ./testdata/ln1 node start --rpc :8621 --p2p :9621 --advertise-p2p http://127.0.0.1:9621

Then from testnet node:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer check http://127.0.0.1:9621
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer sync http://127.0.0.1:9621

Expected:

* rejected network mismatch or chain id mismatch.

Miner template testnet:

Invoke-RestMethod "http://127.0.0.1:8611/miner/template?address=<NODE_A_DKC_ADDR>"

Expected:

* network testnet
* network_id dkc-testnet-1
* chain_id 777101

Controlled service node testnet optional:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 service register --address <NODE_A_DKC_ADDR> --endpoint http://127.0.0.1:9701
go run ./node/cmd/dkcservice --rpc-url http://127.0.0.1:8611 --address <NODE_A_DKC_ADDR> --endpoint http://127.0.0.1:9701 --once
go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8611 service score --address <NODE_A_DKC_ADDR>

Expected:

* service works if service_rpc enabled.
* required stake uses testnet params.
* simulation only, not DKC.

==================================================
15. Done criteria
=================

Phase 3.3 valid jika:

* testnet genesis stable and different from localnet.
* localnet genesis unchanged.
* testnet metadata written.
* node start resolves testnet from datadir metadata.
* mismatch network rejected.
* two testnet nodes can peer/check/sync.
* localnet/testnet peer mismatch rejected.
* miner template and dkcminer work on testnet.
* stake/service profile params use testnet.
* docs/Testnet.md exists.
* all tests pass.
