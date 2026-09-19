DesKaChain Phase 3.3.1 — Peer Sync Network Guard Fix

Context:
Phase 3.3 testnet multi-node bootstrap sudah berjalan:

* testnet genesis stable,
* Node A/B testnet sync berhasil,
* miner testnet berhasil,
* service node testnet berhasil,
* localnet/testnet `peer check` mismatch sudah ditolak.

Namun masih ada bug kecil:
Saat testnet node menjalankan:

peer sync http://127.0.0.1:<LOCALNET_P2P>

output masih:

sync complete
local chain already up to date
imported blocks: 0

Padahal `peer check` ke peer yang sama sudah benar menolak:

error: peer rejected: network id mismatch

Tujuan patch:
Pastikan `peer sync` selalu melakukan profile/handshake validation terhadap target peer sebelum return "already up to date".

Rules:

* `peer sync` harus reject peer dengan network_id mismatch.
* `peer sync` harus reject peer dengan chain_id mismatch.
* `peer sync` harus reject peer dengan genesis_hash mismatch.
* `peer sync` harus reject peer dengan incompatible protocol_version.
* Validation harus dilakukan walaupun local chain sudah sama tinggi atau lebih tinggi.
* Jangan return "local chain already up to date" sebelum peer profile dinyatakan valid.
* Jangan merusak sync testnet-to-testnet yang sudah berhasil.
* Jangan merusak localnet-to-localnet sync.

Implementation hint:
Audit sync path di internal/p2p dan RPC/CLI peer sync handler.

Current likely bug:

* sync code mengecek local height/cumulative work lebih dulu,
* lalu return already up to date,
* sehingga handshake/profile validation dilewati.

Fix order:

1. Fetch peer status/handshake.
2. Validate peer profile against active profile:

   * network_id
   * chain_id
   * genesis_hash
   * protocol_version
3. Only after peer valid:

   * compare height/cumulative work.
   * return already up to date if appropriate.
4. If peer invalid:

   * return error.
   * update peer status/score if system supports it.

Expected manual after patch:

Testnet node -> localnet peer:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer sync http://127.0.0.1:9621

Expected:

error: peer rejected: network id mismatch

or:

error: peer rejected: chain id mismatch

Testnet node -> testnet peer:

go run ./node/cmd/deskachain --rpc-url http://127.0.0.1:8612 peer sync http://127.0.0.1:9611

Expected:

* sync complete if valid,
* local chain already up to date only after peer profile validated.

Tests to add/update:

1. TestPeerSyncRejectsNetworkMismatchEvenWhenLocalUpToDate

* local node is already same height or higher.
* target peer has different network_id.
* peer sync returns mismatch error.

2. TestPeerSyncRejectsGenesisMismatchEvenWhenLocalUpToDate

* target peer has same height but different genesis.
* peer sync returns genesis mismatch.

3. TestPeerSyncUpToDateValidPeer

* target peer profile matches.
* local already up to date.
* returns local chain already up to date.

4. Existing Phase 3.3 sync tests still pass:

* two testnet nodes sync.
* localnet/testnet peer check mismatch rejected.
* miner template testnet still works.

Final commands:
go work sync
go test ./node/...
go test -count=1 ./node/...
go test ./node/internal/p2p -run "PeerSync|NetworkMismatch|GenesisMismatch" -v
go test ./node/internal/rpc -run "PeerSync|NetworkMismatch" -v
