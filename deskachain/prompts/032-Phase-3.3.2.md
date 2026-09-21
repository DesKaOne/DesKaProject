Kamu sedang bekerja pada project Go monorepo IndoChain.

Status saat ini:

* Phase 1 sampai Phase 3.3.1 sudah valid.
* Phase 3.3 sudah valid full.
* go work sync pass.
* go test ./node/... pass.
* go test -count=1 ./node/... pass.
* localnet/testnet profile sudah terpisah.
* testnet genesis stabil dan berbeda dari localnet.
* localnet genesis tetap:
  6e1b3fed63a01109cb665c45c76796b7272b9fec08efcf057b6134a6dc71c22c
* testnet:

  * network: testnet
  * network_id: ind-testnet-1
  * chain_id: 777101
  * genesis hash: db0ec6a6425f3a16241c429e7fdf4f29ee4a40c4a6eead84dab2d0e0f356bbf4
* Multi-node controlled local testnet sudah jalan:

  * Node A testnet bootstrap.
  * Node B testnet pakai bootnode Node A.
  * Node A mine block.
  * Node B sync sampai height sama.
  * chain validate pass di dua node.
* Network mismatch guard sudah valid:

  * peer check testnet -> localnet rejected.
  * peer sync testnet -> localnet rejected even when local already up-to-date.
* Miner testnet valid.
* Service node testnet valid.
* Testnet staking/service params sudah profile-aware:

  * min stake amount: 100 dIDR
  * min service stake: 1000 dIDR
  * unbonding period: 100
* Staking tetap collateral-only.
* Service points tetap simulation-only.
* PoW tetap satu-satunya consensus block production.

Nama patch:
IndoChain Phase 3.3.2 — Multi-node Sync Regression & Peer Persistence Hardening

Tujuan:
Menguatkan fondasi testnet multi-node sebelum masuk faucet/explorer/public testnet planning.

Fokus:

* peer persistence,
* bootnode persistence,
* duplicate peer guard,
* restart node tetap ingat peer,
* sync setelah restart,
* sync setelah peer offline/online,
* peer scoring/status hardening,
* stale peer handling,
* sync result clarity,
* regression tests multi-node.

Non-goals:

* Jangan implement public testnet launch.
* Jangan implement faucet.
* Jangan implement explorer.
* Jangan implement DNS seed production.
* Jangan implement mining pool.
* Jangan implement PoS.
* Jangan implement validator set.
* Jangan implement staking reward.
* Jangan implement slashing.
* Jangan ubah address format dIDR.
* Jangan ubah private key format.
* Jangan ubah localnet genesis.
* Jangan ubah testnet genesis kecuali bug fatal.
* Jangan rewrite P2P besar.

==================================================

1. Peer store persistence
   ==================================================

Pastikan peer store menyimpan peer dengan field minimal:

* url
* node_id
* network
* network_id
* chain_id
* genesis_hash
* protocol_version
* height
* tip_hash
* status
* score
* source
* last_seen
* last_error

Jika struktur saat ini belum lengkap, tambahkan secara backward-compatible.

Rules:

* Peer store lama tetap bisa dibaca.
* Missing field diberi default aman.
* Tidak boleh panic saat membaca peer store lama/partial/corrupt.
* Corrupt peer store harus error jelas atau fallback dengan backup file, jangan silent data loss.

Tambahkan backup optional:

* jika peers.json corrupt, rename ke peers.json.bak.<timestamp> lalu mulai peer store kosong.
* Jika terlalu besar, minimal error jelas.

==================================================
2. Bootnode persistence
=======================

Saat node start dengan:

--bootnode http://127.0.0.1:9611

atau:

--bootnodes [http://127.0.0.1:9611,http://127.0.0.1:9613](http://127.0.0.1:9611,http://127.0.0.1:9613)

Expected:

* bootnode masuk peer store.
* source = bootnode atau flag.
* restart node tanpa flag tetap punya peer tersebut di peer list.
* duplicate bootnode tidak membuat duplicate entry.
* URL dinormalisasi agar trailing slash tidak membuat duplicate.

Contoh duplicate yang harus dianggap sama:

* http://127.0.0.1:9611
* http://127.0.0.1:9611/

==================================================
3. Peer add/list/check UX
=========================

Pastikan commands existing atau tambahkan bila perlu:

peer add <URL>
peer list
peer check <URL>
peer sync <URL>

Behavior:

* peer add valid testnet peer:

  * handshake dulu.
  * validate network_id/chain_id/genesis/protocol.
  * simpan peer jika valid.
* peer add localnet peer dari testnet:

  * reject network mismatch.
  * jangan simpan sebagai active peer.
  * boleh simpan rejected peer hanya jika project punya rejected status; kalau tidak, jangan simpan.
* peer list harus menampilkan:

  * url
  * node_id
  * height
  * network_id
  * chain_id
  * status
  * score
  * last_seen
  * last_error jika ada.

==================================================
4. Sync after restart
=====================

Manual target:

* Start Node A testnet.
* Start Node B testnet dengan bootnode Node A.
* Node B sync.
* Stop Node B.
* Node A mine beberapa block lagi.
* Start Node B lagi tanpa --bootnode.
* Node B masih punya peer A dari peer store.
* Node B bisa sync dari peer A sampai height sama.

Jika auto-sync on start belum ada, jangan over-engineer.
Minimal command manual ini harus berhasil setelah restart:

peer sync http://127.0.0.1:9611

Jika ingin menambah optional auto-sync:

* boleh tambahkan flag:
  --sync-on-start
* default false jika khawatir breaking.
* Kalau sudah ada auto sync, harden saja.

==================================================
5. Offline peer handling
========================

Jika peer offline:

peer check <URL>

Expected:

* error jelas connection refused / timeout.
* peer status menjadi unreachable/offline jika peer sudah ada.
* score turun jika scoring sudah ada.
* node tidak crash.
* peer list menampilkan last_error.

Jika peer kembali online:

* peer check sukses.
* status active.
* last_error cleared atau diganti kosong.
* score recover sebagian jika sistem scoring mendukung.

Jangan membuat peer dihapus otomatis hanya karena sekali offline.
Gunakan threshold jika ingin pruning.

==================================================
6. Sync result clarity
======================

Rapikan output peer sync:

Case valid peer and imported blocks:

sync complete
peer: <URL>
imported blocks: N
height: H
tip hash: <HASH>

Case valid peer but already up to date:

sync complete
local chain already up to date
peer: <URL>
height: H
imported blocks: 0

Case local chain ahead:

sync complete
local chain ahead of peer
peer height: X
local height: Y
imported blocks: 0

Case mismatch:
error: peer rejected: network id mismatch

Case offline:
error: peer unavailable: <reason>

Jangan return "sync complete" jika peer invalid/offline.

==================================================
7. Peer scoring hardening
=========================

Jika peer scoring sudah ada, rapikan rules sederhana:

Positive:

* successful check: +1
* successful sync: +2
* successful block import: +N small capped
* valid profile: +1

Negative:

* timeout/offline: -1 or -2
* network mismatch: strong penalty
* genesis mismatch: strong penalty
* invalid block: strong penalty
* protocol mismatch: penalty or reject

Rules:

* score punya min/max clamp.
* status derived:

  * active
  * stale
  * offline
  * rejected
  * banned optional if score too low

Jangan bikin ban permanen dulu kecuali sudah ada desain.

==================================================
8. Peer URL normalization
=========================

Implement helper if not exist:

NormalizePeerURL(raw string) (string, error)

Rules:

* require http/https scheme.
* require host.
* remove trailing slash.
* reject empty.
* reject invalid URL.
* keep port.
* maybe reject path/query for P2P peer URL unless existing API supports path.

Tests:

* http://127.0.0.1:9611 -> same
* http://127.0.0.1:9611/ -> http://127.0.0.1:9611
* "" rejected
* "127.0.0.1:9611" rejected or normalized only if current UX supports it. Prefer reject with clear message.
* "ftp://..." rejected.

==================================================
9. Multi-node regression tests
==============================

Tambahkan/update tests di internal/p2p dan internal/rpc.

Required tests:

1. TestPeerStorePersistsBootnode

* start/load peer store.
* add bootnode.
* save.
* reload.
* peer still exists.

2. TestPeerStoreDeduplicatesBootnode

* add same URL with and without trailing slash.
* only one peer.

3. TestPeerURLNormalize

* valid URL normalized.
* trailing slash removed.
* invalid URL rejected.

4. TestPeerListIncludesProfileFields

* peer list output/data contains network_id, chain_id, genesis_hash, height, status, score.

5. TestPeerCheckUpdatesLastSeenAndStatus

* successful check sets status active and last_seen non-empty.

6. TestPeerCheckOfflineUpdatesStatusAndLastError

* offline peer check fails.
* peer status offline/unreachable.
* last_error set.
* no crash.

7. TestPeerCheckAfterOfflineRecovery

* peer offline then online.
* check succeeds.
* status active.
* last_error cleared or updated.

8. TestPeerSyncAfterRestart

* Node A has height N.
* Node B has peer A saved in peer store.
* simulate Node B restart by reloading state/peer store.
* Node A mines more blocks.
* Node B syncs from saved peer.
* Node B height equals Node A.

9. TestPeerSyncDoesNotSayCompleteOnMismatch

* mismatch peer returns error.
* response/output does not include sync complete.

10. TestPeerSyncLocalAhead

* local height > peer height.
* valid peer.
* returns local ahead/up-to-date style result.
* imported blocks 0.
* no error.

11. TestPeerSyncInvalidBlockPenalizesPeer

* if test helper exists.
* peer sends invalid block.
* sync fails.
* peer score decreases / status rejected.

12. Existing Phase 3.3 tests still pass:

* TestPeerSyncTestnet
* TestPeerSyncRejectsNetworkMismatch
* TestPeerSyncRejectsNetworkMismatchEvenWhenLocalUpToDate
* TestPeerSyncRejectsGenesisMismatchEvenWhenLocalUpToDate
* TestPeerSyncUpToDateValidPeer
* TestRPCPeerSyncRejectsNetworkMismatch

==================================================
10. Manual validation commands
==============================

After patch:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/p2p -run "Peer|Bootnode|Sync|Normalize|Offline" -v
go test ./node/internal/rpc -run "Peer|Bootnode|Sync|NetworkMismatch" -v
go test ./node/internal/cli -run "Peer|Bootnode|Sync" -v

Manual flow:

Clean:

go run ./node/cmd/indochain --datadir ./testdata/persist_a dev reset --yes
go run ./node/cmd/indochain --datadir ./testdata/persist_b dev reset --yes

Init A:

go run ./node/cmd/indochain --datadir ./testdata/persist_a --network testnet init
go run ./node/cmd/indochain --datadir ./testdata/persist_a wallet new

Start A:

go run ./node/cmd/indochain --datadir ./testdata/persist_a node start --rpc :8711 --p2p :9711 --advertise-p2p http://127.0.0.1:9711

Init B:

go run ./node/cmd/indochain --datadir ./testdata/persist_b --network testnet init

Start B with bootnode:

go run ./node/cmd/indochain --datadir ./testdata/persist_b node start --rpc :8712 --p2p :9712 --advertise-p2p http://127.0.0.1:9712 --bootnode http://127.0.0.1:9711/

Check B peers:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer list

Expected:

* peer url normalized to http://127.0.0.1:9711
* peers: 1
* source bootnode/flag
* status active or known.

Mine A:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8711 --address <A_IND_ADDR> --threads 4 --max-blocks 2

Sync B:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer sync http://127.0.0.1:9711

Expected:

* sync complete
* imported blocks >= 1
* height equals A

Stop B.
Mine A again:

go run ./node/cmd/indominer --rpc-url http://127.0.0.1:8711 --address <A_IND_ADDR> --threads 4 --max-blocks 2

Restart B WITHOUT bootnode flag:

go run ./node/cmd/indochain --datadir ./testdata/persist_b node start --rpc :8712 --p2p :9712 --advertise-p2p http://127.0.0.1:9712

Check B peers:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer list

Expected:

* peer A still exists.

Sync B from saved peer:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer sync http://127.0.0.1:9711

Expected:

* sync complete.
* B catches up to A.

Offline peer test:

* Stop A.
* From B:

  go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer check http://127.0.0.1:9711

Expected:

* error peer unavailable / connection refused.
* peer list shows offline/unreachable and last_error.

Restart A.
From B:

go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer check http://127.0.0.1:9711

Expected:

* peer ok.
* status active.
* last_error cleared or no longer blocking.

Mismatch still rejected:

* Start localnet node C.
* From B testnet:

  go run ./node/cmd/indochain --rpc-url http://127.0.0.1:8712 peer sync http://127.0.0.1:<LOCALNET_P2P_PORT>

Expected:

* error: peer rejected: network id mismatch
* no sync complete.

==================================================
11. Docs update
===============

Update:

* docs/Testnet.md
* docs/Architecture.md
* Roadmap.md
* README.md / README-ID.md if needed.

Document:

* bootnode flag usage.
* peer persistence.
* restart sync flow.
* offline peer handling.
* mismatch rejection.
* controlled local testnet runbook.

Roadmap:

* Current phase: Phase 3.3.2 — Multi-node Sync Regression & Peer Persistence Hardening
* Next possible:

  * Phase 3.4 — Testnet Faucet Planning & Dev Faucet CLI
  * Phase 3.4.1 — Explorer API Read-only Index Planning
  * Phase 3.5 — Public Testnet Packaging

==================================================
12. Done criteria
=================

Phase 3.3.2 valid if:

* peer store persists bootnodes.
* duplicate bootnode normalized/deduplicated.
* node restart keeps peers.
* sync after restart works.
* offline peer does not crash node.
* offline status/last_error visible.
* peer recovery works.
* mismatch sync still rejected before up-to-date check.
* sync output no longer misleading.
* tests pass.
* docs updated.
