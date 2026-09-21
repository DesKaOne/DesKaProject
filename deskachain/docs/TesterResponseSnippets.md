# IndoChain Tester Response Snippets

Testnet dIDR has no monetary value. Mainnet is not available. Do not paste private keys, wallet files, or secrets into issues.

## Ask For Version

Please run:

```powershell
.\indochain.exe version
```

Linux:

```sh
./indochain version
```

Paste the output, but do not include private keys or wallet files.

## Ask For Health

Please open or run:

```text
http://127.0.0.1:9311/health
```

PowerShell:

```powershell
Invoke-RestMethod http://127.0.0.1:9311/health
```

## Ask For Chain Info

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9311 chain info
```

Linux:

```sh
./indochain --rpc-url http://127.0.0.1:9311 chain info
```

## Ask For Peer List

```powershell
.\indochain.exe --rpc-url http://127.0.0.1:9311 peer list
```

Linux:

```sh
./indochain --rpc-url http://127.0.0.1:9311 peer list
```

## Ask For Logs

PowerShell terminal output is enough for most local runs. Please redact public IPs if you prefer and do not paste private keys.

Systemd:

```sh
journalctl -u indochain-testnet -n 200 --no-pager
```

## Private Key Warning

Please do not paste private keys, `wallets.json`, full datadirs, `.env` files with secrets, SSH keys, or tokens. If a wallet issue needs deeper debugging, share only redacted command output first.

## No Monetary Value

Testnet dIDR has no monetary value. It is only for testing network, wallet, explorer, faucet, mining, staking-collateral, and service-node simulation workflows. Mainnet is not available.

## Coinbase Maturity

Mining rewards are confirmed immediately but are not mature/spendable until the coinbase maturity window passes. On testnet RC1, that maturity is 100 blocks.

## Localnet/Testnet Mismatch

Please confirm the node reports:

```text
network: testnet
network_id: ind-testnet-1
chain_id: 777101
```

If the datadir was initialized for localnet, use a separate clean testnet datadir.

## Public RPC Safety

For public nodes, `/health` and `/explorer/status` should show:

```text
public_rpc: true
wallet_rpc: false
admin_rpc: false
```

Do not expose wallet/admin RPC publicly.

