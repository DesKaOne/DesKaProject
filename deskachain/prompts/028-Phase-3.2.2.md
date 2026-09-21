Kamu sedang bekerja pada project Go monorepo IndoChain.

Struktur project:

* node/

  * cmd/indochain/
  * cmd/indominer/
  * cmd/indoservice/
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
* root repo punya:

  * go.work
  * README.md
  * README-ID.md
  * Roadmap.md
  * docs/
  * prompts/
  * testdata/
  * data/

Status saat ini:

* Phase 1 sampai Phase 3.2.1 sudah selesai dan valid.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* Staking regression tests sudah ada dan pass.
* Service node simulation + indoservice agent sudah pass.
* Public RPC hardening sudah pass.
* Standalone miner sudah pass.
* Address final sudah `iND...`.
* Staking tetap collateral-only, bukan PoS.
* Service points tetap simulation only, bukan spendable dIDR.

Nama patch:
IndoChain Phase 3.2.2 — Pre-Testnet Cleanup & Network Profile Plumbing

Tujuan:
Membersihkan repo dan memastikan network profile benar-benar mengalir dari CLI/config ke seluruh layer sebelum masuk Phase 3.3 multi-node testnet bootstrap.

Phase ini fokus pada:

* hapus runtime state file yang tidak seharusnya masuk repo,
* rapikan .gitignore,
* bersihkan docs dari address lama `iND1...`,
* sinkronkan Roadmap phase agar tidak mengarah ke PoS dulu,
* rename typo docs/Arsitecture.md menjadi docs/Architecture.md,
* hilangkan hardcode `config.Localnet()` di runtime path penting,
* pastikan `--network testnet` benar-benar mengubah network, chain_id, network_id, params, staking params, difficulty params, dan health/chain info/P2P handshake,
* tambahkan test localnet vs testnet.

Prinsip penting:

* Jangan ubah address format `iND...`.
* Jangan ubah private key format.
* Jangan ubah PoW consensus.
* Jangan ubah difficulty formula.
* Jangan ubah coinbase maturity behavior.
* Jangan ubah staking collateral behavior.
* Jangan implement PoS.
* Jangan implement validator set.
* Jangan implement staking reward.
* Jangan implement slashing.
* Jangan implement mining pool.
* Jangan implement explorer.
* Jangan rewrite besar.
* Patch ini cleanup + plumbing + tests.
* Semua command lama harus tetap bekerja.
* Semua test harus tetap pass:

  go work sync
  go test ./node/...
  go test -count=1 ./node/...

==================================================

1. Hapus runtime state file dari repo
   ==================================================

Cek root repo apakah ada file runtime lokal seperti:

* indoservice-state.json
* service_nodes.json
* service_challenges.json
* service_rewards.json
* peers.json
* mempool.json
* *.tmp

Jika ada di root repo atau folder non-test fixture:

* hapus dari tracked files.
* jangan hapus file di testdata kalau memang dipakai sebagai fixture test.
* jangan hapus file source/docs.

Tambahkan ke .gitignore:

indoservice-state.json
**/indoservice-state.json
service_nodes.json
service_challenges.json
service_rewards.json
peers.json
mempool.json
*.tmp
*.lock
data/
testdata/
!node/**/testdata/
!**/testdata/fixtures/
!**/testdata/*.json

Catatan:

* Jika project memang butuh testdata runtime untuk tests, jangan ignore/hapus fixture test yang dibutuhkan.
* Pastikan .gitignore tidak merusak test fixtures.

==================================================
2. Rapikan .gitignore prompts
=============================

Cek .gitignore apakah ada:

prompts/

Jika folder prompts/ memang bagian dokumentasi repo, hapus ignore tersebut.

Jika prompts/ hanya lokal, biarkan.
Namun karena project punya folder prompts/ untuk roadmap patch, rekomendasi:

* jangan ignore prompts/
* ignore hanya runtime/cache.

==================================================
3. Bersihkan docs dari address lama iND1...
===========================================

Cari semua kemunculan:

* iND1
* iND1...
* "address": "iND1..."
* "from": "iND1..."
* "to": "iND1..."

Command referensi:

grep -R "iND1" -n README.md README-ID.md Roadmap.md docs node 2>/dev/null

Atau Windows PowerShell:

Select-String -Path README.md,README-ID.md,Roadmap.md,docs* -Pattern "iND1" -Recurse

Ganti contoh address lama menjadi format final:

iND...

Contoh aman untuk docs:

iNDDExampleAddressReplaceWithRealWalletOutput

Atau lebih baik:

<IND_ADDR>

Rules:

* Jangan mengubah parser address.
* Ini docs cleanup saja.
* Jangan mengganti string test yang memang sengaja menguji invalid address lama, kalau ada.

==================================================
4. Rename docs typo
===================

Jika ada:

docs/Arsitecture.md

Rename menjadi:

docs/Architecture.md

Update semua link di README/Roadmap/docs dari:
Arsitecture.md
menjadi:
Architecture.md

Jika file tidak ada, skip.

==================================================
5. Sinkronkan Roadmap phase
===========================

Cek Roadmap.md.

Jika Phase 3.3 masih tertulis PoS / Hybrid Consensus Research, ubah arah roadmap agar sesuai desain aman sekarang.

Ubah menjadi:

Phase 3.2.2 — Pre-Testnet Cleanup & Network Profile Plumbing

* cleanup runtime files
* network profile plumbing
* testnet readiness checks

Phase 3.3 — Testnet Genesis/Profile & Multi-node Bootstrap

* official testnet profile
* testnet genesis
* seed/bootstrap peer
* multi-node runbook
* testnet node config examples

Catatan:

* PoS / Hybrid Consensus Research jangan dijadikan Phase 3.3.
* Kalau tetap mau dicatat, pindahkan ke future research / long-term research.
* Staking saat ini collateral-only, bukan PoS.

==================================================
6. Network profile plumbing audit
=================================

Cari semua hardcode:

config.Localnet()

di folder:

node/internal/
node/cmd/

Klasifikasikan:

A. Test-only usage:

* Boleh tetap `config.Localnet()` jika test memang explicit localnet.

B. Runtime default:

* Boleh hanya di tempat default config/CLI jika user tidak memilih network.

C. Runtime path penting:

* Harus diganti supaya pakai active network profile dari node config.

Runtime path penting termasuk:

* node start config
* RPC server
* P2P server
* chain info
* chain validate if network-specific params are needed
* balance
* staking
* service collateral
* miner template
* miner submit
* nodestate
* p2p handshake/status/sync
* p2p block/tx validation
* mining block candidate
* difficulty calculation
* address validation if version/prefix network-dependent
* health endpoint

Jangan mengganti blindly di tests.
Jangan membuat global mutable profile yang rawan race.
Lebih baik profile dipassing sebagai dependency ke runtime structs.

==================================================
7. Tambahkan/rapikan NetworkProfile di config
=============================================

Pastikan package config punya profile jelas:

type NetworkProfile struct {
Name string
NetworkID string
ChainID uint64
ProtocolVersion uint32
AddressPrefix string
AddressVersion byte
GenesisHash string optional
Consensus ConsensusParams
Staking StakingParams
P2P P2PParams
RPC RPCParams
}

Minimal harus ada:

* Localnet()
* Testnet()

Localnet expected:

* network: localnet
* network_id: ind-local-1
* chain_id: 777001
* address prefix: iND
* target block time: 10s
* retarget window: 10
* min difficulty: 1
* max difficulty: 8
* coinbase maturity: 10
* staking enabled: true
* min stake amount: 10 dIDR
* min service stake: 100 dIDR
* unbonding period: 10

Testnet expected placeholder:

* network: testnet
* network_id: ind-testnet-1
* chain_id: pilih existing kalau sudah ada; kalau belum, gunakan angka berbeda dari localnet, contoh 777101
* address prefix tetap iND untuk Phase ini, kecuali project sudah punya versi address network-specific.
* target block time boleh lebih realistis, contoh 30s atau tetap sesuai existing config.
* retarget window lebih besar dari localnet jika sudah disiapkan.
* coinbase maturity lebih besar dari localnet, contoh 20/50/100 sesuai existing.
* staking enabled true
* min service stake lebih besar dari localnet, contoh 1000 dIDR jika sudah ada.
* unbonding period lebih besar dari localnet.

Rules:

* Jangan ubah localnet params yang sudah dipakai tests kecuali update tests sengaja.
* Pastikan testnet ChainID berbeda dari localnet.
* Pastikan testnet NetworkID berbeda dari localnet.

==================================================
8. CLI network flag
===================

Pastikan `indochain` mendukung network selection konsisten.

Flags:

* --network localnet|testnet
* default localnet

Commands yang harus pakai active profile:

* init
* node start
* chain info remote/local
* balance
* mine/built-in jika ada
* stake info/lock/unlock/list/status
* service score/register if network-specific info needed
* p2p peer commands
* dev commands tetap local/dev oriented tetapi harus jelas.

Behavior:

* init dengan --network testnet harus menyimpan profile/network metadata ke datadir.
* node start dari datadir testnet harus start sebagai testnet.
* Jika datadir sudah init localnet, lalu start dengan --network testnet:

  * harus error jelas:
    datadir initialized for localnet, cannot start as testnet
    atau
  * ignore flag dan pakai datadir profile dengan warning jelas.
* Rekomendasi: error untuk mismatch.

==================================================
9. Datadir network metadata
===========================

Pastikan datadir menyimpan network metadata saat init.

File contoh:
network.json
atau metadata storage existing.

Isi minimal:
{
"network": "localnet",
"network_id": "ind-local-1",
"chain_id": 777001,
"genesis_hash": "..."
}

Untuk testnet:
{
"network": "testnet",
"network_id": "ind-testnet-1",
"chain_id": 777101,
"genesis_hash": "..."
}

Rules:

* Existing datadir localnet lama tetap bisa dibaca/migrated.
* Jika metadata missing pada datadir lama, default ke localnet dan tulis metadata saat next safe load jika tidak berisiko.
* Jangan merusak data test yang sudah ada.

==================================================
10. Genesis harus profile-specific
==================================

Pastikan genesis block/hash untuk localnet dan testnet tidak tertukar.

Options:
A. Testnet punya genesis berbeda karena chain_id/network_id/timestamp/genesis params beda.
B. Kalau belum mau genesis beda, minimal chain_id/network_id endpoint harus beda dan docs jelas testnet genesis TBD.

Rekomendasi:

* Buat genesis profile-specific.
* Localnet genesis tetap jangan berubah kalau tests bergantung padanya.
* Testnet genesis baru dengan network_id testnet.

Important:

* Jangan ubah localnet genesis hash yang sudah muncul di banyak tests/log:
  6e1b3fed63a01109cb665c45c76796b7272b9fec08efcf057b6134a6dc71c22c
  kecuali memang semua tests diupdate sengaja. Rekomendasi jangan ubah.

==================================================
11. RPC harus pakai active profile
==================================

Update RPC server agar menyimpan active NetworkProfile.

Endpoints yang harus mencerminkan active profile:

* /health
* chain info
* chain difficulty
* miner template
* miner submit validation
* balance
* stake info/list/status
* service score collateral params
* p2p status if exposed via RPC

Expected localnet /health:
{
"network": "localnet",
"network_id": "ind-local-1",
"chain_id": 777001
}

Expected testnet /health:
{
"network": "testnet",
"network_id": "ind-testnet-1",
"chain_id": 777101
}

If currently /health lacks network_id, add it.

==================================================
12. P2P harus pakai active profile
==================================

Update P2P server/client/handshake/status/sync agar pakai active NetworkProfile.

Handshake/status response harus mencantumkan:

* network
* network_id
* chain_id
* protocol_version
* genesis_hash
* height
* tip_hash
* cumulative_work

Rules:

* localnet node reject testnet peer.
* testnet node reject localnet peer.
* wrong chain_id rejected.
* wrong network_id rejected.
* peer score penalty if applicable.
* error message jelas:
  network mismatch
  chain id mismatch

Jangan lagi hardcode localnet di P2P runtime.

==================================================
13. NodeState / snapshots harus pakai active profile
====================================================

Cek internal/nodestate.

Jika snapshot/status masih pakai `config.Localnet()`, ubah agar menerima profile.

Node state output harus cocok:

* localnet chain id localnet.
* testnet chain id testnet.
* network_id active.

==================================================
14. Mining / miner template harus pakai active profile
======================================================

Miner template harus pakai active profile:

* chain_id
* network
* network_id
* difficulty params
* coinbase reward/maturity if applicable
* target from active difficulty params

indominer harus tetap compatible:

* localnet works.
* testnet works if RPC target is testnet.
* indominer tidak perlu tahu config selain dari template response.

==================================================
15. Staking/service harus pakai active profile
==============================================

Stake info harus profile-specific:

* localnet min service stake 100.
* testnet min service stake sesuai Testnet().
* unbonding period testnet sesuai Testnet().

Service collateral eligibility harus pakai active profile:

* localnet required stake localnet.
* testnet required stake testnet.

Jangan hardcode localnet staking params di service score.

==================================================
16. Tests wajib
===============

Tambahkan/update tests:

1. TestNetworkProfilesDifferent

* Localnet().ChainID != Testnet().ChainID
* Localnet().NetworkID != Testnet().NetworkID
* names beda.
* staking params boleh beda sesuai config.

2. TestInitWritesNetworkMetadataLocalnet

* init localnet datadir.
* metadata network localnet.
* chain_id localnet.

3. TestInitWritesNetworkMetadataTestnet

* init with --network testnet.
* metadata network testnet.
* chain_id testnet.

4. TestDatadirNetworkMismatchRejected

* init localnet.
* start/open with testnet.
* expect clear error.

5. TestRPCHealthUsesActiveProfileLocalnet

* localnet RPC health returns localnet network_id and chain_id.

6. TestRPCHealthUsesActiveProfileTestnet

* testnet RPC health returns testnet network_id and chain_id.

7. TestChainInfoUsesActiveProfileTestnet

* testnet chain info shows network testnet, chain id testnet.

8. TestStakeInfoUsesActiveProfile

* localnet stake info min service stake localnet.
* testnet stake info min service stake testnet.

9. TestServiceCollateralUsesActiveProfile

* localnet required stake localnet.
* testnet required stake testnet.

10. TestP2PHandshakeUsesActiveProfile

* localnet handshake returns localnet.
* testnet handshake returns testnet.

11. TestP2PRejectsNetworkMismatch

* localnet peer response rejected by testnet node or vice versa.

12. TestMinerTemplateUsesActiveProfile

* testnet miner template returns network testnet, chain_id testnet, network_id testnet.

13. TestNoRuntimeLocalnetHardcode

* Optional grep-style test or documented script.
* Ensure key packages don't call config.Localnet() except allowed files/tests.
* If too brittle, skip as unit test but document command.

14. Existing tests still pass:

* address
* amount
* chain
* cli
* config
* cpuminer
* crypto
* ledger
* mempool
* p2p
* rpc
* serviceagent
* servicenode
* staking
* types
* wallet

==================================================
17. Manual validation commands
==============================

After patch:

go work sync
go mod tidy
go test ./node/...
go test -count=1 ./node/...

Search cleanup:

PowerShell:
Select-String -Path README.md,README-ID.md,Roadmap.md,docs* -Pattern "iND1" -Recurse

Expected:

* no stale docs example, except intentional invalid address tests if searching node too.

Search Localnet runtime hardcode:

PowerShell:
Select-String -Path node\internal*.go,node\internal**.go,node\cmd*.go,node\cmd**.go -Pattern "config.Localnet("

Expected:

* allowed only in config defaults and tests.
* no runtime RPC/P2P path hardcode.

Localnet init:

go run ./node/cmd/indochain --datadir ./testdata/profile_local dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/profile_local --network localnet init
go run ./node/cmd/indochain --datadir ./testdata/profile_local node start --rpc :8511 --p2p :9511 --advertise-p2p http://127.0.0.1:9511

Check:

Invoke-RestMethod http://127.0.0.1:8511/health
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8511 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8511 stake info

Expected:

* network localnet
* network_id ind-local-1
* chain id 777001
* min service stake 100 dIDR

Testnet init:

go run ./node/cmd/indochain --datadir ./testdata/profile_test dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/profile_test --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/profile_test node start --rpc :8521 --p2p :9521 --advertise-p2p http://127.0.0.1:9521

Check:

Invoke-RestMethod http://127.0.0.1:8521/health
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8521 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8521 stake info

Expected:

* network testnet
* network_id ind-testnet-1
* chain id different from 777001
* min service stake testnet value, not 100 localnet if configured differently

Mismatch test:

go run ./node/cmd/indochain --datadir ./testdata/profile_local --network testnet node start --rpc :8531 --p2p :9531 --advertise-p2p http://127.0.0.1:9531

Expected:

* error clear:
  datadir initialized for localnet, cannot start as testnet

Miner template testnet:

go run ./node/cmd/indochain --datadir ./testdata/profile_test wallet new
Invoke-RestMethod "http://127.0.0.1:8521/miner/template?address=<IND_ADDR>"

Expected:

* network testnet
* network_id ind-testnet-1
* chain_id testnet

Optional miner testnet:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8521 --address <IND_ADDR> --threads 4 --once

Expected:

* submit accepted
* chain info testnet height 1

P2P mismatch manual optional:

* start localnet node.
* start testnet node.
* try peer add/connect between them.
  Expected:
* rejected network mismatch or chain id mismatch.

==================================================
18. Docs update
===============

Update:

* README.md
* README-ID.md
* Roadmap.md
* docs/Architecture.md
* docs/Staking.md
* docs/ServiceNode.md
* docs/ServiceAgent.md if needed

Mention:

* Phase 3.2.2 is pre-testnet cleanup.
* localnet and testnet profiles are separate.
* staking remains collateral-only.
* service points remain simulation-only.
* Phase 3.3 will handle multi-node bootstrap/testnet runbook.

==================================================
19. Non-goals
=============

Do not implement:

* public testnet launch yet,
* DNS seed,
* explorer,
* faucet,
* bootstrap server binary,
* PoS,
* validator set,
* staking reward,
* slashing,
* mainnet,
* production economics.

Phase 3.2.2 ends when:

* docs clean,
* runtime state ignored,
* network profile plumbed,
* localnet/testnet clearly different,
* RPC/P2P/miner/stake/service use active profile,
* tests pass.
