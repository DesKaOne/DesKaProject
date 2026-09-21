# Phase 12 — Local IndoChain Node / Client Validation

## Objective

Validate a real two-machine IndoChain testnet deployment before moving to private VPS infrastructure.

Target layout:

```
Mini PC Ubuntu
  ├── IndoChain node/server
  ├── RPC endpoint
  ├── P2P endpoint
  └── persistent datadir
          │
          ▼
       LAN
          │
          ▼
Main PC client
  ├── RPC/CLI client
  ├── wallet/read checks
  └── transaction verification
```

This drill uses the existing IndoChain **testnet** profile:

- Network: `testnet`
- Network ID: `ind-testnet-1`
- Chain ID: `777101`
- Testnet dIDR is experimental and has no monetary value.

Do not expose the node to the public Internet during this drill.

## 12.1 Mini PC Ubuntu preparation

Install the required base tools:

```sh
sudo apt update
sudo apt install -y git build-essential curl jq
go version
```

Clone the repository and enter the project:

```sh
git clone https://github.com/DesKaOne/DesKaProject.git
cd DesKaProject/deskachain/node
```

Verify the frozen Phase 11 merge is present:

```sh
git checkout main
git log -1 --oneline
```

Build the node and miner from source:

```sh
go build -o indochain ./cmd/indochain
go build -o indominer ./cmd/indominer
```

Verify binaries:

```sh
./indochain version
./indominer --help
```

Create a dedicated testnet datadir:

```sh
mkdir -p /var/lib/indochain/local-testnet
sudo chown -R "$USER":"$USER" /var/lib/indochain/local-testnet
```

Initialize the node:

```sh
./indochain --datadir /var/lib/indochain/local-testnet --network testnet init
```

Validate the initial chain:

```sh
./indochain --datadir /var/lib/indochain/local-testnet chain info
./indochain --datadir /var/lib/indochain/local-testnet chain validate
```

## 12.2 Choose the Mini PC LAN address

On the Mini PC:

```sh
ip -4 addr
hostname -I
```

Example only:

```text
Mini PC LAN IP = 192.168.1.50
```

Do not copy the example address into production configuration unless it is actually the Mini PC address.

The peer must reach:

- RPC: TCP `9311`
- P2P: TCP `10311`

Bind to the LAN interface or all local interfaces only as required for the trusted LAN drill.

## 12.3 Start the Mini PC node

Run:

```sh
./indochain \
  --datadir /var/lib/indochain/local-testnet \
  node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://192.168.1.50:10311
```

Replace `192.168.1.50` with the actual Mini PC LAN IP.

The node should expose:

```text
RPC  = http://<MINI_PC_IP>:9311
P2P  = http://<MINI_PC_IP>:10311
```

## 12.4 Verify the server locally

From the Mini PC:

```sh
curl http://127.0.0.1:9311/health
curl http://127.0.0.1:9311/network/info
curl http://127.0.0.1:9311/node/status
curl http://127.0.0.1:9311/chain/info
```

Also verify:

```sh
./indochain --rpc-url http://127.0.0.1:9311 chain validate
./indochain --rpc-url http://127.0.0.1:9311 peer list
```

Expected identity:

```text
network: testnet
network id: ind-testnet-1
chain id: 777101
```

## 12.5 Main PC connectivity test

From the main PC, first test TCP reachability to the Mini PC.

Windows PowerShell:

```powershell
Test-NetConnection <MINI_PC_IP> -Port 9311
Test-NetConnection <MINI_PC_IP> -Port 10311
```

Linux/macOS:

```sh
nc -vz <MINI_PC_IP> 9311
nc -vz <MINI_PC_IP> 10311
```

Then query the server RPC from the main PC:

```text
http://<MINI_PC_IP>:9311/health
http://<MINI_PC_IP>:9311/network/info
http://<MINI_PC_IP>:9311/chain/info
```

The main PC must observe the same:

```text
testnet / ind-testnet-1 / 777101
```

## 12.6 Main PC IndoChain client

Use the repository CLI from a working copy on the main PC:

```sh
go run ./node/cmd/indochain --rpc-url http://<MINI_PC_IP>:9311 node status
go run ./node/cmd/indochain --rpc-url http://<MINI_PC_IP>:9311 chain info
go run ./node/cmd/indochain --rpc-url http://<MINI_PC_IP>:9311 chain validate
go run ./node/cmd/indochain --rpc-url http://<MINI_PC_IP>:9311 peer list
```

Record:

- network identity
- chain ID
- chain height
- tip hash
- peer count
- validation result

## 12.7 P2P join from a second node

For a real two-node chain drill, create a second datadir on the main PC or another Linux host.

Initialize:

```sh
go run ./node/cmd/indochain \
  --datadir ./data/local-client-node \
  --network testnet init
```

Start:

```sh
go run ./node/cmd/indochain \
  --datadir ./data/local-client-node \
  node start \
  --rpc 0.0.0.0:9312 \
  --p2p 0.0.0.0:10312 \
  --advertise-p2p http://<MAIN_PC_REACHABLE_IP>:10312 \
  --seed-peer http://<MINI_PC_IP>:10311
```

Then from the main PC:

```sh
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:9312 peer list
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:9312 peer check http://<MINI_PC_IP>:10311
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:9312 peer sync http://<MINI_PC_IP>:10311
```

Validate both nodes:

```sh
go run ./node/cmd/indochain --rpc-url http://<MINI_PC_IP>:9311 chain validate
go run ./node/cmd/indochain --rpc-url http://127.0.0.1:9312 chain validate
```

Expected after successful synchronization:

- both nodes report the same network identity;
- both nodes report the same chain ID;
- both nodes converge on the same height and tip;
- `chain validate` passes on both.

## 12.8 Transaction drill

Create separate test wallets on a trusted local node only.

Example:

```sh
go run ./node/cmd/indochain --datadir ./data/test-wallet-a --network testnet init
go run ./node/cmd/indochain --datadir ./data/test-wallet-a wallet new

go run ./node/cmd/indochain --datadir ./data/test-wallet-b --network testnet init
go run ./node/cmd/indochain --datadir ./data/test-wallet-b wallet new
```

Use the existing controlled faucet/mining flow documented in `docs/DeployTestnet.md` and `docs/MultiHostTestnet.md`.

The transaction test should prove:

1. a normal testnet transaction can be created;
2. the transaction reaches the mempool;
3. a block confirms it;
4. both nodes observe the resulting canonical state;
5. transaction and chain validation succeed.

Do not copy private keys between independent nodes.

## 12.9 Restart and persistence drill

Stop the Mini PC node gracefully.

Restart it using the same datadir:

```sh
./indochain \
  --datadir /var/lib/indochain/local-testnet \
  node start \
  --rpc 0.0.0.0:9311 \
  --p2p 0.0.0.0:10311 \
  --advertise-p2p http://<MINI_PC_IP>:10311
```

Then verify:

```sh
curl http://127.0.0.1:9311/health
./indochain --rpc-url http://127.0.0.1:9311 node status
./indochain --rpc-url http://127.0.0.1:9311 chain info
./indochain --rpc-url http://127.0.0.1:9311 chain validate
./indochain --rpc-url http://127.0.0.1:9311 peer list --source
```

Check that:

- the same datadir is used;
- node identity persists;
- chain height/tip are preserved;
- peer metadata can recover;
- the main PC can reconnect.

## 12.10 LAN firewall boundary

On Ubuntu, allow only the trusted LAN subnet.

Example:

```sh
sudo ufw allow from <LAN_CIDR> to any port 10311 proto tcp
sudo ufw allow from <LAN_CIDR> to any port 9311 proto tcp
sudo ufw status
```

Do not forward these ports from the router to the public Internet during this drill.

Do not enable public wallet/admin RPC on the LAN deployment unless there is a dedicated controlled reason.

## 12.11 Operator evidence

For each drill record:

- date/time
- Mini PC hostname
- Mini PC LAN IP
- main PC hostname
- RPC port
- P2P port
- datadir path
- node ID fingerprint/identifier if available
- network ID
- chain ID
- genesis identity
- starting height/tip
- ending height/tip
- peer status
- `chain validate` result
- restart result
- transaction result

Never record private keys, seed phrases, or authentication secrets in the evidence.

## 12.12 Exit gate

Local deployment passes when all of the following are true:

- [ ] Mini PC node starts cleanly on testnet.
- [ ] Main PC reaches RPC over LAN.
- [ ] Main PC observes the expected network ID and chain ID.
- [ ] P2P connectivity succeeds.
- [ ] Nodes synchronize.
- [ ] Chain validation passes on both sides.
- [ ] Test transaction propagates and confirms.
- [ ] Restart preserves canonical state.
- [ ] Node identity persists.
- [ ] No public Internet exposure is required.

## Next phase

Only after this local drill passes should the project move to a controlled VPS private testnet.

VPS deployment should reuse the same testnet identity and configuration contract while making RPC/P2P exposure explicit and firewall-controlled.

Future product integrations should use environment/configuration values:

```
INDOCHAIN_RPC_URL
INDOSCAN_BASE_URL
WALLET_BACKEND_URL
DESKAPAY_API_URL
DESKACASH_API_URL
```
