# <img align="center" width=40 src="media/logo-white.jpg"> DOS Client and Core Libraries
[![Go Report Card](https://goreportcard.com/badge/github.com/DOSNetwork/core)](https://goreportcard.com/report/github.com/DOSNetwork/core)
[![Maintainability](https://api.codeclimate.com/v1/badges/a2eb5767f8984835fb3b/maintainability)](https://codeclimate.com/github/DOSNetwork/core/maintainability)
[![GoDoc](https://godoc.org/github.com/DOSNetwork/core?status.svg)](https://godoc.org/github.com/DOSNetwork/core)

## Development Setup:
- [Install](https://golang.org/doc/install) Go (recommended version 1.10+) and setup golang workingspace, specifically by adding environment variable [GOPATH](https://golang.org/doc/code.html#GOPATH) into PATH.
- Install [dep](https://golang.github.io/dep/docs/daily-dep.html#key-takeaways) to manage package dependencies and versions.
  - Run `$ dep ensure` to update missing dependencies/packages.
  - [Visualize package dependencies](https://golang.github.io/dep/docs/daily-dep.html#visualizing-dependencies)
- Download:
  - `$ go get -d github.com/DOSNetwork/core/...` or
  - `$ git clone git@github.com:DOSNetwork/core.git`
- Build:
  - `$ make` or `$ make client` to build release version client.
  - `$ make devClient` to build develoment version client.
  - `$ make updateSubmodule` to fetch latest system contracts from [repo](https://github.com/DOSNetwork/eth-contracts), instead of making contract modifications locally.
  - `$ make gen` to generate binding files for system contracts.
- Dev tips:
  - `$ go fmt ./...` to reformat go source code.
  - `$ golint` to fix style mistakes conflicting with [effective go](https://golang.org/doc/effective_go.html). ([golint](https://github.com/golang/lint) tool for vim users.)
  - `$ make clean` to remove built binaries or unnecessary generated files.



## Running a Beta DOS node on a cloud server or VPS:
### Requirements
- **Cloud Server / VPS Recommendations**
  - [Vultr](https://www.vultr.com/?ref=7806004-4F) - Cloud Compute $5 monthly plan (1CPU, 1GB Memory, 25GB SSD, 1TB Bandwidth)
  - [AWS Lightsail](https://aws.amazon.com/lightsail/pricing/?opdp1=pricing) - $5 monthly plan (1CPU, 1GB Memory, 40GB SSD, 2TB Bandwidth)
  - [Digital Ocean](https://m.do.co/c/a912bdc08b78) - Droplet $5 monthly plan (1CPU, 25GB SSD, 1TB Bandwidth)
  - [Linode](https://www.linode.com/?r=35c0c22d412b3fc8bd98b4c7c6f5ac42ae3bc2e2) - $5 monthly plan (1CPU, 1GB Memory, 25GB SSD, 1TB Bandwidth)
  - .

- **Verified and recommended installation environment**
  - Ubuntu 16.04 x64 LTS or higher 
  - A static IPv4 address with port `7946, 8545, 8546 and 9501` open
  - It's recommended to generate ssh login key pairs and setup public key authentication instead of using password login for server security and funds safety:
    - Learn [how to](https://www.digitalocean.com/community/tutorials/how-to-set-up-ssh-keys-on-ubuntu-1604) setup SSH public key authentication on Ubuntu 16.04 and disable password logins.


- **A node runner controlled keystore file with testnet ether and DOS token**
  - A [keystore file](https://medium.com/@julien.maffre/what-is-an-ethereum-keystore-file-86c8c5917b97) is generated by encrypting the raw private key with a user-specified password, to be used to sign transactions. It can be either created by [using geth](https://medium.com/@julien.maffre/what-is-an-ethereum-keystore-file-86c8c5917b97), or exported through [MEW Chrome Plugin](https://bitcointalk.org/index.php?topic=3014688.0), or generated by using [ethereumjs-wallet library](https://ethereum.stackexchange.com/questions/11166/how-to-generate-a-keystore-utc-file-from-the-raw-private-key).
  - Acquire testnet ether from rinkeby [faucet](https://faucet.rinkeby.io/).
  - Acquire 50,000 [testnet DOS token](https://rinkeby.etherscan.io/address/0x214e79c85744cd2ebbc64ddc0047131496871bee), (and optional - acquire several [testnet DropBurn token](https://rinkeby.etherscan.io/address/0x9bfe8f5749d90eb4049ad94cc4de9b6c4c31f822)).
  - Please fill in [this](https://docs.google.com/forms/d/e/1FAIpQLSe7Kf1RvGa2p5SjP4eGAp-fw2frauOl6CDORnHK0-TNbjho9w/viewform) form to request testnet tokens.



### Prepare the environment
- Download github repo: `$ git clone https://github.com/DOSNetwork/core.git`
- Config following fields in [`dos.setting`](https://github.com/DOSNetwork/core/blob/master/dos.setting) file:
  - `USER`: Username of the remote server/vps.
  - `IP`: Public ip address of the remote server/vps. 
  - `SSHKEY`: VPS ssh private key location
  - `KEYSTORE`: Path to the ethereum keystore file generated by user
  - `GETHPOOL`: Beta node runners may NOT need to modify this field. (User can add more infura endpoins and more geth full nodes here. (Infura endpoins are used to relay transactions and ws (web socket) of full nodes are only for event subscriptions.)
  - Example:
	```
	DOSIMAGE=dosnetwork/dosnode:beta
	GETHPOOL="https://rinkeby.infura.io/v3/<apikey>;ws://<ip-to-ethereum-rinkeby-fullnode>:8546"
	USER=<ubuntu>
	IP=<remote-server-ip>
	SSHKEY=/home/<ubuntu>/.ssh/<local-private-key-to-login-remote-server>
	KEYSTORE=<path-to-local-ethereum-keystore-file-generated-by-user>
	```

### Install and run client node using Docker
- Install and setup docker environment: 
  - `$ ./vps_docker.sh install`
- Start client node: 
  - `$ ./vps_docker.sh run`
- Stop client node: 
  - `$ ./vps_docker.sh stop`
- Check node status: 
  - `$ ./vps_docker.sh clientInfo`


### Build from source and run standalone binary
- `$ git checkout Beta1.1` to use source code of [latest release](https://github.com/DOSNetwork/core/releases/tag/Beta1) and follow [development-setup](#development-setup) to build #beta1.0 client node from scratch.
- You can also build from `master branch` which contains latest features/updates, but they might not be considered as release-ready.
- Install and upload node executable binary file to remote server: 
  - `$ ./vps.sh install`
- Start client node: 
  - `$ ./vps.sh run`
- Stop client node: 
  - `$ ./vps.sh stop`
- Check node status: 
  - `$ ./vps.sh clientInfo`



## Status
- ☑️ Secret Sharing
- ☑️ Distributed Key Generation (Pedersen's DKG approach)
- ☑️ Paring Library and BLS Signature
- ☑️ Distributed Randomness Engine with VRF
- ☑️ Gossip & DHT Implementation
- ☑️ P2P NAT Support
- ☑️ Json / Xml / Html Parser
- ☑️ Dockerize and Deployment Script
- ☑️ Integration with Ethereum On-chain [System Contracts](https://github.com/DOSNetwork/eth-contracts)
- :white_large_square: Enable geth lightnode mode and experiment with parity clients
- :white_large_square: P2P Network Performance Tuning
- :white_large_square: Network Status Scanner/Explorer
- :white_large_square: Staking & Delegation Contracts with a User-friendly Frontend
