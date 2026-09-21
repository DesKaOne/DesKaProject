# Phase 8.8 — CI + Integration Testing

Phase 8.8 is the bounded CI integration gate for the Phase 8 testnet stack.

## Integrated path

1. Miner RPC commits a canonical block.
2. Canonical chain storage validates the committed tip.
3. A pending native transfer is visible through monitoring.
4. The persistent Explorer indexer catches up.
5. Explorer status, indexed statistics, and block views expose the read model.
6. /node/metrics exposes chain, mempool, and indexer observations.
7. Explorer Monitoring presents the /node/metrics contract.
8. POST /node/metrics remains rejected with HTTP 405.

Test source: deskachain/node/internal/rpc/phase8_integration_test.go

Test: TestPhase8IntegrationNodeChainExplorerMonitoring

## CI strategy

The test is deliberately bounded and reuses existing RPC, chain, mempool, and Explorer fixtures. Multi-node convergence and soak/recovery behavior remain covered by Phases 8.5 and 8.6, while monitoring contracts remain covered by Phase 8.7.

## Acceptance

- [x] Consolidated integration test passes in the existing Go test workflow.
- [x] Node → chain → mempool → Explorer indexer → Explorer API → monitoring → Explorer UI is covered.
- [x] Canonical chain validation is integrated.
- [x] Explorer Monitoring consumes versioned /node/metrics.
- [x] Monitoring remains read-only.
- [x] No consensus state is introduced.
- [x] CI is green on the integration commit.

Phase 8.8 is complete when these criteria remain green on phase-8-testnet. The next gate is final Phase 8 testnet readiness/release review; no main-branch changes are implied.
