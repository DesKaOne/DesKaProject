package p2p

import (
	"log"
	"time"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

type BroadcastSummary struct {
	Peers   int               `json:"peers"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Results []BroadcastResult `json:"results,omitempty"`
	Errors  []string          `json:"errors,omitempty"`
}

type BroadcastResult struct {
	Peer    string `json:"peer"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func BroadcastTx(peers []string, tx types.Transaction) BroadcastSummary {
	meta := make([]PeerMetadata, 0, len(peers))
	for _, peer := range peers {
		meta = append(meta, PeerMetadata{URL: peer, Status: PeerStatusActive})
	}
	return BroadcastTxToPeers("", meta, tx)
}

func BroadcastTxToPeers(peerStorePath string, peers []PeerMetadata, tx types.Transaction) BroadcastSummary {
	return broadcastTxToPeers(config.Paths{Peers: peerStorePath}, peers, tx, nil)
}

func BroadcastTxToPeersWithProfile(paths config.Paths, profile config.NetworkConfig, peers []PeerMetadata, tx types.Transaction) BroadcastSummary {
	return broadcastTxToPeers(paths, peers, tx, &profile)
}

func broadcastTxToPeers(paths config.Paths, peers []PeerMetadata, tx types.Transaction, profile *config.NetworkConfig) BroadcastSummary {
	client := NewClient()
	if profile != nil {
		var err error
		client, err = NewClientForProfile(paths, *profile, 5*time.Second)
		if err != nil {
			return BroadcastSummary{Peers: len(peers), Failed: len(peers), Errors: []string{err.Error()}}
		}
	}
	summary := BroadcastSummary{Peers: len(peers)}
	for _, peer := range peers {
		if peer.Status == PeerStatusBad {
			continue
		}
		response, err := client.BroadcastTx(peer.URL, tx)
		if err != nil {
			log.Printf("tx broadcast fail peer=%s tx=%s error=%v", peer.URL, tx.ID, err)
			summary.Failed++
			summary.Errors = append(summary.Errors, err.Error())
			summary.Results = append(summary.Results, BroadcastResult{Peer: peer.URL, OK: false, Message: err.Error()})
			adjustBroadcastScore(paths.Peers, peer.URL, -5, "request failed")
			continue
		}
		if !response.Accepted {
			log.Printf("tx broadcast fail peer=%s tx=%s error=%s", peer.URL, tx.ID, response.Error)
			summary.Failed++
			summary.Errors = append(summary.Errors, response.Error)
			summary.Results = append(summary.Results, BroadcastResult{Peer: peer.URL, OK: false, Message: response.Error})
			adjustBroadcastScore(paths.Peers, peer.URL, -30, "invalid tx")
			continue
		}
		summary.Success++
		summary.Results = append(summary.Results, BroadcastResult{Peer: peer.URL, OK: true, Message: "accepted"})
		adjustBroadcastScore(paths.Peers, peer.URL, 2, "broadcast tx")
		log.Printf("tx broadcast success peer=%s tx=%s", peer.URL, tx.ID)
	}
	return summary
}

func BroadcastBlock(peers []string, block types.Block) BroadcastSummary {
	meta := make([]PeerMetadata, 0, len(peers))
	for _, peer := range peers {
		meta = append(meta, PeerMetadata{URL: peer, Status: PeerStatusActive})
	}
	return BroadcastBlockToPeers("", meta, block)
}

func BroadcastBlockToPeers(peerStorePath string, peers []PeerMetadata, block types.Block) BroadcastSummary {
	return broadcastBlockToPeers(config.Paths{Peers: peerStorePath}, peers, block, nil)
}

func BroadcastBlockToPeersWithProfile(paths config.Paths, profile config.NetworkConfig, peers []PeerMetadata, block types.Block) BroadcastSummary {
	return broadcastBlockToPeers(paths, peers, block, &profile)
}

func broadcastBlockToPeers(paths config.Paths, peers []PeerMetadata, block types.Block, profile *config.NetworkConfig) BroadcastSummary {
	client := NewClientWithTimeout(3 * time.Second)
	if profile != nil {
		var err error
		client, err = NewClientForProfile(paths, *profile, 3*time.Second)
		if err != nil {
			return BroadcastSummary{Peers: len(peers), Failed: len(peers), Errors: []string{err.Error()}}
		}
	}
	summary := BroadcastSummary{Peers: len(peers)}
	for _, peer := range peers {
		if peer.Status == PeerStatusBad {
			continue
		}
		response, err := client.BroadcastBlock(peer.URL, block)
		if err != nil {
			log.Printf("block broadcast fail peer=%s height=%d error=%v", peer.URL, block.Height, err)
			summary.Failed++
			summary.Errors = append(summary.Errors, err.Error())
			summary.Results = append(summary.Results, BroadcastResult{Peer: peer.URL, OK: false, Message: err.Error()})
			adjustBroadcastScore(paths.Peers, peer.URL, -5, "request failed")
			continue
		}
		if !response.Accepted {
			log.Printf("block broadcast fail peer=%s height=%d error=%s", peer.URL, block.Height, response.Error)
			summary.Failed++
			summary.Errors = append(summary.Errors, response.Error)
			summary.Results = append(summary.Results, BroadcastResult{Peer: peer.URL, OK: false, Message: response.Error})
			adjustBroadcastScore(paths.Peers, peer.URL, -30, "invalid block")
			continue
		}
		summary.Success++
		summary.Results = append(summary.Results, BroadcastResult{Peer: peer.URL, OK: true, Message: "accepted"})
		adjustBroadcastScore(paths.Peers, peer.URL, 3, "broadcast block")
		log.Printf("block broadcast success peer=%s height=%d", peer.URL, block.Height)
	}
	return summary
}

func adjustBroadcastScore(path, peer string, delta int, reason string) {
	if path == "" {
		return
	}
	if err := NewPeerStore(path).AdjustPeerScore(peer, delta, reason); err != nil {
		log.Printf("peer score update failed peer=%s error=%v", peer, err)
	}
}

