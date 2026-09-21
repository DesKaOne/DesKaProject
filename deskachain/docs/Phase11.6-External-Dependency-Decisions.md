# Phase 11.6 — External Dependency Decisions

## Purpose

Resolve or explicitly track external dependencies that could block mainnet operation or the wider DesKa Ecosystem.

Engineering readiness does not equal legal, regulatory, banking, payment, or partner approval.

## Dependency categories

- infrastructure/hosting;
- DNS and public endpoints;
- RPC/public access;
- monitoring and alerting;
- backup storage;
- build/release infrastructure;
- wallet/payment/e-wallet integrations;
- banking/card/payment partners;
- KYC/compliance services;
- legal/regulatory requirements;
- operational vendors.

## Required state

Use one of:

`Identified` · `Evaluating` · `Sandbox/Tested` · `Production-ready` · `Approved` · `Blocked` · `Not required`

Sandbox/API access must never be recorded as production partnership or approval.

## Decision record

For each dependency record:

| Field | Required |
|---|---|
| Dependency | Yes |
| Purpose | Yes |
| Owner | Yes |
| Current state | Yes |
| Environment | Yes |
| Contract/approval evidence | Where applicable |
| Failure/fallback | Yes |
| Production prerequisite | Yes |
| Blocker/risk | If applicable |

## DesKa ecosystem boundary

Dependencies for DesKaCash, DesKaBank, DesKaPay, DesKaCard, payment rails, banking partners, card networks, and compliance workflows must remain separate from the technical readiness of IndoChain itself.

A sandbox integration can validate engineering interfaces while production access still requires the relevant provider, operational, contractual, and regulatory approvals.

## Acceptance criteria

- Critical dependencies are identified.
- Each has an explicit state and owner.
- Sandbox and production status are separated.
- Failure/fallback paths are recorded.
- Legal/regulatory requirements are not inferred from engineering tests.
- Blocked dependencies remain visible to the release gate.
