# Phase 11.3 — Economic & Protocol Decision Record

## Purpose

Create the explicit production decision record for IndoChain economic and protocol behavior before any production genesis is approved.

This document records decisions and evidence requirements. It does not authorize mainnet launch.

## Decision areas

| Area | Current state | Required evidence |
|---|---|---|
| Transaction fee policy | TBD | Unit, calculation, minimum/required behavior, accounting |
| Mining/block reward | TBD | Reward rule and block accounting |
| Difficulty | TBD | Initial value, target interval, retarget behavior |
| Issuance rules | TBD | Creation, authorization and accounting boundaries |
| Native dIDR accounting | Defined semantically; production rules TBD | State transition and supply/accounting rules |
| Issued-asset governance | TBD | Authority, lifecycle and constraints |
| Supply assumptions | TBD | Explicit model and invariants |

## dIDR semantic boundary

The established ecosystem relationship remains:

**1 dIDR = 1000 Rupiah**

This is an ecosystem accounting definition. It does not by itself establish a bank deposit, reserve, legal tender status, freely tradable crypto asset, or redemption guarantee.

Any production financial or reserve behavior must be separately defined and approved.

## Consensus-critical behavior

The production decision record must explicitly cover:

- block production and validation;
- chain-work and fork/reorganization behavior;
- transaction validation;
- nonce/accounting rules;
- mempool acceptance boundaries;
- difficulty and retargeting;
- timestamps and validity rules;
- genesis/network/chain identity;
- fee and reward accounting.

A decision is incomplete when implementation behavior and documented production intent diverge.

## Issued assets

Issued assets remain distinct from native dIDR. Production governance must define who or what is authorized to issue, modify, freeze, or otherwise govern an issued asset and what validation rules apply.

No undocumented issuance authority should be inferred from testnet behavior.

## Economic/security assumptions

Record assumptions about:

- miner participation and distribution;
- honest versus adversarial participants;
- peer availability;
- chain-work security;
- spam/resource exhaustion;
- transaction and accounting correctness;
- independent node operation;
- operational concentration and failure modes.

Each material assumption should have an owner, evidence source, or explicit residual-risk status.

## Decision states

- **TBD** — unresolved;
- **PROPOSED** — candidate rule documented;
- **VERIFIED** — implementation/evidence checked against the rule;
- **APPROVED** — authorized production decision;
- **BLOCKED** — dependency or evidence prevents approval.

## Acceptance criteria

- Mainnet-critical economic and protocol rules are explicit.
- No TBD behavior is silently treated as production behavior.
- dIDR semantics remain consistent.
- Fee, reward, difficulty, issuance and accounting rules are traceable to evidence.
- Security/economic assumptions have explicit residual-risk handling.
- No mainnet launch authorization is implied.
- CI status remains separate from the decision record.
