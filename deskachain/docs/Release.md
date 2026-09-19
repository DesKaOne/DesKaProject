# DesKaChain Release Build And Quickstart

Phase 4.10 prepares the `v0.4.10-testnet-rc1` public-testnet mining-stability and difficulty-observation build for external testers and operators. Testnet DKC has no monetary value. Mainnet is not available.

## Requirements

- Go `1.22` or newer.
- PowerShell on Windows for `scripts/build.ps1` and `scripts/package.ps1`.
- POSIX shell, `tar`, and `zip` on Linux/macOS for `scripts/build.sh` and `scripts/package.sh`.

## Build Binaries

Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.10-testnet-rc1
```

Skip tests when doing a local smoke build:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1 -Version v0.4.10-testnet-rc1 -SkipTests
```

Linux/macOS shell:

```sh
sh ./scripts/build.sh v0.4.10-testnet-rc1
```

Outputs:

- `dist/windows-amd64/deskachain.exe`
- `dist/windows-amd64/dkcminer.exe`
- `dist/windows-amd64/dkcservice.exe`
- `dist/linux-amd64/deskachain`
- `dist/linux-amd64/dkcminer`
- `dist/linux-amd64/dkcservice`
- `dist/linux-arm64/...`

## Package Archives

Windows PowerShell:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\package.ps1 -Version v0.4.10-testnet-rc1 -SkipTests
```

Linux/macOS shell:

```sh
SKIP_TESTS=1 sh ./scripts/package.sh v0.4.10-testnet-rc1
```

Archives and checksums are written under `dist/releases/`.

Verify checksums on Windows:

```powershell
Get-FileHash .\dist\releases\deskachain-v0.4.10-testnet-rc1-windows-amd64.zip -Algorithm SHA256
Get-Content .\dist\releases\SHA256SUMS.txt
```

Verify checksums on Linux:

```sh
cd dist/releases
sha256sum -c SHA256SUMS.txt
```

Release archives include the three binaries, health-check scripts, this quickstart, README files, release notes, operator checklist, public quickstart, tester onboarding, GitHub release checklist/body, announcement draft, seed publishing guide, post-release monitoring docs, topology docs, mining docs, deployment/preflight/genesis/multi-host/systemd/long-run/explorer/faucet/staking/service docs, the seed registry example, and example testnet/systemd configs. They must not include runtime datadirs, wallets, chain DBs, mempool, peers, faucet state, service state, private keys, or `.git`. Deployment preparation lives in `docs/DeployTestnet.md`; topology lives in `docs/TestnetTopology.md`; mining lives in `docs/Mining.md`, `docs/MiningStability.md`, and `docs/DifficultyObservation.md`; multi-host setup lives in `docs/MultiHostTestnet.md`; systemd recovery lives in `docs/Systemd.md`; long-run validation lives in `docs/LongRunTestnet.md`; explorer API docs live in `docs/Explorer.md`; faucet operation lives in `docs/Faucet.md`; staking collateral lives in `docs/Staking.md`; service-node simulation lives in `docs/ServiceNode.md`; preflight checks live in `docs/Preflight.md`; the testnet genesis candidate lives in `docs/TestnetGenesis.md`; RC notes live in `docs/ReleaseNotes-v0.4.6-testnet-rc1.md`; external quickstart lives in `docs/PublicTestnetQuickstart.md`; operator checklist lives in `docs/OperatorChecklist.md`; tester onboarding lives in `docs/TesterOnboarding.md`; feedback checklist lives in `docs/TestnetFeedbackChecklist.md`; seed publishing guidance lives in `docs/SeedOperatorPublish.md`; monitoring lives in `docs/PostReleaseMonitoring.md`; triage lives in `docs/IssueTriage.md`; RC2 planning lives in `docs/RC2Planning.md`.

Checksum verification confirms download integrity. It does not replace source review. Download artifacts only from the official repository workflow or release process, and do not run random binaries from unknown sources.

## GitHub Actions CI

The repository includes two workflows:

- `.github/workflows/ci.yml` runs `go work sync`, formatting checks, `go test ./node/...`, and `go test -count=1 ./node/...` on pull requests, pushes to `main`/`master`, and manual dispatch.
- `.github/workflows/release-artifacts.yml` builds Windows/Linux binaries, runs Linux amd64 version and node/miner smoke tests, packages release archives, verifies `SHA256SUMS.txt`, and uploads `dist/releases/*` as workflow artifacts.

To run the release artifact workflow manually:

1. Open the GitHub Actions tab.
2. Select `Release Artifacts`.
3. Choose `Run workflow`.
4. Set `version`, for example `v0.4.10-testnet-rc1`.
5. Leave `skip_tests` disabled for normal release artifacts.

Tag pushes matching `v*` also build artifacts using the tag name as the version. The workflow uploads artifacts only; it does not publish a GitHub Release automatically and does not require secrets.

Downloaded workflow artifacts contain:

- `deskachain-v<version>-windows-amd64.zip`
- `deskachain-v<version>-linux-amd64.tar.gz`
- `deskachain-v<version>-linux-arm64.tar.gz`
- `SHA256SUMS.txt`

Artifacts intentionally exclude `testdata/`, runtime `data/`, wallet files, chain DBs, mempool/peer runtime stores, faucet/service state, private keys, `.git`, and intermediate `dist` directories.

## Smoke Scripts

Run a local release binary smoke test:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\smoke-test.ps1 -BinDir .\dist\windows-amd64
```

```sh
bash ./scripts/smoke-test.sh ./dist/linux-amd64
```

The smoke scripts create temporary datadirs, run `version`, initialize testnet, start a public-safe local node, mine one block with `dkcminer --once`, check `chain info`, `chain validate`, `/explorer/status`, `/explorer/blocks?limit=1`, and `/explorer-ui/`, then clean up unless `--keep-data` is supplied.

## Health Check Script

Check a running RC1 node or public seed RPC endpoint:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\testnet-health.ps1 -RpcUrl http://127.0.0.1:9311 -ExpectedNetwork testnet -ExpectedNetworkID dkc-testnet-1 -ExpectedChainID 777101
```

```sh
bash ./scripts/testnet-health.sh http://127.0.0.1:9311 --expected-network testnet --expected-network-id dkc-testnet-1 --expected-chain-id 777101
```

The script checks `/health`, `/explorer/status`, `/explorer/blocks?limit=1`, `/explorer-ui/`, expected network identity, public RPC safety flags, and optional CLI chain checks.

## Version Smoke

Windows:

```powershell
.\dist\windows-amd64\deskachain.exe version
.\dist\windows-amd64\dkcminer.exe --version
.\dist\windows-amd64\dkcservice.exe --version
```

Linux:

```sh
./dist/linux-amd64/deskachain version
./dist/linux-amd64/dkcminer --version
./dist/linux-amd64/dkcservice --version
```

Expected output includes version, commit, build date, Go version, OS/ARCH, supported networks `localnet,testnet`, and `mainnet: not available`.

## Windows Testnet Quickstart

Initialize and start a public-safe testnet node:

```powershell
.\deskachain.exe --datadir .\data\testnet --network testnet init
.\deskachain.exe --datadir .\data\testnet node start --rpc :9011 --p2p :10011 --advertise-p2p http://127.0.0.1:10011 --public-rpc --enable-miner-rpc --seed-file .\examples\testnet\testnet-seeds.txt
```

Create a miner wallet in a separate datadir:

```powershell
.\deskachain.exe --datadir .\data\miner --network testnet init
.\deskachain.exe --datadir .\data\miner wallet new
```

Mine once:

```powershell
.\dkcminer.exe --rpc-url http://127.0.0.1:9011 --address <ADDR> --threads 2 --once
```

Check the node:

```powershell
.\deskachain.exe --rpc-url http://127.0.0.1:9011 chain info
.\deskachain.exe --rpc-url http://127.0.0.1:9011 chain validate
```

Run service simulation once:

```powershell
.\dkcservice.exe --rpc-url http://127.0.0.1:9011 --address <ADDR> --endpoint http://127.0.0.1:9971 --once
```

## Linux Testnet Quickstart

```sh
./deskachain --datadir ./data/testnet --network testnet init
./deskachain --datadir ./data/testnet node start --rpc :9011 --p2p :10011 --advertise-p2p http://127.0.0.1:10011 --public-rpc --enable-miner-rpc --seed-file ./examples/testnet/testnet-seeds.txt
```

In another shell:

```sh
./deskachain --datadir ./data/miner --network testnet init
ADDR="$(./deskachain --datadir ./data/miner wallet new)"
./dkcminer --rpc-url http://127.0.0.1:9011 --address "$ADDR" --threads 2 --once
./deskachain --rpc-url http://127.0.0.1:9011 chain validate
```

## systemd

Example units and environment files are included under:

- `examples/systemd/deskachain-testnet.service`
- `examples/systemd/dkcservice-testnet.service`
- `examples/testnet/public-node.env`
- `examples/testnet/miner-node.env`
- `examples/testnet/faucet-node.env`

Review paths, ports, seed peers, and RPC flags before installing. Do not expose wallet/admin RPC publicly.

For restart checks, run `sudo systemctl restart deskachain-testnet`, then verify `/health`, `chain info`, and `chain validate`. The unit should stop with SIGINT and release the datadir lock.

## Safety Notes

- Testnet DKC has no monetary value.
- Mainnet is not available.
- Do not expose wallet or admin RPC to the public internet.
- Public nodes should use `--public-rpc`; enable miner RPC only when direct miner traffic is intended.
- Use a separate datadir for miner reward wallets when the node datadir is locked.
- Back up `wallets.json`.
- Seed peers are not trusted authorities. Network ID, chain ID, genesis hash, and protocol validation still apply.
- Do not treat testnet DKC as an investment.
- There are no mining income, staking APY, profit, or reward promises.
- Staking is collateral-only.
- Service points are simulation-only and not spendable.
- Testnet may reset.
