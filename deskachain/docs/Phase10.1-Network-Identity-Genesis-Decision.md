# Phase 10.1 — Network Identity & Genesis Decision Record

## Status

**Decision state: NOT YET APPROVED**

This record separates the existing testnet identity from the future mainnet identity. It intentionally does not invent or silently promote production identifiers.

## Existing testnet identity

The Phase 8.5 operator baseline documents the current testnet values:

| Parameter | Testnet value |
|---|---|
| Network | `testnet` |
| Network ID | `ind-testnet-1` |
| Chain ID | `777101` |
| Genesis | Testnet canonical genesis |

These values are testnet-only and must not be reused for mainnet.

## Mainnet identity

The following values remain explicit placeholders until a separate approval decision is recorded:

| Parameter | Mainnet decision |
|---|---|
| Network name | `mainnet` / final name pending approval |
| Network ID | **TBD — approval required** |
| Chain ID | **TBD — approval required** |
| Genesis artifact | **TBD — production artifact pending** |
| Genesis hash | **TBD — generated only from approved genesis artifact** |
| Protocol version | **TBD — final release decision** |

No production node configuration should be generated from these placeholders.

## Native unit decision

The existing ecosystem decision is retained:

- IndoChain native unit: **dIDR**
- Ecosystem accounting relationship: **1 dIDR = 1000 Rupiah**
- dIDR is the native chain/accounting unit and is not being defined here as a freely tradable issued crypto asset.
- Issued assets remain separate from native dIDR.

This record does not change those existing semantics.

## Genesis requirements

Before mainnet approval, the production genesis artifact must be:

1. deterministically generated from an approved configuration;
2. independently checked;
3. hashed and recorded;
4. distributed through the release evidence package;
5. treated as immutable after release approval.

The recorded genesis hash must be derived from the final approved artifact, not manually entered.

## Identity invariants

Every production node must validate:

- expected network ID;
- expected chain ID;
- expected genesis identity;
- compatible protocol version;
- intended peer/network boundaries.

A node configured for testnet must fail to silently join mainnet, and a mainnet node must not silently accept testnet genesis/state.

## Testnet-to-mainnet separation

The following must remain separate:

- network ID;
- chain ID;
- genesis artifact/hash;
- persistent chain datadir;
- node/P2P identity;
- bootstrap/seed configuration;
- production operator configuration.

A testnet datadir must never be promoted in place to mainnet.

## Approval checklist

- [ ] Mainnet network ID approved.
- [ ] Mainnet chain ID approved.
- [ ] Production genesis configuration approved.
- [ ] Production genesis artifact generated.
- [ ] Genesis hash independently verified.
- [ ] Protocol/release version approved.
- [ ] Testnet/mainnet configuration separation verified.
- [ ] Fresh mainnet datadir provisioning procedure verified.
- [ ] Mainnet bootstrap/seed endpoints approved.
- [ ] Release evidence archive contains the exact genesis artifact and hash.
- [ ] Explicit mainnet release authority recorded.

## Decision boundary

Until all required identity/genesis approvals are complete:

- testnet remains the active engineering network;
- no mainnet genesis is considered canonical;
- no testnet identifier is promoted to production;
- no mainnet launch is implied.

## Evidence

Source baseline:
- `deskachain/docs/Phase8.5-MultiNodeTestnet.md`
- `deskachain/docs/Phase10-PreMainnet-Mainnet-Readiness.md`

The Phase 8.5 source establishes the current testnet network/chain identity and the requirement that peers validate network ID, chain ID, protocol compatibility, and genesis identity. Mainnet values are intentionally not supplied by that source and therefore remain TBD here.
