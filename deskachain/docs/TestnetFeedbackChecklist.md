# Testnet Feedback Checklist

Use this checklist when reporting public testnet RC results.

## Node Sync

- Node started.
- Seed peer reachable.
- Peer list active.
- Chain info height matches seed.
- Chain validate passes.

## Mining

- Miner wallet created.
- Miner submits block.
- Explorer block list updates.
- Address page shows coinbase transaction.

## Explorer

- Dashboard loads.
- Blocks page works.
- Search works.
- Address page works.
- Transaction page works.
- Mobile view is usable.

## Faucet

- Request succeeds when faucet is provided.
- Rate limit works.
- Faucet transaction appears.
- No supply is minted beyond coinbase.

## Service

- Stake lock works.
- Service register works.
- `idrservice --once` works.
- Service eligibility is shown.
- Points are clearly simulation-only.

## Safety

- Wallet/admin RPC is not public.
- Testnet warnings are visible.
- Explorer UI has no write buttons.
- No private keys, wallet files, or secrets are shared in reports.
