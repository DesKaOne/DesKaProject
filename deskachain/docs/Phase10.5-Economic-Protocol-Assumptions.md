# Phase 10.5 — Economic & Protocol Assumptions

## Purpose

Document the economic and protocol assumptions that must be understood before a mainnet release decision. This is an evidence and decision record; it does not change consensus rules.

## 10.5.1 Native unit semantics

The existing project decision is:

- Chain: **IndoChain**
- Native unit: **dIDR**
- Ecosystem accounting relationship: **1 dIDR = 1000 Rupiah**
- dIDR remains the native chain/accounting unit.
- Issued assets remain separate from native dIDR.

This document does not redefine those semantics.

## 10.5.2 Consensus assumptions

Before mainnet approval, explicitly document the assumptions relied upon by the deployed protocol, including:

- block production and validation;
- cumulative chain-work selection;
- fork/reorg handling;
- transaction validation;
- nonce/accounting behavior;
- mempool admission and confirmation;
- difficulty adjustment;
- timestamp handling;
- genesis/network/chain identity.

The assumption record must reference implementation and test evidence where available.

## 10.5.3 Mining and difficulty

Document the approved assumptions for:

- target block time;
- difficulty calculation;
- retarget interval;
- minimum/maximum or bounded difficulty behavior, if implemented;
- chain-work comparison;
- miner/operator responsibilities.

Observability metrics such as current difficulty, recent block intervals, and projected retarget direction are monitoring data and do not themselves alter consensus.

## 10.5.4 Transaction fees

Before mainnet approval, define and approve:

- fee unit;
- fee calculation;
- minimum/required fee behavior;
- fee inclusion in block accounting;
- fee treatment for native transfers;
- fee treatment for issued-asset transactions;
- mempool fee admission policy.

If any item is not yet implemented or approved, record it as **TBD** rather than assuming behavior.

## 10.5.5 Native dIDR accounting

The mainnet decision record must distinguish:

- native dIDR balances;
- transaction fees;
- miner/block rewards, if applicable;
- issued assets;
- application-level Rupiah accounting.

The existing ecosystem relationship of 1 dIDR = 1000 Rupiah must not be silently interpreted as a promise that the blockchain itself is a bank account, deposit, or freely tradable asset.

Any future financial/e-wallet/payment product must define its own operational and regulatory boundaries.

## 10.5.6 Issued assets

Issued assets must remain separate from native dIDR.

Before mainnet approval, document:

- issuance authority;
- asset identity;
- supply rules;
- transfer/burn behavior, where supported;
- fee treatment;
- explorer/indexer representation;
- recovery and administrative controls.

If a rule is not currently implemented, mark it **TBD**.

## 10.5.7 Economic/security assumptions

Document assumptions about:

- miner participation;
- hash-power distribution;
- peer/network availability;
- honest versus adversarial participants;
- chain-work assumptions;
- transaction spam/fee pressure;
- resource exhaustion;
- availability of independent nodes.

These assumptions are not guarantees. They identify conditions under which the deployed protocol is expected to operate as designed.

## 10.5.8 Known limitations

Maintain an explicit list of protocol/economic limitations.

At minimum, review:

| Area | Mainnet decision state |
|---|---|
| Final network/chain identity | Pending 10.1 approval |
| Genesis artifact/hash | Pending approved production genesis |
| Final fee policy | Must be explicitly approved |
| Mining/reward economics | Must be explicitly approved |
| Issued-asset governance | Must be explicitly approved |
| Threat-model assumptions | Covered by 10.4; residual risks pending |
| Regulatory/economic claims | Outside this engineering document |

Do not convert an undocumented behavior into a mainnet guarantee.

## 10.5.9 Evidence requirements

For every mainnet-critical economic/protocol assumption, retain:

1. assumption statement;
2. implementation reference;
3. test evidence;
4. operational evidence where relevant;
5. known limitation;
6. owner/approval status.

If evidence is missing, the item remains unresolved.

## Acceptance criteria

- Native dIDR semantics are explicitly recorded.
- Consensus and chain-selection assumptions are documented.
- Mining/difficulty assumptions are documented.
- Fee behavior is explicitly identified as approved, implemented, or TBD.
- Native dIDR and issued assets remain distinct.
- Economic/security assumptions are documented without presenting them as guarantees.
- Known limitations are explicit.
- Missing evidence remains unresolved.
- No consensus implementation change is required by this documentation phase.
- CI status remains separate from protocol/economic evidence.
