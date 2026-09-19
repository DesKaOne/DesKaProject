# DesKaChain RC1 Issue Triage

Testnet DKC has no monetary value. Mainnet is not available. Never ask testers to paste private keys, wallet files, or secrets.

## Severity

- S0 security/safety: private key leak risk, public wallet/admin RPC exposure, unsafe release artifact, secret disclosure.
- S1 consensus/data loss: chain validation failure, consensus mismatch, corrupted datadir after normal operation, reproducible block/tx validation panic.
- S2 networking/sync: seed unreachable, peers cannot sync, wrong network/genesis rejection confusion, restart catch-up failure.
- S3 explorer/UI: explorer status, block list, search, address view, or static UI failure.
- S4 docs/usability: unclear instructions, typo, missing command, onboarding confusion.

## Labels

- `testnet-rc1`
- `bug`
- `docs`
- `explorer`
- `p2p`
- `mining`
- `faucet`
- `staking`
- `service-node`
- `safety`
- `needs-repro`
- `rc2-candidate`

## Triage Flow

1. Confirm version with `deskachain version`.
2. Confirm network/genesis with `/health`, `/explorer/status`, or `chain info`.
3. Ask for the exact command, OS, artifact name, and redacted logs.
4. Reproduce locally when possible.
5. Classify severity.
6. Add a workaround if available.
7. Mark `rc2-candidate` if the issue blocks wider public testnet.

## Safety Rules

- Never ask testers to paste private keys.
- Never ask testers to upload `wallets.json`.
- Never ask testers to upload full datadirs publicly.
- Do not request secrets, tokens, systemd private env files, or SSH keys.
- Ask testers to redact IPs if needed.
- Keep public RPC advice read-only by default.
- Do not imply testnet DKC has price, profit, APY, or guaranteed future value.

## Close Rules

Close or mark non-actionable when:

- The report is a duplicate and has no new evidence.
- The tester uses an unsupported old build.
- The issue is caused by wrong network/datadir.
- The issue is fixed in a newer RC.
- The behavior is documented as expected RC1 behavior.
- The report requires secrets/private files that the tester should not share publicly.

