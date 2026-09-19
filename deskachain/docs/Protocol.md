# DesKaChain Protocol Notes

Status: Phase 2.7 protocol foundation.

## Address Format

New wallet addresses use:

```text
IDR + Base58Check(version_byte + pubkey_hash)
```

The checksum is the first 4 bytes of `SHA256(SHA256(payload))`.

`pubkey_hash` is:

```text
RIPEMD160(SHA256(compressed secp256k1 public key))
```

Rules:

* `IDR` prefix is uppercase and case-sensitive.
* Address payload is 21 bytes: 1 version byte + 20-byte public key hash.
* Base58 alphabet is Bitcoin-style: `123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz`.
* Wrong checksum, wrong version, wrong prefix, and invalid Base58 characters are rejected.
* The old dev address format is legacy localnet/dev only.

## Keys And Signatures

* Private key export/import is raw 32-byte scalar hex.
* Exported private keys are 64 lowercase hex characters.
* Public keys are compressed secp256k1 public keys, 33 bytes.
* Transactions store compressed public key hex.
* Transaction signatures use secp256k1 ECDSA DER encoding.
* Transaction verification derives the sender address from the stored public key and rejects mismatches.

## Network Profiles

| Network | Chain ID | Network ID | Address Version | Default RPC | Default P2P | Legacy dev address |
| --- | ---: | --- | --- | ---: | ---: | --- |
| localnet | 777001 | `idr-local-1` | `0x1E` | 8331 | 9331 | allowed |
| testnet | 777101 | `idr-testnet-1` | `0x1F` | 18331 | 19331 | rejected |
| mainnet | 777000 | `idr-main-1` | `0x20` | 8333 | 9333 | rejected |

All profiles currently use the visual prefix `IDR`. Network separation is enforced through the address version byte and network id.

## Protocol Versions

Initial versions:

* protocol version: `1`
* block version: `1`
* tx version: `1`
* P2P protocol version: `idr-p2p/1`
* RPC API version: `v1`

`/p2p/handshake` exposes `network_id`, `chain_id`, `protocol_version`, and `p2p_protocol_version`.

`/node/status` and `/chain/info` expose `network`, `chain_id`, `protocol_version`, and `rpc_api_version`.

## Difficulty And Work

Localnet starts at difficulty `4`, targets `10s` block time, and retargets every `10` blocks. Difficulty changes conservatively by at most 1 per retarget window.

Cumulative work uses:

```text
work = 16 ^ difficulty
```

Genesis or difficulty `0` contributes work `1`. Reorg decisions compare cumulative work, not height alone.

`/chain/difficulty` exposes current tip difficulty, next difficulty, retarget params, blocks until retarget, and cumulative work.

## Public Network Notice

Testnet IDR has no monetary value. DesKaChain makes no price promise, APY promise, or profit promise.

Any future mainnet claim process, if implemented, must be capped and time-limited.
