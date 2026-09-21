# Phase 9.6 — Security & Operational Hardening

## Objective

Review the testnet operational surface and separate development conveniences from safer testnet defaults without changing consensus behavior.

This phase is an engineering hardening review, not a formal security audit, penetration test, or cryptographic review.

## 9.6.1 RPC exposure

Review every RPC surface before exposing a node beyond localhost.

Minimum checks:
- public/read-only endpoints are intentionally exposed;
- wallet, admin, mining-control, or transaction-submission RPCs are not exposed on public-safe nodes unless explicitly required;
- monitoring remains read-only;
- `POST /node/metrics` remains rejected;
- health and Explorer endpoints do not become implicit write paths.

The existing mainnet/public-safe launch gate remains the baseline boundary.

## 9.6.2 P2P exposure

Review:
- bind addresses and advertised endpoints;
- seed/bootstrap configuration;
- peer discovery assumptions;
- network ID and chain ID validation;
- genesis identity validation;
- protocol compatibility checks;
- peer failure/cooldown handling.

A seed is a discovery/synchronization aid, not a consensus authority.

## 9.6.3 Configuration and defaults

Separate development conveniences from testnet-safe operation.

Review:
- localhost versus public bindings;
- testnet-specific ports;
- persistent datadir paths;
- node identity paths;
- bootstrap/seed endpoints;
- Explorer enablement;
- mining enablement;
- debug/test flags;
- operator-only configuration.

Any default that broadens exposure must be explicit in operator documentation.

## 9.6.4 Secrets and filesystem permissions

Operator controls must protect:
- node/P2P identity material;
- wallet/private-key material where present;
- configuration containing credentials or sensitive endpoints;
- backups containing identity/private-key material.

Backups must stay outside the live datadir and must not be committed to source control or emitted into CI logs.

Recommended operational boundary:
- least-privilege filesystem ownership;
- private identity/key files not world-readable;
- backup storage restricted to operators;
- no secrets in command output, documentation examples, or test fixtures.

## 9.6.5 Backup and recovery safety

Use the Phase 9.4 backup/restore boundary.

Before recovery:
1. stop the target node;
2. preserve the current state as evidence if failure investigation is needed;
3. restore into a new empty datadir;
4. validate chain/network/genesis identity;
5. run health and smoke checks.

Never copy one node's P2P identity into another independent node.

## 9.6.6 Monitoring and Explorer safety

Monitoring and Explorer remain read-only observability layers.

Review that:
- Explorer index failure cannot invalidate canonical chain state;
- index rebuild does not mutate consensus state;
- monitoring only reads node state;
- public metrics do not expose private keys, wallet secrets, or unnecessary credential material;
- API schema/version changes are explicit.

## 9.6.7 Operator controls

Operational commands should make risky actions explicit.

For any command that can:
- start mining;
- replace persistent state;
- restore a backup;
- change network/chain identity;
- alter public bindings;

the operator should know the affected node, datadir, and network before execution.

Development-only shortcuts should not be presented as production/mainnet procedures.

## Hardening checklist

| Area | Gate |
|---|---|
| RPC | Public-safe surface is read-only where intended |
| P2P | Bindings/bootstrap and identity invariants reviewed |
| Config | Testnet-safe defaults documented |
| Secrets | Identity/key material protected |
| Backups | Backup and restore boundaries preserved |
| Explorer | Read-model failure remains non-consensus |
| Monitoring | Metrics remain read-only |
| Operator controls | Risky state changes are explicit |
| Main isolation | Main branch remains untouched |

## Findings classification

Record findings as:
- **Blocker** — unsafe exposure or state handling that must be fixed before the node is used for the intended testnet role.
- **High** — significant operational risk requiring remediation before the next readiness gate.
- **Medium** — meaningful hardening gap with a documented mitigation.
- **Low** — hygiene or usability improvement.
- **Informational** — observation with no immediate remediation requirement.

This classification is for engineering tracking only and does not constitute a security-audit severity rating.

## Evidence

For each review, record:
- node role;
- endpoint/bindings reviewed;
- relevant config paths;
- identity/key permission checks;
- backup location policy;
- monitoring/Explorer checks;
- finding classification;
- remediation commit or documented mitigation;
- reviewer/operator timestamp.

Do not record private key contents or secret values.

## Acceptance criteria

- RPC and P2P exposure boundaries are documented.
- Development conveniences are separated from testnet-safe defaults.
- Identity, key, and backup handling boundaries are explicit.
- Explorer and monitoring remain read-only.
- Operator controls identify risky persistent-state changes.
- Findings have an explicit classification and remediation/mitigation path.
- No consensus implementation change is required.
- This phase is not represented as a formal security audit.
