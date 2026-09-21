# Phase 11.2 — Genesis Finalization

## Purpose

Define the controlled procedure for producing and verifying the IndoChain production genesis artifact after the production network identity has been explicitly approved.

This phase does not authorize mainnet launch and must not generate a production genesis from unresolved identity placeholders.

## Preconditions

Genesis generation may proceed only after Phase 11.1 has approved:

- final network name;
- final network ID;
- final chain ID;
- production protocol version;
- genesis-critical protocol/configuration inputs.

Until those values are approved, production genesis remains **TBD**.

## 11.2.1 Deterministic inputs

Record every genesis-critical input used by the generator, including as applicable:

- network identity;
- chain identity;
- protocol version;
- initial chain parameters;
- initial accounts/state;
- initial asset/state definitions;
- timestamp or explicitly controlled genesis time;
- difficulty/mining parameters;
- native dIDR initial-state assumptions;
- any genesis metadata that participates in the resulting hash.

No undocumented input may silently affect the production genesis.

## 11.2.2 Generation procedure

1. Start from the approved production configuration.
2. Verify that no testnet identity or persistent state is referenced.
3. Generate the genesis artifact deterministically.
4. Preserve the exact generation inputs.
5. Compute the canonical genesis hash.
6. Independently regenerate or validate the artifact.
7. Compare the resulting hash and identity fields.
8. Record the verified artifact and hash in release evidence.

If independent verification differs, the candidate remains **BLOCKED**.

## 11.2.3 Independence requirement

The verification step should not merely trust the same generated output.

Where practical, use an independent invocation, clean environment, separate operator, or independently reviewed generation procedure to establish that the artifact is reproducible from the approved inputs.

The verification record must identify what was independently checked.

## 11.2.4 Genesis immutability

Once approved, the production genesis artifact and hash become immutable release inputs.

Any change to a genesis-critical input requires:

- regeneration;
- new hash;
- independent verification;
- updated evidence;
- renewed approval.

A modified genesis must never be silently substituted after approval.

## 11.2.5 Mainnet/testnet separation

Production genesis must not reuse the testnet genesis artifact.

The production evidence package must explicitly record:

- production network ID;
- production chain ID;
- production genesis hash;
- production protocol version;
- artifact reference.

Testnet genesis remains a separate testnet artifact.

## 11.2.6 dIDR and initial state

If production genesis contains native dIDR initial state, the exact initialization rule must be approved and recorded before generation.

The existing ecosystem accounting relationship remains:

**1 dIDR = 1000 Rupiah**

This statement alone does not define a genesis allocation, monetary supply, reserve, or financial liability. Any such production rule requires a separate explicit decision.

## 11.2.7 Evidence package

Retain:

- approved identity decision;
- genesis-generation configuration/input reference;
- generated genesis artifact;
- canonical genesis hash;
- independent verification result;
- generator/build version;
- source commit;
- generation timestamp;
- operator/reviewer record;
- known limitations or exceptions.

Never place private keys or credentials in the public evidence package.

## 11.2.8 Gate states

Use:

- **TBD** — identity or inputs not approved;
- **CANDIDATE** — genesis generated but verification/approval incomplete;
- **VERIFIED** — independent verification succeeded;
- **APPROVED** — authorized production genesis decision recorded;
- **BLOCKED** — mismatch, missing evidence, or unresolved critical input.

## Current status

Because Phase 11.1 currently leaves production network identity and genesis-critical inputs unresolved, Phase 11.2 starts in **TBD** state.

No production genesis is generated or approved by this documentation-only step.

## Acceptance criteria

- Genesis-critical inputs are explicitly identified.
- Production genesis is downstream of approved identity.
- Generation is deterministic and reproducible.
- Genesis hash is recorded.
- Independent verification is required.
- Testnet and mainnet genesis artifacts remain isolated.
- dIDR initial-state rules are not invented when unresolved.
- Genesis changes require renewed verification and approval.
- No mainnet launch authorization is implied.
- CI status remains separate from genesis evidence.
