# IndoChain RC1 Known Issues

Testnet dIDR has no monetary value. Mainnet is not available.

## RC1 Known Limitations

- Mainnet is not available.
- Testnet dIDR has no monetary value.
- Explorer uses simple scan mode.
- There is no persistent explorer database yet.
- There is no wallet web UI.
- There is no mobile app.
- There is no Stratum or pool mining support.
- Service points are simulation-only.
- Staking is collateral-only.
- Faucet must be deliberately operated and funded.
- Public seed availability depends on operators.

## Open Issues

No confirmed RC1 blockers are recorded yet.

Use the template below for confirmed reports.

## Workarounds

- Wrong datadir: restart with a dedicated testnet datadir and run `init`.
- Localnet/testnet mismatch: confirm `network_id` is `ind-testnet-1` and chain ID is `777101`.
- RPC bind/firewall issue: test from localhost first, then from another host.
- Explorer unavailable: verify `/health`, `/explorer/status`, and `/explorer-ui/`.
- Coinbase maturity confusion: mining rewards are confirmed immediately but not mature until 100 blocks on testnet.

## Fixed In Next RC

No fixes have been assigned to RC2 yet.

## Not A Bug / Expected Behavior

- Mainnet commands or mainnet availability are not expected in RC1.
- Testnet dIDR has no monetary value and should not be priced.
- Public RPC mode disables wallet/admin/miner/faucet/service write endpoints unless explicitly re-enabled.
- Service points are not dIDR and are not spendable.
- Staking does not create PoS validators, APY, or staking rewards.
- Faucet requests require an operator-run faucet and later mining for confirmation.

## ISSUE-YYYYMMDD-001 - Title

Status:

Severity:

Affected version:

Affected OS:

Summary:

Steps to reproduce:

Workaround:

Fix plan:

Linked GitHub issue:

