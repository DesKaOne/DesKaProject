package p2p

import (
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
)

func CheckPeer(paths config.Paths, peer string) (Handshake, error) {
	return CheckPeerWithProfile(paths, peer, config.Localnet())
}

func CheckPeerWithProfile(paths config.Paths, peer string, profile config.NetworkConfig) (Handshake, error) {
	normalized, err := NormalizePeerURL(peer)
	if err != nil {
		return Handshake{}, err
	}
	peer = normalized
	if profile.Name == "" {
		profile = config.Localnet()
	}
	local := profile
	local.GenesisHash = chain.GenesisBlockForNetwork(profile).Hash
	start := time.Now()
	hs, err := NewClientWithTimeout(2 * time.Second).Handshake(peer)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		_ = NewPeerStore(paths.Peers).UpdateLatency(peer, latency, err.Error())
		_ = notePeerFailure(paths, peer, err.Error(), -5, "request failed")
		return Handshake{}, err
	}
	if err := ValidateHandshake(local, hs); err != nil {
		_ = NewPeerStore(paths.Peers).UpdateLatency(peer, latency, err.Error())
		_ = notePeerFailure(paths, peer, err.Error(), -20, "handshake mismatch")
		return Handshake{}, err
	}
	_ = NewPeerStore(paths.Peers).UpdateLatency(peer, latency, "")
	_ = notePeerSuccess(paths, peer, hs, 1, "peer status ok")
	return hs, nil
}
