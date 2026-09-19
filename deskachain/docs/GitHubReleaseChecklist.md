# GitHub Release Checklist - v0.4.6-testnet-rc1

Use this checklist before publishing DesKaChain Public Testnet RC1.

## Before Release

- Confirm working branch is clean or contains only intentional release changes.
- Confirm tag exists: `v0.4.6-testnet-rc1`.
- Confirm CI is green.
- Confirm release artifact workflow is green.
- Confirm smoke test passed.
- Confirm `SHA256SUMS.txt` exists.
- Confirm release archives contain expected binaries, docs, examples, and config.
- Confirm archives contain no datadir, wallet files, chain DB, mempool, peer store, faucet state, service state, private keys, or `.git`.
- Confirm public docs state that testnet IDR has no monetary value and mainnet is not available.
- Confirm wallet/admin RPC examples remain private or disabled.

## Artifacts To Attach

- `deskachain-v0.4.6-testnet-rc1-windows-amd64.zip`
- `deskachain-v0.4.6-testnet-rc1-linux-amd64.tar.gz`
- `deskachain-v0.4.6-testnet-rc1-linux-arm64.tar.gz`
- `SHA256SUMS.txt`

## Release Metadata

- Title: `DesKaChain Public Testnet RC1 - v0.4.6-testnet-rc1`
- Pre-release: `true`
- Latest release: `false`, if GitHub allows.
- Clearly mark as testnet RC.

## Release Body

- Use `docs/GitHubRelease-v0.4.6-testnet-rc1.md`.
- Keep warnings near the top.
- Link `docs/PublicTestnetQuickstart.md`, `docs/OperatorChecklist.md`, and `docs/Explorer.md`.

## Post Publish

- Download artifacts from the GitHub Release.
- Verify SHA256 from downloaded artifacts.
- Extract to a clean folder.
- Run `deskachain version`.
- Start a local testnet node.
- Open `/explorer-ui/`.
- Run the smoke script from repository checkout if practical.
- Confirm quickstart docs render correctly.
- Confirm warnings are visible.
