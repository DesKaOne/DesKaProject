# IndoChain Architecture

Phase 3.3 separates localnet and controlled testnet at the chain profile layer. Each datadir stores `network.json` with `network`, `network_id`, `chain_id`, and `genesis_hash`; CLI and runtime paths resolve this metadata when `--network` is omitted. Testnet uses a deterministic genesis marker that differs from localnet while preserving the localnet genesis hash.

P2P bootstrap uses the existing peer store plus `node start --bootnode` / `--bootnodes` and Phase 3.6 seed peers. Network profiles expose `SeedPeers`; operators can also pass `--seed-peer`, `--seed-peers`, `IND_SEED_PEERS`, config `p2p.seed_peers`, or `--seed-file` / config `p2p.seed_file`. Peer URLs are normalized before persistence, so bootnodes and seeds survive restart without duplicate trailing-slash variants. Self references are skipped based on `--advertise-p2p`. Peer metadata stores source (`flag`, `bootnode`, or `seed`), node ID, network ID, chain ID, genesis hash, protocol version, last height/tip, score, latency, status check time, and `last_error`. Handshake, status, sync, reorg, miner, staking, and service-node collateral paths validate against the active network profile before importing blocks.

Phase 3.4 adds a dev/testnet faucet RPC surface. Faucet state lives in `faucet_state.json` under the selected datadir. Faucet requests sign normal wallet transactions from the configured faucet address, enter the normal mempool, and require block mining for confirmation; the faucet does not bypass ledger validation or mutate total supply directly.

Phase 3.5 packages the public-testnet operator workflow. Public RPC mode keeps wallet/admin/miner/faucet/service write endpoints disabled by default, with miner RPC enabled only by explicit operator flag. Health/status responses and startup output include network identity, genesis, datadir/listen addresses, advertised P2P URL, and RPC mode toggles. Circulating supply is defined as mature coinbase supply and includes mature coins that are active/unlocking/released stake; spendable balance remains wallet-specific and excludes locked stake.

```txt
IndoChain/
├── node/                         # Core blockchain node Go
│   ├── cmd/
│   │   └── indochain/
│   ├── internal/
│   │   ├── amount/
│   │   ├── chain/
│   │   ├── cli/
│   │   ├── config/
│   │   ├── crypto/
│   │   ├── ledger/
│   │   ├── mempool/
│   │   ├── mining/
│   │   ├── nodestate/
│   │   ├── p2p/
│   │   ├── rpc/
│   │   ├── storage/
│   │   ├── types/
│   │   └── wallet/
│   ├── go.mod
│   ├── go.sum
│   └── README.md
├── apps/
│   ├── wallet_mobile/
│   ├── wallet_web/
│   ├── wallet_desktop/
│   ├── miner_desktop/
│   ├── miner_mobile/
│   ├── service_node_desktop/       # bandwidth/service node
│   └── explorer_web/
│
├── services/
│   ├── explorer_api/
│   ├── faucet/
│   ├── public_rpc/
│   ├── mining_pool/
│   ├── service_verifier/           # verifier bandwidth/uptime/latency
│   ├── service_reward_api/         # reward simulation/accounting
│   ├── swap_api/
│   └── bridge_relayer/
│
├── packages/
│   ├── wallet_core/
│   ├── protocol/
│   ├── sdk_go/
│   ├── sdk_dart/
│   └── sdk_ts/
│
├── contracts/
├── docs/
├── scripts/
├── deployments/
├── prompts/
├── testdata/
├── data/
├── go.work
└── README.md
```
