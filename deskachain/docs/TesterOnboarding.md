# IndoChain Public Testnet Tester Onboarding

This guide is for non-core testers who want to try the public testnet safely.

## What You Need

- Windows x64, Linux x64, or Linux arm64.
- Basic terminal comfort.
- Network connection.
- Optional LAN, Tailscale, or VPS access.

## What This Is

This is a public testnet release candidate. Testnet dIDR has no monetary value. Mainnet is not available. Testnet may reset.

## Download And Verify

1. Download the artifact for your platform from the official repository release or workflow artifacts.
2. Download `SHA256SUMS.txt`.
3. Verify checksum.
4. Run `indochain version`.

Do not trust random binaries from unknown sources.

## Join As Read-Only Or Explorer Tester

1. Initialize a testnet datadir.
2. Start a node with an operator-published seed peer.
3. Open `http://127.0.0.1:9311/explorer-ui/`.
4. Check dashboard, blocks, search, and address pages.

## Join As Miner Tester

1. Create a separate miner wallet datadir.
2. Run `indominer --once`.
3. Confirm the block appears in Explorer.
4. Check the miner reward address page.

Coinbase maturity applies before mined rewards become spendable.

## Join As Faucet, Staking, Or Service Tester

1. Request faucet funds only if a faucet URL is provided by an operator.
2. Lock stake only if you understand this is testnet collateral.
3. Run `indoservice --once` against a controlled service RPC node.
4. Check service score and eligibility.

Service points are simulation-only and are not spendable dIDR. Staking is collateral-only and has no APY.

## What To Report

- Sync issues.
- Wrong height or tip.
- Explorer UI errors.
- Crash or panic.
- Firewall or connectivity issues.
- Faucet rate-limit issues.
- Service agent errors.

Include version, OS, architecture, commands, and logs.

## What Not To Do

- Do not expose wallet/admin RPC publicly.
- Do not use an important or private machine without understanding testnet risk.
- Do not put real funds or unrelated private keys into IndoChain testnet.
- Do not trust random binaries.
- Do not paste private keys, wallet files, or secrets in reports.
