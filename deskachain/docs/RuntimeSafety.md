# Runtime Safety Notes

Phase 2.6.5 adds small localnet safety cleanup: canonical chain stats, atomic mempool writes, duplicate mempool guards, basic per-file mempool locking, request body limits, and initial HTTP server timeouts.

Future public-network hardening still needs:

- stronger RPC rate limiting;
- peer ban thresholds and decay policy;
- per-IP request limits;
- max mempool size;
- transaction fee policy;
- nonce replacement policy;
- config file support for runtime limits;
- graceful shutdown consistency;
- atomic reorg apply transactions;
- full job-based mining RPC instead of long synchronous requests.

These items are intentionally not consensus changes for Phase 2.6.5.
