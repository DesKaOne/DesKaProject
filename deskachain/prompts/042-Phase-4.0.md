Kamu sedang bekerja pada project Go monorepo DesKaChain.

Status saat ini:

* Phase 1 sampai Phase 3.9 sudah valid full.
* Phase 3.9 Public Testnet Genesis Candidate & Seed Node Deployment Prep valid full.
* Public testnet candidate sudah berhasil diuji di Mini PC Linux.
* Seed node A testnet start OK.
* Node B join via seed-peer OK.
* Peer list/check OK.
* Miner wallet testnet OK.
* Mining ke node A OK.
* Block broadcast/import ke node B OK.
* Node B chain info height 1 OK.
* Node B chain validate OK.
* CI dan release artifact workflow sudah hijau.
* Release binary Windows/Linux + SHA256SUMS valid.
* Testnet IDR has no monetary value.
* Mainnet does not exist yet.
* Staking remains collateral-only.
* Service points remain simulation-only.
* PoW remains the only block-production consensus.

Patch name:
DesKaChain Phase 4.0 — Public Testnet Multi-Host Deployment

Goal:
Prepare and validate DesKaChain public testnet deployment across multiple real hosts.

This phase should make it easy and safe to run:

* Mini PC as seed node,
* Windows PC as peer/miner node,
* optional VPS as public seed/relay node,
* optional Tailscale/LAN based testnet before exposing anything to the open internet.

Focus:

* multi-host deployment docs,
* network binding/advertise correctness,
* firewall checklist,
* peer diagnostics,
* systemd production-like unit examples,
* external node join flow,
* block propagation validation across hosts.

Non-goals:

* Do not launch mainnet.
* Do not change testnet genesis.
* Do not change localnet genesis.
* Do not change consensus.
* Do not implement explorer.
* Do not implement Stratum/pool mining.
* Do not implement staking rewards/APY.
* Do not make service points spendable.
* Do not promise profit/rewards.
* Do not expose wallet/admin RPC publicly.
* Do not require centralized seed trust.
* Do not auto-connect to unknown public seeds unless operator explicitly configures them.

==================================================

1. Multi-host deployment docs
   ==================================================

Add or update:

docs/MultiHostTestnet.md

This doc should explain three deployment modes:

1. LAN mode

   * Mini PC seed node using LAN IP.
   * Windows/Linux peer joins using LAN seed peer.

2. Tailscale mode

   * Mini PC seed node advertises Tailscale IP or MagicDNS name.
   * Other machines must be inside same Tailnet.
   * Good for private public-testnet rehearsal without opening router ports.

3. Public VPS mode

   * VPS runs public seed node.
   * P2P port open to internet.
   * RPC should be public read-only or private/firewalled.
   * Wallet/admin RPC must remain disabled.

Include safety notes:

* P2P port can be public.
* Public RPC must be read-only unless specific safe toggles are intended.
* Wallet/admin RPC must never be public.
* Testnet IDR has no monetary value.
* Mainnet is not available.
* Seed peers are not trusted authorities; network ID and genesis are validated.

==================================================
2. Binding vs advertise docs
============================

Make docs very clear about difference:

Bind address:

* where node listens locally.
* examples:

  * `--rpc :9311`
  * `--p2p :10311`
  * `--rpc 0.0.0.0:9311`
  * `--p2p 0.0.0.0:10311`

Advertise address:

* what other peers use to reach this node.
* examples:

  * LAN: `http://192.168.1.5:10311`
  * Tailscale: `http://100.101.251.7:10311`
  * Tailscale MagicDNS: `http://caca.tailxxxx.ts.net:10311`
  * VPS: `http://<VPS_PUBLIC_IP>:10311`

Add examples:

Mini PC seed over Tailscale:

./deskachain --datadir /var/lib/deskachain/testnet --network testnet init

./deskachain --datadir /var/lib/deskachain/testnet node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc

Windows peer joins:

.\deskachain.exe --datadir .\data\testnet --network testnet init

.\deskachain.exe --datadir .\data\testnet node start `    --rpc :9312`
--p2p :10312 `    --advertise-p2p http://100.83.159.107:10312`
--public-rpc `    --enable-miner-rpc`
--seed-peer http://100.101.251.7:10311/

VPS seed example:

./deskachain --datadir /var/lib/deskachain/testnet node start 
--rpc 127.0.0.1:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://<VPS_PUBLIC_IP>:10311 
--public-rpc=false

If current CLI does not support `--public-rpc=false`, document the correct existing behavior instead.

==================================================
3. Firewall checklist
=====================

Add section to docs:

Linux UFW LAN:

sudo ufw allow from 192.168.1.0/24 to any port 10311 proto tcp

Linux UFW Tailscale:

sudo ufw allow in on tailscale0 to any port 10311 proto tcp
sudo ufw allow in on tailscale0 to any port 9311 proto tcp

For safer setup:

* expose P2P 10311 to Tailnet.
* expose RPC 9311 only if needed.
* never expose wallet/admin RPC.

Public VPS:
sudo ufw allow 10311/tcp
sudo ufw deny 9311/tcp

or if public read-only RPC intentionally enabled:
sudo ufw allow from <trusted-ip> to any port 9311 proto tcp

Windows firewall:

* allow inbound TCP for P2P port.
* optionally allow RPC only on private network or Tailnet.
* document PowerShell example if practical.

==================================================
4. Peer diagnostics
===================

Improve or document peer diagnostics.

Existing commands:

* peer list
* peer check
* chain info
* chain validate

Add docs examples:

From peer node:

./deskachain --rpc-url http://127.0.0.1:9312 peer list
./deskachain --rpc-url http://127.0.0.1:9312 peer check http://100.101.251.7:10311
./deskachain --rpc-url http://127.0.0.1:9312 chain info
./deskachain --rpc-url http://127.0.0.1:9312 chain validate

Optional code improvement if simple:
Add command:

deskachain peer diagnose <peer-url>

or improve `peer check` output to include:

* reachability OK/fail
* network id
* chain id
* genesis hash
* height
* tip hash
* advertised address if peer provides it
* clear error on timeout/refused/wrong network/genesis mismatch

Do not over-engineer. If current peer check is enough, just document it.

==================================================
5. Multi-host seed file support
===============================

Add or update:

config/testnet-seeds.example.txt

Include examples commented out:

# LAN seed:

# http://192.168.1.5:10311

# Tailscale seed:

# http://100.101.251.7:10311

# http://caca.tailxxxx.ts.net:10311

# VPS seed:

# http://<VPS_PUBLIC_IP>:10311

Docs should show:

--seed-file ./config/testnet-seeds.txt

and:

--seed-peer http://100.101.251.7:10311/

Ensure seed file:

* ignores blank lines.
* ignores comments.
* normalizes/dedupes URLs.
* rejects invalid URLs.
* does not change genesis/network.

==================================================
6. Systemd unit for real seed node
==================================

Add or update:

examples/systemd/deskachain-testnet.service
examples/systemd/deskachain-testnet.env

Example env:

IDR_DATADIR=/var/lib/deskachain/testnet
IDR_NETWORK=testnet
IDR_RPC_ADDR=0.0.0.0:9311
IDR_P2P_ADDR=0.0.0.0:10311
IDR_ADVERTISE_P2P=http://100.101.251.7:10311
IDR_PUBLIC_RPC=true
IDR_ENABLE_MINER_RPC=true
IDR_ENABLE_FAUCET_RPC=false
IDR_ENABLE_SERVICE_RPC=false
IDR_SEED_PEERS=

Systemd unit:

* After=network-online.target
* Wants=network-online.target
* WorkingDirectory=/opt/deskachain
* EnvironmentFile=/etc/deskachain/testnet.env
* ExecStart should use env variables.
* Restart=always
* RestartSec=5
* LimitNOFILE=65535
* KillSignal=SIGINT
* TimeoutStopSec=30
* StandardOutput=journal
* StandardError=journal

Add install commands:
sudo useradd --system --home /var/lib/deskachain --shell /usr/sbin/nologin deskachain
sudo mkdir -p /opt/deskachain /var/lib/deskachain/testnet /etc/deskachain
sudo chown -R deskachain:deskachain /var/lib/deskachain
sudo systemctl daemon-reload
sudo systemctl enable --now deskachain-testnet
journalctl -u deskachain-testnet -f

==================================================
7. Multi-host manual validation plan
====================================

Add docs checklist for real host test.

Host A: Mini PC seed node

* IP examples:

  * LAN: 192.168.1.5
  * Tailscale: 100.101.251.7
* Start node with:

  * RPC: 0.0.0.0:9311
  * P2P: 0.0.0.0:10311
  * advertise: reachable host/IP
  * public rpc true
  * miner rpc true

Host B: Windows/Linux peer

* init clean datadir
* start node with seed-peer pointing to Host A
* check peer list
* check peer check
* mine on A or B
* verify opposite node imports block

Commands:

Host A:

./deskachain --datadir ./data/seed --network testnet init

./deskachain --datadir ./data/seed node start 
--rpc 0.0.0.0:9311 
--p2p 0.0.0.0:10311 
--advertise-p2p http://100.101.251.7:10311 
--public-rpc 
--enable-miner-rpc

Host B:

./deskachain --datadir ./data/node_b --network testnet init

./deskachain --datadir ./data/node_b node start 
--rpc 0.0.0.0:9312 
--p2p 0.0.0.0:10312 
--advertise-p2p http://100.83.159.107:10312 
--public-rpc 
--enable-miner-rpc 
--seed-peer http://100.101.251.7:10311/

Host B checks:

./deskachain --rpc-url http://127.0.0.1:9312 peer list
./deskachain --rpc-url http://127.0.0.1:9312 peer check http://100.101.251.7:10311

Mine from Host B to Host B or Host A:

./deskachain --datadir ./data/miner --network testnet init
ADDR=$(./deskachain --datadir ./data/miner wallet new)

./idrminer --rpc-url http://127.0.0.1:9312 --address "$ADDR" --threads 2 --once

Check Host A:

./deskachain --rpc-url http://127.0.0.1:9311 chain info
./deskachain --rpc-url http://127.0.0.1:9311 chain validate

Expected:

* both nodes same height.
* both nodes same tip hash.
* both nodes chain valid.
* peer list active.
* no wallet/admin public exposure.

==================================================
8. Optional Tailscale notes
===========================

Add docs section:

Tailscale deployment notes:

* every peer that uses Tailscale IP/MagicDNS must be inside the same Tailnet.
* advertise address can use Tailscale IPv4 or MagicDNS.
* if using UFW, allow tailscale0 interface.
* this is similar to a private overlay network, not open internet.
* good for testnet rehearsal before VPS/public seed.

Example:

Mini PC:
advertise-p2p http://100.101.251.7:10311

Windows:
seed-peer http://100.101.251.7:10311/

VPS:
seed-peer http://100.101.251.7:10311/ if VPS also joins Tailnet.

==================================================
9. Tests
========

Required:

go work sync
go test ./node/...
go test -count=1 ./node/...

Target tests:

go test ./node/internal/cli -run "Seed|Peer|Public|Advertise|Systemd|Deploy|MultiHost" -count=1 -v
go test ./node/internal/config -run "Seed|Profile|Testnet|Advertise|Env" -count=1 -v
go test ./node/internal/p2p -run "Seed|Peer|Advertise|NetworkMismatch|GenesisMismatch|Sync" -count=1 -v
go test ./node/internal/rpc -run "Public|Health|ChainInfo|Peer|Miner" -count=1 -v

If env/systemd docs are not testable, skip specific test names but keep full package tests passing.

==================================================
10. Manual validation
=====================

Validate at least one real multi-host mode:

Option A: LAN

* Mini PC seed: 192.168.1.5
* Windows peer: 192.168.1.x

Option B: Tailscale

* Mini PC seed: 100.101.251.7 or MagicDNS
* Windows/VPS peer: Tailnet IP/MagicDNS

Required manual proof:

1. Host A starts testnet seed node.
2. Host B starts with seed-peer to Host A.
3. Host B peer list shows Host A active.
4. Host B peer check Host A passes.
5. Mine one block on Host A or Host B.
6. Opposite host imports block.
7. Both hosts show:

   * same height
   * same tip hash
   * chain valid
8. Wallet/admin RPC remains disabled in public mode.
9. Restart one node and confirm peer persistence still works.

==================================================
11. Done criteria
=================

Phase 4.0 valid if:

* multi-host deployment docs exist.
* LAN/Tailscale/VPS modes documented.
* bind vs advertise is documented clearly.
* firewall checklist exists.
* systemd seed node example exists.
* seed file example supports LAN/Tailscale/VPS.
* all tests pass.
* at least one real multi-host manual test passes.
* two different hosts sync the same mined block.
* both nodes show same height and tip.
* public safety remains intact.
