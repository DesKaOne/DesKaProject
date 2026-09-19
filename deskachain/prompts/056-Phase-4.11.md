Phase 4.11 — Peer Reputation & Auto Peer Selection

Implement a lightweight peer reputation system.

Goals:

* Add peer score based on successful syncs, latency, uptime, and failures.
* Prefer higher-score peers when selecting outbound connections.
* Temporarily penalize peers after timeout, invalid responses, or failed sync.
* Automatically recover penalized peers after a cooldown period.
* Persist reputation data in peers.json.
* Expose read-only RPC endpoints:

  * GET /p2p/peers
  * GET /p2p/reputation

Requirements:

* No consensus changes.
* No mining or block validation changes.
* Backward compatible with existing peers.json.
* Keep implementation lightweight.
* Add unit tests.
* Update README and README-ID if new RPC endpoints are added.
