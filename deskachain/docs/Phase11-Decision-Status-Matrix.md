# Phase 11 — Decision & Genesis Status Matrix

## Purpose

Provide one operational view of the production decision gates after completion of the Phase 11 documentation set.

This matrix is a tracking artifact. It does not approve mainnet and does not replace the individual decision records.

## Current status

| Gate | State | Evidence | Next action |
|---|---|---|---|
| 11.1 Production identity | OPEN / TBD | Phase11.1 | Explicitly approve network name, Network ID, Chain ID and protocol version |
| 11.2 Genesis | BLOCKED by 11.1 | Phase11.2 | Generate only after approved identity/config |
| 11.3 Economic/protocol | OPEN / TBD | Phase11.3 | Decide fee, reward, difficulty, issuance and accounting rules |
| 11.4 Security | OPEN | Phase11.4 | Obtain/record required security assessment and close or own findings |
| 11.5 Infrastructure | OPEN | Phase11.5 | Confirm production hosts, roles, ownership, access and recovery |
| 11.6 External dependencies | OPEN | Phase11.6 | Resolve each critical dependency state |
| 11.7 Release candidate | BLOCKED by upstream gates | Phase11.7 + Phase9.8 evidence index | Assemble candidate after sufficient evidence exists |
| 11.8 Final decision | READY FOR REVIEW framework | Phase11.8 | Assemble evidence, blockers, exceptions and release authority |

## Phase 9 handoff

Phase 9.8 is the operational evidence handoff into Phase 11. In particular, the Phase 9.5 upgrade evidence record is an execution template and becomes evidence only after an actual drill is performed and archived.

Phase 9 completion must not be used to mark any Phase 11 production gate approved.

## Explicitly unresolved production values

The following remain intentionally unresolved:

- Mainnet network name
- Mainnet Network ID
- Mainnet Chain ID
- Production protocol version
- Production genesis artifact
- Production genesis hash
- Final fee policy
- Mining/block reward
- Difficulty parameters
- Issuance rules
- Production infrastructure endpoints/owners
- Security assessment outcome
- Critical external dependency states
- Release authority

No placeholder is a production value.

## Evidence rule

A status can only move forward when the corresponding evidence exists and is traceable to an identified source commit, configuration, artifact, test, review, or explicit authority decision.

Documentation completion is not equivalent to approval.

## Mainnet boundary

Until the unresolved production decisions are explicitly approved:

- do not generate production genesis;
- do not publish production node configuration;
- do not reuse testnet genesis or node identities;
- do not label a release candidate as approved;
- do not treat testnet operation as mainnet authorization.

## dIDR continuity

The established ecosystem accounting definition remains:

**1 dIDR = 1000 Rupiah**

This matrix does not establish any separate monetary, reserve, redemption, legal, or financial guarantee.

## CI tracking

The repository currently has no applicable CI workflow evidence for this documentation progression. Project tracking may therefore use **GREEN** as the local documentation/progress convention.

GREEN must not be interpreted as:

- successful GitHub Actions execution;
- independent security approval;
- production approval;
- legal/regulatory approval;
- mainnet release authorization.

## Next execution order

1. Approve production identity.
2. Resolve economic/protocol decisions.
3. Complete required security assessment.
4. Confirm production infrastructure and ownership.
5. Resolve critical external dependencies.
6. Generate and independently verify production genesis.
7. Assemble and verify release candidate.
8. Execute explicit final release decision.

## Acceptance criteria

- Every production gate has one visible state.
- Unresolved values are explicit.
- Evidence dependencies are traceable.
- Genesis remains blocked until identity/configuration approval.
- CI status is not conflated with engineering evidence.
- No mainnet launch is implied.
