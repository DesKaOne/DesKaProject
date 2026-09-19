# DesKaChain RC2 Planning

Testnet IDR has no monetary value. Mainnet is not available.

## RC2 Goal

RC2 should fix issues found during RC1 monitoring and external tester onboarding without changing consensus or testnet genesis unless a critical safety issue explicitly requires a new testnet candidate.

## RC2 Triggers

- Consensus bug.
- Seed sync bug.
- Public RPC safety issue.
- Explorer usability blocker.
- Packaging/checksum issue.
- Serious docs/onboarding confusion.

## RC2 Non-Triggers

- Cosmetic UI issue.
- Minor docs typo.
- Expected faucet rate limit.
- Expected coinbase maturity confusion when docs can clarify it.
- Local firewall issue that is already covered by docs.

## Candidate Fixes

| Priority | Area | Issue | Fix | Test needed |
| -------- | ---- | ----- | --- | ----------- |
| | | | | |

## RC2 Release Process

1. Patch the minimal fix.
2. Run full Go tests.
3. Run targeted RPC/CLI/config/P2P tests.
4. Run smoke tests.
5. Build/package release artifacts.
6. Generate checksums.
7. Update release notes.
8. Publish as a GitHub pre-release.
9. Update `KnownIssues.md` and tester onboarding links.

## Version Suggestion

- Use `v0.4.6-testnet-rc2` if only RC fixes are included.
- Use `v0.4.7-testnet-rc1` if new features are added.

