# Phase 10.7 — External Dependencies

## Purpose

Document external dependencies that can affect IndoChain mainnet readiness without treating third-party availability as an internal protocol guarantee.

This phase identifies dependency ownership, readiness evidence, failure boundaries, and unresolved decisions. It does not authorize a mainnet launch.

## 10.7.1 Dependency categories

Review external dependencies across:

- infrastructure and hosting;
- DNS/domain services where applicable;
- RPC/public access infrastructure;
- monitoring and alerting;
- backup storage;
- release/build infrastructure;
- source-code and artifact distribution;
- external wallet/payment/e-wallet integrations;
- banking/payment partners;
- card/payment-network partners;
- identity/KYC or compliance providers where applicable;
- legal/regulatory dependencies;
- operational vendors and support services.

A dependency must be explicitly identified rather than assumed available.

## 10.7.2 Dependency record

For each mainnet-critical dependency, retain:

1. dependency name and purpose;
2. owner/provider;
3. interface or operational boundary;
4. environment affected;
5. required availability or service expectation;
6. credentials/secret ownership boundary;
7. fallback or failure procedure;
8. current readiness status;
9. evidence/reference;
10. responsible owner.

Secrets and credentials must never be stored in this document.

## 10.7.3 Internal versus external responsibility

The following remain IndoChain engineering responsibilities:

- consensus implementation;
- chain/network/genesis identity;
- node persistence;
- P2P protocol behavior;
- transaction validation;
- Explorer/indexer read-model behavior;
- release artifact integrity;
- documented operational recovery.

External providers do not become consensus authorities merely because IndoChain depends on their infrastructure.

## 10.7.4 Infrastructure dependencies

Before mainnet approval, confirm:

- production hosting and node placement;
- network connectivity and required ports;
- persistent storage capacity and protection;
- backup storage and retention;
- monitoring/alerting path;
- DNS or endpoint ownership where required;
- time synchronization expectations;
- recovery access.

Failure of an infrastructure provider must have a documented operational response where the dependency is material.

## 10.7.5 External application integrations

Future DesKa ecosystem services may depend on external partners for wallets, payments, banking, cards, QR/payment rails, or other regulated services.

Those integrations must be treated as separate dependency gates. Their existence must not be represented as completed merely because an API or sandbox integration exists.

For each integration, record whether it is:

- planned;
- sandbox-tested;
- contract/partnership pending;
- production-approved;
- operationally live.

## 10.7.6 Legal and regulatory dependencies

Where a mainnet deployment or an ecosystem service depends on legal, licensing, registration, financial-service, payment, or other regulatory requirements, those requirements must be identified and assigned to the appropriate owner.

Engineering readiness must not be represented as legal or regulatory approval.

Unverified legal/regulatory assumptions remain open dependencies.

## 10.7.7 Dependency failure handling

For each critical external dependency, define:

- detection signal;
- affected service or node role;
- immediate containment;
- fallback/manual procedure;
- recovery procedure;
- evidence to retain;
- escalation owner.

An external service outage must not trigger unsafe consensus or persistent-state changes.

## 10.7.8 Readiness states

Use explicit dependency states:

- **Identified** — dependency and owner documented.
- **Evaluating** — requirements or provider capability under review.
- **Sandbox/Tested** — integration or operational path tested in a non-production environment.
- **Production-ready** — required technical and operational evidence complete.
- **Approved** — responsible authority has explicitly approved production use.
- **Blocked** — dependency prevents the intended release path.
- **Not required** — dependency explicitly excluded from the release scope.

Do not treat an API key, account creation, or sandbox success as production approval.

## 10.7.9 Evidence package

Before the final release gate, retain:

- dependency inventory;
- provider/owner references;
- interface and configuration references;
- sandbox or integration evidence where applicable;
- production-readiness evidence;
- failure/fallback procedures;
- contractual or approval references where applicable;
- unresolved dependency list.

Sensitive credentials must remain outside the evidence package unless stored through an approved secret-management process.

## Acceptance criteria

- Mainnet-critical external dependencies are inventoried.
- Ownership and responsibility boundaries are explicit.
- Production versus sandbox status is distinguishable.
- Failure and fallback procedures are defined for material dependencies.
- Legal/regulatory dependencies are explicitly separated from engineering readiness.
- No external provider is treated as a consensus authority.
- Unresolved dependencies remain visible.
- No consensus implementation change is required.
- No mainnet launch authorization is implied.
- CI status remains separate from external-dependency evidence.
