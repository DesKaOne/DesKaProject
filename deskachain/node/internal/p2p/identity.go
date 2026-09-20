package p2p

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/types"
)

const (
	NodeIdentityVersion uint32 = 1
	nodeIdentityDomain         = "DesKaChain/p2p-handshake/v1"
)

type NodeIdentity struct {
	NodeID     string
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

func LoadOrCreateNodeID(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err == nil {
		id := string(raw)
		if id != "" {
			return id, nil
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	id := hex.EncodeToString(buf)
	return id, os.WriteFile(path, []byte(id), 0644)
}

func LoadOrCreateNodeIdentity(nodeIDPath string) (NodeIdentity, error) {
	nodeID, err := LoadOrCreateNodeID(nodeIDPath)
	if err != nil {
		return NodeIdentity{}, err
	}
	keyPath := nodeIDPath + ".ed25519"
	raw, err := os.ReadFile(keyPath)
	if err == nil {
		// Existing key files may have been created with broader permissions by older releases.
		// Tighten them on load so private key material remains owner-only on Unix-like systems.
		if chmodErr := os.Chmod(keyPath, 0600); chmodErr != nil && !errors.Is(chmodErr, os.ErrPermission) {
			return NodeIdentity{}, chmodErr
		}
		privateKeyRaw, decodeErr := hex.DecodeString(string(raw))
		if decodeErr != nil || len(privateKeyRaw) != ed25519.PrivateKeySize {
			return NodeIdentity{}, errors.New("invalid node identity private key")
		}
		privateKey := ed25519.PrivateKey(append([]byte(nil), privateKeyRaw...))
		publicKey := privateKey.Public().(ed25519.PublicKey)
		return NodeIdentity{
			NodeID:     nodeID,
			PublicKey:  append(ed25519.PublicKey(nil), publicKey...),
			PrivateKey: privateKey,
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return NodeIdentity{}, err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		return NodeIdentity{}, err
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return NodeIdentity{}, err
	}
	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(privateKey)), 0600); err != nil {
		return NodeIdentity{}, err
	}
	return NodeIdentity{NodeID: nodeID, PublicKey: publicKey, PrivateKey: privateKey}, nil
}

func (identity NodeIdentity) IdentityVersion() uint32 {
	if len(identity.PublicKey) == ed25519.PublicKeySize && len(identity.PrivateKey) == ed25519.PrivateKeySize {
		return NodeIdentityVersion
	}
	return 0
}

func handshakeSigningBytes(hs Handshake) []byte {
	var buf bytes.Buffer
	writeIdentityString(&buf, nodeIdentityDomain)
	writeIdentityUint32(&buf, hs.IdentityVersion)
	writeIdentityString(&buf, hs.NodeID)
	writeIdentityString(&buf, hs.NodePublicKey)
	writeIdentityString(&buf, hs.AuthChallenge)
	writeIdentityString(&buf, hs.NetworkID)
	writeIdentityUint64(&buf, hs.ChainID)
	writeIdentityString(&buf, hs.GenesisHash)
	writeIdentityUint32(&buf, hs.ProtocolVersion)
	writeIdentityString(&buf, hs.P2PProtocolVersion)
	writeIdentityUint32(&buf, hs.MinProtocolVersion)
	writeIdentityUint64(&buf, hs.Height)
	writeIdentityString(&buf, hs.TipHash)
	writeIdentityUint64(&buf, hs.CumulativeWork)
	writeIdentityString(&buf, hs.P2PListen)
	writeIdentityString(&buf, hs.P2PAdvertise)
	return buf.Bytes()
}

func SignHandshake(identity NodeIdentity, hs Handshake) (string, error) {
	if identity.IdentityVersion() != NodeIdentityVersion {
		return "", errors.New("unsupported node identity version")
	}
	hs.IdentityVersion = NodeIdentityVersion
	hs.NodeID = identity.NodeID
	hs.NodePublicKey = hex.EncodeToString(identity.PublicKey)
	signature := ed25519.Sign(identity.PrivateKey, handshakeSigningBytes(hs))
	return hex.EncodeToString(signature), nil
}

func VerifyHandshakeIdentity(hs Handshake) error {
	if hs.IdentityVersion != NodeIdentityVersion {
		return errors.New("peer rejected: unsupported node identity version")
	}
	publicKey, err := hex.DecodeString(hs.NodePublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return errors.New("peer rejected: invalid node identity public key")
	}
	signature, err := hex.DecodeString(hs.NodeSignature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("peer rejected: invalid node identity signature")
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), handshakeSigningBytes(hs), signature) {
		return errors.New("peer rejected: invalid node identity signature")
	}
	return nil
}

func ValidateHandshake(local config.NetworkConfig, peer Handshake) error {
	if peer.NetworkID != local.NetworkID {
		return errors.New("peer rejected: network id mismatch")
	}
	if peer.ChainID != local.ChainID {
		return errors.New("peer rejected: chain id mismatch")
	}
	expectedGenesis := chain.GenesisHashForNetwork(local)
	if peer.GenesisHash != expectedGenesis {
		return errors.New("peer rejected: genesis hash mismatch")
	}
	if peer.ProtocolVersion < local.MinProtocolVersion || local.ProtocolVersion < peer.MinProtocolVersion {
		return errors.New("peer rejected: incompatible protocol version")
	}
	if peer.IdentityVersion != 0 || peer.NodePublicKey != "" || peer.NodeSignature != "" {
		if err := VerifyHandshakeIdentity(peer); err != nil {
			return err
		}
	} else if local.RequireAuthenticatedNode {
		return errors.New("peer rejected: authenticated node identity required")
	}
	return nil
}

func ValidateStatus(local config.NetworkConfig, peer Status) error {
	if peer.NetworkID != local.NetworkID {
		return errors.New("peer rejected: network id mismatch")
	}
	if peer.ChainID != local.ChainID {
		return errors.New("peer rejected: chain id mismatch")
	}
	if peer.GenesisHash != local.GenesisHash {
		return errors.New("peer rejected: genesis hash mismatch")
	}
	if peer.ProtocolVersion < local.MinProtocolVersion {
		return errors.New("peer rejected: incompatible protocol version")
	}
	return nil
}

func HeaderFromBlock(block types.Block) BlockHeader {
	return BlockHeader{
		Version:      block.Version,
		Height:       block.Height,
		Hash:         block.Hash,
		PreviousHash: block.PreviousHash,
		Timestamp:    block.Timestamp,
		Difficulty:   block.Difficulty,
		MerkleRoot:   block.MerkleRoot,
		StateRoot:    block.StateRoot,
		TxCount:      len(block.Transactions),
		MinerAddress: block.MinerAddress,
	}
}

func writeIdentityString(buf *bytes.Buffer, value string) {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], uint32(len(value)))
	buf.Write(raw[:])
	buf.WriteString(value)
}

func writeIdentityUint32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}

func writeIdentityUint64(buf *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buf.Write(raw[:])
}
