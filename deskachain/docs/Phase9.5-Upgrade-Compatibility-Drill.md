# Phase 9.5 — Upgrade & Compatibility Drill

## Objective

Exercise a controlled IndoChain node upgrade on testnet without changing consensus rules merely for the drill.

The drill validates:
- binary replacement with persistent state retained;
- configuration compatibility;
- clean restart;
- chain resynchronization/convergence;
- Explorer/indexer recovery;
- public monitoring schema compatibility;
- rollback boundaries.

This is an operational compatibility exercise, not a release certification or security audit.

## Drill boundary

Use one disposable testnet node or a non-critical validator/full-node role.

Never perform the first drill against main.

Preserve the pre-upgrade datadir and configuration as a protected rollback point. Do not copy a node identity to another independent node.

## Pre-upgrade evidence

Record before stopping the old binary:

- binary identifier/version or build commit;
- configuration file checksum;
- network ID;
- chain ID;
- genesis identity;
- node/P2P identity;
- current height and tip hash;
- peer count;
- mempool pending count;
- Explorer indexer height/lag/readiness;
- `/node/metrics` schema version;
- `chain validate` result.

The Phase 9.4 stopped-node backup procedure should be used before the upgrade when rollback requires state restoration.

## Controlled upgrade sequence

1. Confirm the target node is healthy with the Phase 9.3 smoke flow.
2. Record pre-upgrade evidence.
3. Stop the node cleanly.
4. Create/protect the pre-upgrade backup.
5. Replace the binary only; retain the intended persistent datadir.
6. Apply only explicitly reviewed configuration changes.
7. Start the upgraded node.
8. Verify network ID, chain ID, and genesis identity.
9. Verify chain height/tip and run chain validation.
10. Verify peer health and resynchronization.
11. Verify Explorer indexer readiness and lag.
12. Verify `/node/metrics` remains schema `v1`.
13. Run the Phase 9.3 operator smoke flow.
14. Observe the node through at least one normal sync/recovery cycle before declaring the drill complete.

## Compatibility checks

### Chain compatibility

The upgraded node must not silently adopt a different network, chain, or genesis identity.

A changed chain identity is a failed compatibility gate unless the change was explicitly intended and separately approved.

### Persistent-state compatibility

The upgrade must preserve the ability to open and validate the existing chain state and node identity.

If a state migration is introduced later, it must have an explicit migration and rollback procedure rather than being assumed compatible.

### Explorer compatibility

Explorer is a rebuildable read model. An Explorer index incompatibility may be recovered by rebuilding from canonical chain state.

Explorer schema/read-model changes must not be used to mutate consensus state.

### Monitoring compatibility

The public `/node/metrics` contract remains `schema_version: v1` during this phase unless a separately scoped API-version change is approved.

The smoke gate must continue to reject write attempts to the monitoring endpoint.

## Rollback boundary

Rollback means restoring the previous known-good binary/configuration and, only when necessary, restoring the protected pre-upgrade datadir backup.

Do not blindly downgrade a newer persistent state into an older binary if the newer release introduced an incompatible on-disk format.

If persistent-format compatibility is uncertain:
1. stop the node;
2. preserve the current datadir as an evidence copy;
3. restore the protected pre-upgrade backup into a new datadir;
4. start the previous binary against that restored state;
5. validate chain/network identity;
6. run health and smoke checks;
7. record the rollback result.

## Failure classification

| Observation | Classification |
|---|---|
| Node starts, validates chain, and converges | Pass |
| Explorer index needs rebuild but canonical chain is valid | Operational recovery |
| Monitoring response changes only within an approved contract update | Compatibility review |
| Network/chain/genesis identity changes unexpectedly | Upgrade failure |
| Existing chain state cannot be opened/validated | Upgrade failure |
| Previous binary cannot safely open upgraded state | Rollback blocked; restore protected pre-upgrade state |

## Evidence record

For each drill record:

- drill ID and timestamp;
- node role;
- pre/post binary identifiers;
- config checksums;
- pre/post network/chain/genesis identity;
- pre/post height/tip;
- peer and mempool observations;
- Explorer indexer status;
- monitoring schema;
- validation/smoke results;
- restart duration if measured;
- incidents and recovery actions;
- rollback result if exercised.

## Acceptance criteria

- A controlled upgrade sequence is documented.
- Pre-upgrade backup/evidence is explicitly required.
- Chain/network/genesis compatibility checks are explicit.
- Persistent-state and Explorer recovery boundaries are explicit.
- Monitoring API compatibility remains observable and read-only.
- Rollback avoids unsafe in-place downgrade of unknown persistent formats.
- The drill produces reproducible evidence.
- No consensus implementation change is required.
- CI status is recorded separately from operational drill evidence.
