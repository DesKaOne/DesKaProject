# Phase 10.4 — Security & Threat-Model Gate

## Purpose

Consolidate the existing Phase 9 operational hardening boundaries into a mainnet-oriented threat-model and security evidence gate.

This is an engineering security gate, not a formal security audit, penetration test, or cryptographic certification.

## Assets to protect

The review must explicitly consider:
- canonical chain state;
- node/P2P identity;
- wallet/private-key material where present;
- mining/operator controls;
- RPC/API surfaces;
- Explorer/indexer state;
- backups;
- production configuration;
- release artifacts and signing material if signing is used.

Private keys, credentials, and secret values must never be placed in the threat-model document.

## Trust boundaries

### Consensus boundary

Canonical chain state and protocol validation are consensus-critical.

Explorer, monitoring, and operational tooling must not become alternate consensus authorities.

### P2P boundary

Peers must validate the intended network identity, chain identity, genesis identity, and protocol compatibility.

Seed/bootstrap nodes provide discovery/synchronization assistance and are not consensus authorities.

### RPC boundary

Public-facing RPC must expose only approved surfaces.

The established operational baseline requires monitoring and Explorer surfaces to remain read-only; wallet/admin/mining-control/write surfaces remain restricted unless separately approved.

### Filesystem boundary

Persistent chain state, node identity, keys, configuration, and backups require controlled filesystem access.

A node identity must not be cloned into another independent production node.

### Release boundary

Only traceable release artifacts with recorded source/build provenance, checksums, configuration provenance, and verification evidence may progress through the release states defined by Phase 10.3.

## Threat categories

The following categories must be reviewed with concrete evidence rather than assumptions:

| Category | Review focus |
|---|---|
| Unauthorized RPC access | Exposed methods, bindings, authentication/control boundaries |
| P2P abuse | Peer admission, identity checks, discovery/bootstrap exposure |
| Identity compromise | P2P identity and private-key protection |
| Chain-state corruption | Persistence, backup, restore, validation |
| Malicious/invalid transactions | Transaction validation and mempool boundaries |
| Mining/operator compromise | Mining controls, key custody, operator access |
| Explorer compromise | Read-model isolation and rebuild behavior |
| Monitoring disclosure | Sensitive data exposure through metrics |
| Release compromise | Artifact provenance, checksum/signing controls |
| Backup compromise | Access to persistent state and key material |
| Operational error | Wrong network, wrong datadir, unsafe restore/rollback |

## Threat-model evidence

For each material threat, record:
1. asset affected;
2. trust boundary;
3. threat/event;
4. existing control;
5. evidence supporting the control;
6. residual risk;
7. mitigation or explicit acceptance owner.

Do not claim a threat is eliminated solely because a control is documented; evidence must match the actual implementation or operational procedure.

## Existing evidence carried forward

Phase 9.6 establishes:
- RPC/P2P exposure review;
- network/chain/genesis identity checks;
- secret and filesystem protection boundaries;
- backup/recovery safety;
- read-only Explorer/monitoring boundaries;
- explicit operator controls;
- engineering finding classification.

Phase 10.3 establishes:
- release traceability;
- build provenance;
- checksums;
- signing boundary;
- configuration provenance;
- migration and rollback controls.

These are inputs to the gate, not proof of a formal security audit.

## Required mainnet security decisions

Before the final mainnet gate, explicitly decide:
- key custody model;
- operator access model;
- public RPC exposure policy;
- P2P exposure policy;
- backup retention/protection;
- signing-key custody if release signing is used;
- required independent security review/audit;
- accepted residual risks and owners;
- incident escalation and response ownership.

## Security finding states

Use:
- **Open — Blocker**
- **Open — High**
- **Open — Medium**
- **Open — Low**
- **Mitigated — awaiting verification**
- **Accepted — explicit owner**
- **Closed — evidence verified**

A finding is not closed merely because a code change exists; the relevant verification evidence must be recorded.

## Mainnet security gate

The security gate is ready for final review only when:
- assets and trust boundaries are documented;
- material threat categories have evidence;
- public exposure is reviewed;
- key/backup custody is defined;
- release integrity controls are defined;
- unresolved blockers are explicitly listed;
- any required independent review is identified;
- residual risks have explicit owners.

## Acceptance criteria

- Threat-model scope is documented.
- Assets and trust boundaries are explicit.
- Existing Phase 9 hardening evidence is carried forward.
- Release-integrity controls are included.
- Security findings have explicit lifecycle states.
- Mainnet security decisions requiring approval are identified.
- No claim of formal security audit or certification is made.
- No consensus implementation change is required.
- CI status remains separate from security evidence.
