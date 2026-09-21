Kamu sedang bekerja pada project Go monorepo IndoChain.

Status saat ini:

* Phase 1 sampai Phase 3.2.1 sudah valid.
* Phase 3.2.2 sudah dipatch sebagian besar.
* Network profile plumbing sudah mulai masuk:

  * config.Localnet()
  * config.Testnet()
  * localnet chain_id 777001
  * testnet chain_id 777101
  * localnet network_id ind-local-1
  * testnet network_id ind-testnet-1
  * network metadata network.json
  * RPC/P2P/profile tests sebagian sudah ada.
* Namun sebelum lanjut Phase 3.3, masih ada blocker kecil:

  1. file runtime `indoservice-state.json` masih ikut root repo/zip.
  2. file typo `docs/Arsitecture.md` masih ada.
  3. `node/internal/p2p/reorg.go` masih hardcode `config.Localnet()` pada runtime path.
  4. RPC built-in `/mine` masih perlu validasi address memakai active profile.

Nama patch:
IndoChain Phase 3.2.2.1 — Pre-Testnet Final Cleanup & Profile Reorg Fix

Tujuan:
Menutup sisa blocker Phase 3.2.2 agar aman lanjut ke Phase 3.3 multi-node testnet bootstrap.

Non-goals:

* Jangan implement Phase 3.3 dulu.
* Jangan implement seed node.
* Jangan implement explorer.
* Jangan implement faucet.
* Jangan implement PoS.
* Jangan implement validator set.
* Jangan implement staking reward.
* Jangan implement slashing.
* Jangan ubah address format `iND...`.
* Jangan ubah private key format.
* Jangan ubah localnet genesis hash.
* Jangan ubah staking behavior.
* Jangan rewrite besar.

==================================================

1. Hapus runtime state file
   ==================================================

Pastikan file ini tidak ada di repo root:

indoservice-state.json

Jika ada:

* hapus dari working tree.
* jika tracked git, untrack.

Command manual referensi:

git rm --cached indoservice-state.json

atau jika belum tracked:

rm indoservice-state.json

Pastikan .gitignore tetap punya:

indoservice-state.json
**/indoservice-state.json
service_nodes.json
service_challenges.json
service_rewards.json
peers.json
mempool.json
*.tmp
*.lock

Jangan hapus test fixture yang memang dipakai tests.

==================================================
2. Hapus file typo docs/Arsitecture.md
======================================

Saat ini ada dua file:

docs/Architecture.md
docs/Arsitecture.md

Hapus file typo:

docs/Arsitecture.md

Pastikan semua link di README.md, README-ID.md, Roadmap.md, dan docs lain hanya mengarah ke:

docs/Architecture.md

Cari:

Arsitecture

Expected:

* tidak ada hasil.

==================================================
3. Profile-aware P2P reorg
==========================

Audit:

node/internal/p2p/reorg.go

Masalah:

* masih ada runtime hardcode:

  config.Localnet()

Contoh yang harus diganti:

* net := config.Localnet()
* ledger.ReplayMature(..., config.Localnet().Consensus)

Buat versi profile-aware.

Tambahkan function baru:

BuildReorgPlanWithProfile(paths NodePaths, peer string, maxDepth int, profile config.NetworkProfile) (*ReorgPlan, error)

ApplyReorgWithProfile(paths NodePaths, peer string, maxDepth int, yes bool, profile config.NetworkProfile) error

RevalidateMempoolAgainstLedgerWithProfile(paths NodePaths, profile config.NetworkProfile) error

Nama dan signature boleh menyesuaikan kode existing, tapi intinya:

* semua reorg validation/replay pakai profile aktif.
* chain_id/network_id/consensus params dari profile aktif.
* no runtime `config.Localnet()` di path reorg utama.

Wrapper lama boleh tetap ada untuk backward compatibility:

func BuildReorgPlan(...) {
return BuildReorgPlanWithProfile(..., config.Localnet())
}

func ApplyReorg(...) {
return ApplyReorgWithProfile(..., config.Localnet())
}

func RevalidateMempoolAgainstLedger(...) {
return RevalidateMempoolAgainstLedgerWithProfile(..., config.Localnet())
}

Tapi RPC/CLI runtime wajib pakai versi WithProfile.

==================================================
4. Update RPC reorg handlers
============================

Cari pemanggilan di RPC seperti:

p2p.BuildReorgPlan(...)
p2p.ApplyReorg(...)
p2p.RevalidateMempoolAgainstLedger(...)

Ganti ke versi profile-aware:

p2p.BuildReorgPlanWithProfile(..., h.profile())
p2p.ApplyReorgWithProfile(..., h.profile())
p2p.RevalidateMempoolAgainstLedgerWithProfile(..., h.profile())

Sesuaikan dengan cara handler menyimpan active profile saat ini.

Expected:

* RPC reorg preview/apply pada testnet memakai testnet consensus params.
* RPC localnet tetap memakai localnet.

==================================================
5. Update CLI reorg commands
============================

Cari pemanggilan CLI reorg/dev sync yang memakai:

p2p.BuildReorgPlan(...)
p2p.ApplyReorg(...)
p2p.RevalidateMempoolAgainstLedger(...)

Ganti ke versi profile-aware dengan active profile CLI/datadir:

p2p.BuildReorgPlanWithProfile(..., activeProfile)
p2p.ApplyReorgWithProfile(..., activeProfile)
p2p.RevalidateMempoolAgainstLedgerWithProfile(..., activeProfile)

Jika CLI belum punya active profile di scope command:

* load dari datadir metadata.
* fallback localnet hanya untuk datadir lama tanpa metadata.
* jangan hardcode localnet untuk datadir testnet.

Mismatch behavior:

* kalau datadir network testnet tapi command dipaksa localnet, error jelas.
* kalau datadir metadata missing, default localnet dengan backward-compatible warning jika project sudah punya warning pattern.

==================================================
6. RPC `/mine` address validation pakai active profile
======================================================

Audit RPC built-in mine endpoint.

Masalah:

* kemungkinan masih memakai:

  crypto.ValidateAddress(req.Address)

Ganti ke profile-aware validation jika project sudah punya:

crypto.ValidateAddressForNetwork(req.Address, h.profile())

atau equivalent dari package address/crypto.

Rules:

* localnet address tetap valid.
* testnet address validation pakai active profile.
* Jika testnet masih pakai prefix/version iND yang sama, hasil tetap sama, tapi path sudah future-proof.
* Error jelas jika address invalid untuk network aktif.

Endpoint yang harus dicek:

* `/mine`
* `/miner/template`
* `/miner/submit` jika ada address validation tambahan
* service register jika address validation network-aware tersedia
* stake lock/unlock jika address validation network-aware tersedia

Jangan breaking existing iND address.

==================================================
7. Search hardcode runtime Localnet
===================================

Setelah patch, jalankan search:

PowerShell:

Select-String -Path node\internal*.go,node\internal**.go,node\cmd*.go,node\cmd**.go -Pattern "config.Localnet("

Expected:

* allowed:

  * config defaults
  * tests
  * compatibility wrapper functions
  * old datadir fallback
* not allowed:

  * RPC runtime main path
  * P2P handshake/status/sync/reorg runtime path
  * miner template runtime path
  * stake/service profile runtime path
  * node state runtime path

Tambahkan komentar pada wrapper/fallback agar jelas intentional.

==================================================
8. Tests wajib
==============

Tambahkan/update tests:

1. TestP2PReorgUsesActiveProfile

* buat testnet profile.
* panggil BuildReorgPlanWithProfile atau helper reorg dengan profile testnet.
* pastikan consensus params yang dipakai berasal dari testnet, bukan localnet.
* Bisa pakai test spy/helper jika sulit.

2. TestP2PReorgRejectsNetworkMismatch

* localnet branch/peer data ditolak oleh testnet profile, atau sebaliknya.
* error mengandung network mismatch atau chain id mismatch.

3. TestRPCReorgUsesActiveProfile

* RPC handler testnet memanggil reorg dengan testnet profile.
* Minimal cek output/error memuat testnet network/chain_id jika endpoint mendukung.

4. TestMineRPCValidateAddressWithActiveProfile

* localnet mine valid address works.
* invalid address rejected.
* testnet profile path dipakai.
* Jika address version sama antar network, test cukup memastikan handler memakai profile by checking no Localnet hardcode or using a test profile with different address version if available.

5. TestNoRuntimeStateFilesInRepo

* optional lightweight test/script.
* fail jika root `indoservice-state.json` ada.
* Jika tidak cocok sebagai Go test, dokumentasikan manual check.

6. Existing tests tetap pass:
   go test ./node/...
   go test -count=1 ./node/...

==================================================
9. Manual validation
====================

Setelah patch:

go work sync
go mod tidy
go test ./node/...
go test -count=1 ./node/...

Search cleanup:

PowerShell:

Test-Path .\indoservice-state.json

Expected:
False

Test-Path .\docs\Arsitecture.md

Expected:
False

Select-String -Path README.md,README-ID.md,Roadmap.md,docs* -Pattern "Arsitecture" -Recurse

Expected:
no result

Runtime Localnet search:

Select-String -Path node\internal*.go,node\internal**.go,node\cmd*.go,node\cmd**.go -Pattern "config.Localnet("

Expected:

* no runtime reorg hardcode.
* only tests/default wrappers/fallback.

Localnet check:

go run ./node/cmd/indochain --datadir ./testdata/final_local dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/final_local --network localnet init
go run ./node/cmd/indochain --datadir ./testdata/final_local node start --rpc :8511 --p2p :9511 --advertise-p2p http://127.0.0.1:9511

Then:

Invoke-RestMethod http://127.0.0.1:8511/health

Expected:

* network localnet
* network_id ind-local-1
* chain_id 777001

Testnet check:

go run ./node/cmd/indochain --datadir ./testdata/final_test dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/final_test --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/final_test node start --rpc :8521 --p2p :9521 --advertise-p2p http://127.0.0.1:9521

Then:

Invoke-RestMethod http://127.0.0.1:8521/health
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8521 chain info
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8521 stake info

Expected:

* network testnet
* network_id ind-testnet-1
* chain_id 777101
* stake params testnet, not localnet

Miner template testnet:

go run ./node/cmd/indochain --datadir ./testdata/final_test wallet new

Invoke-RestMethod "http://127.0.0.1:8521/miner/template?address=<IND_ADDR>"

Expected:

* network testnet
* network_id ind-testnet-1
* chain_id 777101

Optional indominer testnet:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8521 --address <IND_ADDR> --threads 4 --once

Expected:

* submit accepted
* chain info still testnet
* chain validate pass

P2P mismatch optional:

* start one localnet node
* start one testnet node
* try peer add/connect/sync.
  Expected:
* rejected network mismatch / chain id mismatch.

==================================================
10. Docs update
===============

Update README/Roadmap/docs to mention:

* Phase 3.2.2.1 finalizes pre-testnet cleanup.
* runtime state files are ignored.
* Architecture doc path fixed.
* reorg path is profile-aware.
* testnet profile is now safe for Phase 3.3 bootstrap.

==================================================
11. Done criteria
=================

Patch valid jika:

* root indoservice-state.json hilang.
* docs/Arsitecture.md hilang.
* no stale Arsitecture links.
* p2p reorg runtime path no longer hardcodes localnet.
* RPC/CLI reorg use active profile.
* RPC /mine address validation uses active profile.
* localnet health/profile OK.
* testnet health/profile OK.
* testnet miner template OK.
* tests pass.
