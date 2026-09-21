# Fork and Reorg Notes

IndoChain Phase 2.4 detects forks and prepares the data flow needed for a future safe reorg, but it does not automatically reorganize the local chain.

## What Is a Fork

A fork happens when two nodes share an ancestor block but disagree on one or more later blocks. For example, two local nodes can both start from the same genesis block and mine a different block at height 1.

Forks can happen because:

- two miners find competing blocks at the same height;
- a node was offline and later sees a different branch;
- test nodes were reset or mined independently;
- a peer sends blocks that do not extend the local tip.

## Phase 2.4 Behavior

Phase 2.4 only detects forks.

- Nodes build a block locator from tip back to genesis.
- Nodes can ask a peer for the first common ancestor.
- Sync refuses a peer branch that does not extend the local tip.
- Automatic reorg is disabled.
- Fork check reports local height, peer height, common ancestor, branch distance, and `reorg_supported: false`.

This means a longer peer chain is not automatically accepted if it forks from the local chain. The node fails safely and leaves local chain state, total supply, and mempool state unchanged.

## Phase 2.5 Direction

Future safe reorg work should include:

- common ancestor search;
- reorg depth limits;
- rollback of ledger state;
- rollback or revalidation of mempool transactions;
- applying the peer branch after validation;
- fork choice by cumulative work, not height alone.

Height alone is not enough for fork choice. IndoChain now has placeholder helpers for block work and cumulative work, but the consensus rule is not changed in Phase 2.4.
