Phase 4.12 — Peer Discovery & Auto Bootstrap

Implement automatic peer discovery and bootstrap improvements.

Goals:

* Add peer gossip so connected peers exchange known peer lists.
* Automatically discover and connect to new peers.
* Maintain peer metadata:

  * first_seen
  * last_seen
  * last_success
  * version
  * protocol
  * advertised services
* Remove expired peers using configurable TTL.
* Retry failed peers with exponential backoff.
* Prevent duplicate peers.
* Limit peers from the same IP/subnet.
* Keep peers.json backward compatible.
* Add background peer maintenance worker.
* Add read-only RPC endpoints:

  * GET /p2p/discovery
  * GET /p2p/bootstrap
  * GET /p2p/known-peers

Requirements:

* No consensus changes.
* No mining changes.
* No block validation changes.
* No wallet changes.
* No transaction format changes.
* Add unit tests.
* Update README and README-ID for the new endpoints.
